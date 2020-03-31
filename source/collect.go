package source

import (
	"os"
	"path/filepath"
)

// A Filter filters source files prior to adding them to a file list.
//
// Filter is called with paths relative to the root.
type Filter func(path string) bool

// ExcludeFile excludes a file from the source file list.
//
// The path should be relative to the root.
func ExcludeFile(filename string) Filter {
	return func(path string) bool { return path != filename }
}

// ExcludeHidden excludes files and directories that start with a dot.
func ExcludeHidden() Filter {
	return func(path string) bool {
		return filepath.Base(path)[0] != '.'
	}
}

// Collect collects all files into a file list from the given directory.
//
// If any filter returns false, the file is excluded. If the filter returns
// false for a directory, all files within the directory are excluded.
func Collect(dir string, filters ...Filter) (*FileList, error) {
	var files []string
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if dir == path {
			// Skip self.
			// If we don't include this, rel will return . for the current
			// path, which will match the hidden filter.
			return nil
		}
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		for _, filter := range filters {
			if !filter(rel) {
				if info.IsDir() {
					// Filter matched dir, skip contents
					return filepath.SkipDir
				}
				// Skip file
				return nil
			}
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, rel)
		return nil
	}); err != nil {
		return nil, err
	}

	return &FileList{
		Root:  dir,
		Files: files,
	}, nil
}
