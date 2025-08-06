package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/config"
	"github.com/pshima/consul-snapshot/consul"
	"github.com/pshima/consul-snapshot/interfaces"
)

// LoggerAdapter is a simple implementation for the service
type LoggerAdapter struct{}

func (l *LoggerAdapter) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func (l *LoggerAdapter) Print(args ...interface{}) {
	fmt.Print(args...)
}

func (l *LoggerAdapter) Fatalf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	// Don't actually exit for service layer
}

func (l *LoggerAdapter) Fatal(args ...interface{}) {
	fmt.Print(args...)
	// Don't actually exit for service layer
}

// BackupService handles backup operations with dependency injection
type BackupService struct {
	Config     *config.Config
	Consul     *consul.Consul
	Storage    interfaces.StorageClient
	FileSystem interfaces.FileSystem
	Archiver   interfaces.Archiver
	Logger     interfaces.Logger
}

// NewBackupService creates a new backup service
func NewBackupService(config *config.Config, consul *consul.Consul, storage interfaces.StorageClient, fs interfaces.FileSystem, archiver interfaces.Archiver, logger interfaces.Logger) *BackupService {
	return &BackupService{
		Config:     config,
		Consul:     consul,
		Storage:    storage,
		FileSystem: fs,
		Archiver:   archiver,
		Logger:     logger,
	}
}

// BackupData represents the data being backed up
type BackupData struct {
	KVData       consulapi.KVPairs
	PQData       []*consulapi.PreparedQueryDefinition
	ACLData      []*consulapi.ACLEntry
	KVJSONData   []byte
	PQJSONData   []byte
	ACLJSONData  []byte
	StartTime    int64
	LocalPath    string
	RemotePath   string
	Checksums    map[string]string
}

// CollectData collects data from consul
func (s *BackupService) CollectData() (*BackupData, error) {
	data := &BackupData{
		StartTime: time.Now().Unix(),
		Checksums: make(map[string]string),
	}

	s.Logger.Print("[INFO] Listing keys from consul")
	if err := s.Consul.ListKeys(); err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	data.KVData = s.Consul.KeyData

	s.Logger.Print("[INFO] Listing Prepared Queries from consul")
	if err := s.Consul.ListPQs(); err != nil {
		return nil, fmt.Errorf("failed to list PQs: %w", err)
	}
	data.PQData = s.Consul.PQData

	s.Logger.Print("[INFO] Listing ACLs from consul")
	if err := s.Consul.ListACLs(); err != nil {
		return nil, fmt.Errorf("failed to list ACLs: %w", err)
	}
	data.ACLData = s.Consul.ACLData

	return data, nil
}

// SerializeData converts consul data to JSON
func (s *BackupService) SerializeData(data *BackupData) error {
	var err error

	s.Logger.Printf("[INFO] Converting %d keys to JSON", len(data.KVData))
	data.KVJSONData, err = json.Marshal(data.KVData)
	if err != nil {
		return fmt.Errorf("failed to marshal KV data: %w", err)
	}

	s.Logger.Printf("[INFO] Converting %d PQs to JSON", len(data.PQData))
	data.PQJSONData, err = json.Marshal(data.PQData)
	if err != nil {
		return fmt.Errorf("failed to marshal PQ data: %w", err)
	}

	s.Logger.Printf("[INFO] Converting %d ACLs to JSON", len(data.ACLData))
	data.ACLJSONData, err = json.Marshal(data.ACLData)
	if err != nil {
		return fmt.Errorf("failed to marshal ACL data: %w", err)
	}

	return nil
}

