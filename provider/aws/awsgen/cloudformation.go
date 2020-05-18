package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type CloudFormationSpec struct {
	Version   string
	Resources []CloudFormationResource
}

type CloudFormationResource struct {
	Type          string
	Documentation string
	Properties    Struct
	Attributes    Struct
}

func LoadCloudFormation(loc string) (*CloudFormationSpec, error) {
	rc, err := getSpec(loc)
	if err != nil {
		return nil, fmt.Errorf("Get spec: %w", err)
	}
	defer rc.Close()

	model := &cfspec{}
	if err := json.NewDecoder(rc).Decode(model); err != nil {
		return nil, err
	}

	spec := &CloudFormationSpec{
		Version:   model.ResourceSpecificationVersion,
		Resources: make([]CloudFormationResource, 0, len(model.ResourceTypes)),
	}
	for name, res := range model.ResourceTypes {
		spec.Resources = append(spec.Resources, CloudFormationResource{
			Type:          name,
			Documentation: res.Documentation,
			Properties:    model.resolveStruct(name, "props", res.Properties),
			Attributes:    model.resolveStruct(name, "attrs", res.Attributes),
		})
	}
	sort.Slice(spec.Resources, func(i, j int) bool {
		return spec.Resources[i].Type < spec.Resources[j].Type
	})
	return spec, nil
}

func getSpec(src string) (io.ReadCloser, error) {
	u, err := url.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	switch u.Scheme {
	case "http", "https":
		resp, err := http.Get(src)
		if err != nil {
			return nil, fmt.Errorf("download: %w", err)
		}
		return resp.Body, nil
	case "", "file":
		f := filepath.Join(u.Host, u.Path)
		return os.Open(f)
	default:
		return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
}

type cfspec struct {
	ResourceSpecificationVersion string
	PropertyTypes                map[string]propertyType
	ResourceTypes                map[string]resourceType

	structs map[string]*Struct
}

func (s *CloudFormationSpec) Resource(serviceID, resourceName string) CloudFormationResource {
	combined := serviceID + "::" + resourceName
	for _, res := range s.Resources {
		if strings.HasSuffix(res.Type, combined) {
			return res
		}
	}
	return CloudFormationResource{}
}

func (s *cfspec) resolveStruct(resource string, id string, props map[string]property) Struct {
	id = resource + ":" + id
	if existing, ok := s.structs[id]; ok {
		return *existing
	}
	strct := make(Struct, 0, len(props))
	if s.structs == nil {
		s.structs = make(map[string]*Struct)
	}
	s.structs[id] = &strct

	for propName, prop := range props {
		strct = append(strct, Field{
			Name:     propName,
			Required: prop.Required,
			Type:     s.resolveType(resource, prop),
		})
	}
	sort.Slice(strct, func(i, j int) bool {
		return strct[i].Name < strct[j].Name
	})

	return strct
}

func (s *cfspec) resolveType(resource string, prop property) Type {
	if prop.PrimitiveType != "" {
		return s.primitiveType(prop.PrimitiveType)
	}
	if prop.PrimitiveItemType != "" {
		return s.primitiveType(prop.PrimitiveItemType)
	}
	if prop.Type == "List" {
		prop.Type = ""
		return List{
			Element: s.resolveType(resource, prop),
		}
	}
	if prop.ItemType == "Tag" {
		return Tag{}
	}
	if nested, ok := s.PropertyTypes[resource+"."+prop.Type]; ok {
		return s.resolveStruct(resource, prop.Type, nested.Properties)
	}
	if nested, ok := s.PropertyTypes[prop.Type]; ok {
		return s.resolveStruct(resource, prop.Type, nested.Properties)
	}
	if nested, ok := s.PropertyTypes[resource+"."+prop.ItemType]; ok {
		return s.resolveStruct(resource, prop.ItemType, nested.Properties)
	}
	if nested, ok := s.PropertyTypes[prop.ItemType]; ok {
		return s.resolveStruct(resource, prop.ItemType, nested.Properties)
	}
	// Special cases:
	if (resource == "AWS::ImageBuilder::Image" && prop.Type == "OutputResources") ||
		(resource == "AWS::SSM::Association" && prop.ItemType == "ParameterValues") ||
		(resource == "AWS::Macie::FindingsFilter" && prop.ItemType == "FindingsFilterListItem") {
		return nil
	}
	panic(fmt.Sprintf("Cannot resolve %s type from:\n%#v", resource, prop))
}

func (cfspec) primitiveType(typename string) Type {
	switch typename {
	case "String":
		return String{}
	case "Integer", "Long":
		return Int{}
	case "Double", "Float":
		return Float{}
	case "Boolean":
		return Bool{}
	case "Timestamp":
		return Timestamp{}
	case "Json":
		return JSON{}
	default:
		panic("Unknown primitive type: " + typename)
	}
}

type propertyType struct {
	Documentation string
	Properties    map[string]property
}

type resourceType struct {
	Documentation string
	Properties    map[string]property
	Attributes    map[string]property
}

type property struct {
	Documentation      string
	ItemType           string
	PrimitiveItemType  string
	PrimitiveType      string
	Required           bool
	Type               string
	PrimitiveTypes     []string
	PrimitiveItemTypes []string
	ItemTypes          []string
	Types              []string
}
