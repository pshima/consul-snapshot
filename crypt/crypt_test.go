package crypt

import (
	"io/ioutil"
	"os"
	"testing"
)

const (
	passphrase   = "testingpassphrase"
	filecontents = "woweezowee"
	filetestpath = "/tmp/encryptdecrypttest"
)

func TestEncryptDecrypt(t *testing.T) {
	//Write a temp file and make sure we get back what we expect
	if err := ioutil.WriteFile(filetestpath, []byte(filecontents), os.FileMode(0644)); err != nil {
		t.Errorf("Error writing test file to %s: %v", filetestpath, err)
	}

	EncryptFile(filetestpath, passphrase)
	CheckEncryption(filetestpath)
	DecryptFile(filetestpath, passphrase)

	data, err := ioutil.ReadFile(filetestpath)
	if err != nil {
		t.Errorf("[ERR] Unable to read testfile: %v", err)
	}

	if string(data) != filecontents {
		t.Errorf("Encrypt Decrypt returned bad results!\n Expected: %v \n Got: %v", filecontents, data)
	}
}
