package backup

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/mholt/archiver"
	"github.com/pshima/consul-snapshot/config"
	"github.com/pshima/consul-snapshot/consul"
	"github.com/pshima/consul-snapshot/health"
)

// Backup is the backup itself including configuration and data
type Backup struct {
	StartTime        int64
	EndTime          int64
	KVJSONData       []byte
	LocalKVFileName  string
	KVFileChecksum   []byte
	PQJSONData       []byte
	LocalPQFileName  string
	PQFileChecksum   []byte
	ACLJSONData      []byte
	LocalACLFileName string
	ACLFileChecksum  []byte
	LocalFilePath    string
	RemoteFilePath   string
	Config           config.Config
	Client           *consul.Consul
	FullFilename     string
}

// backupMeta holds the meta struct to write inside the compressed data
type backupMeta struct {
	ConsulSnapshotVersion string
	KVChecksum            []byte
	PQChecksum            []byte
	ACLChecksum           []byte
	StartTime             int64
	EndTime               int64
}

func calcMD5(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	calc := md5.New()
	_, err = io.Copy(calc, file)
	if err != nil {
		return nil, err
	}

	return calc.Sum(nil), nil
}

// Runner is the main runner for a backup
func Runner(version string) int {
	// Start up the http server health checks
	go health.StartServer()

	conf := config.ParseConfig(false)
	conf.Version = version
	client := &consul.Consul{Client: *consul.Client()}

	if conf.Acceptance {
		doWork(conf, client)
	} else {
		log.Printf("[DEBUG] Backup starting on interval: %v", conf.BackupInterval)
		ticker := time.NewTicker(conf.BackupInterval)
		for range ticker.C {
			doWork(conf, client)
		}
	}
	return 0
}

func doWork(conf config.Config, client *consul.Consul) {

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
		log.Fatalf("[ERR] Unable to write file %s/%s: %v", b.LocalFilePath, b.LocalKVFileName, err)
	}

	kvchecksum, err := calcMD5(filepath.Join(b.LocalFilePath, b.LocalKVFileName))
	if err != nil {
		log.Fatalf("[ERR] to generate checksum for file %s: %v", b.LocalKVFileName, err)
	}
	b.KVFileChecksum = kvchecksum

	log.Print("[INFO] Writing PQs to local backup file")
	if err := writeFileLocal(b.LocalFilePath, b.LocalPQFileName, b.PQJSONData); err != nil {
		log.Fatalf("[ERR] Unable to write file %s/%s: %v", b.LocalFilePath, b.LocalPQFileName, err)
	}

	pqchecksum, err := calcMD5(filepath.Join(b.LocalFilePath, b.LocalPQFileName))
	if err != nil {
		log.Fatalf("Unable to generate checksum for file %s: %v", b.LocalPQFileName, err)
	}
	b.PQFileChecksum = pqchecksum

	log.Print("[INFO] Writing ACLs to local backup file")
	if err := writeFileLocal(b.LocalFilePath, b.LocalACLFileName, b.ACLJSONData); err != nil {
		log.Fatalf("[ERR] Unable to write file %s/%s: %v", b.LocalFilePath, b.LocalACLFileName, err)
	}

	aclchecksum, err := calcMD5(filepath.Join(b.LocalFilePath, b.LocalACLFileName))
	if err != nil {
		log.Fatalf("[ERR] Unable to generate checksum for file %s: %v", b.LocalACLFileName, err)
	}
	b.ACLFileChecksum = aclchecksum

	b.writeMetaLocal()
	b.compressStagedBackup()

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
	dir := filepath.Join(b.Config.TmpDir, "consul-snapshot", startString)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		log.Fatalf("Unable to create tmpdir %s: %v", b.Config.TmpDir, err)
	}

	b.LocalKVFileName = fmt.Sprintf("consul.kv.%s.gz", startString)
	b.LocalPQFileName = fmt.Sprintf("consul.pq.%s.gz", startString)
	b.LocalACLFileName = fmt.Sprintf("consul.acl.%s.gz", startString)

	b.LocalFilePath = dir

}

// writeMetaLocal is used to write metadata about the backup into the
// tarball for further inspection later, such as consul-snapshot rev
func (b *Backup) writeMetaLocal() {
	meta := &backupMeta{
		KVChecksum:            b.KVFileChecksum,
		PQChecksum:            b.PQFileChecksum,
		ACLChecksum:           b.ACLFileChecksum,
		ConsulSnapshotVersion: b.Config.Version,
		StartTime:             b.StartTime,
		EndTime:               b.EndTime,
	}

	metajsonData, err := json.Marshal(meta)
	if err != nil {
		log.Fatalf("[ERR] Could not encode meta to json!: %v", err)
	}

	if err := writeFileLocal(b.LocalFilePath, "meta", metajsonData); err != nil {
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

	// Create a new gzip writer
	gz := gzip.NewWriter(handle)
	defer gz.Close()

	// Actually write the json to the file
	bytesWritten, err := gz.Write([]byte(contents))
	if err != nil {
		return fmt.Errorf("Could not write data to file!: %v", err)
	}

	log.Printf("[DEBUG] Wrote %v bytes to file, %v", bytesWritten, writepath)
	return nil
}

func (b *Backup) compressStagedBackup() {
	startString := fmt.Sprintf("%v", b.StartTime)
	finalfile := fmt.Sprintf("consul.snapshot.%s.gz", startString)
	finalpath := filepath.Join(b.Config.TmpDir, finalfile)
	b.FullFilename = finalpath
	source := []string{b.LocalFilePath}
	err := archiver.TarGz(finalpath, source)
	if err != nil {
		log.Fatalf("[INFO] Unable to write compressed archive to %s: %v", finalpath, err)
	}
}

// Write the local backup file to S3.
// There are no tests for this remote operation
func (b *Backup) writeBackupRemote() {
	s3Conn := session.New(&aws.Config{Region: aws.String(string(b.Config.S3Region))})

	t := time.Unix(b.StartTime, 0)
	remotePath := fmt.Sprintf("backups/%v/%d/%v/%v", t.Year(), t.Month(), t.Day(), filepath.Base(b.FullFilename))

	b.RemoteFilePath = remotePath

	// re-read the compressed file.  There is probably a better way to do this
	localFileContents, err := ioutil.ReadFile(b.FullFilename)
	if err != nil {
		log.Fatalf("[ERR] Could not read compressed file!: %v", err)
	}

	// Create the params to pass into the actual uploader
	params := &s3manager.UploadInput{
		Bucket: &b.Config.S3Bucket,
		Key:    &b.RemoteFilePath,
		Body:   bytes.NewReader(localFileContents),
	}

	log.Printf("[INFO] Uploading %v/%v to S3 in %v", string(b.Config.S3Bucket), b.RemoteFilePath, string(b.Config.S3Region))
	uploader := s3manager.NewUploader(s3Conn)
	_, err = uploader.Upload(params)
	if err != nil {
		log.Fatalf("[ERR] Could not upload to S3!: %v", err)
	}
}

// Run post processing on the backup, acking the key and removing and temp files.
// There are no tests for the remote operation.
func (b *Backup) postProcess() {
	// endtime is used just for meta and if needed to calc how long it actually took
	b.EndTime = time.Now().Unix()

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
