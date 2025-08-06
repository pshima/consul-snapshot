package adapters

import (
	"io/ioutil"
	"os"

	"github.com/pshima/consul-snapshot/interfaces"
)

// FileSystemAdapter implements the FileSystem interface
type FileSystemAdapter struct{}

// NewFileSystemAdapter creates a new filesystem adapter
func NewFileSystemAdapter() interfaces.FileSystem {
	return &FileSystemAdapter{}
}

// WriteFile writes data to a file
func (f *FileSystemAdapter) WriteFile(filename string, data []byte, perm int) error {
	return ioutil.WriteFile(filename, data, os.FileMode(perm))
}

// ReadFile reads data from a file
func (f *FileSystemAdapter) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

// MkdirAll creates directories
func (f *FileSystemAdapter) MkdirAll(path string, perm int) error {
	return os.MkdirAll(path, os.FileMode(perm))
}

// Remove removes a file
func (f *FileSystemAdapter) Remove(path string) error {
	return os.Remove(path)
}

// RemoveAll removes a directory and all its contents
func (f *FileSystemAdapter) RemoveAll(path string) error {
	return os.RemoveAll(path)
}