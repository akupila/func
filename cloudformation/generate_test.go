package cloudformation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func TestGenerate_empty(t *testing.T) {
	got, diags := Generate(resource.List{}, nil)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	equalAsJSON(t, got, `{
		"AWSTemplateFormatVersion": "2010-09-09"
	}`)
}

func TestGenerate_arguments(t *testing.T) {
	type cfg struct {
		testConfig
		String string                 `input:"string" cloudformation:"StringValue"`
		Slice  []interface{}          `input:"slice" cloudformation:"SliceValue"`
		Map    map[string]interface{} `input:"map" cloudformation:"MapValue"`
		Ptr    *string                `input:"ptr" cloudformation:"Ptr"`
	}

	strptr := func(v string) *string { return &v }

	list := resource.List{
		{
			Name: "test_resource",
			Config: cfg{
				String: "foo",
				Slice:  []interface{}{"foo", "bar"},
				Map:    map[string]interface{}{"foo": "bar"},
				Ptr:    strptr("val"),
			},
		},
	}

	got, diags := Generate(list, nil)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	equalAsJSON(t, got, `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "CloudFormation::TestResource",
				"Properties": {
					"StringValue": "foo",
					"SliceValue":  ["foo", "bar"],
					"MapValue":    {"foo": "bar"},
					"Ptr":         "val"
				}
			}
		}
	}`)
}

func TestGenerate_customEncoder(t *testing.T) {
	type cfg struct {
		testConfig
		Value jsonValue            `input:"value" cloudformation:"Value"`
		List  []jsonValue          `input:"list" cloudformation:"List"`
		Map   map[string]jsonValue `input:"map" cloudformation:"Map"`
	}

	list := resource.List{
		{
			Name: "custom_encoder",
			Config: cfg{
				Value: jsonValue{SomeValue: "foo"},
				List:  []jsonValue{{SomeValue: "foo"}, {SomeValue: "bar"}},
				Map:   map[string]jsonValue{"foo": {SomeValue: "bar"}},
			},
		},
	}

	got, diags := Generate(list, nil)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	equalAsJSON(t, got, `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"CustomEncoder": {
				"Type": "CloudFormation::TestResource",
				"Properties": {
					"Value": {"some_value": "foo"},
					"List": [
						{"some_value": "foo"},
						{"some_value": "bar"}
					],
					"Map": {
						"foo": {"some_value": "bar"}
					}
				}
			}
		}
	}`)
}

func TestGenerate_references(t *testing.T) {
	type a struct {
		testConfig
		RefOut string `output:"out1" cloudformation:"Out1,ref"`
		AttOut string `output:"out2" cloudformation:"Out2,att"`
	}

	type b struct {
		testConfig
		Input1 string `input:"in1" cloudformation:"In1"`
		Input2 string `input:"in2" cloudformation:"In2"`
	}

	list := resource.List{
		{
			Name:   "a",
			Type:   "a",
			Config: a{testConfig: testConfig{Type: "test:a"}},
		},
		{
			Name:   "b",
			Type:   "b",
			Config: b{testConfig: testConfig{Type: "test:b"}},
			Refs: []resource.Reference{
				{Field: cty.GetAttrPath("in1"), Expression: parseExpr(t, "a.out1")},
				{Field: cty.GetAttrPath("in2"), Expression: parseExpr(t, "a.out2")},
			},
		},
	}

	got, diags := Generate(list, nil)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	equalAsJSON(t, got, `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"A": {
				"Type": "test:a"
			},
			"B": {
				"Type": "test:b",
				"Properties": {
					"In1": {
						"Fn::Ref": "A"
					},
					"In2": {
						"Fn::GetAtt": "A.Out2"
					}
				}
			}
		}
	}`)
}

