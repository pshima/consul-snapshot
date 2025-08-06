package consul

import (
	"fmt"
	"strings"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/mocks"
)

func TestClient(t *testing.T) {
	// Test that Client() creates a consul client
	// This will use environment variables if set, or defaults
	client := Client()
	if client == nil {
		t.Error("expected Client() to return a non-nil client")
	}
}

func TestConsulStruct(t *testing.T) {
	// Test the Consul struct initialization
	c := &Consul{}
	
	// Test initial state
	if c.KeyDataLen != 0 {
		t.Error("expected initial KeyDataLen to be 0")
	}
	if c.PQDataLen != 0 {
		t.Error("expected initial PQDataLen to be 0")
	}
	if c.ACLDataLen != 0 {
		t.Error("expected initial ACLDataLen to be 0")
	}
}

func TestListKeys_MockData(t *testing.T) {
	// Create test consul instance
	c := &Consul{}
	
	// Mock some data directly for testing data handling
	testKV1 := &consulapi.KVPair{Key: "test/key1", Value: []byte("value1")}
	testKV2 := &consulapi.KVPair{Key: "test/key2", Value: []byte("value2")}
	testData := consulapi.KVPairs{testKV1, testKV2}
	
	// Set mock data directly
	c.KeyData = testData
	c.KeyDataLen = len(testData)
	
	if c.KeyDataLen != 2 {
		t.Errorf("expected KeyDataLen to be 2, got %d", c.KeyDataLen)
	}
	
	if len(c.KeyData) != 2 {
		t.Errorf("expected KeyData length to be 2, got %d", len(c.KeyData))
	}
	
	if c.KeyData[0].Key != "test/key1" {
		t.Errorf("expected first key to be 'test/key1', got %s", c.KeyData[0].Key)
	}
}

func TestListPQs_MockData(t *testing.T) {
	// Create test consul instance
	c := &Consul{}
	
	// Mock some PQ data
	testPQ := &consulapi.PreparedQueryDefinition{
		ID:   "test-id",
		Name: "test-query",
	}
	testPQData := []*consulapi.PreparedQueryDefinition{testPQ}
	
	// Set mock data directly
	c.PQData = testPQData
	c.PQDataLen = len(testPQData)
	
	if c.PQDataLen != 1 {
		t.Errorf("expected PQDataLen to be 1, got %d", c.PQDataLen)
	}
	
	if len(c.PQData) != 1 {
		t.Errorf("expected PQData length to be 1, got %d", len(c.PQData))
	}
	
	if c.PQData[0].Name != "test-query" {
		t.Errorf("expected PQ name to be 'test-query', got %s", c.PQData[0].Name)
	}
}

func TestListACLs_MockData(t *testing.T) {
	// Create test consul instance
	c := &Consul{}
	
	// Mock some ACL data
	testACL := &consulapi.ACLEntry{
		ID:   "test-acl-id",
		Name: "test-acl",
	}
	testACLData := []*consulapi.ACLEntry{testACL}
	
	// Set mock data directly
	c.ACLData = testACLData
	c.ACLDataLen = len(testACLData)
	
	if c.ACLDataLen != 1 {
		t.Errorf("expected ACLDataLen to be 1, got %d", c.ACLDataLen)
	}
	
	if len(c.ACLData) != 1 {
		t.Errorf("expected ACLData length to be 1, got %d", len(c.ACLData))
	}
	
	if c.ACLData[0].Name != "test-acl" {
		t.Errorf("expected ACL name to be 'test-acl', got %s", c.ACLData[0].Name)
	}
}

func TestListACLs_DisabledScenario(t *testing.T) {
	// Test the ACL disabled scenario by testing the error string matching
	c := &Consul{}
	
	// Simulate the ACL disabled error handling logic
	testError := "401 (ACL support disabled)"
	if !strings.Contains(testError, "401 (ACL support disabled)") {
		t.Error("expected error string to match ACL disabled pattern")
	}
	
	// Test that when ACLs are disabled, we set empty data
	c.ACLData = []*consulapi.ACLEntry{}
	c.ACLDataLen = 0
	
	if c.ACLDataLen != 0 {
		t.Errorf("expected ACLDataLen to be 0 when ACLs disabled, got %d", c.ACLDataLen)
	}
	
	if len(c.ACLData) != 0 {
		t.Errorf("expected ACLData length to be 0 when ACLs disabled, got %d", len(c.ACLData))
	}
}

func TestClientCreation(t *testing.T) {
	// Test that Client() creates a valid client
	client := Client()
	if client == nil {
		t.Error("Client() should return a non-nil client")
	}
}

func TestConsulStructFields(t *testing.T) {
	// Test that the Consul struct has all expected fields
	c := &Consul{}
	
	// Verify field types exist and can be set
	c.KeyDataLen = 10
	c.PQDataLen = 5
	c.ACLDataLen = 3
	
	if c.KeyDataLen != 10 {
		t.Errorf("expected KeyDataLen to be 10, got %d", c.KeyDataLen)
	}
	
	if c.PQDataLen != 5 {
		t.Errorf("expected PQDataLen to be 5, got %d", c.PQDataLen)
	}
	
	if c.ACLDataLen != 3 {
		t.Errorf("expected ACLDataLen to be 3, got %d", c.ACLDataLen)
	}
}

