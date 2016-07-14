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
	var isencrypted bool
	var err error

	//Write a temp file and make sure we get back what we expect
	if err := ioutil.WriteFile(filetestpath, []byte(filecontents), os.FileMode(0644)); err != nil {
		t.Errorf("Error writing test file to %s: %v", filetestpath, err)
	}

	// at this point its just a regular file and it should not be encrypted
	isencrypted, err = CheckEncryption(filetestpath)
	if isencrypted == true {
		t.Error("File detected as encrypted before it was encrypted!")
	}

	// encrypt the file in place
	EncryptFile(filetestpath, passphrase)

	// now the file should be encrypted
	isencrypted, err = CheckEncryption(filetestpath)
	if isencrypted == false {
		t.Error("File detected as not encrypted right after it was encrypted!")
	}

	// decrypt the file
	DecryptFile(filetestpath, passphrase)

	// read it back
	data, err := ioutil.ReadFile(filetestpath)
	if err != nil {
		t.Errorf("[ERR] Unable to read testfile: %v", err)
	}

	// the source and the data we read from the file should again match as strings
	if string(data) != filecontents {
		t.Errorf("Encrypt Decrypt returned bad results!\n Expected: %v \n Got: %v", filecontents, data)
	}
}
