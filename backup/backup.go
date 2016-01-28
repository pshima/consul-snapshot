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
}

func BackupRunner(t string) int {
	consulClient := &consul.Consul{Client: *consul.ConsulClient()}

	conf := config.ParseConfig()

	// Start up the http server health checks
	go health.StartServer()

	if t == "test" {
		doWork(conf, consulClient, t)
	} else {
		log.Printf("[DEBUG] Backup starting on interval: %v", conf.BackupInterval)
		ticker := time.NewTicker(conf.BackupInterval)
		for range ticker.C {
			doWork(conf, consulClient, t)
		}
	}
	return 0
}

func doWork(conf config.Config, c *consul.Consul, t string) {
	// Loop over and over at interval time.
	backup := &Backup{}
	backup.StartTime = time.Now().Unix()

	if t == "test" {
		backup.LocalFileName = "acceptancetest.gz"
	}

	startstring := fmt.Sprintf("%v", backup.StartTime)
	log.Printf("[INFO] Starting Backup At: %s", startstring)

	log.Print("[INFO] Listing keys from consul")
	c.ListKeys()
	log.Print("[INFO] Converting keys to JSON")
	backup.KeysToJSON(c)
	log.Print("[INFO] Writing Local Backup File")
	writeBackupLocal(backup)
	if t != "test" {
		log.Print("[INFO] Writing Backup to Remote File")
		writeBackupRemote(backup, conf)
	} else {
		log.Print("[INFO] Skipping remove back during testing")
	}
	if t != "test" {
		log.Print("[INFO] Running post processing")
		postProcess(backup, c)
	} else {
		log.Print("[INFO] Skipping post processing during testing")
	}
}

// Marshall all the keys to JSON
func (b *Backup) KeysToJSON(c *consul.Consul) {
	jsonData, err := json.Marshal(c.KeyData)
	if err != nil {
		log.Fatalf("[ERR] Could not encode keys to json!: %v", err)
	}
	b.JSONData = jsonData
}

// Write a local gzipped file in to tmp
func writeBackupLocal(b *Backup) {
	// Create a filename with a unix timestamp
	startstring := fmt.Sprintf("%v", b.StartTime)
	filename := fmt.Sprintf("consul.backup.%s.gz", startstring)
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
func writeBackupRemote(b *Backup, conf config.Config) {
	s3Conn := session.New(&aws.Config{Region: aws.String(string(conf.S3Region))})

	filepath := fmt.Sprintf("%v/%v", b.LocalFilePath, b.LocalFileName)

	b.RemoteFilePath = b.LocalFileName

	// re-read the compressed file.  There is probably a better way to do this
	localFileContents, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("[ERR] Could not read compressed file!: %v", err)
	}

	// Create the params to pass into the actual uploader
	params := &s3manager.UploadInput{
		Bucket: &conf.S3Bucket,
		Key:    &b.RemoteFilePath,
		Body:   bytes.NewReader(localFileContents),
	}

	log.Printf("[INFO] Uploading %v/%v to S3 in %v", string(conf.S3Bucket), b.RemoteFilePath, string(conf.S3Region))
	uploader := s3manager.NewUploader(s3Conn)
	_, err = uploader.Upload(params)
	if err != nil {
		log.Fatalf("[ERR] Could not upload to S3!: %v", err)
	}
}

// Run post processing on the backup, acking the key and removing and temp files.
// There are no tests for the remote operation.
func postProcess(b *Backup, c *consul.Consul) {
	// Mark a key in consul for our last backup time.
	writeOpt := &consulapi.WriteOptions{}
	startstring := fmt.Sprintf("%v", b.StartTime)

	var err error

	lastbackup := &consulapi.KVPair{Key: "service/consul-snapshot/lastbackup", Value: []byte(startstring)}
	_, err = c.Client.KV().Put(lastbackup, writeOpt)
	if err != nil {
		log.Fatalf("[ERR] Failed writing last backup timestamp to consul: %v", err)
	}

	filepath := fmt.Sprintf("%v/%v", b.LocalFilePath, b.LocalFileName)
	err = os.Remove(filepath)
	if err != nil {
		log.Printf("Unable to remove temporary backup file!", err)
	}

}
