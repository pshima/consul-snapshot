package main

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
)

const (
	version = "0.1.0"
)

// Backup is the backup itself including configuration and data
type Backup struct {
	StartTime      int64
	JSONData       []byte
	LocalFileName  string
	LocalFilePath  string
	RemoteFilePath string
}

// Consul struct is used for consul client such as the client
// and the actual key data.
type Consul struct {
	Client     consulapi.Client
	KeyData    consulapi.KVPairs
	KeyDataLen int
}

// Just our main function to kick things off in a loop.
func main() {
	log.Printf("[INFO] v%v: Starting Consul Snapshot", version)

	log.Print("[DEBUG] Parsing Configuration")
	conf := ParseConfig()
	consul := &Consul{Client: *consulClient()}

	log.Printf("[DEBUG] Backup starting on interval: %v", conf.BackupInterval)
	ticker := time.NewTicker(conf.BackupInterval)
	for range ticker.C {
		doWork(conf, consul)
	}
}

// Create a consul client.
func consulClient() *consulapi.Client {
	consul, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Fatalf("[ERR] Unable to create a consul client: %v", err)
	}
	return consul
}

// List all the keys from consul with no prefix.
func (c *Consul) ListKeys() {
	listOpt := &consulapi.QueryOptions{}
	keys, _, err := c.Client.KV().List("/", listOpt)
	if err != nil {
		log.Fatalf("[ERR] Unable to list keys: %v", err)
	}
	c.KeyData = keys
	c.KeyDataLen = len(keys)
}

// Marshall all the keys to JSON
func (b *Backup) KeysToJSON(c *Consul) {
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
	b.LocalFileName = filename
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
func writeBackupRemote(b *Backup, conf Config) {
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
func postProcess(b *Backup, c *Consul) {
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

func doWork(conf Config, c *Consul) {
	// Loop over and over at interval time.
	backup := &Backup{}
	backup.StartTime = time.Now().Unix()

	startstring := fmt.Sprintf("%v", backup.StartTime)
	log.Printf("[INFO] Starting Backup At: %s", startstring)

	log.Print("[INFO] Listing keys from consul")
	c.ListKeys()
	log.Print("[INFO] Converting keys to JSON")
	backup.KeysToJSON(c)
	log.Print("[INFO] Writing Local Backup File")
	writeBackupLocal(backup)
	log.Print("[INFO] Writing Backup to Remote File")
	writeBackupRemote(backup, conf)
	log.Print("[INFO] Running post processing")
	postProcess(backup, c)

}
