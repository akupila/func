package resource

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// Decode decodes a configuration body to a resource graph.
// Resources must be registered in the schema registry.
func Decode(body hcl.Body, registry *Registry) (*Graph, hcl.Diagnostics) {
	rootBodySchema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"name"}},
		},
	}

	content, diags := body.Content(rootBodySchema)
	if diags.HasErrors() {
		return nil, diags
	}

	dec := &decoder{
		Registry:  registry,
		Resources: make(map[string]*decoderResource),
	}

	diags = append(diags, dec.DecodeResources(content)...)
	if diags.HasErrors() {
		return nil, diags
	}

	diags = append(diags, dec.ResolveStatic()...)
	if diags.HasErrors() {
		return nil, diags
	}

	diags = append(diags, dec.ValidateReferences()...)
	if diags.HasErrors() {
		return nil, diags
	}

	g := &Graph{
		Resources: make(map[string]Resource, len(dec.Resources)),
	}

	for name, res := range dec.Resources {
		cfg, _ := registry.New(res.Type)
		setValue(res.Config, cfg)
		g.Resources[name] = Resource{
			Type:       res.Type,
			Definition: res.Definition,
			SourceCode: res.SourceCode,
			Config:     cfg.Interface(),
			Refs:       res.Refs,
		}
	}

	return g, diags
}

type decoder struct {
	Registry  *Registry
	Resources map[string]*decoderResource
}

type decoderResource struct {
	Type       string
	Definition hcl.Range
	SourceCode *SourceCode
	Config     cty.Value
	Refs       []Reference
	Input      cty.Type
	Output     cty.Type
}

func (d *decoder) DecodeResources(content *hcl.BodyContent) hcl.Diagnostics {
	blocks := content.Blocks.OfType("resource")

	var diags hcl.Diagnostics
	for _, b := range blocks {
		name := b.Labels[0]
		if name == "" {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Resource name not set",
				Detail:   "A resource name cannot be blank.",
				Subject:  b.LabelRanges[0].Ptr(),
			})
			continue
		}

		if prev, ok := d.Resources[name]; ok {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate resource",
				Detail: fmt.Sprintf(
					"Another resource named %q was defined in %s on line %d.",
					name, prev.Definition.Filename, prev.Definition.Start.Line,
				),
				Subject: b.DefRange.Ptr(),
			})
			continue
		}

		res, morediags := d.DecodeResource(b)
		diags = append(diags, morediags...)
		d.Resources[name] = res
	}

	return diags
}

func (d *decoder) DecodeResource(block *hcl.Block) (*decoderResource, hcl.Diagnostics) {
	spec := hcldec.ObjectSpec{
		"type": &hcldec.AttrSpec{
			Name:     "type",
			Type:     cty.String,
			Required: true,
		},
		"source": &hcldec.BlockSpec{
			TypeName: "source",
			Required: false,
			Nested: hcldec.ObjectSpec{
				"dir": &hcldec.AttrSpec{
					Name:     "dir",
					Type:     cty.String,
					Required: true,
				},
			},
		},
	}

	content, body, diags := hcldec.PartialDecode(block.Body, spec, nil)
	if diags.HasErrors() {
		// The type of source did not match the common resource spec.
		// Don't continue so we can assume the content is valid.
		return nil, diags
	}

	typename := content.GetAttr("type").AsString()
	cfg, err := d.Registry.New(typename)
	if err != nil {
		rng := hcldec.SourceRange(block.Body, spec["type"])
		diag := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported resource",
			Detail:   fmt.Sprintf("Resources of type %q are not supported.", typename),
			Subject:  rng.Ptr(),
		}
		if suggestion, ok := suggest(d.Registry.Types(), typename); ok {
			diag.Detail += fmt.Sprintf(" Did you mean %q?", suggestion)
		}
		diags = append(diags, diag)
		return nil, diags
	}

	var source *SourceCode
	if srcAttr := content.GetAttr("source"); !srcAttr.IsNull() {
		filename := body.MissingItemRange().Filename
		resourceDir := filepath.Dir(filename)
		cont, _, _ := block.Body.PartialContent(hcldec.ImpliedSchema(spec["source"]))
		rng := cont.Blocks[0].DefRange
		dir := filepath.Join(resourceDir, srcAttr.GetAttr("dir").AsString())
		source = &SourceCode{
			Definition: rng,
			Dir:        dir,
		}
	}

	inputSpec := impliedSpec(cfg.Type())
	config, morediags := hcldec.Decode(body, inputSpec, nil)
	diags = append(diags, morediags...)

	input := inputType(cfg.Type())
	output := outputType(cfg.Type())

	return &decoderResource{
		Type:       typename,
		Definition: block.DefRange,
		SourceCode: source,
		Config:     config,
		Input:      input,
		Output:     output,
	}, diags
}

