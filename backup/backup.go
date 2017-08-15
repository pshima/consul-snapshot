package backup

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/mholt/archiver"
	"github.com/pshima/consul-snapshot/config"
	"github.com/pshima/consul-snapshot/consul"
	"github.com/pshima/consul-snapshot/crypt"
	"github.com/pshima/consul-snapshot/health"
)

// Backup is the backup itself including configuration and data
type Backup struct {
	ACLFileChecksum  string
	ACLJSONData      []byte
	Client           *consul.Consul
	Config           *config.Config
	FullFilename     string
	KVFileChecksum   string
	KVJSONData       []byte
	LocalACLFileName string
	LocalFilePath    string
	LocalKVFileName  string
	LocalPQFileName  string
	PQFileChecksum   string
	PQJSONData       []byte
	RemoteFilePath   string
	StartTime        int64
}

// Meta holds the meta struct to write inside the compressed data
type Meta struct {
	ACLSha256             string
	ConsulSnapshotVersion string
	EndTime               int64
	KVSha256              string
	NodeName              string
	PQSha256              string
	StartTime             int64
}

func calcSha256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	calc := sha256.New()
	_, err = io.Copy(calc, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(calc.Sum(nil)), nil
}

// Runner is the main runner for a backup
func Runner(version string, once bool) int {

	conf := config.ParseConfig(false)
	conf.Version = version
	client := &consul.Consul{Client: *consul.Client()}

	if once {
		err := doWork(conf, client)
		if err != nil {
			return 1
		}
	} else {
		// Start up the http server health checks, only needed for daemon-mode
		go health.StartServer()

		log.Printf("[DEBUG] Backup starting on interval: %v", conf.BackupInterval)
		ticker := time.NewTicker(conf.BackupInterval)
		for range ticker.C {
			err := doWork(conf, client)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
	}

	return 0
}

func doWork(conf *config.Config, client *consul.Consul) error {

	b := &Backup{
		Config: conf,
		Client: client,
	}

	// Loop over and over at interval time.
	b.StartTime = time.Now().Unix()

	startString := fmt.Sprintf("%v", b.StartTime)

	log.Printf("[INFO] Starting Backup At: %s", startString)

	log.Print("[INFO] Listing keys from consul")
	b.Client.ListKeys()
	log.Printf("[INFO] Converting %v keys to JSON", b.Client.KeyDataLen)
	b.KeysToJSON()

	log.Print("[INFO] Listing Prepared Queries from consul")
	b.Client.ListPQs()
	log.Printf("[INFO] Converting %v keys to JSON", b.Client.PQDataLen)
	b.PQsToJSON()

	log.Print("[INFO] Listing ACLs from consul")
	b.Client.ListACLs()
	log.Printf("[INFO] Converting %v ACLs to JSON", b.Client.ACLDataLen)
	b.ACLsToJSON()

	log.Print("[INFO] Preparing temporary directory for backup staging")
	b.preProcess()

	log.Print("[INFO] Writing KVs to local backup file")
	if err := writeFileLocal(b.LocalFilePath, b.LocalKVFileName, b.KVJSONData); err != nil {
		return fmt.Errorf("[ERR] Unable to write file %s/%s: %v", b.LocalFilePath, b.LocalKVFileName, err)
	}

	kvchecksum, err := calcSha256(filepath.Join(b.LocalFilePath, b.LocalKVFileName))
	if err != nil {
		return fmt.Errorf("[ERR] to generate checksum for file %s: %v", b.LocalKVFileName, err)
	}
	b.KVFileChecksum = kvchecksum

	log.Print("[INFO] Writing PQs to local backup file")
	if err := writeFileLocal(b.LocalFilePath, b.LocalPQFileName, b.PQJSONData); err != nil {
		return fmt.Errorf("[ERR] Unable to write file %s/%s: %v", b.LocalFilePath, b.LocalPQFileName, err)
	}

	pqchecksum, err := calcSha256(filepath.Join(b.LocalFilePath, b.LocalPQFileName))
	if err != nil {
		return fmt.Errorf("Unable to generate checksum for file %s: %v", b.LocalPQFileName, err)
	}
	b.PQFileChecksum = pqchecksum

	log.Print("[INFO] Writing ACLs to local backup file")
	if err := writeFileLocal(b.LocalFilePath, b.LocalACLFileName, b.ACLJSONData); err != nil {
		return fmt.Errorf("[ERR] Unable to write file %s/%s: %v", b.LocalFilePath, b.LocalACLFileName, err)
	}

	aclchecksum, err := calcSha256(filepath.Join(b.LocalFilePath, b.LocalACLFileName))
	if err != nil {
		return fmt.Errorf("[ERR] Unable to generate checksum for file %s: %v", b.LocalACLFileName, err)
	}
	b.ACLFileChecksum = aclchecksum

	b.writeMetaLocal()
	b.compressStagedBackup()

	if b.Config.Encryption != "" {
		crypt.EncryptFile(b.LocalFilePath, b.Config.Encryption)
	}

	if conf.Acceptance {
		log.Print("[INFO] Skipping remote backup during testing")
		log.Print("[INFO] Skipping post processing during testing")
	} else {
		log.Print("[INFO] Writing Backup to Remote File")
		b.writeBackupRemote()
		log.Print("[INFO] Running post processing")
		b.postProcess()
	}

	log.Print("[INFO] Backup completed successfully")
	return nil
}

// KeysToJSON used to marshall the data and put it on a Backup object
func (b *Backup) KeysToJSON() {
	jsonData, err := json.Marshal(b.Client.KeyData)
	if err != nil {
		log.Fatalf("[ERR] Could not encode keys to json!: %v", err)
	}
	b.KVJSONData = jsonData
}

// PQsToJSON used to marshall the data and put it on a Backup object
func (b *Backup) PQsToJSON() {
	jsonData, err := json.Marshal(b.Client.PQData)
	if err != nil {
		log.Fatalf("[ERR] Could not encode keys to json!: %v", err)
	}
	b.PQJSONData = jsonData
}

// ACLsToJSON used to marshall the data and put it on a Backup object
func (b *Backup) ACLsToJSON() {
	jsonData, err := json.Marshal(b.Client.ACLData)
	if err != nil {
		log.Fatalf("[ERR] Could not encode keys to json!: %v", err)
	}
	b.ACLJSONData = jsonData
}

// preProcess is used to prepare the backup temp location
func (b *Backup) preProcess() {
	startString := fmt.Sprintf("%v", b.StartTime)
	var prefix string
	if b.Config.Acceptance {
		prefix = "acceptancetest"
	} else {
		prefix = fmt.Sprintf("%s.consul.snapshot.%s", b.Config.Hostname, startString)
	}

	dir := filepath.Join(b.Config.TmpDir, prefix)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		log.Fatalf("Unable to create tmpdir %s: %v", b.Config.TmpDir, err)
	}

	b.LocalKVFileName = fmt.Sprintf("consul.kv.%s.json", startString)
	b.LocalPQFileName = fmt.Sprintf("consul.pq.%s.json", startString)
	b.LocalACLFileName = fmt.Sprintf("consul.acl.%s.json", startString)

	b.LocalFilePath = dir
}

