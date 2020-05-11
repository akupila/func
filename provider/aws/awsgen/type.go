package main

import (
	"sort"
	"strings"
)

type Path []string

func (p Path) String() string { return strings.Join(p, ".") }

// A Type represents a type in the AWS APIs.
type Type interface {
	isType()
}

// Struct is a struct in an AWS API request or response.
type Struct []Field

// FieldNames returns the names of all fields in the struct. The returned slice
// is sorted alphabetically.
func (s Struct) FieldNames() []string {
	fields := make([]string, len(s))
	for i, f := range s {
		fields[i] = f.Name
	}
	sort.Strings(fields)
	return fields
}

func (s Struct) FieldByName(name string) (Field, bool) {
	for _, f := range s {
		if strings.EqualFold(f.Name, name) {
			return f, true
		}
	}
	return Field{}, false
}

func (s Struct) FieldByPath(path Path) Field {
	var f Field
	for _, p := range path {
		field, ok := s.FieldByName(p)
		if !ok {
			return Field{}
		}
		f = field
		switch nested := f.Type.(type) {
		case Struct:
			s = nested
		case List:
			if strct, ok := nested.Element.(Struct); ok {
				s = strct
			}
		}
	}
	return f
}

// Exclude creates a copy of the struct without the given field names.
func (s Struct) Exclude(fieldNames []string) Struct {
	out := make(Struct, 0, len(s))
outer:
	for _, f := range s {
		for _, name := range fieldNames {
			if f.Name == name {
				continue outer
			}
		}
		out = append(out, f)
	}
	return out
}

func (Struct) isType() {}

func (s Struct) HasTimestamp() bool {
	for _, f := range s {
		switch nested := f.Type.(type) {
		case Timestamp:
			return true
		case Struct:
			if nested.HasTimestamp() {
				return true
			}
		case List:
			if strct, ok := nested.Element.(Struct); ok {
				if strct.HasTimestamp() {
					return true
				}
			}
		}
	}
	return false
}

// Field is a field within a struct.
type Field struct {
	Name     string
	Required bool
	Doc      *Doc
	Type     Type
}

// String is a string type.
type String struct {
	MinLen  *int
	MaxLen  *int
	Pattern *string
	Enum    []string
}

func (String) isType() {}

// Int is an integer or long type.
type Int struct {
	Min *int64
	Max *int64
}

func (Int) isType() {}

// Float is a float or double type.
type Float struct {
	Min *float64
	Max *float64
}

func (Float) isType() {}

// List is a list (slice) of an element type.
type List struct {
	MinItems *int
	MaxItems *int
	Element  Type
}

func (List) isType() {}

// Map is a map of key to values.
type Map struct {
	MinItems *int
	MaxItems *int
	Key      Type
	Value    Type
}

func (Map) isType() {}

// Binary represents binary data.
type Binary struct{}

func (Binary) isType() {}

// Timestamp is a timestamp.
type Timestamp struct{}

func (Timestamp) isType() {}

// Bool is a boolean type.
type Bool struct{}

func (Bool) isType() {}

// JSON is a json encoded byte slice.
type JSON struct{}

func (JSON) isType() {}

// Tag is a AWS resource tag
type Tag struct{}

func (Tag) isType() {}
