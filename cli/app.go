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
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/func/func/cloudformation"
	"github.com/func/func/provider/aws"
	"github.com/func/func/resource"
	"github.com/func/func/source"
	"github.com/func/func/ui"
	"github.com/func/func/version"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/hcl/v2"
	"golang.org/x/sync/errgroup"
)

// App encapsulates all cli business logic.
type App struct {
	Log    *logger
	Stdout io.Writer

	loader *resource.Loader
}

// NewApp creates a new cli app.
func NewApp(verbose bool) *App {
	return &App{
		Log:    newLogger(os.Stderr, verbose),
		Stdout: os.Stdout,
	}
}

func (a *App) loadResources(dir string) (resource.List, hcl.Diagnostics) {
	if a.loader == nil {
		reg := &resource.Registry{}
		reg.Add("aws:iam_role", reflect.TypeOf(&aws.IAMRole{}))
		reg.Add("aws:lambda_function", reflect.TypeOf(&aws.LambdaFunction{}))

		a.loader = &resource.Loader{
			Registry: reg,
		}
	}

	return a.loader.LoadDir(dir)
}

type sourcecode struct {
	Resource *resource.Resource
	Source   *source.Code
	Checksum string
	Key      string
}

func sources(resources resource.List) ([]sourcecode, error) {
	sources := resources.WithSource()
	out := make([]sourcecode, len(sources))
	g, _ := errgroup.WithContext(context.Background())
	for i, res := range sources {
		i, res := i, res
		g.Go(func() error {
			sum, err := res.SourceCode.Checksum()
			if err != nil {
				return fmt.Errorf("%s: compute source checksum: %v", res.Name, err)
			}
			out[i] = sourcecode{
				Resource: res,
				Source:   res.SourceCode,
				Checksum: sum,
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

func ensureSource(ctx context.Context, src sourcecode, s3 *source.S3, step *logStep) error {
	step.Icon = true
	step.Verbosef("Dir:      %s", src.Source.Files.Root)
	step.Verbosef("Checksum: %s", src.Checksum[0:12])

	exists, err := s3.Has(ctx, src.Key)
	if err != nil {
		return fmt.Errorf("check existing source: %w", err)
	}
	if exists {
		return nil
	}

	files := src.Source.Files

	if len(src.Source.Build) > 0 {
		buildScriptStep := step.Step("Build")

		buildDir, err := ioutil.TempDir("", "func-build")
		if err != nil {
			return err
		}
		defer func() {
			_ = os.RemoveAll(buildDir)
		}()
		if err := files.Copy(buildDir); err != nil {
			return err
		}

		for i, s := range src.Source.Build {
			buildStep := buildScriptStep.Step("$ " + string(s))
			io := &window{MaxLines: 3}
			buildStep.Push(io)
			buildContext := &source.BuildContext{
				Dir:    buildDir,
				Stdout: io,
				Stderr: io,
			}
			if err := s.Exec(ctx, buildContext); err != nil {
				buildStep.Errorf("Step failed: %v", err)
				return fmt.Errorf("exec step %d: %s: %w", i, s, err)
			}
			// buildStep.Remove(io)
			buildStep.Done()
		}

		collectStep := step.Step("Collect build artifacts")
		output, err := source.Collect(buildDir)
		if err != nil {
			return err
		}
		files = output
		collectStep.Verbosef("Got %d build artifacts", len(files.Files))
		collectStep.Done()

		buildScriptStep.Done()
	}

	compressStep := step.Step("Compress")
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
	compressStep.Done()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	uploadStep := step.Step("Upload")
	progress := newProgressBar(16)
	progress.SetProgress(0.75)
	uploadStep.Push(progress)
	r := &uploadReader{
		File:     f,
		Progress: progress,
		Size:     stat.Size(),
	}
	err = s3.Upload(ctx, src.Key, r)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	uploadStep.Remove(progress)
	uploadStep.Done()
	return nil
}

type uploadReader struct {
	File     *os.File
	Progress *progressBar
	Size     int64
	read     int64
}

func (r *uploadReader) Read(p []byte) (int, error) {
	return r.File.Read(p)
}

func (r *uploadReader) ReadAt(p []byte, off int64) (int, error) {
	n, err := r.File.ReadAt(p, off)
	if err != nil {
		return n, err
	}

	r.read += int64(n)

	time.Sleep(50 * time.Millisecond)

	// Data appears to be read twice, possibly for signing the request
	percent := float64(r.read-r.Size) / float64(r.Size)
	r.Progress.SetProgress(percent)

	return n, err
}

func (r *uploadReader) Seek(offset int64, whence int) (int64, error) {
	return r.File.Seek(offset, whence)
}

func sourceLocations(sources []sourcecode, bucket string) map[string]cloudformation.S3Location {
	out := make(map[string]cloudformation.S3Location, len(sources))
	for _, src := range sources {
		out[src.Resource.Name] = cloudformation.S3Location{
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
	step := a.Log.Step("Load resource configurations")
	resources, diags := a.loadResources(dir)
	step.PrintDiags(diags, a.loader.Files())
	if diags.HasErrors() {
		return 1
	}
	step.Done()

	srcs, err := sources(resources)
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

		srcStep := a.Log.Step("Process source code")
		g, gctx := errgroup.WithContext(ctx)
		for _, src := range srcs {
			src := src
			g.Go(func() error {
				if err := ensureSource(gctx, src, s3, srcStep); err != nil {
					return fmt.Errorf("%s: %w", src.Resource.Name, err)
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			a.Log.Errorf("Could not process source: %v", err)
			return 1
		}

		srcStep.Done()
	}

	step = a.Log.Step("Generate CloudFormation template")
	locs := sourceLocations(srcs, opts.SourceBucket)
	tmpl, diags := cloudformation.Generate(resources, locs)
	step.PrintDiags(diags, a.loader.Files())
	if diags.HasErrors() {
		os.Exit(1)
	}
	step.Done()

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

	// Better way to ensure render completes
	time.Sleep(100 * time.Millisecond)

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

	defer func() {
		// Better way to ensure render completes
		time.Sleep(200 * time.Millisecond)
	}()

	a.Log.Infof(ui.Format("func ", ui.Bold) +
		ui.Format(version.Version, ui.Dim) + "\n",
	)

	step := a.Log.Step("Load resource configurations")
	resources, diags := a.loadResources(dir)
	step.PrintDiags(diags, a.loader.Files())
	if diags.HasErrors() {
		return 1
	}
	step.Done()

	srcs, err := sources(resources)
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

	srcStep := a.Log.Step("Process source code")
	locs := sourceLocations(srcs, opts.SourceBucket)
	tmpl, diags := cloudformation.Generate(resources, locs)
	srcStep.PrintDiags(diags, a.loader.Files())
	if diags.HasErrors() {
		return 1
	}

	// Concurrently process sources and create change set.
	//   Sources may require build/upload time,
	//   Change set creation can take a couple seconds.
	g, gctx := errgroup.WithContext(ctx)

	// 1/2: Process & upload source code
	var wg sync.WaitGroup
	for _, src := range srcs {
		src := src
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			step := srcStep.Step(src.Resource.Name)
			if err := ensureSource(gctx, src, s3, step); err != nil {
				return fmt.Errorf("%s: %w", src.Resource.Name, err)
			}
			step.Done()
			return nil
		})
	}
	go func() {
		wg.Wait()
		srcStep.Done()
	}()

	// 2/2: Change set
	var changeset *cloudformation.ChangeSet
	g.Go(func() error {
		stack, err := cf.StackByName(gctx, opts.StackName)
		if err != nil {
			return fmt.Errorf("get stack: %w", err)
		}

		cs, err := cf.CreateChangeSet(gctx, stack, tmpl)
		if err != nil {
			return fmt.Errorf("create change set: %w", err)
		}
		changeset = cs

		return nil
	})

	if err := g.Wait(); err != nil {
		a.Log.Errorf("Error: %v", err)
		return 1
	}

	if len(changeset.Changes) == 0 {
		a.Log.Infof(ui.Format("\nNo changes", ui.Dim))
		if err := cf.DeleteChangeSet(ctx, changeset); err != nil {
			// Safe to ignore
			a.Log.Errorf("Error cleaning up change set: %v\n", err)
			return 3
		}
		return 0
	}

	step = a.Log.Step("Deploy")

	deployment, err := cf.ExecuteChangeSet(ctx, changeset)
	if err != nil {
		a.Log.Errorf("Could not execute change set: %v", err)
		return 1
	}

	deploySteps := make(map[string]*logStep)
	for ev := range cf.Events(ctx, deployment) {
		switch e := ev.(type) {
		case cloudformation.ErrorEvent:
			step.Errorf("Deployment error: %v", e.Error)
			return 1
		case cloudformation.ResourceEvent:
			name := tmpl.LookupResource(e.LogicalID)
			if name == "" {
				// No mapping for resources that are being deleted
				name = e.LogicalID
			}
			resStep, ok := deploySteps[name]
			if !ok {
				resStep = step.Step(e.Operation.String() + " " + name)
				resStep.Icon = true
				deploySteps[name] = resStep
			}
			switch e.State {
			case cloudformation.StateComplete:
				resStep.Done()
			case cloudformation.StateFailed:
				resStep.Errorf("%s failed because %s", e.Operation, e.Reason)
			}
		case cloudformation.StackEvent:
			if e.State == cloudformation.StateComplete {
				if e.Operation == cloudformation.StackRollback {
					step.Errorf("Deployment failed: %s", e.Reason)
					return 1
				}
			}
		}
	}

	step.Done()

	return 0
}
