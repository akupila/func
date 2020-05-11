package main

import (
	"sort"
	"strings"
)

type Resource struct {
	Name    string
	Service *Service
	Create  *Operation
	Update  *Operation
	Delete  *Operation
}

func ResolveResources(svc *Service) []*Resource {
	m := make(map[string]*Resource)
	for _, op := range svc.Operations {
		action, name := splitOp(op.Name)
		if action == actionUnknown {
			continue
		}
		res, ok := m[name]
		if !ok {
			res = &Resource{
				Name:    name,
				Service: svc,
			}
		}
		switch action {
		case actionCreate:
			res.Create = op.Ptr()
		case actionUpdate:
			res.Update = op.Ptr()
		case actionDelete:
			res.Delete = op.Ptr()
		}
		m[name] = res
	}
	out := make([]*Resource, 0, len(m))
	for _, res := range m {
		if res.Create != nil && res.Delete != nil {
			out = append(out, res)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

type action int

const (
	actionUnknown action = iota
	actionCreate
	actionUpdate
	actionDelete
)

func splitOp(op string) (action, string) {
	switch {
	case strings.HasPrefix(op, "Create"):
		return actionCreate, op[6:]
	case strings.HasPrefix(op, "Add"):
		return actionCreate, op[3:]
	case strings.HasPrefix(op, "Update"):
		return actionUpdate, op[6:]
	case strings.HasPrefix(op, "Delete"):
		return actionDelete, op[6:]
	case strings.HasPrefix(op, "Remove"):
		return actionDelete, op[6:]
	}
	return actionUnknown, ""
}
