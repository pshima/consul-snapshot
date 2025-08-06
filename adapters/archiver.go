package adapters

import (
	"github.com/mholt/archiver"
	"github.com/pshima/consul-snapshot/interfaces"
)

// ArchiverAdapter implements the Archiver interface
type ArchiverAdapter struct{}

// NewArchiverAdapter creates a new archiver adapter
func NewArchiverAdapter() interfaces.Archiver {
	return &ArchiverAdapter{}
}

// TarGz creates a tar.gz archive
func (a *ArchiverAdapter) TarGz(destination string, sources []string) error {
	tgz := archiver.NewTarGz()
	return tgz.Archive(sources, destination)
}

// UnTarGz extracts a tar.gz archive
func (a *ArchiverAdapter) UnTarGz(source, destination string) error {
	tgz := archiver.NewTarGz()
	return tgz.Unarchive(source, destination)
}