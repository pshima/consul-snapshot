package restore

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/mholt/archiver"
	"github.com/pshima/consul-snapshot/backup"
	"github.com/pshima/consul-snapshot/config"
	"github.com/pshima/consul-snapshot/consul"
	"golang.org/x/crypto/scrypt"
)

// Restore is a struct to hold data about a single restore
type Restore struct {
	Config        *config.Config
	StartTime     int64
	JSONData      consulapi.KVPairs
	PQData        []*consulapi.PreparedQueryDefinition
	ACLData       []*consulapi.ACLEntry
	LocalFilePath string
	RestorePath   string
	RawData       []byte
	Encrypted     bool
	Meta          *backup.Meta
	ExtractedPath string
	Version       string
}

// Runner is the base level to start a restore and is called from command
func Runner(restorepath string) int {
	consulClient := &consul.Consul{Client: *consul.Client()}

	conf := config.ParseConfig(false)

	log.Printf("[DEBUG] Starting restore of %s/%s", conf.S3Bucket, restorepath)
	doWork(conf, consulClient, restorepath)
	return 0
}

// doWork this is the main function to start a restore
func doWork(conf *config.Config, c *consul.Consul, restorePath string) {
	restore := &Restore{}
	restore.StartTime = time.Now().Unix()
	restore.RestorePath = restorePath
	restore.Config = conf

	// if we are running an Acceptance test then we need to restore from local
	if conf.Acceptance {
		restore.LocalFilePath = fmt.Sprintf("%v/acceptancetest.gz", conf.TmpDir)
	} else {
		getRemoteBackup(restore, conf)
	}

	log.Print("[INFO] Checking encryption status of backup")
	restore.checkEncryption()

	log.Print("[INFO] Encrypted backup detected, decrypting")
	if restore.Encrypted {
		restore.decrypt()
	}

	log.Print("[INFO] Extracting backup")
	restore.extractBackup()

	log.Print("[INFO] Inspecting backup contents")
	restore.inspectBackup()

	// if during the backup inspection if we found it was v1 we
	// already have the kv data in the restore struct
	if restore.Version != "0.0.1" {
		log.Print("[INFO] Parsing KV Data")
		restore.loadKVData()
		log.Print("[INFO] Parsing PQ Data")
		restore.loadPQData()
		log.Print("[INFO] Parsing ACL Data")
		restore.loadACLData()
	}

	restoreKV(restore, c)
	restorePQs(restore, c)
	restoreACLs(restore, c)

	log.Print("[INFO] Restore completed.")

}

// checkEncryption peeks into the backup to see if it encrypted
// if it is, then we need to have the CRYPTO_PASSWORD env var
// or we cant restore it at all
func (r *Restore) checkEncryption() {
	backupData, err := ioutil.ReadFile(r.LocalFilePath)
	if err != nil {
		log.Fatalf("[ERR] Unable to read backupfile: %v", err)
	}
	// try and peek in to see if we have an encrypted backup
	if bytes.HasPrefix(backupData, []byte(r.Config.EncryptionPrefix)) {
		r.Encrypted = true
		if r.Config.Encryption == "" {
			log.Fatal("[ERR] No passphrase set and backup is encrypted, exiting")
		}
	} else {
		r.Encrypted = false
	}
}

// getRemoteBackup is used to pull backups from S3
func getRemoteBackup(r *Restore, conf *config.Config) {
	s3Conn := session.New(&aws.Config{Region: aws.String(string(conf.S3Region))})

	r.LocalFilePath = fmt.Sprintf("%v/%v", conf.TmpDir, r.RestorePath)

	localFileDir := filepath.Dir(r.LocalFilePath)

	err := os.MkdirAll(localFileDir, 0755)
	if err != nil {
		log.Fatalf("[ERR] Unable to create local restore directory!: %v", err)
	}

	outFile, err := os.Create(r.LocalFilePath)
	if err != nil {
		log.Fatalf("[ERR] Unable to create local restore temp file!: %v", err)
	}

	// Create the params to pass into the actual downloader
	params := &s3.GetObjectInput{
		Bucket: &conf.S3Bucket,
		Key:    &r.RestorePath,
	}

	log.Printf("[INFO] Downloading %v%v from S3 in %v", string(conf.S3Bucket), r.RestorePath, string(conf.S3Region))
	downloader := s3manager.NewDownloader(s3Conn)
	_, err = downloader.Download(outFile, params)
	if err != nil {
		log.Fatalf("[ERR] Could not download file from S3!: %v", err)
	}
	outFile.Close()
	log.Print("[INFO] Download completed")
}

