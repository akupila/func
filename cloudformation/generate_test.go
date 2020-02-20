package cloudformation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/func/func/resource"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func TestGenerate_empty(t *testing.T) {
	g := &resource.Graph{}

	gen := &Generator{S3Client: &mockS3{}}
	got, diags := gen.Generate(context.Background(), g)
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

	g := &resource.Graph{
		Resources: map[string]resource.Resource{
			"test_resource": {
				Config: cfg{
					String: "foo",
					Slice:  []interface{}{"foo", "bar"},
					Map:    map[string]interface{}{"foo": "bar"},
					Ptr:    strptr("val"),
				},
			},
		},
	}

	gen := &Generator{S3Client: &mockS3{}}
	got, diags := gen.Generate(context.Background(), g)
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

	g := &resource.Graph{
		Resources: map[string]resource.Resource{
			"custom_encoder": {
				Config: cfg{
					Value: jsonValue{SomeValue: "foo"},
					List:  []jsonValue{{SomeValue: "foo"}, {SomeValue: "bar"}},
					Map:   map[string]jsonValue{"foo": {SomeValue: "bar"}},
				},
			},
		},
	}

	gen := &Generator{S3Client: &mockS3{}}
	got, diags := gen.Generate(context.Background(), g)
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

	g := &resource.Graph{
		Resources: map[string]resource.Resource{
			"a": {
				Type:   "a",
				Config: a{testConfig: testConfig{Type: "test:a"}},
			},
			"b": {
				Type:   "b",
				Config: b{testConfig: testConfig{Type: "test:b"}},
				Refs: []resource.Reference{
					{Field: cty.GetAttrPath("in1"), Expression: parseExpr(t, "a.out1")},
					{Field: cty.GetAttrPath("in2"), Expression: parseExpr(t, "a.out2")},
				},
			},
		},
	}

	gen := &Generator{S3Client: &mockS3{}}
	got, diags := gen.Generate(context.Background(), g)
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

	g := &resource.Graph{
		Resources: map[string]resource.Resource{
			"test_resource": {
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
		},
	}

	gen := &Generator{S3Client: &mockS3{}}
	got, diags := gen.Generate(context.Background(), g)
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

	g := &resource.Graph{
		Resources: map[string]resource.Resource{
			"test_resource": {
				Type:       "test:resource",
				Definition: def,
				Config: notCFResource{
					String: "foo",
				},
			},
		},
	}

	gen := &Generator{}
	_, diags := gen.Generate(context.Background(), g)

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

var testFile = []byte("test")
var testFileKey string

func init() {
	s := sha256.New()
	s.Write(testFile)
	testFileKey = hex.EncodeToString(s.Sum(nil)) + ".zip"
}

func TestGenerate_uploadSource(t *testing.T) {
	temp, done := tempdir(t)
	defer done()

	writeFiles(t, temp, map[string][]byte{
		"index.js": testFile,
	})

	g := &resource.Graph{
		Resources: map[string]resource.Resource{
			"test_resource": {
				Config: &testSourceConfig{},
				SourceCode: &resource.SourceCode{
					Dir: temp,
				},
			},
		},
	}

	cache, done := tempdir(t)
	defer done()

	uploaded := false

	bucket := "testbucket"
	gen := &Generator{
		S3Client: &mockS3{
			onPut: func(input *s3.PutObjectInput) {
				if *input.Bucket != bucket {
					t.Errorf("Uploaded bucket does not match, got %q, want %q", *input.Bucket, bucket)
				}
				if *input.Key != testFileKey {
					t.Errorf("Uploaded key does not match, got %q, want %q", *input.Key, testFileKey)
				}
				data, err := ioutil.ReadAll(input.Body)
				if err != nil {
					t.Fatal(err)
				}
				uploaded = true
				zipLen := 158 // number of bytes of zip of testFile
				if len(data) != zipLen {
					t.Errorf("Uploaded length does not match, got %d, want %d", len(data), zipLen)
				}
			},
		},
		S3Bucket:   bucket,
		CacheDir: cache,
	}
	got, diags := gen.Generate(context.Background(), g)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	if !uploaded {
		t.Error("File was not uploaded")
	}

	equalAsJSON(t, got, fmt.Sprintf(`{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "CloudFormation::TestResourceWithSource",
				"Properties": {
					"Bucket": "testbucket",
					"Key":    %q
				}
			}
		}
	}`, testFileKey))
}

// File available locally in cache but not remotely
func TestGenerate_uploadSource_cached(t *testing.T) {
	temp, done := tempdir(t)
	defer done()

	writeFiles(t, temp, map[string][]byte{
		"index.js": testFile,
	})

	g := &resource.Graph{
		Resources: map[string]resource.Resource{
			"test_resource": {
				Config: &testSourceConfig{},
				SourceCode: &resource.SourceCode{
					Dir: temp,
				},
			},
		},
	}

	cache, done := tempdir(t)
	defer done()

	// Write dummy file to cache
	cached := []byte("cached")
	if err := ioutil.WriteFile(filepath.Join(cache, testFileKey), cached, 0644); err != nil {
		t.Fatal(err)
	}

	uploaded := false

	bucket := "testbucket"
	gen := &Generator{
		S3Client: &mockS3{
			onPut: func(input *s3.PutObjectInput) {
				data, err := ioutil.ReadAll(input.Body)
				if err != nil {
					t.Fatal(err)
				}
				uploaded = true

				if !bytes.Equal(data, cached) {
					t.Helper()
					t.Errorf(
						"Uploaded data does not equal cached data\nGot\n%s\nWant\n%s",
						hex.Dump(data), hex.Dump(cached),
					)
				}
			},
		},
		S3Bucket:   bucket,
		CacheDir: cache,
	}
	got, diags := gen.Generate(context.Background(), g)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	if !uploaded {
		t.Error("File was not uploaded")
	}

	equalAsJSON(t, got, fmt.Sprintf(`{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "CloudFormation::TestResourceWithSource",
				"Properties": {
					"Bucket": "testbucket",
					"Key":    %q
				}
			}
		}
	}`, testFileKey))
}

func TestGenerate_sourceExists(t *testing.T) {
	temp, done := tempdir(t)
	defer done()

	writeFiles(t, temp, map[string][]byte{
		"index.js": testFile,
	})

	g := &resource.Graph{
		Resources: map[string]resource.Resource{
			"test_resource": {
				Config: &testSourceConfig{},
				SourceCode: &resource.SourceCode{
					Dir: temp,
				},
			},
		},
	}

	cache, done := tempdir(t)
	defer done()

	gen := &Generator{
		S3Client: &mockS3{
			files: map[string][]byte{
				testFileKey: testFile, // File already exists
			},
			onPut: func(input *s3.PutObjectInput) {
				t.Errorf("Want no uploads, got upload for %s", *input.Key)
			},
		},
		S3Bucket:   "testbucket",
		CacheDir: cache,
	}
	got, diags := gen.Generate(context.Background(), g)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	equalAsJSON(t, got, fmt.Sprintf(`{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": {
			"TestResource": {
				"Type": "CloudFormation::TestResourceWithSource",
				"Properties": {
					"Bucket": "testbucket",
					"Key":    %q
				}
			}
		}
	}`, testFileKey))
}

func TestGenerate_noSource(t *testing.T) {
	def := hcl.Range{
		Filename: "file.hcl",
		Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
		End:      hcl.Pos{Line: 1, Column: 2, Byte: 1},
	}

	g := &resource.Graph{
		Resources: map[string]resource.Resource{
			"test_resource": {
				Type:       "test:resource",
				Definition: def,
				Config:     &testSourceConfig{}, // Requires source
				SourceCode: nil,                 // .. but not set
			},
		},
	}

	gen := &Generator{}
	_, diags := gen.Generate(context.Background(), g)

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

type mockS3 struct {
	s3iface.ClientAPI
	files map[string][]byte

	onPut func(input *s3.PutObjectInput)
}

func (m *mockS3) req() *aws.Request {
	return &aws.Request{
		HTTPRequest: &http.Request{
			URL:    &url.URL{},
			Header: make(http.Header),
		},
		HTTPResponse: &http.Response{},
		Retryer:      aws.NewDefaultRetryer(),
	}
}

func (m *mockS3) ListObjectsV2Request(input *s3.ListObjectsV2Input) s3.ListObjectsV2Request {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		objects := make([]s3.Object, 0, len(m.files))
		for name := range m.files {
			objects = append(objects, s3.Object{
				Key: aws.String(name),
			})
		}
		sort.Slice(objects, func(i, j int) bool {
			return *objects[i].Key < *objects[j].Key
		})
		r.Data = &s3.ListObjectsV2Output{
			Contents: objects,
		}
	})
	return s3.ListObjectsV2Request{Request: req}
}

func (m *mockS3) PutObjectRequest(input *s3.PutObjectInput) s3.PutObjectRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		if m.onPut != nil {
			m.onPut(input)
		}
		b, err := ioutil.ReadAll(input.Body)
		if err != nil {
			r.Error = err
			return
		}
		if m.files == nil {
			m.files = make(map[string][]byte)
		}
		m.files[*input.Key] = b
		r.Data = &s3.PutObjectOutput{}
	})
	return s3.PutObjectRequest{Request: req}
}

func (m *mockS3) HeadObjectRequest(input *s3.HeadObjectInput) s3.HeadObjectRequest {
	req := m.req()
	req.Handlers.Send.PushBack(func(r *aws.Request) {
		if _, ok := m.files[*input.Key]; ok {
			r.Data = &s3.HeadObjectOutput{}
			return
		}
		r.Error = awserr.New("NotFound", "Not Found", nil)
	})
	return s3.HeadObjectRequest{Request: req}
}
