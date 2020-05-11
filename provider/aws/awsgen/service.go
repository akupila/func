package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// A Service is a definition for a service in botocore.
type Service struct {
	Metadata   Metadata
	Operations []Operation
}

// Metadata contains service meta data.
type Metadata struct {
	APIVersion          string
	EndpointPrefix      string
	SigningName         string
	ServiceAbbreviation string
	ServiceFullName     string
	SignatureVersion    string
	JSONVersion         string
	TargetPrefix        string
	Protocol            string
	UID                 string
	EndpointsID         string
	ServiceID           string
}

// An Operation is a supported operation for a service.
type Operation struct {
	Name       string
	HTTP       HTTPInfo
	Doc        *Doc
	Deprecated bool
	Input      Struct
	Output     Struct
	Errors     []Struct
}

func (op Operation) Ptr() *Operation {
	return &op
}

// HTTPInfo contains http request info for executing an operation.
type HTTPInfo struct {
	Method       string
	RequestURI   string
	ResponseCode int
}

// ParseServiceID parses the service id from a service.
func ParseServiceID(r io.Reader) (string, error) {
	var s struct {
		Metadata struct {
			ServiceID string
		}
	}
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return "", err
	}
	return s.Metadata.ServiceID, nil
}

// ParseService parses a botocore service model definition and resolves nested
// references.
func ParseService(r io.Reader) (*Service, error) {
	d := json.NewDecoder(r)
	svc := &serviceModel{}
	if err := d.Decode(svc); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	resolver := &resolver{
		Model: svc,
	}

	ops := make([]Operation, 0, len(svc.Operations))
	for name, op := range svc.Operations {
		ops = append(ops, resolver.resolveOp(name, op))
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].Name < ops[j].Name
	})

	return &Service{
		Metadata:   svc.Metadata,
		Operations: ops,
	}, nil
}

type resolver struct {
	Model *serviceModel

	structs map[string]*Struct
}

func (r *resolver) resolveOp(name string, model operationModel) Operation {
	op := Operation{
		Name:       name,
		HTTP:       model.HTTP,
		Deprecated: model.Deprecated,
		Doc:        ParseDoc(model.Documentation),
	}

	if s := r.resolveShape(model.Input); s != nil {
		op.Input = s.(Struct)
	}
	if s := r.resolveShape(model.Output); s != nil {
		op.Output = s.(Struct)
	}
	op.Errors = make([]Struct, len(model.Errors))
	for i, e := range model.Errors {
		op.Errors[i] = r.resolveShape(e).(Struct)
	}

	return op
}

func (r *resolver) resolveShape(ref *shapeRef) Type {
	if ref == nil {
		return nil
	}
	shape, ok := r.Model.Shapes[ref.Shape]
	if !ok {
		panic("shape not defined: " + ref.Shape)
	}

	switch shape.Type {
	case "structure":
		if s, ok := r.structs[ref.Shape]; ok {
			return *s
		}

		s := make(Struct, 0, len(shape.Members))
		if r.structs == nil {
			r.structs = make(map[string]*Struct, 1)
		}
		r.structs[ref.Shape] = &s

		for name, member := range shape.Members {
			typ := r.resolveShape(member)
			s = append(s, Field{
				Name:     name,
				Doc:      ParseDoc(member.Documentation),
				Required: shape.IsRequired(name),
				Type:     typ,
			})
		}
		sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
		return s
	case "string":
		return String{
			Enum:    shape.Enum,
			MinLen:  shape.MinInt(),
			MaxLen:  shape.MaxInt(),
			Pattern: shape.Pattern,
		}
	case "integer", "long":
		return Int{
			Min: shape.MinInt64(),
			Max: shape.MaxInt64(),
		}
	case "double", "float":
		return Float{
			Min: shape.MinFloat64(),
			Max: shape.MaxFloat64(),
		}
	case "boolean":
		return Bool{}
	case "list":
		member := r.resolveShape(shape.Member)
		v := List{
			MinItems: shape.MinInt(),
			MaxItems: shape.MaxInt(),
			Element:  member,
		}
		return v
	case "blob":
		return Binary{}
	case "map":
		return Map{
			MinItems: shape.MinInt(),
			MaxItems: shape.MaxInt(),
			Key:      r.resolveShape(shape.Key),
			Value:    r.resolveShape(shape.Value),
		}
	case "timestamp":
		return Timestamp{}
	default:
		panic("Unsupported: " + shape.Type)
	}
}

type serviceModel struct {
	Version    string
	Metadata   Metadata
	Operations map[string]operationModel
	Shapes     map[string]shapeModel
}

type operationModel struct {
	Name          string
	HTTP          HTTPInfo
	Input         *shapeRef
	Output        *shapeRef
	Errors        []*shapeRef
	Documentation string
	Deprecated    bool
}

type shapeRef struct {
	Shape         string
	Documentation string
	Location      string
	LocationName  string
}

type shapeModel struct {
	Type          string
	Documentation string

	// Struct
	Required []string
	Members  map[string]*shapeRef

	// List
	Member *shapeRef

	// Map
	Key   *shapeRef
	Value *shapeRef

	// Validation
	Min     *float64
	Max     *float64
	Pattern *string

	Enum []string
}

func (m shapeModel) IsRequired(member string) bool {
	for _, req := range m.Required {
		if req == member {
			return true
		}
	}
	return false
}

func (m shapeModel) MinInt() *int {
	if m.Min == nil {
		return nil
	}
	v := int(*m.Min)
	return &v
}

func (m shapeModel) MaxInt() *int {
	if m.Max == nil {
		return nil
	}
	v := int(*m.Max)
	return &v
}

func (m shapeModel) MinInt64() *int64 {
	if m.Min == nil {
		return nil
	}
	v := int64(*m.Min)
	return &v
}

func (m shapeModel) MaxInt64() *int64 {
	if m.Max == nil {
		return nil
	}
	v := int64(*m.Max)
	return &v
}

func (m shapeModel) MinFloat64() *float64 {
	return m.Min
}

func (m shapeModel) MaxFloat64() *float64 {
	return m.Max
}