// decrypt is used to read a file, decrypt it, and write it back to the same path
func (r *Restore) decrypt() {
	ciphertext, err := ioutil.ReadFile(r.LocalFilePath)
	if err != nil {
		log.Fatalf("[ERR] Unable to read backupfile: %v", err)
	}
	ciphertext = ciphertext[len(r.Config.EncryptionPrefix):]
	salt := ciphertext[:r.Config.EncryptionSaltLen]
	ciphertext = ciphertext[r.Config.EncryptionSaltLen:]

	key, err := scrypt.Key([]byte(r.Config.Encryption), salt, 16384, 8, 1, r.Config.EncryptionSaltLen)
	if err != nil {
		log.Fatalf("[ERR] Unable to generate scrypt key: %v", err)
	}

	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("[ERR] Unable to generate aes cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		log.Fatalf("[ERR] Unable to create GCM: %v", err)
	}

	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	output, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Fatalf("[ERR] Unable to decrypt data (possible bad CRYPTO_PASSWORD: %v", err)
	}

	if err := ioutil.WriteFile(r.LocalFilePath, output, os.FileMode(0644)); err != nil {
		log.Fatalf("Error decrypting file to %s: %v", r.LocalFilePath, err)
	}
}

// extractBackup uses archiver to extract the backup locally.
func (r *Restore) extractBackup() {
	dest := filepath.Dir(r.LocalFilePath)
	archiver.UntarGz(r.LocalFilePath, dest)
}

// parsev1data is used if we have detected the backup has no metadata
// then it is likely a v1 style backup, so open it, gunzip it, and
// then unmarshall the contents and return them as kvpairs.
// v1 backups only contained kvpairs.
func parsev1data(path string) (consulapi.KVPairs, error) {
	handle, err := os.Open(path)
	if err != nil {
		log.Fatalf("[ERR] Could not open local gzipped file: %v", err)
	}

	// Create a new gzip writer
	gz, err := gzip.NewReader(handle)
	if err != nil {
		log.Fatalf("[ERR] Could not read local gzipped file: %v", err)
	}

	outData := new(bytes.Buffer)
	_, err = io.Copy(outData, gz)

	bytestosend := outData.Bytes()

	kvpairs := consulapi.KVPairs{}
	if err := json.Unmarshal(bytestosend, &kvpairs); err != nil {
		return nil, err
	}
	totalKeys := len(kvpairs)
	log.Printf("[INFO] Extracted %v keys to restore", totalKeys)
	return kvpairs, nil
}

// inspectBackup takes a look at the metadata of the backup and
// tries to determine more information about it from the meta.
// if we find its a v1 backup, just process it and return
func (r *Restore) inspectBackup() {
	// first we need to fix the pathing to the extracted location
	var extractedpath string
	extractedpath = strings.Replace(r.LocalFilePath, ".tar.gz", "", 1)
	extractedpath = strings.Replace(extractedpath, ".gz", "", 1)
	r.ExtractedPath = extractedpath

	metaPath := filepath.Join(r.ExtractedPath, "meta.json")
	metaData, err := ioutil.ReadFile(metaPath)
	if err != nil {
		log.Print("[INFO] No meta file found, assuming 0.1.x backup")
		r.JSONData, err = parsev1data(r.LocalFilePath)
		r.Version = "0.0.1"
		if err != nil {
			log.Fatalf("[ERR] Failed to parse v1 data, possible bad backup file: %v", err)
		}
		return
	}

	metaExtract := &backup.Meta{}

	if err := json.Unmarshal(metaData, metaExtract); err != nil {
		log.Fatalf("[ERR] Unable to unmarshal metadata: %v", err)
	}

	log.Printf("[INFO] Found valid metadata of snapshot version %v with unix_timestamp %v",
		metaExtract.ConsulSnapshotVersion, metaExtract.StartTime)

	r.Version = metaExtract.ConsulSnapshotVersion
	r.Meta = metaExtract
}

