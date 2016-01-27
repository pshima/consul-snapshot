package config

import (
	"os"
	"testing"
	"time"
)

func TestCheckEmptyOneEmpty(t *testing.T) {
	emptyfalse := checkEmpty([]string{"yesyes", "", "nono"})
	if emptyfalse != false {
		t.Error("Empty String Checking Failed")
	}
	emptytrue := checkEmpty([]string{"yesyes", "maybemaybe", "nono"})
	if emptytrue != true {
		t.Error("Full String Checking Failed")
	}
}

func TestSetEnvVars(t *testing.T) {
	var c Config
	os.Clearenv()
	os.Setenv("S3BUCKET", "buckettest")
	os.Setenv("S3REGION", "regiontest")
	os.Setenv("AWS_ACCESS_KEY_ID", "accesskeytest")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secretkeytest")
	os.Setenv("BACKUPINTERVAL", "60")

	_ = setEnvVars(&c)

	if c.S3Bucket != "buckettest" {
		t.Errorf("Expected %v got %v", "buckettest", c.S3Bucket)
	}
	if c.S3Region != "regiontest" {
		t.Errorf("Expected %v got %v", "regiontest", c.S3Region)
	}
	if c.S3AccessKey != "accesskeytest" {
		t.Errorf("Expected %v got %v", "accesskeytest", c.S3AccessKey)
	}
	if c.S3SecretKey != "secretkeytest" {
		t.Errorf("Expected %v got %v", "secretkeytest", c.S3SecretKey)
	}
	if c.BackupInterval != 60*time.Second {
		t.Errorf("Expected %v got %v", 60, c.BackupInterval)
	}
}
