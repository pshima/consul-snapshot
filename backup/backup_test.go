package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/consul"
)

const (
	writeBackupChecksum = "cd77faecc2e0dd4eb6e03e87354837763d4f6eb0742446ee6670d5944982c4e1"
)

var (
	testkvpair1 = consulapi.KVPair{Key: "test1", Value: []byte("yes")}
	testkvpair2 = consulapi.KVPair{Key: "test2", Value: []byte("no")}
	kvpairlist  = []*consulapi.KVPair{&testkvpair1, &testkvpair2}
)

// Setup some basic structs we can use across tests
func testingStructs() (*consul.Consul, *Backup) {
	consulClient := &consul.Consul{}
	consulClient.KeyData = kvpairlist
	backup := &Backup{}
	backup.StartTime = time.Now().Unix()

	return consulClient, backup

}

// just see if we get the same encoded results back
func TestKeysToJSON(t *testing.T) {
	consulClient, backup := testingStructs()
	backup.KeysToJSON(consulClient)

	marshallSouce, err := json.Marshal(kvpairlist)
	if err != nil {
		t.Errorf("Unable to marshall source testing data: %v", err)
	}

	reflecttest := reflect.DeepEqual(backup.JSONData, marshallSouce)

	if reflecttest != true {
		t.Errorf("JSON marshall did not equal. Got %v, expected %v", marshallSouce, backup.JSONData)
	}

}

// Write the file locally and then check that we get the same checksum back
func TestWriteBackupLocal(t *testing.T) {
	consulClient, backup := testingStructs()
	backup.KeysToJSON(consulClient)

	writeBackupLocal(backup)
	shacheck := sha256.New()

	filepath := fmt.Sprintf("%v/%v", backup.LocalFilePath, backup.LocalFileName)
	f, err := ioutil.ReadFile(filepath)
	shacheck.Write(f)
	if err != nil {
		t.Errorf("Unable to read local backup file", err)
	}

	expectedSum := writeBackupChecksum
	receivedSum := hex.EncodeToString(shacheck.Sum(nil))

	if expectedSum != receivedSum {
		t.Errorf("Expected checksum %s, recieved %s", expectedSum, receivedSum)
	}

	err = os.Remove(filepath)
	if err != nil {
		t.Errorf("Unable to remove temporary backup file!", err)
	}

}
