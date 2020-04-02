package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/func/func/cloudformation"
	"github.com/func/func/provider/aws"
	"github.com/func/func/resource"
	"github.com/func/func/source"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/hcl/v2"
	"golang.org/x/sync/errgroup"
)

// App encapsulates all cli business logic.
type App struct {
	*logger
	Stdout io.Writer
	Stderr io.Writer
}

// NewApp creates a new App with the given log level.
func NewApp(logLevel LogLevel) *App {
	return &App{
		logger: &logger{
			Level:  logLevel,
			Output: os.Stderr,
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

type diagPrinter func(diags hcl.Diagnostics)

func (a *App) loadResources(dir string) (resource.List, diagPrinter, hcl.Diagnostics) {
	reg := &resource.Registry{}
	reg.Add("aws:iam_role", reflect.TypeOf(&aws.IAMRole{}))
	reg.Add("aws:lambda_function", reflect.TypeOf(&aws.LambdaFunction{}))

	loader := &resource.Loader{
		Registry: reg,
	}

	a.Verbosef("Loading config from %s\n", dir)

	list, diags := loader.LoadDir(dir)
	printer := func(diags hcl.Diagnostics) {
		loader.PrintDiagnostics(a.Stderr, diags)
	}
	a.Tracef("Loaded %d resources\n", len(list))
	return list, printer, diags
}

type sourcecode struct {
	Resource string
	Source   *source.Code
	Key      string
}

func (a *App) sources(resources resource.List) ([]sourcecode, error) {
	sources := resources.WithSource()
	out := make([]sourcecode, len(sources))
	g, _ := errgroup.WithContext(context.Background())
	for i, res := range sources {
		i, res := i, res
		g.Go(func() error {
			a.Verbosef("  %s: Computing source checksum\n", res.Name)
			sum, err := res.SourceCode.Checksum()
			if err != nil {
				return fmt.Errorf("  %s: compute source checksum: %v", res.Name, err)
			}
			a.Tracef("  %s: Source checksum = %s\n", res.Name, sum)
			out[i] = sourcecode{
				Resource: res.Name,
				Source:   res.SourceCode,
				Key:      sum + ".zip",
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *App) ensureSource(ctx context.Context, src sourcecode, s3 *source.S3) error {
	name := src.Resource
	a.Verbosef("  %s: Checking if source exists\n", name)

	exists, err := s3.Has(ctx, src.Key)
	if err != nil {
		return fmt.Errorf("%s: check existing source: %w", name, err)
	}
	if exists {
		a.Verbosef("  %s: Source ok\n", name)
		return nil
	}

	files := src.Source.Files

	if len(src.Source.Build) > 0 {
		tmp, err := ioutil.TempDir("", "func-build")
		if err != nil {
			return err
		}
		defer func() {
			_ = os.RemoveAll(tmp)
		}()

		if err := files.Copy(tmp); err != nil {
			return err
		}

		buildContext := &source.BuildContext{
			Dir:    tmp,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		if err := src.Source.Build.Exec(ctx, buildContext); err != nil {
			return err
		}

		output, err := source.Collect(tmp)
		if err != nil {
			return err
		}
		files = output
	}

	a.Verbosef("  %s: Creating source zip\n", name)
	f, err := ioutil.TempFile("", src.Key)
	if err != nil {
		return fmt.Errorf("%s: create source file: %w", name, err)
	}
	defer func() {
		_ = os.Remove(f.Name())
	}()

	if err := files.Zip(f); err != nil {
		return fmt.Errorf("%s: zip: %w", name, err)
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}

	a.Infof("  %s: Uploading\n", name)

	err = s3.Upload(ctx, src.Key, f)
	if err != nil {
		return fmt.Errorf("%s: upload: %w", name, err)
	}

	a.Verbosef("  %s: Upload complete\n", name)
	return nil
}

func sourceLocations(sources []sourcecode, bucket string) map[string]cloudformation.S3Location {
	out := make(map[string]cloudformation.S3Location, len(sources))
	for _, src := range sources {
		out[src.Resource] = cloudformation.S3Location{
			Bucket: bucket,
			Key:    src.Key,
		}
	}
	return out
}

// GenerateCloudFormationOpts contains options for generating a CloudFormation
// template.
type GenerateCloudFormationOpts struct {
	Format       string
	SourceBucket string
}

// GenerateCloudFormation generates a CloudFormation template from the
// resources in the given directory.
func (a *App) GenerateCloudFormation(dir string, opts GenerateCloudFormationOpts) int {
	resources, printDiags, diags := a.loadResources(dir)
	printDiags(diags)
	if diags.HasErrors() {
		return 1
	}

	srcs, err := a.sources(resources)
	if err != nil {
		a.Errorf("Could not collect source files: %v\n", err)
		return 1
	}
	if len(srcs) > 0 && opts.SourceBucket == "" {
		a.Errorln("Source bucket not set")
		return 2
	}

	locs := sourceLocations(srcs, opts.SourceBucket)
	tmpl, diags := cloudformation.Generate(resources, locs)
	printDiags(diags)
	if diags.HasErrors() {
		os.Exit(1)
	}

	switch strings.ToLower(opts.Format) {
	case "json":
		out, err := json.MarshalIndent(tmpl, "", "    ")
		if err != nil {
			a.Errorln(err)
			return 1
		}
		fmt.Fprintln(a.Stdout, string(out))
	case "json-compact":
		out, err := json.Marshal(tmpl)
		if err != nil {
			a.Errorln(err)
			return 1
		}
		fmt.Fprintln(a.Stdout, string(out))
	case "yaml", "yml":
		j, err := json.Marshal(tmpl)
		if err != nil {
			a.Errorln(err)
			return 1
		}
		y, err := yaml.JSONToYAML(j)
		if err != nil {
			a.Errorln(err)
			return 1
		}
		fmt.Fprint(a.Stdout, string(y)) // Output already has trailing newline
	default:
		a.Errorf("Unsupported output format %q. Supported: [json, json-compact, yaml, yml]", opts.Format)
		return 2
	}
	return 0
}

// DeploymentOpts provides options for deploying the project using
// CloudFormation.
type DeploymentOpts struct {
	StackName    string
	SourceBucket string
}

// DeployCloudFormation deploys the project using CloudFormation.
func (a *App) DeployCloudFormation(ctx context.Context, dir string, opts DeploymentOpts) int { // nolint: gocyclo
	if opts.StackName == "" {
		a.Errorln("Stack name not set")
		return 2
	}

	resources, printDiags, diags := a.loadResources(dir)
	printDiags(diags)
	if diags.HasErrors() {
		return 1
	}

	a.Infoln("Processing source files")

	srcs, err := a.sources(resources)
	if err != nil {
		a.Errorf("Could not collect source files: %v\n", err)
		return 1
	}
	if len(srcs) > 0 && opts.SourceBucket == "" {
		a.Errorln("Source bucket not set")
		return 2
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		a.Errorf("Could not load aws config: %v\n", err)
	}
	cf := cloudformation.NewClient(cfg)
	s3 := source.NewS3(cfg, opts.SourceBucket)

	// Concurrently process sources and create change set.
	//   Sources may require build/upload time,
	//   Change set creation takes a ~3 seconds.
	g, gctx := errgroup.WithContext(ctx)

	// 1/2: Process & upload source code
	for _, src := range srcs {
		src := src
		g.Go(func() error {
			return a.ensureSource(gctx, src, s3)
		})
	}

	a.Infoln("Generating CloudFormation template")

	locs := sourceLocations(srcs, opts.SourceBucket)
	tmpl, diags := cloudformation.Generate(resources, locs)
	if diags.HasErrors() {
		printDiags(diags)
		return 1
	}

	// 2/2: Change set
	var changeset *cloudformation.ChangeSet
	g.Go(func() error {
		a.Infoln("Creating CloudFormation change set")

		a.Verboseln("Getting CloudFormation stack")
		stack, err := cf.StackByName(gctx, opts.StackName)
		if err != nil {
			return fmt.Errorf("get stack: %w", err)
		}
		if stack.ID == "" {
			a.Traceln("Stack does not exist")
		} else {
			a.Tracef("Got CloudFormation stack: %s\n", stack.ID)
		}

		cs, err := cf.CreateChangeSet(gctx, stack, tmpl)
		if err != nil {
			return fmt.Errorf("create change set: %w", err)
		}
		a.Tracef("Created CloudFormation change set %s\n", cs.ID)
		changeset = cs

		return nil
	})

	if err := g.Wait(); err != nil {
		a.Errorln(err)
		return 1
	}

	if len(changeset.Changes) == 0 {
		a.Infoln("No changes")
		if err := cf.DeleteChangeSet(ctx, changeset); err != nil {
			// Safe to ignore
			a.Errorf("Error cleaning up change set: %v\n", err)
			return 3
		}
		return 0
	}

	a.Verboseln("Deploying")

	deployment, err := cf.ExecuteChangeSet(ctx, changeset)
	if err != nil {
		a.Errorf("Could not execute change set: %v", err)
		return 1
	}

	for ev := range cf.Events(ctx, deployment) {
		switch e := ev.(type) {
		case cloudformation.ErrorEvent:
			a.Errorf("Deployment error: %v", e.Error)
			return 1
		case cloudformation.ResourceEvent:
			name := tmpl.LookupResource(e.LogicalID)
			a.Verbosef("  %s: %s %s %s\n", name, e.Operation, e.State, e.Reason)
		case cloudformation.StackEvent:
			if e.State == cloudformation.StateComplete {
				if e.Operation == cloudformation.StackRollback {
					a.Errorf("Deployment failed: %s\n", e.Reason)
					return 1
				}
			}
		}
	}

	a.Infoln("Done")

	return 0
}
