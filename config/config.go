package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

var hostname string

type Config struct {
	S3Bucket       string
	S3Region       string
	S3AccessKey    string
	S3SecretKey    string
	Hostname       string
	BackupInterval time.Duration
}

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Fatalf("Unable to determine hostname: %v", err)
	}
}

func checkEmpty(s []string) {
	for _, item := range s {
		if len(item) < 1 {
			log.Fatal("Required env var missing, exiting")
		}
	}
}

func ParseConfig() Config {
	conf := Config{}
	conf.S3Bucket = os.Getenv("S3BUCKET")
	conf.S3Region = os.Getenv("S3REGION")
	conf.S3AccessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	conf.S3SecretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	backupInterval := os.Getenv("BACKUPINTERVAL")

	envChecks := []string{conf.S3Bucket, conf.S3Region, conf.S3AccessKey, conf.S3SecretKey, backupInterval}
	checkEmpty(envChecks)

	backupStrToInt, err := strconv.Atoi(backupInterval)
	if err != nil {
		log.Fatalf("Unable to conver BACKUPINTERVAL environment var to integer: %v", err)
	}

	backupTimeDuration := time.Duration(backupStrToInt) * time.Second

	conf.BackupInterval = backupTimeDuration
	conf.Hostname = hostname
	return conf
}
