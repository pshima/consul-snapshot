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
	version        = "0.0.1"
	backupInterval = 180
)

func main() {
	// TODO refactor into functions and add tests
	// TODO web interface to view backups
	// TODO add metrics
	// TODO back up single key history
	// TODO add pathing options
	// TODO restore functionality
	// TODO register as a consul service

	log.Printf("[INFO] v%v: Starting Consul Snapshot", version)
	log.Printf("[INFO] Connecting to Consul")

	// TODO load the config, detect where we are and output some of the info
	consul, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Printf("[ERR] Unable to create a consul client: %v", err)
	}

	// Options used when reading or writing to consul
	// TODO make sure the right options are set here
	queryOpt := &consulapi.QueryOptions{}
	writeOpt := &consulapi.WriteOptions{}

	// TODO better validation on data retrieved from consul
	// TODO decode the values

	// Check a key to see if we should run or not.
	enabled, _, err := consul.KV().Get("/service/consul-snapshot/enabled", queryOpt)
	if err != nil || enabled == nil {
		log.Fatalf("[ERR] Unable to determine if backups are enabled: %v", err)
	}

	// Get the bucket to use
	s3Bucket, _, err := consul.KV().Get("/service/consul-snapshot/s3bucket", queryOpt)
	if err != nil {
		log.Fatal("[ERR] Unable to get s3 bucket: %v", err)
	}

	log.Printf("[INFO] S3 Bucket configured as: %v", string(s3Bucket.Value))

	// Pass this object into the s3 uploader later.
	s3BucketString := string(s3Bucket.Value)

	// Get the region to use
	s3Region, _, err := consul.KV().Get("/service/consul-snapshot/s3region", queryOpt)
	if err != nil {
		log.Fatal("[ERR] Unable to get s3 region: %v", err)
	}

	// Acquire a lock to ensure we only have 1 node doing a backup at once
	// Expire the lock after 5 minutes, renew after each backup.
	// TODO make TTL and name configurable
	sessionEntry := consulapi.SessionEntry{Name: "consul-snapshot", TTL: "300s"}

	sessionID, _, err := consul.Session().Create(&sessionEntry, writeOpt)
	if err != nil {
		log.Fatalf("[ERR] Unable to create session: %v", err)
	}

	// Use the hostname as the data for the lock
	hostname, err := os.Hostname()

	// Let's setup a KV pair to use for locking
	lockData := &consulapi.KVPair{
		Key:     "service/consul-snapshot/lock",
		Value:   []byte(hostname),
		Session: sessionID,
	}

	// Now let's acquire the lock
	lock, _, err := consul.KV().Acquire(lockData, writeOpt)
	if lock == true {
		log.Printf("[INFO] Acquired Lock with sessionID: %v", sessionID)
	} else {
		log.Fatalf("[WARN] Could not acquire lock: %v", err)
	}

	// Release the lock when we don't need it.
	defer consul.KV().Release(lockData, writeOpt)

	// Loop over and over at interval time.
	ticker := time.NewTicker(time.Second * backupInterval)
	for range ticker.C {
		// Let's calc the time here at the start of the upload.
		start := time.Now().Unix()
		startstring := fmt.Sprintf("%v", start)

		// List all the keys in consul
		log.Printf("[INFO] Listing keys")
		keys, _, err := consul.KV().List("/", queryOpt)
		if err != nil {
			log.Println("[ERR] Unable to list keys: %v", err)
		}

		numKeys := len(keys)
		log.Printf("[INFO] Found %v keys", numKeys)

		// Take the data and convert it to JSON
		jsonData, err := json.Marshal(keys)
		if err != nil {
			log.Fatalf("[ERR] Could not encode keys to json!: %v", err)
		}

		// Create a filename with a unix timestamp
		filename := fmt.Sprintf("consul.backup.%s.gz", startstring)

		// Write the json to a gzip
		handle, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("[ERR] Could not open file for writing!: %v", err)
		}

		// Create a new gzip writer
		gz := gzip.NewWriter(handle)

		// Actually write the json to the file
		bytesWritten, err := gz.Write([]byte(jsonData))
		if err != nil {
			log.Fatalf("[ERR] Could not write data to file!: %v", err)
		}

		// explicitly close the file handles
		gz.Close()
		handle.Close()

		log.Printf("[INFO] Finished writing backup locally to %v (%v bytes)", filename, bytesWritten)

		// Upload the data to S3.
		// This requires env vars to be set
		// AWS_ACCESS_KEY_ID=AKID1234567890
		// AWS_SECRET_ACCESS_KEY=MY-SECRET-KEY
		// TODO use vault to get temporary credentials
		log.Print("[INFO] Connecting to S3")
		s3Conn := session.New(&aws.Config{Region: aws.String(string(s3Region.Value))})

		// re-read the compressed file.  There is probably a better way to do this
		filer, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatalf("[ERR] Could not read compressed file!: %v", err)
		}

		// Create the params to pass into the actual uploader
		params := &s3manager.UploadInput{
			Bucket: &s3BucketString,
			Key:    &filename,
			Body:   bytes.NewReader(filer),
		}

		log.Printf("[INFO] Uploading %v/%v to S3 in %v", string(s3Bucket.Value), filename, string(s3Region.Value))
		uploader := s3manager.NewUploader(s3Conn)
		_, err = uploader.Upload(params)
		if err != nil {
			log.Fatalf("[ERR] Could not upload to S3!: %v", err)
		}

		// Mark a key in consul for our last backup time.
		lastbackup := &consulapi.KVPair{Key: "service/consul-snapshot/lastbackup", Value: []byte(startstring)}
		_, err = consul.KV().Put(lastbackup, writeOpt)
		if err != nil {
			log.Fatalf("[ERR] Failed writing last backup timestamp to consul: %v", err)
		}

		log.Println("[INFO] Backup Completed")

		// Renew the session
		sessionID, _, err := consul.Session().Renew(sessionID, writeOpt)
		if err != nil {
			log.Printf("[WARN] Unable to renew session: %v", err)
		}
	}
}
