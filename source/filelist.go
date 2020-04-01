package source

import (
	"archive/zip"
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

// Write writes the contents of all files to the given writer. The files are
// processed in a deterministic order.
//
// This can be used to hash the contents of all files.
func (l FileList) Write(w io.Writer) error {
	sort.Strings(l.Files) // Ensure consistent order
	for _, name := range l.Files {
		f, err := os.Open(filepath.Join(l.Root, name))
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return err
		}
		_ = f.Close()
	}
	return nil
}

// Zip compresses the file list to a zip archive.
func (l FileList) Zip(w io.Writer) error {
	zf := zip.NewWriter(w)
	for _, f := range l.Files {
		if err := addFileToZip(zf, l.Root, f); err != nil {
			return err
		}
	}
	if err := zf.Close(); err != nil {
		return err
	}
	return nil
}

func addFileToZip(z *zip.Writer, root, filename string) error {
	f, err := os.Open(filepath.Join(root, filename))
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
