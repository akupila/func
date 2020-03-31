package resource

import (
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
	SourceCode *source.Code
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
	out := make(List, 0, len(l))
	for _, r := range l {
		if r.Type == typename {
			out = append(out, r)
		}
	}
	return out
}

// WithSource returns a list of resources that have source code.
func (l List) WithSource() List {
	out := make(List, 0, len(l))
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
