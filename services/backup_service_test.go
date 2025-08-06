package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/config"
	"github.com/pshima/consul-snapshot/consul"
	"github.com/pshima/consul-snapshot/mocks"
)

func TestNewBackupService(t *testing.T) {
	cfg := &config.Config{}
	consulClient := &consul.Consul{}
	storage := mocks.NewMockStorageClient()
	fs := mocks.NewMockFileSystem()
	archiver := mocks.NewMockArchiver()
	logger := mocks.NewMockLogger()

	service := NewBackupService(cfg, consulClient, storage, fs, archiver, logger)

	if service.Config != cfg {
		t.Error("expected config to be set")
	}
	if service.Storage != storage {
		t.Error("expected storage to be set")
	}
}

func TestCollectData(t *testing.T) {
	// Setup mocks
	mockConsulClient := mocks.NewMockConsulClient()
	mockConsulClient.KeyData = consulapi.KVPairs{
		&consulapi.KVPair{Key: "test/key1", Value: []byte("value1")},
		&consulapi.KVPair{Key: "test/key2", Value: []byte("value2")},
	}
	mockConsulClient.PQData = []*consulapi.PreparedQueryDefinition{
		{ID: "pq1", Name: "query1"},
	}
	mockConsulClient.ACLData = []*consulapi.ACLEntry{
		{ID: "acl1", Name: "policy1"},
	}

	consul := consul.NewConsul(mockConsulClient)
	logger := mocks.NewMockLogger()

	service := &BackupService{
		Consul: consul,
		Logger: logger,
	}

	// Test data collection
	data, err := service.CollectData()
	if err != nil {
		t.Fatalf("CollectData failed: %v", err)
	}

	if len(data.KVData) != 2 {
		t.Errorf("expected 2 KV pairs, got %d", len(data.KVData))
	}

	if len(data.PQData) != 1 {
		t.Errorf("expected 1 PQ, got %d", len(data.PQData))
	}

	if len(data.ACLData) != 1 {
		t.Errorf("expected 1 ACL, got %d", len(data.ACLData))
	}

	if data.StartTime == 0 {
		t.Error("expected StartTime to be set")
	}

	// Verify logging
	if len(logger.LogEntries) == 0 {
		t.Error("expected log entries")
	}

	logFound := false
	for _, entry := range logger.LogEntries {
		if entry.Level == "print" && len(entry.Args) > 0 {
			if msg, ok := entry.Args[0].(string); ok && strings.Contains(msg, "Listing keys") {
				logFound = true
				break
			}
		}
	}
	if !logFound {
		t.Error("expected 'Listing keys' log entry")
	}
}

func TestCollectDataError(t *testing.T) {
	// Setup mocks with error
	mockConsulClient := mocks.NewMockConsulClient()
	mockConsulClient.KeyError = fmt.Errorf("consul connection failed")

	consul := consul.NewConsul(mockConsulClient)
	logger := mocks.NewMockLogger()

	service := &BackupService{
		Consul: consul,
		Logger: logger,
	}

	// Test error handling
	_, err := service.CollectData()
	if err == nil {
		t.Fatal("expected error when consul fails")
	}

	if !strings.Contains(err.Error(), "failed to list keys") {
		t.Errorf("expected 'failed to list keys' error, got: %v", err)
	}
}

func TestSerializeData(t *testing.T) {
	logger := mocks.NewMockLogger()
	service := &BackupService{Logger: logger}

	data := &BackupData{
		KVData: consulapi.KVPairs{
			&consulapi.KVPair{Key: "test", Value: []byte("value")},
		},
		PQData: []*consulapi.PreparedQueryDefinition{
			{ID: "pq1", Name: "query1"},
		},
		ACLData: []*consulapi.ACLEntry{
			{ID: "acl1", Name: "policy1"},
		},
	}

	err := service.SerializeData(data)
	if err != nil {
		t.Fatalf("SerializeData failed: %v", err)
	}

	// Verify JSON data was created
	if len(data.KVJSONData) == 0 {
		t.Error("expected KVJSONData to be populated")
	}

	if len(data.PQJSONData) == 0 {
		t.Error("expected PQJSONData to be populated")
	}

	if len(data.ACLJSONData) == 0 {
		t.Error("expected ACLJSONData to be populated")
	}

	// Verify JSON is valid
	var kvData consulapi.KVPairs
	if err := json.Unmarshal(data.KVJSONData, &kvData); err != nil {
		t.Errorf("KV JSON is invalid: %v", err)
	}

	if len(kvData) != 1 || kvData[0].Key != "test" {
		t.Error("KV JSON content is incorrect")
	}
}

