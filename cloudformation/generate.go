package cloudformation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/func/func/resource"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// A Template represents an AWS CloudFormation template.
type Template struct {
	AWSTemplateFormatVersion string              `json:"AWSTemplateFormatVersion"`
	Description              string              `json:"Description,omitempty"`
	Resources                map[string]Resource `json:"Resources,omitempty"`

	logicalMapping map[string]string // CloudFormation logical ID -> resource name
}

// A Resource is a CloudFormation encoded resource.
type Resource struct {
	Type       string                 `json:"Type"`
	Properties map[string]interface{} `json:"Properties,omitempty"`
}

// SupportedResource is implemented by resource configs that have a
// corresponding CloudFormation resource.
type SupportedResource interface {
	// Returns the CloudFormation type, such as "AWS::Lambda::Function".
	CloudFormationType() string
}

// SourceSetter is implemented by resources that support getting their
// source code from S3. If implemented, the resource's source code is
// processed, zipped and uploaded to S3. The key is then passed to the resource
// prior to encoding it.
//
// Implementing the interface makes the source code required. In case the user
// does not provide source code, error diagnostics are returned.
type SourceSetter interface {
	SetS3SourceCode(bucket, key string)
}

// Encoder can be implemented on fields that produce custom output for
// CloudFormation, for example json.
// Fields that implement Encoder are not allowed to have references on them. If
// the user provides a reference, error diagnostics are produced.
type Encoder interface {
	CloudFormation() (interface{}, error)
}

// S3Location is the location of a file in AWS S3.
type S3Location struct {
	Bucket string
	Key    string
}

// Generate generates a CloudFormation template from a resource graph.
//
// The source codes for all resources must be prepared in advanced and uploaded
// to S3. The corresponding locations should be provided in sources, keyed by
// resource name.
//
// For Lambda functions, the region of the S3 bucket must be in the same region
// as the Lambda function.
func Generate(resources resource.List, sources map[string]S3Location) (*Template, hcl.Diagnostics) {
	gen := &generator{
		Sources:   sources,
		Resources: resources,
	}
	return gen.Generate()
}

type generator struct {
	Sources   map[string]S3Location
	Resources resource.List
}

func (g *generator) Generate() (*Template, hcl.Diagnostics) {
	template := &Template{
		AWSTemplateFormatVersion: "2010-09-09",
		Resources:                make(map[string]Resource, len(g.Resources)),
		logicalMapping:           make(map[string]string, len(g.Resources)),
	}

	var diags hcl.Diagnostics
	for _, input := range g.Resources {
		res, morediags := g.processResource(input)
		diags = append(diags, morediags...)
		logicalName := resourceName(input.Name)
		template.Resources[logicalName] = res
		template.logicalMapping[logicalName] = input.Name
	}

	return template, diags
}

func (g *generator) processResource(input *resource.Resource) (Resource, hcl.Diagnostics) {
	t, ok := input.Config.(SupportedResource)
	if !ok {
		return Resource{}, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "Incompatible resource",
			Detail:   fmt.Sprintf("A CloudFormation resource cannot be generated from %s.", input.Type),
			Subject:  input.Definition.Ptr(),
		}}
	}

	res := Resource{
		Type: t.CloudFormationType(),
	}

	enc := &encoder{
		Resources: g.Resources,
		Refs:      input.Refs,
	}

	if s3src, ok := input.Config.(SourceSetter); ok {
		loc, ok := g.Sources[input.Name]
		if !ok {
			return Resource{}, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Source code not provided",
				Detail:   fmt.Sprintf("Source code must be provided for %s.", input.Type),
				Subject:  input.Definition.Ptr(),
			}}
		}
		s3src.SetS3SourceCode(loc.Bucket, loc.Key)
	}

	props, err := enc.Encode(reflect.ValueOf(input.Config), nil)
	if err != nil {
		var diags hcl.Diagnostics
		if errors.As(err, &diags) {
			return Resource{}, diags
		}
		return Resource{}, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "Could not create CloudFormation resource",
			Detail:   fmt.Sprintf("Encoding properties failed: %v", err),
			Subject:  input.Definition.Ptr(),
		}}
	}

	if pp, ok := props.(map[string]interface{}); ok {
		res.Properties = pp
	}

	return res, nil
}

