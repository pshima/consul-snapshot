package mocks

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
)

// MockConsulClient implements ConsulClient for testing
type MockConsulClient struct {
	KeyData       consulapi.KVPairs
	PQData        []*consulapi.PreparedQueryDefinition
	ACLData       []*consulapi.ACLEntry
	KeyError      error
	PQError       error
	ACLError      error
	ACLDisabled   bool
	PutKVError    error
	CreatePQError error
	CreateACLError error
}

// NewMockConsulClient creates a new mock consul client
func NewMockConsulClient() *MockConsulClient {
	return &MockConsulClient{}
}

// ListKeys returns mock key data
func (m *MockConsulClient) ListKeys() (consulapi.KVPairs, error) {
	if m.KeyError != nil {
		return nil, m.KeyError
	}
	return m.KeyData, nil
}

// ListPQs returns mock prepared query data
func (m *MockConsulClient) ListPQs() ([]*consulapi.PreparedQueryDefinition, error) {
	if m.PQError != nil {
		return nil, m.PQError
	}
	return m.PQData, nil
}

// ListACLs returns mock ACL data
func (m *MockConsulClient) ListACLs() ([]*consulapi.ACLEntry, error) {
	if m.ACLError != nil {
		return nil, m.ACLError
	}
	if m.ACLDisabled {
		return []*consulapi.ACLEntry{}, nil
	}
	return m.ACLData, nil
}

// PutKV mocks putting a key-value pair
func (m *MockConsulClient) PutKV(key string, value []byte) error {
	if m.PutKVError != nil {
		return m.PutKVError
	}
	// Add to mock data
	m.KeyData = append(m.KeyData, &consulapi.KVPair{Key: key, Value: value})
	return nil
}

// CreatePQ mocks creating a prepared query
func (m *MockConsulClient) CreatePQ(pq *consulapi.PreparedQueryDefinition) error {
	if m.CreatePQError != nil {
		return m.CreatePQError
	}
	m.PQData = append(m.PQData, pq)
	return nil
}

// CreateACL mocks creating an ACL
func (m *MockConsulClient) CreateACL(acl *consulapi.ACLEntry) error {
	if m.CreateACLError != nil {
		return m.CreateACLError
	}
	m.ACLData = append(m.ACLData, acl)
	return nil
}

// MockStorageClient implements StorageClient for testing
type MockStorageClient struct {
	Data        map[string][]byte
	UploadError error
	DownloadError error
	UploadCalls []UploadCall
	DownloadCalls []DownloadCall
}

type UploadCall struct {
	Bucket string
	Key    string
	Data   []byte
}

type DownloadCall struct {
	Bucket string
	Key    string
}

// NewMockStorageClient creates a new mock storage client
func NewMockStorageClient() *MockStorageClient {
	return &MockStorageClient{
		Data: make(map[string][]byte),
	}
}

// Upload mocks uploading data
func (m *MockStorageClient) Upload(bucket, key string, data []byte) error {
	m.UploadCalls = append(m.UploadCalls, UploadCall{Bucket: bucket, Key: key, Data: data})
	if m.UploadError != nil {
		return m.UploadError
	}
	m.Data[bucket+"/"+key] = data
	return nil
}

// Download mocks downloading data
func (m *MockStorageClient) Download(bucket, key string) ([]byte, error) {
	m.DownloadCalls = append(m.DownloadCalls, DownloadCall{Bucket: bucket, Key: key})
	if m.DownloadError != nil {
		return nil, m.DownloadError
	}
	data, exists := m.Data[bucket+"/"+key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s/%s", bucket, key)
	}
	return data, nil
}

// MockFileSystem implements FileSystem for testing
type MockFileSystem struct {
	Files         map[string][]byte
	Dirs          map[string]bool
	WriteError    error
	ReadError     error
	MkdirError    error
	RemoveError   error
	WriteCalls    []WriteCall
	ReadCalls     []string
	MkdirCalls    []MkdirCall
	RemoveCalls   []string
}

