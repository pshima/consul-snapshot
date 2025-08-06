package services

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/pshima/consul-snapshot/config"
	"github.com/pshima/consul-snapshot/consul"
	"github.com/pshima/consul-snapshot/interfaces"
)

// RestoreService handles restore operations with dependency injection
type RestoreService struct {
	Config     *config.Config
	Consul     *consul.Consul
	Storage    interfaces.StorageClient
	FileSystem interfaces.FileSystem
	Archiver   interfaces.Archiver
	Logger     interfaces.Logger
}

// NewRestoreService creates a new restore service
func NewRestoreService(config *config.Config, consul *consul.Consul, storage interfaces.StorageClient, fs interfaces.FileSystem, archiver interfaces.Archiver, logger interfaces.Logger) *RestoreService {
	return &RestoreService{
		Config:     config,
		Consul:     consul,
		Storage:    storage,
		FileSystem: fs,
		Archiver:   archiver,
		Logger:     logger,
	}
}

// RestoreData represents the data being restored
type RestoreData struct {
	RestorePath   string
	LocalPath     string
	ExtractedPath string
	KVData        consulapi.KVPairs
	PQData        []*consulapi.PreparedQueryDefinition
	ACLData       []*consulapi.ACLEntry
	Metadata      map[string]interface{}
}

// DownloadBackup downloads backup from cloud storage
func (s *RestoreService) DownloadBackup(restorePath string) (*RestoreData, error) {
	data := &RestoreData{
		RestorePath: restorePath,
	}

	if s.Config.Acceptance {
		// In acceptance mode, use local file
		data.LocalPath = filepath.Join(s.Config.TmpDir, "acceptancetest.tar.gz")
		return data, nil
	}

	// Download from cloud storage
	var bucket string
	if s.Config.S3Bucket != "" {
		bucket = s.Config.S3Bucket
	} else if s.Config.GCSBucket != "" {
		bucket = s.Config.GCSBucket
	} else {
		return nil, fmt.Errorf("no storage bucket configured")
	}

	s.Logger.Printf("[INFO] Downloading %s from %s", restorePath, bucket)
	
	backupData, err := s.Storage.Download(bucket, restorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to download backup: %w", err)
	}

	// Write to local file
	filename := filepath.Base(restorePath)
	data.LocalPath = filepath.Join(s.Config.TmpDir, filename)
	
	if err := s.FileSystem.WriteFile(data.LocalPath, backupData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write local backup file: %w", err)
	}

	s.Logger.Print("[INFO] Download completed")
	return data, nil
}

// ExtractBackup extracts the backup archive
func (s *RestoreService) ExtractBackup(data *RestoreData) error {
	s.Logger.Print("[INFO] Extracting backup")
	
	// Create extraction directory
	data.ExtractedPath = data.LocalPath + ".extracted"
	if err := s.FileSystem.MkdirAll(data.ExtractedPath, 0755); err != nil {
		return fmt.Errorf("failed to create extraction directory: %w", err)
	}

	// Extract archive
	if err := s.Archiver.UnTarGz(data.LocalPath, data.ExtractedPath); err != nil {
		return fmt.Errorf("failed to extract backup: %w", err)
	}

	return nil
}

