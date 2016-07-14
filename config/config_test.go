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
	os.Setenv("SNAPSHOT_TMP_DIR", "/foo")
	os.Setenv("CRYPTO_PASSWORD", "bar")

	_ = setEnvVars(&c, true)

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
	if c.TmpDir != "/foo" {
		t.Errorf("Expected %v got %v", "/foo", c.TmpDir)
	}
	if c.Acceptance != false {
		t.Errorf("Expected %v got %v", "false", c.Acceptance)
	}
}

func TestSetEnvVarsAcceptanceTrue(t *testing.T) {
	var c Config
	os.Clearenv()
	os.Setenv("S3BUCKET", "buckettest")
	os.Setenv("S3REGION", "regiontest")
	os.Setenv("AWS_ACCESS_KEY_ID", "accesskeytest")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secretkeytest")
	os.Setenv("BACKUPINTERVAL", "60")
	os.Setenv("SNAPSHOT_TMP_DIR", "/foo")
	os.Setenv("ACCEPTANCE_TEST", "asdf")
	os.Setenv("CRYPTO_PASSWORD", "bar")

	_ = setEnvVars(&c, true)

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
	if c.TmpDir != "/foo" {
		t.Errorf("Expected %v got %v", "/foo", c.TmpDir)
	}
	if c.Acceptance != true {
		t.Errorf("Expected %v got %v", "true", c.Acceptance)
	}
}

func TestEmptyTmpDir(t *testing.T) {
	var c Config
	os.Clearenv()
	os.Setenv("TMPDIR", "")
	_ = setEnvVars(&c, true)
	if c.TmpDir != "/tmp" {
		t.Errorf("Expected tmp dir to = /tmp, got %v", c.TmpDir)
	}
}

func TestParseConfig(t *testing.T) {
	os.Clearenv()
	os.Setenv("BACKUPINTERVAL", "60")
	conf := ParseConfig(true)
	if conf.EncryptionSaltLen != 32 {
		t.Error("Default encryption salt length not 32!")
	}
	if conf.EncryptionPrefix != "v0:" {
		t.Error("Encryption prefix not set correctly!")
	}
	hostname, _ = os.Hostname()
	if conf.Hostname != hostname {
		t.Error("Hostname not being set correctly!")
	}

}
