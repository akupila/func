package resource

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
)

func impliedSpec(ty reflect.Type) hcldec.Spec {
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	return impliedStructSpec(ty, "input", true)
}

func inputType(ty reflect.Type) cty.Type {
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	spec := impliedStructSpec(ty, "input", false)
	return hcldec.ImpliedType(spec)
}

func outputType(ty reflect.Type) cty.Type {
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	spec := impliedStructSpec(ty, "output", false)
	return hcldec.ImpliedType(spec)
}

func inputFields(ty reflect.Type) map[string]int {
	inputs := make(map[string]int)
	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, _ := parseTag(field.Tag.Get("input"))
		if name == "" {
			continue
		}
		inputs[name] = i
	}
	return inputs
}

func impliedStructSpec(ty reflect.Type, fieldName string, decodeToExpression bool) hcldec.Spec {
	spec := make(hcldec.ObjectSpec, ty.NumField())
	var labelIndex int
	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, opts := parseTag(field.Tag.Get(fieldName))
		if name == "" {
			continue
		}

		ft := field.Type
		var ptr bool
		if ft.Kind() == reflect.Ptr {
			ptr = true
			ft = ft.Elem()
		}

		switch ft.Kind() {
		case reflect.Struct:
			spec[name] = &hcldec.BlockSpec{
				TypeName: name,
				Nested:   impliedStructSpec(ft, fieldName, decodeToExpression),
				Required: !ptr,
			}
		case reflect.Map:
			spec[name] = &hcldec.AttrSpec{
				Name:     name,
				Type:     ctyType(ft, decodeToExpression),
				Required: false,
			}
		case reflect.Slice:
			et := ft.Elem()
			if et.Kind() == reflect.Ptr {
				et = et.Elem()
			}
			switch et.Kind() {
			case reflect.Struct:
				spec[name] = &hcldec.BlockListSpec{
					TypeName: name,
					Nested:   impliedStructSpec(et, fieldName, decodeToExpression),
					MinItems: parseIntOrZero(field.Tag.Get("min")),
					MaxItems: parseIntOrZero(field.Tag.Get("max")),
				}
			default:
				spec[name] = &hcldec.AttrSpec{
					Name:     name,
					Type:     ctyType(ft, decodeToExpression),
					Required: false,
				}
			}
		case reflect.Array:
			spec[name] = &hcldec.AttrSpec{
				Name:     name,
				Type:     ctyType(ft, decodeToExpression),
				Required: true,
			}
		default:
			if opts.contains("label") {
				spec[name] = &hcldec.BlockLabelSpec{
					Name:  name,
					Index: labelIndex,
				}
				labelIndex++
				continue
			}
			spec[name] = &hcldec.AttrSpec{
				Name:     name,
				Type:     ctyType(ft, decodeToExpression),
				Required: !ptr,
			}
		}
	}
	return spec
}

func ctyType(rt reflect.Type, decodeToExpression bool) cty.Type {
	if decodeToExpression {
		return customdecode.ExpressionType
	}

	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	switch rt.Kind() {
	case reflect.Bool:
		return cty.Bool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cty.Number
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cty.Number
	case reflect.Float32, reflect.Float64:
		return cty.Number
	case reflect.String:
		return cty.String
	case reflect.Slice, reflect.Array:
		et := ctyType(rt.Elem(), decodeToExpression)
		return cty.List(et)
	case reflect.Map:
		et := ctyType(rt.Elem(), decodeToExpression)
		return cty.Map(et)
	default:
		panic("Invalid type " + rt.Kind().String())
	}
}

type options []string

func (opts options) contains(key string) bool {
	for _, opt := range opts {
		if opt == key {
			return true
		}
	}
	return false
}

func parseTag(tag string) (string, options) {
	if len(tag) == 0 {
		return "", nil
	}
	parts := strings.Split(tag, ",")
	return parts[0], parts[1:]
}

func parseIntOrZero(val string) int {
	if val == "" {
		return 0
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		// If set, must be a valid number
		panic(err)
	}
	return v
}