func TestWriteLocalFiles(t *testing.T) {
	cfg := &config.Config{
		TmpDir:   "/tmp",
		Hostname: "test-host",
	}
	fs := mocks.NewMockFileSystem()
	logger := mocks.NewMockLogger()

	service := &BackupService{
		Config:     cfg,
		FileSystem: fs,
		Logger:     logger,
	}

	data := &BackupData{
		StartTime:   1234567890,
		KVJSONData:  []byte(`[{"Key":"test","Value":"dmFsdWU="}]`),
		PQJSONData:  []byte(`[{"ID":"pq1","Name":"query1"}]`),
		ACLJSONData: []byte(`[{"ID":"acl1","Name":"policy1"}]`),
		Checksums:   make(map[string]string),
	}

	err := service.WriteLocalFiles(data)
	if err != nil {
		t.Fatalf("WriteLocalFiles failed: %v", err)
	}

	// Verify directory was created
	if len(fs.MkdirCalls) == 0 {
		t.Error("expected directory creation")
	}

	// Verify files were written
	expectedFiles := 4 // KV, PQ, ACL, meta
	if len(fs.WriteCalls) != expectedFiles {
		t.Errorf("expected %d files to be written, got %d", expectedFiles, len(fs.WriteCalls))
	}

	// Verify checksums were calculated
	if len(data.Checksums) != 3 {
		t.Errorf("expected 3 checksums, got %d", len(data.Checksums))
	}

	// Verify local path was set
	if data.LocalPath == "" {
		t.Error("expected LocalPath to be set")
	}

	if !strings.Contains(data.LocalPath, "test-host") {
		t.Error("expected LocalPath to contain hostname")
	}
}

func TestCompressBackup(t *testing.T) {
	cfg := &config.Config{
		TmpDir:   "/tmp",
		Hostname: "test-host",
	}
	archiver := mocks.NewMockArchiver()
	logger := mocks.NewMockLogger()

	service := &BackupService{
		Config:   cfg,
		Archiver: archiver,
		Logger:   logger,
	}

	data := &BackupData{
		StartTime: 1234567890,
		LocalPath: "/tmp/test-path",
	}

	err := service.CompressBackup(data)
	if err != nil {
		t.Fatalf("CompressBackup failed: %v", err)
	}

	// Verify archiver was called
	if len(archiver.TarGzCalls) != 1 {
		t.Error("expected one TarGz call")
	}

	call := archiver.TarGzCalls[0]
	if len(call.Sources) != 1 || call.Sources[0] != data.LocalPath {
		t.Error("expected correct source path")
	}

	if !strings.Contains(call.Destination, "test-host") {
		t.Error("expected destination to contain hostname")
	}

	// Verify remote path was set
	if data.RemotePath == "" {
		t.Error("expected RemotePath to be set")
	}
}

func TestCompressBackupAcceptanceMode(t *testing.T) {
	cfg := &config.Config{
		TmpDir:     "/tmp",
		Hostname:   "test-host",
		Acceptance: true,
	}
	archiver := mocks.NewMockArchiver()
	logger := mocks.NewMockLogger()

	service := &BackupService{
		Config:   cfg,
		Archiver: archiver,
		Logger:   logger,
	}

	data := &BackupData{
		StartTime: 1234567890,
		LocalPath: "/tmp/test-path",
	}

	err := service.CompressBackup(data)
	if err != nil {
		t.Fatalf("CompressBackup failed: %v", err)
	}

	// In acceptance mode, should use fixed filename
	call := archiver.TarGzCalls[0]
	if !strings.Contains(call.Destination, "acceptancetest.tar.gz") {
		t.Error("expected acceptancetest.tar.gz filename in acceptance mode")
	}
}