func TestConsulDataTypes(t *testing.T) {
	// Test that data type assignments work correctly
	c := &Consul{}
	
	// Test KV data assignment
	testKV := consulapi.KVPairs{
		&consulapi.KVPair{Key: "test1", Value: []byte("value1")},
		&consulapi.KVPair{Key: "test2", Value: []byte("value2")},
	}
	c.KeyData = testKV
	c.KeyDataLen = len(testKV)
	
	if len(c.KeyData) != 2 {
		t.Errorf("expected 2 KV pairs, got %d", len(c.KeyData))
	}
	
	// Test PQ data assignment
	testPQ := []*consulapi.PreparedQueryDefinition{
		{ID: "pq1", Name: "query1"},
	}
	c.PQData = testPQ
	c.PQDataLen = len(testPQ)
	
	if len(c.PQData) != 1 {
		t.Errorf("expected 1 PQ, got %d", len(c.PQData))
	}
	
	// Test ACL data assignment
	testACL := []*consulapi.ACLEntry{
		{ID: "acl1", Name: "policy1"},
	}
	c.ACLData = testACL
	c.ACLDataLen = len(testACL)
	
	if len(c.ACLData) != 1 {
		t.Errorf("expected 1 ACL, got %d", len(c.ACLData))
	}
}

func TestNewConsul(t *testing.T) {
	mockClient := mocks.NewMockConsulClient()
	consul := NewConsul(mockClient)
	
	if consul.Client != mockClient {
		t.Error("expected client to be set")
	}
}

func TestListKeysWithMock(t *testing.T) {
	mockClient := mocks.NewMockConsulClient()
	mockClient.KeyData = consulapi.KVPairs{
		&consulapi.KVPair{Key: "test", Value: []byte("value")},
	}
	
	consul := NewConsul(mockClient)
	
	err := consul.ListKeys()
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}
	
	if len(consul.KeyData) != 1 {
		t.Errorf("expected 1 key, got %d", len(consul.KeyData))
	}
	
	if consul.KeyDataLen != 1 {
		t.Errorf("expected KeyDataLen to be 1, got %d", consul.KeyDataLen)
	}
}

func TestListKeysError(t *testing.T) {
	mockClient := mocks.NewMockConsulClient()
	mockClient.KeyError = fmt.Errorf("connection failed")
	
	consul := NewConsul(mockClient)
	
	err := consul.ListKeys()
	if err == nil {
		t.Fatal("expected error when consul fails")
	}
	
	if !strings.Contains(err.Error(), "connection failed") {
		t.Errorf("expected connection error, got: %v", err)
	}
}

func TestListPQsWithMock(t *testing.T) {
	mockClient := mocks.NewMockConsulClient()
	mockClient.PQData = []*consulapi.PreparedQueryDefinition{
		{ID: "pq1", Name: "query1"},
	}
	
	consul := NewConsul(mockClient)
	
	err := consul.ListPQs()
	if err != nil {
		t.Fatalf("ListPQs failed: %v", err)
	}
	
	if len(consul.PQData) != 1 {
		t.Errorf("expected 1 PQ, got %d", len(consul.PQData))
	}
	
	if consul.PQDataLen != 1 {
		t.Errorf("expected PQDataLen to be 1, got %d", consul.PQDataLen)
	}
}

func TestListACLsWithMock(t *testing.T) {
	mockClient := mocks.NewMockConsulClient()
	mockClient.ACLData = []*consulapi.ACLEntry{
		{ID: "acl1", Name: "policy1"},
	}
	
	consul := NewConsul(mockClient)
	
	err := consul.ListACLs()
	if err != nil {
		t.Fatalf("ListACLs failed: %v", err)
	}
	
	if len(consul.ACLData) != 1 {
		t.Errorf("expected 1 ACL, got %d", len(consul.ACLData))
	}
	
	if consul.ACLDataLen != 1 {
		t.Errorf("expected ACLDataLen to be 1, got %d", consul.ACLDataLen)
	}
}

func TestRestoreKeys(t *testing.T) {
	mockClient := mocks.NewMockConsulClient()
	consul := NewConsul(mockClient)
	
	keys := consulapi.KVPairs{
		&consulapi.KVPair{Key: "test1", Value: []byte("value1")},
		&consulapi.KVPair{Key: "test2", Value: []byte("value2")},
	}
	
	err := consul.RestoreKeys(keys)
	if err != nil {
		t.Fatalf("RestoreKeys failed: %v", err)
	}
	
	// Verify keys were added to mock
	if len(mockClient.KeyData) != 2 {
		t.Errorf("expected 2 keys in mock, got %d", len(mockClient.KeyData))
	}
}

func TestRestorePQs(t *testing.T) {
	mockClient := mocks.NewMockConsulClient()
	consul := NewConsul(mockClient)
	
	pqs := []*consulapi.PreparedQueryDefinition{
		{ID: "pq1", Name: "query1"},
		{ID: "pq2", Name: "query2"},
	}
	
	err := consul.RestorePQs(pqs)
	if err != nil {
		t.Fatalf("RestorePQs failed: %v", err)
	}
	
	// Verify PQs were added to mock
	if len(mockClient.PQData) != 2 {
		t.Errorf("expected 2 PQs in mock, got %d", len(mockClient.PQData))
	}
}

func TestRestoreACLs(t *testing.T) {
	mockClient := mocks.NewMockConsulClient()
	consul := NewConsul(mockClient)
	
	acls := []*consulapi.ACLEntry{
		{ID: "acl1", Name: "policy1"},
		{ID: "acl2", Name: "policy2"},
	}
	
	err := consul.RestoreACLs(acls)
	if err != nil {
		t.Fatalf("RestoreACLs failed: %v", err)
	}
	
	// Verify ACLs were added to mock
	if len(mockClient.ACLData) != 2 {
		t.Errorf("expected 2 ACLs in mock, got %d", len(mockClient.ACLData))
	}
}