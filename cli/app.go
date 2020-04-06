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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/func/func/cloudformation"
	"github.com/func/func/provider/aws"
	"github.com/func/func/resource"
	"github.com/func/func/source"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/hcl/v2"
	"golang.org/x/sync/errgroup"
)

// A Logger is used for logging activity to the user.
type Logger interface {
	Errorf(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Tracef(format string, args ...interface{})

	Writer(level LogLevel) io.Writer
}

// PrefixLogger allows creating sub loggers with a prefix.
type PrefixLogger interface {
	Logger
	WithPrefix(name string) PrefixLogger
}

// App encapsulates all cli business logic.
type App struct {
	Log    Logger
	Stdout io.Writer
}

// NewApp creates a new App with the given log level.
func NewApp(verbosity int) *App {
	logger := &StdLogger{
		Output: os.Stderr,
		Level:  LogLevel(verbosity),
	}
	return &App{
		Log:    logger,
		Stdout: os.Stdout,
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

	a.Log.Debugf("Loading config from %s", dir)

	list, diags := loader.LoadDir(dir)
	printer := func(diags hcl.Diagnostics) {
		out := a.Log.Writer(allLevels)
		loader.PrintDiagnostics(out, diags)
	}
	a.Log.Tracef("Loaded %d resources", len(list))
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
			log := a.Log
			if pfx, ok := log.(PrefixLogger); ok {
				log = pfx.WithPrefix(fmt.Sprintf("  [%s] ", res.Name))
			}
			log.Tracef("Computing source checksum")
			sum, err := res.SourceCode.Checksum()
			if err != nil {
				return fmt.Errorf("%s: compute source checksum: %v", res.Name, err)
			}
			log.Tracef("Source checksum = %s", sum)
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
	log := a.Log
	if pfx, ok := log.(PrefixLogger); ok {
		log = pfx.WithPrefix(fmt.Sprintf("  %s ", src.Resource))
	}
	log.Tracef("Checking if source exists")

	exists, err := s3.Has(ctx, src.Key)
	if err != nil {
		return fmt.Errorf("check existing source: %w", err)
	}
	if exists {
		log.Tracef("Source ok")
		return nil
	}

	files := src.Source.Files

	if len(src.Source.Build) > 0 {
		log.Infof("Building")
		buildDir, err := ioutil.TempDir("", "func-build")
		if err != nil {
			return err
		}
		log.Tracef("Build dir: %s", buildDir)
		defer func() {
			log.Tracef("Removing temporary build dir %s", buildDir)
			_ = os.RemoveAll(buildDir)
		}()
		log.Tracef("Copying source %d files to build dir", len(files.Files))
		if err := files.Copy(buildDir); err != nil {
			return err
		}

		log.Tracef("Executing build")

		buildTime := time.Now()
		n := len(src.Source.Build)
		for i, s := range src.Source.Build {
			line, stdout, stderr := log, log, log
			if pfx, ok := line.(PrefixLogger); ok {
				indexPrefix := pfx.WithPrefix(fmt.Sprintf("  %d/%d ", i+1, n))
				line = indexPrefix
				stdout = indexPrefix.WithPrefix("| ")
				stderr = indexPrefix.WithPrefix("| ")
			}
			buildContext := &source.BuildContext{
				Dir:    buildDir,
				Stdout: stdout.Writer(Info),
				Stderr: stderr.Writer(allLevels),
			}
			line.Debugf("$ %s", s)
			stepTime := time.Now()
			if err := s.Exec(ctx, buildContext); err != nil {
				return fmt.Errorf("exec step %d: %s: %w", i, s, err)
			}
			line.Tracef("Done in %s", time.Since(stepTime).Round(time.Millisecond))
		}
		log.Debugf("Build completed in %s", time.Since(buildTime).Round(time.Millisecond))

		log.Tracef("Collecting build artifacts")
		output, err := source.Collect(buildDir)
		if err != nil {
			return err
		}
		log.Tracef("Got %d build artifacts", len(output.Files))
		files = output
	}

	log.Debugf("Creating source zip")
	zipfile := strings.Replace(src.Key, ".", "-*.", 1)
	f, err := ioutil.TempFile("", zipfile)
	if err != nil {
		return fmt.Errorf("create source file: %w", err)
	}
	defer func() {
		_ = os.Remove(f.Name())
	}()

	if err := files.Zip(f); err != nil {
		return fmt.Errorf("zip: %w", err)
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}

	log.Infof("Uploading")
	err = s3.Upload(ctx, src.Key, f)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	log.Debugf("Upload complete")
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
	Format        string
	SourceBucket  string
	ProcessSource bool
}

// GenerateCloudFormation generates a CloudFormation template from the
// resources in the given directory.
func (a *App) GenerateCloudFormation(ctx context.Context, dir string, opts GenerateCloudFormationOpts) int {
	resources, printDiags, diags := a.loadResources(dir)
	printDiags(diags)
	if diags.HasErrors() {
		return 1
	}

	srcs, err := a.sources(resources)
	if err != nil {
		a.Log.Errorf("Could not collect source files: %v", err)
		return 1
	}
	if len(srcs) > 0 && opts.SourceBucket == "" {
		a.Log.Errorf("Source bucket not set")
		return 2
	}

	if opts.ProcessSource {
		cfg, err := external.LoadDefaultAWSConfig()
		if err != nil {
			a.Log.Errorf("Could not load aws config: %v", err)
		}
		s3 := source.NewS3(cfg, opts.SourceBucket)

		g, gctx := errgroup.WithContext(ctx)
		for _, src := range srcs {
			src := src
			g.Go(func() error {
				if err := a.ensureSource(gctx, src, s3); err != nil {
					return fmt.Errorf("%s: %w", src.Resource, err)
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			a.Log.Errorf("Could not process source: %v", err)
			return 1
		}
	}

	locs := sourceLocations(srcs, opts.SourceBucket)
	tmpl, diags := cloudformation.Generate(resources, locs)
	printDiags(diags)
	if diags.HasErrors() {
		os.Exit(1)
	}

	var out []byte
	switch strings.ToLower(opts.Format) {
	case "json":
		out, err = json.MarshalIndent(tmpl, "", "    ")
	case "json-compact":
		out, err = json.Marshal(tmpl)
	case "yaml", "yml":
		out, err = json.Marshal(tmpl)
		if err == nil {
			out, err = yaml.JSONToYAML(out)
		}
	default:
		a.Log.Errorf("Unsupported output format %q. Supported: [json, json-compact, yaml, yml]", opts.Format)
		return 2
	}
	if err != nil {
		a.Log.Errorf(err.Error())
		return 1
	}
	outstr := string(out)
	if !strings.HasSuffix(outstr, "\n") {
		outstr += "\n"
	}
	fmt.Fprint(a.Stdout, outstr)
	return 0
}

// DeploymentOpts provides options for deploying the project using
// CloudFormation.
type DeploymentOpts struct {
	StackName    string
	SourceBucket string
}

// DeployCloudFormation deploys the project using CloudFormation.
func (a *App) DeployCloudFormation(ctx context.Context, dir string, opts DeploymentOpts) int {
	if opts.StackName == "" {
		a.Log.Errorf("Stack name not set")
		return 2
	}

	resources, printDiags, diags := a.loadResources(dir)
	printDiags(diags)
	if diags.HasErrors() {
		return 1
	}

	a.Log.Debugf("Processing source files")

	srcs, err := a.sources(resources)
	if err != nil {
		a.Log.Errorf("Could not collect source files: %v", err)
		return 1
	}
	if len(srcs) > 0 && opts.SourceBucket == "" {
		a.Log.Errorf("Source bucket not set")
		return 2
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		a.Log.Errorf("Could not load aws config: %v", err)
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
			if err := a.ensureSource(gctx, src, s3); err != nil {
				return fmt.Errorf("%s: %w", src.Resource, err)
			}
			return nil
		})
	}

	a.Log.Debugf("Generating CloudFormation template")

	locs := sourceLocations(srcs, opts.SourceBucket)
	tmpl, diags := cloudformation.Generate(resources, locs)
	if diags.HasErrors() {
		printDiags(diags)
		return 1
	}

	// 2/2: Change set
	var changeset *cloudformation.ChangeSet
	g.Go(func() error {
		a.Log.Debugf("Creating CloudFormation change set")

		a.Log.Tracef("Getting CloudFormation stack")
		stack, err := cf.StackByName(gctx, opts.StackName)
		if err != nil {
			return fmt.Errorf("get stack: %w", err)
		}
		if stack.ID == "" {
			a.Log.Tracef("Stack does not exist")
		} else {
			a.Log.Tracef("Got CloudFormation stack: %s", stack.ID)
		}

		cs, err := cf.CreateChangeSet(gctx, stack, tmpl)
		if err != nil {
			return fmt.Errorf("create change set: %w", err)
		}
		a.Log.Tracef("Created CloudFormation change set %s\n", cs.ID)
		changeset = cs

		return nil
	})

	if err := g.Wait(); err != nil {
		a.Log.Errorf("Error: %v", err)
		return 1
	}

	if len(changeset.Changes) == 0 {
		a.Log.Infof("No changes")
		if err := cf.DeleteChangeSet(ctx, changeset); err != nil {
			// Safe to ignore
			a.Log.Errorf("Error cleaning up change set: %v\n", err)
			return 3
		}
		return 0
	}

	a.Log.Debugf("Deploying")

	deployment, err := cf.ExecuteChangeSet(ctx, changeset)
	if err != nil {
		a.Log.Errorf("Could not execute change set: %v", err)
		return 1
	}

	for ev := range cf.Events(ctx, deployment) {
		switch e := ev.(type) {
		case cloudformation.ErrorEvent:
			a.Log.Errorf("Deployment error: %v", e.Error)
			return 1
		case cloudformation.ResourceEvent:
			name := tmpl.LookupResource(e.LogicalID)
			if name == "" {
				// No mapping for resources that are being deleted
				name = e.LogicalID
			}
			line := a.Log
			if pfx, ok := line.(PrefixLogger); ok {
				line = pfx.WithPrefix(fmt.Sprintf("  [%s] ", name))
			}
			line.Debugf("%s %s %s", e.Operation, e.State, e.Reason)
		case cloudformation.StackEvent:
			if e.State == cloudformation.StateComplete {
				if e.Operation == cloudformation.StackRollback {
					a.Log.Errorf("Deployment failed: %s", e.Reason)
					return 1
				}
			}
		}
	}

	a.Log.Infof("Done")

	return 0
}
