package interfaces

import (
	consulapi "github.com/hashicorp/consul/api"
)

// ConsulClient interface for mocking consul operations
type ConsulClient interface {
	ListKeys() (consulapi.KVPairs, error)
	ListPQs() ([]*consulapi.PreparedQueryDefinition, error)
	ListACLs() ([]*consulapi.ACLEntry, error)
	PutKV(key string, value []byte) error
	CreatePQ(pq *consulapi.PreparedQueryDefinition) error
	CreateACL(acl *consulapi.ACLEntry) error
}

// StorageClient interface for mocking cloud storage operations
type StorageClient interface {
	Upload(bucket, key string, data []byte) error
	Download(bucket, key string) ([]byte, error)
}

// FileSystem interface for mocking file operations
type FileSystem interface {
	WriteFile(filename string, data []byte, perm int) error
	ReadFile(filename string) ([]byte, error)
	MkdirAll(path string, perm int) error
	Remove(path string) error
	RemoveAll(path string) error
}

// Archiver interface for mocking archive operations
type Archiver interface {
	TarGz(destination string, sources []string) error
	UnTarGz(source, destination string) error
}

// Logger interface for mocking logging
type Logger interface {
	Printf(format string, args ...interface{})
	Print(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatal(args ...interface{})
}