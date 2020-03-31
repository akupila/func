package source

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// A FileList contains a list of source files.
type FileList struct {
	Root  string   // Root directory.
	Files []string // Files relative to root directory.
}

// NewFileList creates a new file list with the given root directory.
func NewFileList(root string) *FileList {
	return &FileList{
		Root: root,
	}
}

// Add adds a new path to the file list. The caller is responsible for ensuring
// the file exists. The path must be relative to the root of the file list.
func (fl *FileList) Add(path string) {
	if filepath.IsAbs(path) {
		panic("Path must be relative to root")
	}
	fl.Files = append(fl.Files, path)
}

// Checksum computes a checksum of the contents of all the files.
func (fl *FileList) Checksum() (string, error) {
	s := sha256.New()
	sort.Strings(fl.Files) // Ensure consistent order
	for _, name := range fl.Files {
		f, err := os.Open(filepath.Join(fl.Root, name))
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(s, f); err != nil {
			_ = f.Close()
			return "", err
		}
		_ = f.Close()
	}
	return hex.EncodeToString(s.Sum(nil)), nil
}

// Zip compresses the file list to a zip archive.
func (fl *FileList) Zip(w io.Writer) error {
	zf := zip.NewWriter(w)
	for _, f := range fl.Files {
		if err := fl.addFileToZip(zf, f); err != nil {
			return err
		}
	}
	if err := zf.Close(); err != nil {
		return err
	}
	return nil
}

func (fl *FileList) addFileToZip(z *zip.Writer, filename string) error {
	f, err := os.Open(filepath.Join(fl.Root, filename))
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filename
	header.Method = zip.Deflate
	w, err := z.CreateHeader(header)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, f); err != nil {
		return err
	}
	return nil
}
