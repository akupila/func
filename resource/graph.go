package resource

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// A Graph represents a decoded resource graph.
type Graph struct {
	Resources map[string]Resource
}

// A Resource is the configuration for a resource.
type Resource struct {
	Type       string
	Definition hcl.Range
	SourceCode *SourceCode
	Config     interface{} // Shape depends on Type
	Refs       []Reference
}

// SourceCode contains the source code defined for a resource.
type SourceCode struct {
	Dir string
}

// A Reference represents a referenced value between two resources.
type Reference struct {
	Field      cty.Path
	Expression hcl.Expression
}