func TestGenerate_ignoreFields(t *testing.T) {
	type cfg struct {
		testConfig
		String     string             `input:"string" cloudformation:"StringValue"`
		EmptySlice []string           `input:"empty_slice" cloudformation:"EmptySlice"`
		EmptyMap   map[string]string  `input:"empty_map" cloudformation:"EmptyMap"`
		SliceNil   []*string          `input:"slice_nil" cloudformation:"SliceNil"`
		MapNil     map[string]*string `input:"map_nil" cloudformation:"MapNil"`
		Ptr        *string            `input:"optinal_string" cloudformation:"OptionalString"`
		NotCF      string             `input:"not_cf"`                                   // No CloudFormation tag -> ignore
		NotInput   string             `output:"not_input" cloudformation:"NotInput,ref"` // Output tag -> ignore
		OnlyCF     string             `cloudformation:"CFOut"`                           // Only CloudFormation tag -> do NOT ignore
	}

	list := resource.List{
		{
			Name: "test_resource",
			Config: cfg{
				String:     "", // Do not output empty strings
				EmptySlice: []string{},
				EmptyMap:   map[string]string{},
				SliceNil:   []*string{nil},
				MapNil:     map[string]*string{"foo": nil},
				Ptr:        nil,   // Skip nil pointer
				NotCF:      "foo", // Skip field without CloudFormation tag
				NotInput:   "bar", // Marked as output
				OnlyCF:     "xxx",
			},
		},
	}

	got, diags := Generate(list, nil)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	equalAsJSON(t, got, `{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "CloudFormation::TestResource",
				"Properties": {
					"CFOut": "xxx"
				}
			}
		}
	}`)
}

func TestGenerate_notCloudFormation(t *testing.T) {
	type notCFResource struct {
		String string `input:"string"`
	}

	def := hcl.Range{
		Filename: "file.hcl",
		Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
		End:      hcl.Pos{Line: 1, Column: 2, Byte: 1},
	}

	list := resource.List{
		{
			Name:       "test_resource",
			Type:       "test:resource",
			Definition: def,
			Config: notCFResource{
				String: "foo",
			},
		},
	}

	_, diags := Generate(list, nil)

	wantDiags := hcl.Diagnostics{{
		Severity: hcl.DiagError,
		Summary:  "Incompatible resource",
		Detail:   "A CloudFormation resource cannot be generated from test:resource.",
		Subject:  &def,
	}}
	if diff := cmp.Diff(diags, wantDiags); diff != "" {
		t.Fatalf("Diff (-got +want):\n%s", diff)
	}
}

func TestGenerate_WithSource(t *testing.T) {
	list := resource.List{
		{
			Name:   "test_resource",
			Config: &testSourceConfig{},
		},
	}

	bucket, key := "bucket", "file.zip"
	source := map[string]S3Location{
		"test_resource": {
			Bucket: bucket,
			Key:    key,
		},
	}

	got, diags := Generate(list, source)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	equalAsJSON(t, got, fmt.Sprintf(`{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "CloudFormation::TestResourceWithSource",
				"Properties": {
					"Bucket": %q,
					"Key":    %q
				}
			}
		}
	}`, bucket, key))
}

func TestGenerate_noSource(t *testing.T) {
	def := hcl.Range{
		Filename: "file.hcl",
		Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
		End:      hcl.Pos{Line: 1, Column: 2, Byte: 1},
	}

	list := resource.List{
		{
			Name:       "test_resource",
			Type:       "test:resource",
			Definition: def,
			Config:     &testSourceConfig{}, // Requires source
		},
	}

	_, diags := Generate(list, map[string]S3Location{
		"bar": {}, // no source set for "test_resource"
	})

	wantDiags := hcl.Diagnostics{{
		Severity: hcl.DiagError,
		Summary:  "Source code not provided",
		Detail:   "Source code must be provided for test:resource.",
		Subject:  &def,
	}}
	if diff := cmp.Diff(diags, wantDiags); diff != "" {
		t.Fatalf("Diff (-got +want):\n%s", diff)
	}
}

func tempdir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	return dir, func() {
		_ = os.RemoveAll(dir)
	}
}

func writeFiles(t *testing.T, dir string, files map[string][]byte) {
	for name, data := range files {
		if err := ioutil.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

type testConfig struct {
	Type string

	Bucket string `cloudformation:"Bucket"`
	Key    string `cloudformation:"Key"`
}

func (t testConfig) CloudFormationType() string {
	if t.Type == "" {
		return "CloudFormation::TestResource"
	}
	return t.Type
}

type testSourceConfig struct {
	Bucket string `cloudformation:"Bucket"`
	Key    string `cloudformation:"Key"`
}

func (t *testSourceConfig) CloudFormationType() string {
	return "CloudFormation::TestResourceWithSource"
}

func (t *testSourceConfig) SetS3SourceCode(bucket, key string) {
	t.Bucket = bucket
	t.Key = key
}

type jsonValue struct {
	SomeValue string `json:"some_value"`
}

func (t jsonValue) CloudFormation() (interface{}, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
