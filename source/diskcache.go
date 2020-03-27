package source

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiskCache caches source code on disk.
type DiskCache struct {
	Dir string
}

// NewDiskCache creates a new cache on disk. The cached data is stored within a
// func directory in the user's cache directory. The path is created if needed.
func NewDiskCache() (*DiskCache, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get user cache dir: %w", err)
	}
	dir := filepath.Join(cacheDir, "func")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("ensure application specific dir: %w", err)
	}
	return &DiskCache{Dir: dir}, nil
}

// Create creates a new file in cache. If the file already exists, it is truncated.
func (d *DiskCache) Create(filename string) (*os.File, error) {
	return os.Create(filepath.Join(d.Dir, filename))
}

// Get returns a pointer to a file in the cache. Returns nil if the file does
// not exist.
func (d *DiskCache) Get(filename string) *os.File {
	f, err := os.Open(filepath.Join(d.Dir, filename))
	if err != nil {
		return nil
	}
	return f
}