// WriteLocalFiles writes JSON data to local files
func (s *BackupService) WriteLocalFiles(data *BackupData) error {
	startString := fmt.Sprintf("%v", data.StartTime)
	
	// Create local directory
	prefix := fmt.Sprintf("%s.consul.snapshot.%s", s.Config.Hostname, startString)
	data.LocalPath = filepath.Join(s.Config.TmpDir, prefix)
	
	if err := s.FileSystem.MkdirAll(data.LocalPath, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Write KV file
	kvFile := fmt.Sprintf("consul.kv.%s.json", startString)
	kvPath := filepath.Join(data.LocalPath, kvFile)
	if err := s.FileSystem.WriteFile(kvPath, data.KVJSONData, 0644); err != nil {
		return fmt.Errorf("failed to write KV file: %w", err)
	}
	data.Checksums["kv"] = s.calculateChecksum(data.KVJSONData)

	// Write PQ file
	pqFile := fmt.Sprintf("consul.pq.%s.json", startString)
	pqPath := filepath.Join(data.LocalPath, pqFile)
	if err := s.FileSystem.WriteFile(pqPath, data.PQJSONData, 0644); err != nil {
		return fmt.Errorf("failed to write PQ file: %w", err)
	}
	data.Checksums["pq"] = s.calculateChecksum(data.PQJSONData)

	// Write ACL file
	aclFile := fmt.Sprintf("consul.acl.%s.json", startString)
	aclPath := filepath.Join(data.LocalPath, aclFile)
	if err := s.FileSystem.WriteFile(aclPath, data.ACLJSONData, 0644); err != nil {
		return fmt.Errorf("failed to write ACL file: %w", err)
	}
	data.Checksums["acl"] = s.calculateChecksum(data.ACLJSONData)

	// Write metadata
	if err := s.writeMetadata(data); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// CompressBackup creates a compressed archive
func (s *BackupService) CompressBackup(data *BackupData) error {
	startString := fmt.Sprintf("%v", data.StartTime)
	var filename string
	
	if s.Config.Acceptance {
		filename = "acceptancetest.tar.gz"
	} else {
		filename = fmt.Sprintf("%s.consul.snapshot.%s.tar.gz", s.Config.Hostname, startString)
	}
	
	archivePath := filepath.Join(s.Config.TmpDir, filename)
	sources := []string{data.LocalPath}
	
	if err := s.Archiver.TarGz(archivePath, sources); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	
	data.RemotePath = archivePath
	return nil
}

// UploadBackup uploads the backup to cloud storage
func (s *BackupService) UploadBackup(data *BackupData) error {
	if s.Config.Acceptance {
		s.Logger.Print("[INFO] Skipping remote backup during acceptance testing")
		return nil
	}

	if s.Storage == nil {
		return fmt.Errorf("no storage client configured")
	}

	// Read the archive file
	archiveData, err := s.FileSystem.ReadFile(data.RemotePath)
	if err != nil {
		return fmt.Errorf("failed to read archive file: %w", err)
	}

	// Generate remote path
	t := time.Unix(data.StartTime, 0)
	year, month, day := t.Year(), int(t.Month()), t.Day()
	
	prefix := s.Config.ObjectPrefix
	if prefix == "" {
		prefix = "backups"
	}
	
	filename := filepath.Base(data.RemotePath)
	remotePath := fmt.Sprintf("%s/%d/%d/%d/%s", prefix, year, month, day, filename)

	// Upload to storage
	var bucket string
	if s.Config.S3Bucket != "" {
		bucket = s.Config.S3Bucket
	} else if s.Config.GCSBucket != "" {
		bucket = s.Config.GCSBucket
	} else {
		return fmt.Errorf("no storage bucket configured")
	}

	if err := s.Storage.Upload(bucket, remotePath, archiveData); err != nil {
		return fmt.Errorf("failed to upload backup: %w", err)
	}

	s.Logger.Printf("[INFO] Uploaded backup to %s/%s", bucket, remotePath)
	return nil
}

// Cleanup removes temporary files
func (s *BackupService) Cleanup(data *BackupData) error {
	if data.LocalPath != "" {
		if err := s.FileSystem.RemoveAll(data.LocalPath); err != nil {
			s.Logger.Printf("[WARN] Failed to cleanup local path %s: %v", data.LocalPath, err)
		}
	}
	
	if data.RemotePath != "" && !s.Config.Acceptance {
		if err := s.FileSystem.Remove(data.RemotePath); err != nil {
			s.Logger.Printf("[WARN] Failed to cleanup archive file %s: %v", data.RemotePath, err)
		}
	}
	
	return nil
}

// RunBackup executes a complete backup workflow
func (s *BackupService) RunBackup() error {
	s.Logger.Printf("[INFO] Starting backup at: %d", time.Now().Unix())

	// Collect data from consul
	data, err := s.CollectData()
	if err != nil {
		return err
	}

	// Serialize to JSON
	if err := s.SerializeData(data); err != nil {
		return err
	}

	// Write local files
	if err := s.WriteLocalFiles(data); err != nil {
		return err
	}

	// Compress
	if err := s.CompressBackup(data); err != nil {
		return err
	}

	// Upload to cloud storage
	if err := s.UploadBackup(data); err != nil {
		return err
	}

	// Cleanup
	defer s.Cleanup(data)

	s.Logger.Print("[INFO] Backup completed successfully")
	return nil
}

// Helper functions
func (s *BackupService) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *BackupService) writeMetadata(data *BackupData) error {
	meta := map[string]interface{}{
		"consul_snapshot_version": s.Config.Version,
		"start_time":              data.StartTime,
		"end_time":                time.Now().Unix(),
		"node_name":               s.Config.Hostname,
		"kv_checksum":             data.Checksums["kv"],
		"pq_checksum":             data.Checksums["pq"],
		"acl_checksum":            data.Checksums["acl"],
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	metaPath := filepath.Join(data.LocalPath, "meta.json")
	return s.FileSystem.WriteFile(metaPath, metaJSON, 0644)
}