func (d *decoder) ResolveStatic() hcl.Diagnostics {
	var diags hcl.Diagnostics
	for _, res := range d.Resources {
		res.Config, _ = cty.Transform(res.Config, func(path cty.Path, value cty.Value) (cty.Value, error) {
			if !value.Type().Equals(customdecode.ExpressionType) {
				// Wrapper type (object, list, etc)
				return value, nil
			}
			if value.IsNull() {
				// Input value not set
				return value, nil
			}

			expr := customdecode.ExpressionFromVal(value)
			if len(expr.Variables()) > 0 {
				// Reference to other resource
				res.Refs = append(res.Refs, Reference{
					Field:      path,
					Expression: expr,
				})
				return cty.DynamicVal, nil
			}

			// Can be statically resolved
			val, _ := expr.Value(nil)

			// Convert if needed
			wantType := applyTypePath(res.Input, path)

			if !val.Type().Equals(wantType) {
				converted, err := convert.Convert(val, wantType)
				if err != nil {
					diags = append(diags, &hcl.Diagnostic{
						Severity:   hcl.DiagError,
						Summary:    "Incorrect attribute value type",
						Detail:     fmt.Sprintf("Inappropriate value for attribute: %v.", err),
						Subject:    expr.Range().Ptr(),
						Expression: expr,
					})
					val = cty.UnknownVal(wantType)
				} else {
					if wantType.IsPrimitiveType() {
						// Add warning that conversion was necessary.
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagWarning,
							Summary: fmt.Sprintf(
								"Value is converted from %s to %s",
								val.Type().FriendlyName(),
								wantType.FriendlyNameForConstraint(),
							),
							Subject:    expr.Range().Ptr(),
							Expression: expr,
						})
					}
					val = converted
				}
			}

			return val, nil
		})
	}
	return diags
}

func applyTypePath(ty cty.Type, path cty.Path) cty.Type {
	for _, p := range path {
		switch e := p.(type) {
		case cty.GetAttrStep:
			switch {
			case ty.IsMapType():
				ty = ty.ElementType()
			case ty.IsObjectType():
				if !ty.HasAttribute(e.Name) {
					return cty.NilType
				}
				ty = ty.AttributeType(e.Name)
			default:
				return cty.NilType
			}
		case cty.IndexStep:
			switch {
			case ty.IsListType():
				ty = ty.ElementType()
				continue
			case ty.IsTupleType():
				bf := e.Key.AsBigFloat()
				index, _ := bf.Int64()
				ty = ty.TupleElementType(int(index))
				continue
			default:
				return cty.NilType
			}
		}
	}
	return ty
}

func (d *decoder) ValidateReferences() hcl.Diagnostics {
	io := make(map[string]cty.Value, len(d.Resources))
	names := make([]string, 0, len(d.Resources))

	for name, res := range d.Resources {
		names = append(names, name)
		kv := make(map[string]cty.Value)
		for k, v := range res.Config.AsValueMap() {
			kv[k] = v
		}
		for k, t := range res.Output.AttributeTypes() {
			kv[k] = cty.UnknownVal(t)
		}
		io[name] = cty.ObjectVal(kv)
	}
	sort.Strings(names)

	var diags hcl.Diagnostics
	for _, name := range names {
		res := d.Resources[name]
		for _, ref := range res.Refs {
			for _, trav := range ref.Expression.Variables() {
				split := trav.SimpleSplit()
				parentName := trav.RootName()
				parent, ok := io[parentName]
				if !ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "No such resource",
						Detail:   fmt.Sprintf("A resource named %q has not been declared.", parentName),
						Subject:  split.Abs.SourceRange().Ptr(),
					})
					continue
				}

				parentVal, _ := split.Rel.TraverseRel(parent)
				if parentVal.Type().Equals(cty.DynamicPseudoType) {
					// Reference is not a valid input or output within the parent resource.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid reference",
						Detail: fmt.Sprintf(
							"The resource %q (%s) does not have such a field.",
							parentName, d.Resources[parentName].Type,
						),
						Subject: split.Rel.SourceRange().Ptr(),
					})
					continue
				}

				if parentVal.IsNull() {
					// Input reference is valid but not value was set.
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Input value not set",
						Detail: fmt.Sprintf(
							"A value has not been set for this field in %q.",
							parentName,
						),
						Subject: trav.SourceRange().Ptr(),
					})
					continue
				}
			}
		}
	}
	return diags
}

func setValue(value cty.Value, target reflect.Value) {
	ty := value.Type()

	if target.Type().Kind() == reflect.Ptr {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		target = target.Elem()
	}

	switch ty {
	case cty.Bool:
		target.SetBool(value.True())
		return
	case cty.Number:
		bf := value.AsBigFloat()
		switch target.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i64, _ := bf.Int64()
			target.SetInt(i64)
			return
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			u64, _ := bf.Uint64()
			target.SetUint(u64)
			return
		case reflect.Float32, reflect.Float64:
			f64, _ := bf.Float64()
			target.SetFloat(f64)
			return
		}
	case cty.String:
		target.SetString(value.AsString())
		return
	}

	if ty.IsMapType() {
		mapVal := reflect.MakeMap(target.Type())
		et := target.Type().Elem()

		for k, v := range value.AsValueMap() {
			mapElem := reflect.New(et)
			setValue(v, mapElem.Elem())
			mapVal.SetMapIndex(
				reflect.ValueOf(k),
				mapElem.Elem(),
			)
		}
		target.Set(mapVal)
		return
	}

	if ty.IsTupleType() || ty.IsListType() {
		n := value.LengthInt()
		sliceVal := reflect.MakeSlice(target.Type(), n, n)
		for i, ev := range value.AsValueSlice() {
			tv := sliceVal.Index(i)
			setValue(ev, tv)
		}
		target.Set(sliceVal)
		return
	}

	if ty.IsObjectType() {
		fields := inputFields(target.Type())

		strct := reflect.New(target.Type()).Elem()
		for k, v := range value.AsValueMap() {
			if v.IsNull() {
				continue
			}
			index := fields[k]
			setValue(v, strct.Field(index))
		}
		target.Set(strct)
		return
	}

	if ty.Equals(cty.DynamicPseudoType) {
		return
	}

	// This should not happen, all possible types should be covered above.
	panic("Unsupported value to set: " + value.Type().FriendlyName())
}
