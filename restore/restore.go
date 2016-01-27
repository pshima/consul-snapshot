package restore

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/config"
	"github.com/pshima/consul-snapshot/consul"
)

type Restore struct {
	StartTime     int64
	JSONData      consulapi.KVPairs
	LocalFilePath string
	RestorePath   string
}

// Just the runner to call from the command line
func RestoreRunner(restorepath string) int {
	consulClient := &consul.Consul{Client: *consul.ConsulClient()}

	conf := config.ParseConfig()

	log.Print("[DEBUG] Restore starting")
	doWork(conf, consulClient, restorepath)
	return 0
}

// actually do the work here.
func doWork(conf config.Config, c *consul.Consul, restorePath string) {
	restore := &Restore{}
	restore.StartTime = time.Now().Unix()
	restore.RestorePath = restorePath

	getRemoteBackup(restore, conf)
	extractBackup(restore)
	runRestore(restore, c)

}

// Get the backup from S3
func getRemoteBackup(r *Restore, conf config.Config) {
	s3Conn := session.New(&aws.Config{Region: aws.String(string(conf.S3Region))})

	r.LocalFilePath = fmt.Sprintf("%v/%v", "/tmp", r.RestorePath)

	outFile, err := os.Create(r.LocalFilePath)
	if err != nil {
		log.Fatalf("[ERR] Unable to create local restore temp file!: %v", err)
	}

	// Create the params to pass into the actual downloader
	params := &s3.GetObjectInput{
		Bucket: &conf.S3Bucket,
		Key:    &r.RestorePath,
	}

	log.Printf("[INFO] Downloading %v/%v from S3 in %v", string(conf.S3Bucket), r.LocalFilePath, string(conf.S3Region))
	downloader := s3manager.NewDownloader(s3Conn)
	_, err = downloader.Download(outFile, params)
	if err != nil {
		log.Fatalf("[ERR] Could not download file from S3!: %v", err)
	}
	outFile.Close()
}

// extract the backup to the Restore struct
func extractBackup(r *Restore) {
	// Write the json to a gzip
	handle, err := os.Open(r.LocalFilePath)
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

	json.Unmarshal(bytestosend, &r.JSONData)

	// explicitly close the file handles
	gz.Close()
	handle.Close()

}

// put the keys back in to consul.
func runRestore(r *Restore, c *consul.Consul) {
	for _, data := range r.JSONData {
		_, err := c.Client.KV().Put(data, nil)
		if err != nil {
			log.Printf("Unable to restore key: %s, %v", data.Key, err)
		}
	}
}
