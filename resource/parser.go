package resource

import (
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// A Parser loads resource configurations from disk.
type Parser struct {
	parser *hclparse.Parser
}

// Body returns a merged body of all loaded files.
func (p *Parser) Body() hcl.Body {
	if p.parser == nil {
		return hcl.EmptyBody()
	}
	files := p.parser.Files()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	list := make([]*hcl.File, len(names))
	for i, name := range names {
		list[i] = files[name]
	}
	return hcl.MergeFiles(list)
}

// ParseHCLFile parses a HCL configuration from the given file.
func (p *Parser) ParseHCLFile(filename string) hcl.Diagnostics {
	if p.parser == nil {
		p.parser = hclparse.NewParser()
	}
	_, diags := p.parser.ParseHCLFile(filename)
	return diags
}

// Files returns a map of all loaded files, keyed by file name.
func (p *Parser) Files() map[string]*hcl.File {
	if p.parser == nil {
		return nil
	}
	return p.parser.Files()
}