func TestUploadBackup(t *testing.T) {
	cfg := &config.Config{
		S3Bucket:     "test-bucket",
		ObjectPrefix: "backups",
	}
	storage := mocks.NewMockStorageClient()
	fs := mocks.NewMockFileSystem()
	logger := mocks.NewMockLogger()

	// Setup filesystem mock
	archiveData := []byte("test archive data")
	fs.Files["/tmp/test.tar.gz"] = archiveData

	service := &BackupService{
		Config:     cfg,
		Storage:    storage,
		FileSystem: fs,
		Logger:     logger,
	}

	data := &BackupData{
		StartTime:  1234567890,
		RemotePath: "/tmp/test.tar.gz",
	}

	err := service.UploadBackup(data)
	if err != nil {
		t.Fatalf("UploadBackup failed: %v", err)
	}

	// Verify storage was called
	if len(storage.UploadCalls) != 1 {
		t.Error("expected one upload call")
	}

	call := storage.UploadCalls[0]
	if call.Bucket != "test-bucket" {
		t.Errorf("expected bucket 'test-bucket', got '%s'", call.Bucket)
	}

	if !strings.Contains(call.Key, "backups/") {
		t.Error("expected key to contain 'backups/' prefix")
	}

	if string(call.Data) != string(archiveData) {
		t.Error("expected correct archive data to be uploaded")
	}
}

func TestUploadBackupAcceptanceMode(t *testing.T) {
	cfg := &config.Config{
		Acceptance: true,
	}
	logger := mocks.NewMockLogger()

	service := &BackupService{
		Config: cfg,
		Logger: logger,
	}

	data := &BackupData{}

	err := service.UploadBackup(data)
	if err != nil {
		t.Fatalf("UploadBackup failed in acceptance mode: %v", err)
	}

	// Should skip upload in acceptance mode
	logFound := false
	for _, entry := range logger.LogEntries {
		if entry.Level == "print" && len(entry.Args) > 0 {
			if msg, ok := entry.Args[0].(string); ok && strings.Contains(msg, "Skipping remote backup") {
				logFound = true
				break
			}
		}
	}
	if !logFound {
		t.Error("expected 'Skipping remote backup' log entry in acceptance mode")
	}
}

func TestRunBackup(t *testing.T) {
	// Setup complete mock environment
	cfg := &config.Config{
		TmpDir:     "/tmp",
		Hostname:   "test-host",
		S3Bucket:   "test-bucket",
		Acceptance: true, // Skip upload for test
	}

	mockConsulClient := mocks.NewMockConsulClient()
	mockConsulClient.KeyData = consulapi.KVPairs{
		&consulapi.KVPair{Key: "test", Value: []byte("value")},
	}

	consul := consul.NewConsul(mockConsulClient)
	storage := mocks.NewMockStorageClient()
	fs := mocks.NewMockFileSystem()
	archiver := mocks.NewMockArchiver()
	logger := mocks.NewMockLogger()

	service := NewBackupService(cfg, consul, storage, fs, archiver, logger)

	// Run complete backup
	err := service.RunBackup()
	if err != nil {
		t.Fatalf("RunBackup failed: %v", err)
	}

	// Verify all steps were executed
	if len(fs.MkdirCalls) == 0 {
		t.Error("expected directory creation")
	}

	if len(fs.WriteCalls) == 0 {
		t.Error("expected file writes")
	}

	if len(archiver.TarGzCalls) == 0 {
		t.Error("expected archive creation")
	}

	// Verify successful completion log
	completionLogFound := false
	for _, entry := range logger.LogEntries {
		if entry.Level == "print" && len(entry.Args) > 0 {
			if msg, ok := entry.Args[0].(string); ok && strings.Contains(msg, "Backup completed successfully") {
				completionLogFound = true
				break
			}
		}
	}
	if !completionLogFound {
		t.Error("expected 'Backup completed successfully' log entry")
	}
}