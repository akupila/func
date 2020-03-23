package resource

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/func/func/source"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// List is a list of decoded resources.
type List []*Resource

// A Resource is the configuration for a resource.
type Resource struct {
	Name       string
	Type       string
	Definition hcl.Range
	SourceCode *SourceCode
	Config     interface{} // Shape depends on Type
	Refs       []Reference
}

// ByName returns a resource by name. The name must exactly match the resource
// name. If no such resource exists, nil is returned.
func (l List) ByName(name string) *Resource {
	for _, r := range l {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// OfType returns a new list of all resources that have a certain type.
func (l List) OfType(typename string) List {
	var out List
	for _, r := range l {
		if r.Type == typename {
			out = append(out, r)
		}
	}
	return out
}

// WithSource returns a list of resources that have source code.
func (l List) WithSource() List {
	var out List
	for _, r := range l {
		if r.SourceCode == nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

// A Reference represents a referenced value between two resources.
type Reference struct {
	Field      cty.Path
	Expression hcl.Expression
}

// SourceCode contains the source code defined for a resource.
type SourceCode struct {
	Definition hcl.Range
	Dir        string
}

// SourceFiles collects all source code files for the resource.
// If the source directory was set on the same directory as the resource config
// file, the config file itself is not included in the returned slice.
//
// Directories starting with . are ignored.
func (r *Resource) SourceFiles() (*source.FileList, error) {
	if r.SourceCode == nil {
		dir := filepath.Dir(r.Definition.Filename)
		return source.NewFileList(dir), nil
	}
	srcDir := r.SourceCode.Dir
	files := source.NewFileList(srcDir)
	if err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(path), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if path == r.Definition.Filename {
			// Skip config file
			return nil
		}
		rel, _ := filepath.Rel(srcDir, path)
		files.Add(rel)
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}
