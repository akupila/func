package source

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// ArchiveFormat defines the format of a source archive.
type ArchiveFormat string

const (
	// Zip compresses the archive to a zip file.
	Zip ArchiveFormat = "zip"
)

// An Archive represents an archive for source code.
type Archive struct {
	ResourceName string

	format   ArchiveFormat
	files    *FileList
	checksum string
	key      string
}

// NewArchive creates a new source archive.
func NewArchive(files *FileList, format ArchiveFormat) (*Archive, error) {
	sum, err := files.Checksum()
	if err != nil {
		return nil, fmt.Errorf("compute checksum: %w", err)
	}
	return &Archive{
		format:   format,
		files:    files,
		checksum: sum,
		key:      fmt.Sprintf("%s.%s", sum, format),
	}, nil
}

// Extension returns the file extension for the archive. The extension is
// returned without a dot (such as "zip").
func (a Archive) Extension() string {
	switch a.format {
	case Zip:
		return "zip"
	default:
		return "invalid"
	}
}

// FileName returns a complete file name for the archive.
func (a Archive) FileName() string {
	return fmt.Sprintf("%s.%s", a.checksum, a.Extension())
}

// Write writes archive to the given writer.
func (a *Archive) Write(w io.Writer) error {
	switch a.format {
	case Zip:
		return a.files.Zip(w)
	default:
		return fmt.Errorf("invalid format %q", a.format)
	}
}

// File writes the archive to a temporary file on disk and returns the file
// pointer to it. The file is open with the offset at the beginning of the
// file.
func (a *Archive) File() (*os.File, error) {
	f, err := ioutil.TempFile("", fmt.Sprintf("*.%s", a.Extension()))
	if err != nil {
		return nil, fmt.Errorf("create zip file: %w", err)
	}
	if err := a.Write(f); err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}
	if err := f.Sync(); err != nil {
		return nil, fmt.Errorf("sync: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}
	return f, nil
}
