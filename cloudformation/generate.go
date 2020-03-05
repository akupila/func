package cloudformation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
	"github.com/func/func/resource"
	"github.com/func/func/source"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"golang.org/x/sync/errgroup"
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

// S3SourceSetter is implemented by resources that support getting their
// source code from S3. If implemented, the resource's source code is
// processed, zipped and uploaded to S3. The key is then passed to the resource
// prior to encoding it.
//
// Implementing the interface makes the source code required. In case the user
// does not provide source code, error diagnostics are returned.
type S3SourceSetter interface {
	SetS3SourceCode(bucket, key string)
}

// Encoder can be implemented on fields that produce custom output for
// CloudFormation, for example json.
// Fields that implement Encoder are not allowed to have references on them. If
// the user provides a reference, error diagnostics are produced.
type Encoder interface {
	CloudFormation() (interface{}, error)
}

// A Generator generates CloudFormation templates from a resource graph.
type Generator struct {
	S3Client s3iface.ClientAPI
	S3Bucket string

	// CacheDir sets the cache directory to use for generated source zip files.
	// If not set, a directory within the user's cache directory is used.
	CacheDir string
}

// Generate generates a CloudFormation template from a resource graph.
func (g *Generator) Generate(ctx context.Context, graph *resource.Graph) (*Template, hcl.Diagnostics) {
	cacheDir := g.CacheDir
	if cacheDir == "" {
		cacheDir = defaultCacheDir()
	}
	gen := &generator{
		Resources: graph.Resources,
		S3Client:  g.S3Client,
		S3Bucket:  g.S3Bucket,
		CacheDir:  cacheDir,
	}

	template := &Template{
		AWSTemplateFormatVersion: "2010-09-09",
		Resources:                make(map[string]Resource, len(graph.Resources)),
		logicalMapping:           make(map[string]string, len(graph.Resources)),
	}
	diags := gen.Generate(ctx, template)

	return template, diags
}

func defaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "func")
	if err := os.MkdirAll(dir, 0700); err != nil {
		panic(err)
	}
	return dir
}

type generator struct {
	Resources map[string]resource.Resource
	S3Client  s3iface.ClientAPI
	S3Bucket  string
	CacheDir  string
}

func (g *generator) Generate(ctx context.Context, tmpl *Template) hcl.Diagnostics {
	var mu sync.Mutex
	var diags hcl.Diagnostics

	eg, ctx := errgroup.WithContext(ctx)
	for name, input := range g.Resources {
		name, input := name, input
		eg.Go(func() error {
			res, morediags := g.processResource(ctx, input)
			mu.Lock()
			diags = append(diags, morediags...)
			logicalName := resourceName(name)
			tmpl.Resources[logicalName] = res
			tmpl.logicalMapping[logicalName] = name
			mu.Unlock()
			if morediags.HasErrors() {
				return morediags
			}
			return nil
		})
	}

	// Any errors are added to diagnostics, safe to ignore error
	_ = eg.Wait()

	return diags
}

func (g *generator) processResource(ctx context.Context, input resource.Resource) (Resource, hcl.Diagnostics) {
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

	if s3src, ok := input.Config.(S3SourceSetter); ok {
		if input.SourceCode == nil {
			return Resource{}, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Source code not provided",
				Detail:   fmt.Sprintf("Source code must be provided for %s.", input.Type),
				Subject:  input.Definition.Ptr(),
			}}
		}
		key, err := g.sourceKey(ctx, input)
		if err != nil {
			return Resource{}, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Could not process source code",
				Detail:   fmt.Sprintf("Error: %v", err),
				Subject:  input.SourceCode.Definition.Ptr(),
			}}
		}
		s3src.SetS3SourceCode(g.S3Bucket, key)
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

func (g *generator) sourceKey(ctx context.Context, res resource.Resource) (string, error) {
	src, err := res.SourceFiles()
	if err != nil {
		return "", err
	}
	sum, err := src.Checksum()
	if err != nil {
		return "", err
	}
	key := sum + ".zip"

	exists, err := g.sourceExists(ctx, key)
	if err != nil {
		return "", err
	}
	if !exists {
		r, err := g.getSource(key, src)
		if err != nil {
			return "", fmt.Errorf("get source: %w", err)
		}
		if err := g.upload(ctx, key, r); err != nil {
			return "", fmt.Errorf("upload: %w", err)
		}
	}

	return key, nil
}

func (g *generator) getSource(key string, files *source.FileList) (*os.File, error) {
	filename := filepath.Join(g.CacheDir, key)
	f, err := os.Open(filename)
	if err == nil {
		// File was cached
		return f, nil
	}
	if !os.IsNotExist(err) {
		// Other error
		return nil, err
	}

	f, err = os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("create zip file: %w", err)
	}
	if err := files.Zip(f); err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}
	if err := f.Sync(); err != nil {
		return nil, fmt.Errorf("sync: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}

	return f, nil
}

func (g *generator) upload(ctx context.Context, key string, body io.Reader) error {
	mgr := s3manager.NewUploaderWithClient(g.S3Client)
	_, err := mgr.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(g.S3Bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	return err
}

func (g *generator) sourceExists(ctx context.Context, key string) (bool, error) {
	_, err := g.S3Client.HeadObjectRequest(&s3.HeadObjectInput{
		Bucket: aws.String(g.S3Bucket),
		Key:    aws.String(key),
	}).Send(ctx)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "NotFound" {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}

type encoder struct {
	Resources map[string]resource.Resource
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
	res, ok := e.Resources[name]
	if !ok {
		return nil
	}
	return res.Config
}

func (t Template) LookupResource(logicalName string) (string, bool) {
	name, ok := t.logicalMapping[logicalName]
	return name, ok
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
