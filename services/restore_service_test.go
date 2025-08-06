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

func TestNewRestoreService(t *testing.T) {
	cfg := &config.Config{}
	consulClient := &consul.Consul{}
	storage := mocks.NewMockStorageClient()
	fs := mocks.NewMockFileSystem()
	archiver := mocks.NewMockArchiver()
	logger := mocks.NewMockLogger()

	service := NewRestoreService(cfg, consulClient, storage, fs, archiver, logger)

	if service.Config != cfg {
		t.Error("expected config to be set")
	}
	if service.Storage != storage {
		t.Error("expected storage to be set")
	}
}

func TestDownloadBackupAcceptanceMode(t *testing.T) {
	cfg := &config.Config{
		TmpDir:     "/tmp",
		Acceptance: true,
	}
	logger := mocks.NewMockLogger()

	service := &RestoreService{
		Config: cfg,
		Logger: logger,
	}

	data, err := service.DownloadBackup("test-backup.tar.gz")
	if err != nil {
		t.Fatalf("DownloadBackup failed: %v", err)
	}

	if data.RestorePath != "test-backup.tar.gz" {
		t.Error("expected RestorePath to be set")
	}

	expectedPath := "/tmp/acceptancetest.tar.gz"
	if data.LocalPath != expectedPath {
		t.Errorf("expected LocalPath to be '%s', got '%s'", expectedPath, data.LocalPath)
	}
}

func TestDownloadBackupFromStorage(t *testing.T) {
	cfg := &config.Config{
		TmpDir:   "/tmp",
		S3Bucket: "test-bucket",
	}
	storage := mocks.NewMockStorageClient()
	fs := mocks.NewMockFileSystem()
	logger := mocks.NewMockLogger()

	// Setup mock data
	backupData := []byte("test backup data")
	storage.Data["test-bucket/backups/test.tar.gz"] = backupData

	service := &RestoreService{
		Config:     cfg,
		Storage:    storage,
		FileSystem: fs,
		Logger:     logger,
	}

	data, err := service.DownloadBackup("backups/test.tar.gz")
	if err != nil {
		t.Fatalf("DownloadBackup failed: %v", err)
	}

	// Verify download was called
	if len(storage.DownloadCalls) != 1 {
		t.Error("expected one download call")
	}

	downloadCall := storage.DownloadCalls[0]
	if downloadCall.Bucket != "test-bucket" {
		t.Errorf("expected bucket 'test-bucket', got '%s'", downloadCall.Bucket)
	}

	if downloadCall.Key != "backups/test.tar.gz" {
		t.Errorf("expected key 'backups/test.tar.gz', got '%s'", downloadCall.Key)
	}

	// Verify data was returned
	if data.RestorePath != "backups/test.tar.gz" {
		t.Error("expected RestorePath to be set correctly")
	}

	// Verify file was written locally
	if len(fs.WriteCalls) != 1 {
		t.Error("expected one file write")
	}

	writeCall := fs.WriteCalls[0]
	if string(writeCall.Data) != string(backupData) {
		t.Error("expected correct backup data to be written")
	}
}

func TestExtractBackup(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	archiver := mocks.NewMockArchiver()
	logger := mocks.NewMockLogger()

	service := &RestoreService{
		FileSystem: fs,
		Archiver:   archiver,
		Logger:     logger,
	}

	data := &RestoreData{
		LocalPath: "/tmp/backup.tar.gz",
	}

	err := service.ExtractBackup(data)
	if err != nil {
		t.Fatalf("ExtractBackup failed: %v", err)
	}

	// Verify directory creation
	if len(fs.MkdirCalls) != 1 {
		t.Error("expected one directory creation")
	}

	// Verify extraction
	if len(archiver.UnTarGzCalls) != 1 {
		t.Error("expected one extraction call")
	}

	extractCall := archiver.UnTarGzCalls[0]
	if extractCall.Source != data.LocalPath {
		t.Error("expected correct source path")
	}

	if data.ExtractedPath == "" {
		t.Error("expected ExtractedPath to be set")
	}

	expectedExtracted := data.LocalPath + ".extracted"
	if data.ExtractedPath != expectedExtracted {
		t.Errorf("expected ExtractedPath to be '%s', got '%s'", expectedExtracted, data.ExtractedPath)
	}
}