// LoadMetadata loads and validates backup metadata
func (s *RestoreService) LoadMetadata(data *RestoreData) error {
	s.Logger.Print("[INFO] Inspecting backup contents")
	
	metaPath := filepath.Join(data.ExtractedPath, "meta.json")
	metaContent, err := s.FileSystem.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	if err := json.Unmarshal(metaContent, &data.Metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Log metadata info
	if version, ok := data.Metadata["consul_snapshot_version"].(string); ok {
		s.Logger.Printf("[INFO] Found backup version: %s", version)
	}
	if startTime, ok := data.Metadata["start_time"].(float64); ok {
		s.Logger.Printf("[INFO] Backup timestamp: %.0f", startTime)
	}

	return nil
}

// LoadBackupData loads the actual backup data from JSON files
func (s *RestoreService) LoadBackupData(data *RestoreData) error {
	// This is simplified - in real implementation you'd scan the directory
	// For now, assume standard naming convention
	if startTime, ok := data.Metadata["start_time"].(float64); ok {
		timestamp := fmt.Sprintf("%.0f", startTime)
		
		// Load KV data
		kvFile := filepath.Join(data.ExtractedPath, fmt.Sprintf("consul.kv.%s.json", timestamp))
		if kvData, err := s.FileSystem.ReadFile(kvFile); err == nil {
			if err := json.Unmarshal(kvData, &data.KVData); err != nil {
				return fmt.Errorf("failed to parse KV data: %w", err)
			}
			s.Logger.Printf("[INFO] Loaded %d keys", len(data.KVData))
		}

		// Load PQ data
		pqFile := filepath.Join(data.ExtractedPath, fmt.Sprintf("consul.pq.%s.json", timestamp))
		if pqData, err := s.FileSystem.ReadFile(pqFile); err == nil {
			if err := json.Unmarshal(pqData, &data.PQData); err != nil {
				return fmt.Errorf("failed to parse PQ data: %w", err)
			}
			s.Logger.Printf("[INFO] Loaded %d prepared queries", len(data.PQData))
		}

		// Load ACL data
		aclFile := filepath.Join(data.ExtractedPath, fmt.Sprintf("consul.acl.%s.json", timestamp))
		if aclData, err := s.FileSystem.ReadFile(aclFile); err == nil {
			if err := json.Unmarshal(aclData, &data.ACLData); err != nil {
				return fmt.Errorf("failed to parse ACL data: %w", err)
			}
			s.Logger.Printf("[INFO] Loaded %d ACLs", len(data.ACLData))
		}
	}

	return nil
}

// RestoreToConsul restores the data to consul
func (s *RestoreService) RestoreToConsul(data *RestoreData) error {
	errorCount := 0

	// Restore KV data
	if len(data.KVData) > 0 {
		s.Logger.Printf("[INFO] Restoring %d keys", len(data.KVData))
		if err := s.Consul.RestoreKeys(data.KVData); err != nil {
			s.Logger.Printf("[ERR] Failed to restore keys: %v", err)
			errorCount++
		}
	}

	// Restore PQ data
	if len(data.PQData) > 0 {
		s.Logger.Printf("[INFO] Restoring %d prepared queries", len(data.PQData))
		if err := s.Consul.RestorePQs(data.PQData); err != nil {
			s.Logger.Printf("[ERR] Failed to restore prepared queries: %v", err)
			errorCount++
		}
	}

	// Restore ACL data
	if len(data.ACLData) > 0 {
		s.Logger.Printf("[INFO] Restoring %d ACLs", len(data.ACLData))
		if err := s.Consul.RestoreACLs(data.ACLData); err != nil {
			s.Logger.Printf("[ERR] Failed to restore ACLs: %v", err)
			errorCount++
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("restore completed with %d errors", errorCount)
	}

	return nil
}

// Cleanup removes temporary files
func (s *RestoreService) Cleanup(data *RestoreData) error {
	if data.LocalPath != "" {
		if err := s.FileSystem.Remove(data.LocalPath); err != nil {
			s.Logger.Printf("[WARN] Failed to cleanup local file %s: %v", data.LocalPath, err)
		}
	}
	
	if data.ExtractedPath != "" {
		if err := s.FileSystem.RemoveAll(data.ExtractedPath); err != nil {
			s.Logger.Printf("[WARN] Failed to cleanup extracted path %s: %v", data.ExtractedPath, err)
		}
	}
	
	return nil
}

// RunRestore executes a complete restore workflow
func (s *RestoreService) RunRestore(restorePath string) error {
	s.Logger.Printf("[INFO] Starting restore of %s", restorePath)

	// Download backup
	data, err := s.DownloadBackup(restorePath)
	if err != nil {
		return err
	}

	// Cleanup on exit
	defer s.Cleanup(data)

	// Extract backup
	if err := s.ExtractBackup(data); err != nil {
		return err
	}

	// Load metadata
	if err := s.LoadMetadata(data); err != nil {
		return err
	}

	// Load backup data
	if err := s.LoadBackupData(data); err != nil {
		return err
	}

	// Restore to consul
	if err := s.RestoreToConsul(data); err != nil {
		return err
	}

	s.Logger.Print("[INFO] Restore completed successfully")
	return nil
}