// writeMetaLocal is used to write metadata about the backup into the
// tarball for further inspection later, such as consul-snapshot rev
func (b *Backup) writeMetaLocal() {
	endTime := time.Now().Unix()

	nodename, err := b.Client.Client.Agent().NodeName()
	if err != nil {
		nodename = ""
	}

	meta := &Meta{
		KVSha256:              b.KVFileChecksum,
		PQSha256:              b.PQFileChecksum,
		ACLSha256:             b.ACLFileChecksum,
		ConsulSnapshotVersion: b.Config.Version,
		StartTime:             b.StartTime,
		EndTime:               endTime,
		NodeName:              nodename,
	}

	metajsonData, err := json.Marshal(meta)
	if err != nil {
		log.Fatalf("[ERR] Could not encode meta to json!: %v", err)
	}

	if err := writeFileLocal(b.LocalFilePath, "meta.json", metajsonData); err != nil {
		log.Fatalf("[ERR] Could not write meta to local dir: %v", err)
	}
}

// writeFilesLocal writes the kv, pq and acl files locally
func writeFileLocal(path string, filename string, contents []byte) error {
	writepath := filepath.Join(path, filename)

	// Write the json to a gzip
	handle, err := os.OpenFile(writepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Could not open local file for writing!: %v", err)
	}
	defer handle.Close()

	// Actually write the json to the file
	bytesWritten, err := handle.Write(contents)
	if err != nil {
		return fmt.Errorf("Could not write data to file!: %v", err)
	}

	log.Printf("[DEBUG] Wrote %v bytes to file, %v", bytesWritten, writepath)
	return nil
}