func TestLoadMetadata(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	logger := mocks.NewMockLogger()

	// Setup mock metadata
	metadata := map[string]interface{}{
		"consul_snapshot_version": "0.2.5",
		"start_time":              float64(1234567890),
		"node_name":               "test-node",
	}
	metaJSON, _ := json.Marshal(metadata)
	fs.Files["/tmp/extracted/meta.json"] = metaJSON

	service := &RestoreService{
		FileSystem: fs,
		Logger:     logger,
	}

	data := &RestoreData{
		ExtractedPath: "/tmp/extracted",
	}

	err := service.LoadMetadata(data)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	// Verify metadata was loaded
	if data.Metadata == nil {
		t.Fatal("expected metadata to be loaded")
	}

	if version, ok := data.Metadata["consul_snapshot_version"].(string); !ok || version != "0.2.5" {
		t.Error("expected correct version in metadata")
	}

	if startTime, ok := data.Metadata["start_time"].(float64); !ok || startTime != 1234567890 {
		t.Error("expected correct start_time in metadata")
	}

	// Verify file was read
	if len(fs.ReadCalls) != 1 {
		t.Error("expected one file read")
	}

	if fs.ReadCalls[0] != "/tmp/extracted/meta.json" {
		t.Error("expected correct metadata file path")
	}
}

func TestLoadBackupData(t *testing.T) {
	fs := mocks.NewMockFileSystem()
	logger := mocks.NewMockLogger()

	// Setup mock data files
	kvData := consulapi.KVPairs{
		&consulapi.KVPair{Key: "test", Value: []byte("value")},
	}
	kvJSON, _ := json.Marshal(kvData)
	fs.Files["/tmp/extracted/consul.kv.1234567890.json"] = kvJSON

	pqData := []*consulapi.PreparedQueryDefinition{
		{ID: "pq1", Name: "query1"},
	}
	pqJSON, _ := json.Marshal(pqData)
	fs.Files["/tmp/extracted/consul.pq.1234567890.json"] = pqJSON

	aclData := []*consulapi.ACLEntry{
		{ID: "acl1", Name: "policy1"},
	}
	aclJSON, _ := json.Marshal(aclData)
	fs.Files["/tmp/extracted/consul.acl.1234567890.json"] = aclJSON

	service := &RestoreService{
		FileSystem: fs,
		Logger:     logger,
	}

	data := &RestoreData{
		ExtractedPath: "/tmp/extracted",
		Metadata: map[string]interface{}{
			"start_time": float64(1234567890),
		},
	}

	err := service.LoadBackupData(data)
	if err != nil {
		t.Fatalf("LoadBackupData failed: %v", err)
	}

	// Verify KV data was loaded
	if len(data.KVData) != 1 {
		t.Errorf("expected 1 KV pair, got %d", len(data.KVData))
	}

	if data.KVData[0].Key != "test" {
		t.Error("expected correct KV data")
	}

	// Verify PQ data was loaded
	if len(data.PQData) != 1 {
		t.Errorf("expected 1 PQ, got %d", len(data.PQData))
	}

	if data.PQData[0].Name != "query1" {
		t.Error("expected correct PQ data")
	}

	// Verify ACL data was loaded
	if len(data.ACLData) != 1 {
		t.Errorf("expected 1 ACL, got %d", len(data.ACLData))
	}

	if data.ACLData[0].Name != "policy1" {
		t.Error("expected correct ACL data")
	}

	// Verify logging
	logFound := false
	for _, entry := range logger.LogEntries {
		if entry.Level == "printf" && strings.Contains(entry.Format, "Loaded %d keys") {
			logFound = true
			break
		}
	}
	if !logFound {
		t.Error("expected 'Loaded keys' log entry")
	}
}

