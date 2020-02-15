package resource

import (
	"fmt"
	"reflect"
	"sort"
)

// The Registry maintains a list of supported resources.
type Registry struct {
	types map[string]reflect.Type
}

// Add adds a new resource to the registry.
func (r *Registry) Add(typename string, typ reflect.Type) {
	if r.types == nil {
		r.types = make(map[string]reflect.Type)
	}
	r.types[typename] = typ
}

// New creates a new resource with the given type and returns a pointer to it.
func (r *Registry) New(typename string) (reflect.Value, error) {
	t, ok := r.types[typename]
	if !ok {
		return reflect.Value{}, fmt.Errorf("not supported")
	}
	v := reflect.New(t).Elem()
	if t.Kind() == reflect.Ptr {
		el := reflect.New(t.Elem())
		v.Set(el)
	}
	return v, nil
}

// Types returns all registered resource types. The types are sorted
// alphabetically.
func (r *Registry) Types() []string {
	names := make([]string, 0, len(r.types))
	for name := range r.types {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