func (b *Backup) compressStagedBackup() {
	startString := fmt.Sprintf("%v", b.StartTime)
	var finalfile string
	if b.Config.Acceptance {
		finalfile = "acceptancetest.tar.gz"
	} else {
		finalfile = fmt.Sprintf("%s.consul.snapshot.%s.tar.gz", b.Config.Hostname, startString)
	}
	finalpath := filepath.Join(b.Config.TmpDir, finalfile)
	b.FullFilename = finalpath
	source := []string{b.LocalFilePath}
	err := archiver.TarGz(finalpath, source)
	if err != nil {
		log.Fatalf("[ERR] Unable to write compressed archive to %s: %v", finalpath, err)
	}
}

// Write the local backup file to S3.
// There are no tests for this remote operation
func (b *Backup) writeBackupRemoteS3(localFileContents []byte) {
	s3Conn := session.New(&aws.Config{Region: aws.String(string(b.Config.S3Region))})
	// Create the params to pass into the actual uploader
	params := &s3manager.UploadInput{
		Bucket: &b.Config.S3Bucket,
		Key:    &b.RemoteFilePath,
		Body:   bytes.NewReader(localFileContents),
	}

	if b.Config.S3ServerSideEncryption != "" {
		params.ServerSideEncryption = &b.Config.S3ServerSideEncryption
	}

	if b.Config.S3KmsKeyID != "" {
		params.SSEKMSKeyId = &b.Config.S3KmsKeyID
	}

	log.Printf("[INFO] Uploading %v/%v to S3 in %v", string(b.Config.S3Bucket), b.RemoteFilePath, string(b.Config.S3Region))
	uploader := s3manager.NewUploader(s3Conn)
	_, err := uploader.Upload(params)
	if err != nil {
		log.Fatalf("[ERR] Could not upload to S3!: %v", err)
	}
}

// Write the local backup file to Google Cloud Storage.
// There are no tests for this remote operation
func (b *Backup) writeBackupRemoteGoogleStorage(localFileContents []byte) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("[ERR] Could not initialize connection with Google Cloud Storage!: %v", err)
	}
	wc := client.Bucket(b.Config.GCSBucket).Object(b.RemoteFilePath).NewWriter(ctx)
	log.Printf("[INFO] Uploading %v/%v to GCS", string(b.Config.GCSBucket), b.RemoteFilePath)
	wc.ContentType = "text/plain"
	// wc.ACL = []storage.ACLRule{{AllUsers: storage.AllUsers, RoleReader: storage.RoleReader}}
	if _, err := wc.Write(localFileContents); err != nil {
		log.Fatalf("[ERR] Could not upload to GCS!: %v", err)
	}
	if err := wc.Close(); err != nil {
		log.Fatalf("[ERR] Could not upload to GCS!: %v", err)
	}
}

func (b *Backup) writeBackupRemote() {
	t := time.Unix(b.StartTime, 0)

	b.RemoteFilePath = fmt.Sprintf("%s/%v/%d/%v/%v", b.Config.ObjectPrefix, t.Year(), t.Month(), t.Day(), filepath.Base(b.FullFilename))

	// re-read the compressed file.  There is probably a better way to do this
	localFileContents, err := ioutil.ReadFile(b.FullFilename)
	if err != nil {
		log.Fatalf("[ERR] Could not read compressed file!: %v", err)
	}

	if len(b.Config.S3Bucket) > 1 {
		b.writeBackupRemoteS3(localFileContents)
	}

	if len(b.Config.GCSBucket) > 1 {
		b.writeBackupRemoteGoogleStorage(localFileContents)
	}
}

// Run post processing on the backup, acking the key and removing and temp files.
// There are no tests for the remote operation.
func (b *Backup) postProcess() {
	// Mark a key in consul for our last backup time.
	writeOpt := &consulapi.WriteOptions{}
	startstring := fmt.Sprintf("%v", b.StartTime)

	var err error

	lastbackup := &consulapi.KVPair{Key: "service/consul-snapshot/lastbackup", Value: []byte(startstring)}
	_, err = b.Client.Client.KV().Put(lastbackup, writeOpt)
	if err != nil {
		log.Fatalf("[ERR] Failed writing last backup timestamp to consul: %v", err)
	}

	// Remove the compressed archive
	err = os.Remove(b.FullFilename)
	if err != nil {
		log.Printf("Unable to remove temporary backup file: %v", err)
	}

	// Remove the staging path
	err = os.RemoveAll(b.LocalFilePath)
	if err != nil {
		log.Printf("Unable to remove temporary backup file: %v", err)
	}

}