type WriteCall struct {
	Filename string
	Data     []byte
	Perm     int
}

type MkdirCall struct {
	Path string
	Perm int
}

// NewMockFileSystem creates a new mock filesystem
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		Files: make(map[string][]byte),
		Dirs:  make(map[string]bool),
	}
}

// WriteFile mocks writing a file
func (m *MockFileSystem) WriteFile(filename string, data []byte, perm int) error {
	m.WriteCalls = append(m.WriteCalls, WriteCall{Filename: filename, Data: data, Perm: perm})
	if m.WriteError != nil {
		return m.WriteError
	}
	m.Files[filename] = data
	return nil
}

// ReadFile mocks reading a file
func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	m.ReadCalls = append(m.ReadCalls, filename)
	if m.ReadError != nil {
		return nil, m.ReadError
	}
	data, exists := m.Files[filename]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", filename)
	}
	return data, nil
}

// MkdirAll mocks creating directories
func (m *MockFileSystem) MkdirAll(path string, perm int) error {
	m.MkdirCalls = append(m.MkdirCalls, MkdirCall{Path: path, Perm: perm})
	if m.MkdirError != nil {
		return m.MkdirError
	}
	m.Dirs[path] = true
	return nil
}

// Remove mocks removing a file
func (m *MockFileSystem) Remove(path string) error {
	m.RemoveCalls = append(m.RemoveCalls, path)
	if m.RemoveError != nil {
		return m.RemoveError
	}
	delete(m.Files, path)
	return nil
}

// RemoveAll mocks removing a directory
func (m *MockFileSystem) RemoveAll(path string) error {
	return m.Remove(path) // Simplified for testing
}

// MockArchiver implements Archiver for testing
type MockArchiver struct {
	TarGzError   error
	UnTarGzError error
	TarGzCalls   []TarGzCall
	UnTarGzCalls []UnTarGzCall
}

type TarGzCall struct {
	Destination string
	Sources     []string
}

type UnTarGzCall struct {
	Source      string
	Destination string
}

// NewMockArchiver creates a new mock archiver
func NewMockArchiver() *MockArchiver {
	return &MockArchiver{}
}

// TarGz mocks creating a tar.gz archive
func (m *MockArchiver) TarGz(destination string, sources []string) error {
	m.TarGzCalls = append(m.TarGzCalls, TarGzCall{Destination: destination, Sources: sources})
	return m.TarGzError
}

// UnTarGz mocks extracting a tar.gz archive
func (m *MockArchiver) UnTarGz(source, destination string) error {
	m.UnTarGzCalls = append(m.UnTarGzCalls, UnTarGzCall{Source: source, Destination: destination})
	return m.UnTarGzError
}

// MockLogger implements Logger for testing
type MockLogger struct {
	LogEntries   []LogEntry
	ShouldFatal  bool
	FatalCalled  bool
}

type LogEntry struct {
	Level   string
	Format  string
	Args    []interface{}
}

// NewMockLogger creates a new mock logger
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

// Printf mocks formatted logging
func (m *MockLogger) Printf(format string, args ...interface{}) {
	m.LogEntries = append(m.LogEntries, LogEntry{Level: "printf", Format: format, Args: args})
}

// Print mocks logging
func (m *MockLogger) Print(args ...interface{}) {
	m.LogEntries = append(m.LogEntries, LogEntry{Level: "print", Args: args})
}

// Fatalf mocks fatal logging with format
func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.LogEntries = append(m.LogEntries, LogEntry{Level: "fatalf", Format: format, Args: args})
	m.FatalCalled = true
	if m.ShouldFatal {
		panic(fmt.Sprintf(format, args...))
	}
}

// Fatal mocks fatal logging
func (m *MockLogger) Fatal(args ...interface{}) {
	m.LogEntries = append(m.LogEntries, LogEntry{Level: "fatal", Args: args})
	m.FatalCalled = true
	if m.ShouldFatal {
		panic(fmt.Sprint(args...))
	}
}