func TestRestoreToConsul(t *testing.T) {
	mockConsulClient := mocks.NewMockConsulClient()
	consul := consul.NewConsul(mockConsulClient)
	logger := mocks.NewMockLogger()

	service := &RestoreService{
		Consul: consul,
		Logger: logger,
	}

	data := &RestoreData{
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

	err := service.RestoreToConsul(data)
	if err != nil {
		t.Fatalf("RestoreToConsul failed: %v", err)
	}

	// Verify data was restored to mock consul
	if len(mockConsulClient.KeyData) != 1 {
		t.Error("expected KV data to be restored")
	}

	if len(mockConsulClient.PQData) != 1 {
		t.Error("expected PQ data to be restored")
	}

	if len(mockConsulClient.ACLData) != 1 {
		t.Error("expected ACL data to be restored")
	}
}

func TestRestoreToConsulWithErrors(t *testing.T) {
	mockConsulClient := mocks.NewMockConsulClient()
	mockConsulClient.PutKVError = fmt.Errorf("KV restore failed")
	mockConsulClient.CreatePQError = fmt.Errorf("PQ restore failed")

	consul := consul.NewConsul(mockConsulClient)
	logger := mocks.NewMockLogger()

	service := &RestoreService{
		Consul: consul,
		Logger: logger,
	}

	data := &RestoreData{
		KVData: consulapi.KVPairs{
			&consulapi.KVPair{Key: "test", Value: []byte("value")},
		},
		PQData: []*consulapi.PreparedQueryDefinition{
			{ID: "pq1", Name: "query1"},
		},
	}

	err := service.RestoreToConsul(data)
	if err == nil {
		t.Fatal("expected error when restore fails")
	}

	if !strings.Contains(err.Error(), "restore completed with") {
		t.Errorf("expected error message about completion with errors, got: %v", err)
	}

	// Verify error logging
	errorLogFound := false
	for _, entry := range logger.LogEntries {
		if entry.Level == "printf" && strings.Contains(entry.Format, "Failed to restore") {
			errorLogFound = true
			break
		}
	}
	if !errorLogFound {
		t.Error("expected error log entry")
	}
}

func TestRunRestore(t *testing.T) {
	// Setup complete mock environment
	cfg := &config.Config{
		TmpDir:     "/tmp",
		Acceptance: true, // Use local file
	}

	mockConsulClient := mocks.NewMockConsulClient()
	consul := consul.NewConsul(mockConsulClient)
	storage := mocks.NewMockStorageClient()
	fs := mocks.NewMockFileSystem()
	archiver := mocks.NewMockArchiver()
	logger := mocks.NewMockLogger()

	// Setup mock files
	metadata := map[string]interface{}{
		"consul_snapshot_version": "0.2.5",
		"start_time":              float64(1234567890),
	}
	metaJSON, _ := json.Marshal(metadata)
	fs.Files["/tmp/acceptancetest.tar.gz.extracted/meta.json"] = metaJSON

	kvData := consulapi.KVPairs{
		&consulapi.KVPair{Key: "test", Value: []byte("value")},
	}
	kvJSON, _ := json.Marshal(kvData)
	fs.Files["/tmp/acceptancetest.tar.gz.extracted/consul.kv.1234567890.json"] = kvJSON

	service := NewRestoreService(cfg, consul, storage, fs, archiver, logger)

	// Run complete restore
	err := service.RunRestore("test-backup.tar.gz")
	if err != nil {
		t.Fatalf("RunRestore failed: %v", err)
	}

	// Verify all steps were executed
	if len(fs.MkdirCalls) == 0 {
		t.Error("expected directory creation for extraction")
	}

	if len(archiver.UnTarGzCalls) == 0 {
		t.Error("expected archive extraction")
	}

	// Verify data was restored
	if len(mockConsulClient.KeyData) == 0 {
		t.Error("expected KV data to be restored")
	}

	// Verify successful completion log
	completionLogFound := false
	for _, entry := range logger.LogEntries {
		if entry.Level == "print" && len(entry.Args) > 0 {
			if msg, ok := entry.Args[0].(string); ok && strings.Contains(msg, "Restore completed successfully") {
				completionLogFound = true
				break
			}
		}
	}
	if !completionLogFound {
		t.Error("expected 'Restore completed successfully' log entry")
	}
}