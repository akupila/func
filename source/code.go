package source

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// Code holds the source code defined for a resource.
type Code struct {
	Files *FileList
	Build BuildScript
}

// Checksum computes the checksum of the source code.
//
// The checksum is based on the contents of all the source files and the build
// script. Files are processed in lexicographical order. File names do not
// affect the result, except possibly changing the order the files are
// processed.
func (c *Code) Checksum() (string, error) {
	sha := sha256.New()
	for _, f := range c.Files.Absolute() {
		f, err := os.Open(f)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(sha, f); err != nil {
			_ = f.Close()
			return "", err
		}
		_ = f.Close()
	}
	if _, err := sha.Write([]byte(c.Build.String())); err != nil {
		return "", err
	}
	return hex.EncodeToString(sha.Sum(nil)), nil
}
