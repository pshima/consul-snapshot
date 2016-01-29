package backup

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/config"
	"github.com/pshima/consul-snapshot/consul"
	"github.com/pshima/consul-snapshot/health"
)

// Backup is the backup itself including configuration and data
type Backup struct {
	StartTime      int64
	JSONData       []byte
	LocalFileName  string
	LocalFilePath  string
	RemoteFilePath string
	Config         config.Config
	Client         *consul.Consul
}

// Runner is the main runner for a backup
func (b *Backup) Runner(t string) int {
	// Start up the http server health checks
	go health.StartServer()

	if t == "test" {
		b.doWork(t)
	} else {
		log.Printf("[DEBUG] Backup starting on interval: %v", b.Config.BackupInterval)
		ticker := time.NewTicker(b.Config.BackupInterval)
		for range ticker.C {
			b.doWork(t)
		}
	}
	return 0
}

func (b *Backup) doWork(t string) {
	// Loop over and over at interval time.
	b.StartTime = time.Now().Unix()

	if t == "test" {
		b.LocalFileName = "acceptancetest.gz"
	}

	startString := fmt.Sprintf("%v", b.StartTime)

	log.Printf("[INFO] Starting Backup At: %s", startString)
	log.Print("[INFO] Listing keys from consul")

	b.Client.ListKeys()
	log.Print("[INFO] Converting keys to JSON")
	b.KeysToJSON()
	log.Print("[INFO] Writing Local Backup File")
	b.writeBackupLocal()
	if t != "test" {
		log.Print("[INFO] Writing Backup to Remote File")
		b.writeBackupRemote()
	} else {
		log.Print("[INFO] Skipping remote backup during testing")
	}
	if t != "test" {
		log.Print("[INFO] Running post processing")
		b.postProcess()
	} else {
		log.Print("[INFO] Skipping post processing during testing")
	}
}

// KeysToJSON used to marshall the data and put it on a Backup object
func (b *Backup) KeysToJSON() {
	jsonData, err := json.Marshal(b.Client.KeyData)
	if err != nil {
		log.Fatalf("[ERR] Could not encode keys to json!: %v", err)
	}
	b.JSONData = jsonData
}

// Write a local gzipped file in to tmp
func (b *Backup) writeBackupLocal() {
	// Create a filename with a unix timestamp
	startString := fmt.Sprintf("%v", b.StartTime)
	filename := fmt.Sprintf("consul.backup.%s.gz", startString)
	if b.LocalFileName == "" {
		b.LocalFileName = filename
	}
	b.LocalFilePath = "/tmp"

	filepath := fmt.Sprintf("%v/%v", b.LocalFilePath, b.LocalFileName)

	// Write the json to a gzip
	handle, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("[ERR] Could not open file for writing!: %v", err)
	}

	// Create a new gzip writer
	gz := gzip.NewWriter(handle)

	// Actually write the json to the file
	bytesWritten, err := gz.Write([]byte(b.JSONData))
	if err != nil {
		log.Fatalf("[ERR] Could not write data to file!: %v", err)
	}

	log.Printf("[DEBUG] Wrote %v bytes to file, %v", bytesWritten, filepath)

	// explicitly close the file handles
	gz.Close()
	handle.Close()
}

// Write the local backup file to S3.
// There are no tests for this remote operation
func (b *Backup) writeBackupRemote() {
	s3Conn := session.New(&aws.Config{Region: aws.String(string(b.Config.S3Region))})

	filepath := fmt.Sprintf("%v/%v", b.LocalFilePath, b.LocalFileName)

	b.RemoteFilePath = b.LocalFileName

	// re-read the compressed file.  There is probably a better way to do this
	localFileContents, err := ioutil.ReadFile(filepath)
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
	// Mark a key in consul for our last backup time.
	writeOpt := &consulapi.WriteOptions{}
	startstring := fmt.Sprintf("%v", b.StartTime)

	var err error

	lastbackup := &consulapi.KVPair{Key: "service/consul-snapshot/lastbackup", Value: []byte(startstring)}
	_, err = b.Client.Client.KV().Put(lastbackup, writeOpt)
	if err != nil {
		log.Fatalf("[ERR] Failed writing last backup timestamp to consul: %v", err)
	}

	filepath := fmt.Sprintf("%v/%v", b.LocalFilePath, b.LocalFileName)
	err = os.Remove(filepath)
	if err != nil {
		log.Printf("Unable to remove temporary backup file: %v", err)
	}
}
