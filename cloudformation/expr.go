package cloudformation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

type exprKind string

const (
	exprRef exprKind = "Ref"
	exprAtt exprKind = "Fn::GetAtt"
	exprSub exprKind = "Fn::Sub"
	exprLit exprKind = "literal"
)

type exprFn struct {
	Kind  exprKind
	Value string
}

func (e exprFn) MarshalJSON() ([]byte, error) {
	if e.Kind == exprLit {
		return []byte(fmt.Sprintf("%q", e.Value)), nil
	}
	return json.Marshal(map[string]string{
		string(e.Kind): e.Value,
	})
}

// configProvider returns the input configuration for the given resource. If
// the resource has not been defined, nil should be returned.
type configProvider interface {
	config(name string) interface{}
}

func convertExpr(expr hcl.Expression, configs configProvider) (*exprFn, hcl.Diagnostics) {
	expr = hcl.UnwrapExpression(expr)

	switch v := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		split := v.Traversal.SimpleSplit()
		parent := configs.config(split.RootName())
		if parent == nil {
			return nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   fmt.Sprintf("No resource named %q.", split.RootName()),
				Subject:  split.Abs.SourceRange().Ptr(),
			}}
		}
		parentType := reflect.Indirect(reflect.ValueOf(parent)).Type()
		cfname, otype := findCFName(parentType, split.Rel)
		switch otype {
		case outputRef:
			return &exprFn{
				Kind:  exprRef,
				Value: resourceName(split.RootName()),
			}, nil
		case outputAtt:
			return &exprFn{
				Kind:  exprAtt,
				Value: fmt.Sprintf("%s.%s", resourceName(split.RootName()), cfname),
			}, nil
		case notCFOutput:
			return nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   "CloudFormation does not support reading this field.",
				Subject:  split.Rel.SourceRange().Ptr(),
			}}
		default:
			return nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Invalid reference",
				Detail:   "No such output.",
				Subject:  split.Rel.SourceRange().Ptr(),
			}}
		}
	case *hclsyntax.TemplateWrapExpr:
		return convertExpr(v.Wrapped, configs)
	case *hclsyntax.TemplateExpr:
		var buf bytes.Buffer
		for _, p := range v.Parts {
			v, err := convertExpr(p, configs)
			if err != nil {
				return nil, err
			}
			if v.Kind != exprLit {
				buf.WriteString("${")
			}
			buf.WriteString(v.Value)
			if v.Kind != exprLit {
				buf.WriteString("}")
			}
		}
		return &exprFn{
			Kind:  exprSub,
			Value: buf.String(),
		}, nil
	case *hclsyntax.LiteralValueExpr:
		// A value in an expression is always a string or a number that can be
		// converted to a string; safe to ignore error.
		val, _ := convert.Convert(v.Val, cty.String)
		return &exprFn{
			Kind:  exprLit,
			Value: val.AsString(),
		}, nil
	default:
		// All possible types should be supported above.
		panic(fmt.Sprintf("cannot create ref for %T", expr))
	}
}

type outputType int

const (
	notOutput outputType = iota
	notCFOutput
	outputRef
	outputAtt
)

func matchesIO(tag reflect.StructTag, name string) bool {
	if strings.Split(tag.Get("input"), ",")[0] == name {
		return true
	}
	if strings.Split(tag.Get("output"), ",")[0] == name {
		return true
	}
	return false
}

func findCFName(typ reflect.Type, trav hcl.Traversal) (string, outputType) {
	attr := trav[0].(hcl.TraverseAttr)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			// Unexported
			continue
		}

		if !matchesIO(field.Tag, attr.Name) {
			continue
		}

		tag, ok := field.Tag.Lookup("cloudformation")
		if !ok {
			// Not CloudFormation field
			return "", notCFOutput
		}

		parts := strings.Split(tag, ",")

		if len(parts) <= 1 {
			// Neither ref or att
			return "", notCFOutput
		}

		name, typ := parts[0], parts[1]

		switch typ {
		case "ref":
			return name, outputRef
		case "att":
			return name, outputAtt
		default:
			panic(fmt.Sprintf("Invalid output %q", parts[1]))
		}
	}
	return "", notOutput
}