// loadKVData loads data from an uncompressed kv backup file into an object
func (r *Restore) loadKVData() {
	startstring := fmt.Sprintf("%v", r.Meta.StartTime)
	kvFileName := fmt.Sprintf("consul.kv.%s.json", startstring)
	kvPath := filepath.Join(r.ExtractedPath, kvFileName)
	kvData, err := ioutil.ReadFile(kvPath)
	if err != nil {
		log.Fatalf("[ERR] Unable to read kv backup file at %s: %v", kvPath, err)
	}

	if err := json.Unmarshal(kvData, &r.JSONData); err != nil {
		log.Fatalf("[ERR] Unable to unmarshal kv data: %v", err)
	}
	log.Printf("[INFO] Loaded %v keys to restore", len(r.JSONData))
}

// loadPQData loads data from an uncompressed PQ backup file into an object
func (r *Restore) loadPQData() {
	startstring := fmt.Sprintf("%v", r.Meta.StartTime)
	pqFileName := fmt.Sprintf("consul.pq.%s.json", startstring)
	pqPath := filepath.Join(r.ExtractedPath, pqFileName)
	pqData, err := ioutil.ReadFile(pqPath)
	if err != nil {
		log.Fatalf("[ERR] Unable to read pq backup file at %s: %v", pqPath, err)
	}

	if err := json.Unmarshal(pqData, &r.PQData); err != nil {
		log.Fatalf("[ERR] Unable to unmarshal pq data: %v", err)
	}
	log.Printf("[INFO] Loaded %v Prepared Queries to restore", len(r.PQData))
}

// loadACLData loads data from an uncompressed ACL backup file into an object
func (r *Restore) loadACLData() {
	startstring := fmt.Sprintf("%v", r.Meta.StartTime)
	aclFileName := fmt.Sprintf("consul.acl.%s.json", startstring)
	aclPath := filepath.Join(r.ExtractedPath, aclFileName)
	aclData, err := ioutil.ReadFile(aclPath)
	if err != nil {
		log.Fatalf("[ERR] Unable to read acl backup file at %s: %v", aclPath, err)
	}

	if err := json.Unmarshal(aclData, &r.ACLData); err != nil {
		log.Fatalf("[ERR] Unable to unmarshal kv data: %v", err)
	}
	log.Printf("[INFO] Loaded %v ACLs to restore", len(r.ACLData))
}

// restoreKV takes the restored kv data and puts it back in to consul
func restoreKV(r *Restore, c *consul.Consul) {
	restoredKeyCount := 0
	errorCount := 0
	for _, data := range r.JSONData {
		_, err := c.Client.KV().Put(data, nil)
		if err != nil {
			errorCount++
			log.Printf("Unable to restore key: %s, %v", data.Key, err)
		}
		restoredKeyCount++
	}
	log.Printf("[INFO] Restored %v keys with %v errors", restoredKeyCount, errorCount)
}

// This needs a bit more testing before we can do PQ restores
func restorePQs(r *Restore, c *consul.Consul) {
	log.Println("[WARN] PQ restoration currently unsupported")
}

// This needs a bit more testing before we can do ACL restores
func restoreACLs(r *Restore, c *consul.Consul) {
	log.Println("[WARN] ACL restoration currently unsupported")
}
