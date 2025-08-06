package adapters

import (
	"context"
	"io"
	"os"
	"path/filepath"
	
	"github.com/mholt/archives"
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
	// Create output file
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()
	
	// Map source files for archiving
	ctx := context.Background()
	fileMap := make(map[string]string)
	for _, source := range sources {
		fileMap[source] = "" // Add to root of archive
	}
	
	files, err := archives.FilesFromDisk(ctx, nil, fileMap)
	if err != nil {
		return err
	}
	
	// Create compressed tar.gz archive
	format := archives.CompressedArchive{
		Compression: archives.Gz{},
		Archival:    archives.Tar{},
	}
	
	return format.Archive(ctx, out, files)
}

// UnTarGz extracts a tar.gz archive
func (a *ArchiverAdapter) UnTarGz(source, destination string) error {
	// Open the compressed archive
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Create decompressor for gzip
	decompressor := archives.Gz{}
	decompressed, err := decompressor.OpenReader(file)
	if err != nil {
		return err
	}
	defer decompressed.Close()
	
	// Extract tar archive
	ctx := context.Background()
	return archives.Tar{}.Extract(ctx, decompressed, func(ctx context.Context, f archives.FileInfo) error {
		// Create the full path for the file
		fullPath := filepath.Join(destination, f.NameInArchive)
		
		// Create directory if needed
		if f.IsDir() {
			return os.MkdirAll(fullPath, 0755)
		}
		
		// Ensure parent directory exists
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		
		// Create and write file
		out, err := os.Create(fullPath)
		if err != nil {
			return err
		}
		defer out.Close()
		
		// Open file from archive and copy contents
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		
		_, err = io.Copy(out, rc)
		return err
	})
}