type encoder struct {
	Resources resource.List
	Refs      []resource.Reference
}

func (e *encoder) ref(path cty.Path) (resource.Reference, bool) {
	for _, ref := range e.Refs {
		if ref.Field.Equals(path) {
			return ref, true
		}
	}
	return resource.Reference{}, false
}

func (e *encoder) Encode(value reflect.Value, path cty.Path) (interface{}, error) {
	ref, hasRef := e.ref(path)
	if enc, ok := value.Interface().(Encoder); ok {
		// Disallow references for fields that use a custom encoder.
		// This is because the output of the custom encoder is not known so
		// references cannot be replaced within it.
		if hasRef {
			return nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "References are not allowed here",
				Subject:  ref.Expression.Variables()[0].SourceRange().Ptr(),
				Context:  ref.Expression.Range().Ptr(),
			}}
		}

		v, err := enc.CloudFormation()
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	if hasRef {
		return e.makeRef(ref)
	}

	v := reflect.Indirect(value)
	t := v.Type()

	if v.IsZero() {
		// Omit empty
		return nil, nil
	}

	switch t.Kind() {
	case reflect.Struct:
		props := make(map[string]interface{})

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				// Unexported
				continue
			}

			if _, ok := field.Tag.Lookup("output"); ok {
				// Exclude outputs
				continue
			}

			cftag, ok := field.Tag.Lookup("cloudformation")
			if !ok {
				// Not CloudFormation field
				continue
			}

			fieldname := field.Tag.Get("input")
			fieldVal := v.Field(i)

			if fieldVal.Kind() == reflect.Ptr {
				if fieldVal.IsNil() {
					continue
				}
				fieldVal = fieldVal.Elem()
			}

			parts := strings.Split(cftag, ",")
			cfname := parts[0]

			val, err := e.Encode(fieldVal, path.GetAttr(fieldname))
			if err != nil {
				return nil, fmt.Errorf("encode %s: %w", field.Name, err)
			}
			if isEmpty(val) {
				continue
			}
			props[cfname] = val
		}
		return props, nil
	case reflect.Slice:
		// Iterate elements in slice, nested elements may have custom encoders
		// or references.
		n := v.Len()
		slice := reflect.MakeSlice(reflect.TypeOf([]interface{}{}), 0, n)
		for i := 0; i < n; i++ {
			val := v.Index(i)
			if val.Kind() == reflect.Ptr && val.IsNil() {
				continue
			}
			ev, err := e.Encode(val, path.Index(cty.NumberIntVal(int64(i))))
			if err != nil {
				return nil, err
			}
			slice = reflect.Append(slice, reflect.ValueOf(ev))
		}
		if slice.Len() == 0 {
			return nil, nil
		}
		return slice.Interface(), nil
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			panic(fmt.Errorf("map key must be a string, not %s", t.Key()))
		}
		// Iterate elements in map, nested elements may have custom encoders or
		// references.
		keys := v.MapKeys()
		if len(keys) == 0 {
			return nil, nil
		}
		mapVal := reflect.MakeMapWithSize(reflect.TypeOf(map[string]interface{}{}), len(keys))
		for _, key := range keys {
			keyStr := key.Interface().(string)
			val := v.MapIndex(key)
			if val.Kind() == reflect.Ptr && val.IsNil() {
				continue
			}
			mv, err := e.Encode(val, path.GetAttr(keyStr))
			if err != nil {
				return nil, err
			}
			mapVal.SetMapIndex(key, reflect.ValueOf(mv))
		}
		return mapVal.Interface(), nil
	default:
		return value.Interface(), nil
	}
}

func (e *encoder) makeRef(ref resource.Reference) (interface{}, error) {
	expr, diags := convertExpr(ref.Expression, e)
	if diags.HasErrors() {
		return nil, diags
	}
	return expr, nil
}

func (e *encoder) config(name string) interface{} {
	res := e.Resources.ByName(name)
	if res == nil {
		return nil
	}
	return res.Config
}

// LookupResource looks up a resource by logical name. The returned string is
// the user defined name. Returns an empty string if the resource does not
// exist.
func (t Template) LookupResource(logicalName string) string {
	return t.logicalMapping[logicalName]
}

func isEmpty(val interface{}) bool {
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return v.Len() == 0
	default:
		return false
	}
}
