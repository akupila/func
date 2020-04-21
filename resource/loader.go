package resource

import (
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
)

// A Loader is a convenience wrapper around the parser and decoder.
type Loader struct {
	Registry *Registry

	parser *Parser
}

// LoadDir loads the resource graph from a given directory and all sub
// directories. All .hcl files are parsed and decoded to the resulting graph.
func (l *Loader) LoadDir(dir string) (List, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	var files []string
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(info.Name()) == ".hcl" {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Could not read config files",
			Detail:   err.Error(),
		})
	}

	l.parser = &Parser{}
	for _, file := range files {
		diags = append(diags, l.parser.ParseHCLFile(file)...)
	}
	body := l.parser.Body()

	g, morediags := Decode(body, l.Registry)
	diags = append(diags, morediags...)

	return g, diags
}

// Files returns all the loaded files, keyed by file name.
func (l *Loader) Files() map[string]*hcl.File {
	return l.parser.Files()
}
