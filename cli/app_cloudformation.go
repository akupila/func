package cli

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/func/func/cloudformation"
	"github.com/func/func/resource"
	"github.com/func/func/source"
)

// GenerateCloudFormation generates a CloudFormation template from the the
// resources in the given directory.
func (a *App) GenerateCloudFormation(ctx context.Context, dir string) (*cloudformation.Template, int) {
	loader := &resource.Loader{
		Registry: a.Registry,
	}

	a.Logger.Verbosef("Loading config from %s\n", dir)

	resources, diags := loader.LoadDir(dir)
	loader.PrintDiagnostics(os.Stderr, diags)
	if diags.HasErrors() {
		return nil, 1
	}

	if len(resources) == 0 {
		a.Logger.Errorf("No resources found in %s\n", dir)
		return nil, 2
	}
	a.Logger.Tracef("Found %d resources\n", len(resources))

	a.Logger.Verboseln("Checking source code that needs processing")

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		a.Logger.Errorln(err)
		return nil, 1
	}
	s3 := source.NewS3(cfg, a.SourceS3Bucket)

	cache, err := source.NewDiskCache()
	if err != nil {
		a.Logger.Errorln(err)
		return nil, 1
	}

	a.Logger.Infoln("Checking source code")

	srcs := resources.WithSource()
	var wg sync.WaitGroup
	archives := make([]*source.Archive, len(srcs))
	for i, res := range srcs {
		i, res := i, res
		wg.Add(1)
		go func() {
			defer wg.Done()

			a.Logger.Verbosef("%s: Collecting source files\n", res.Name)

			files, err := res.SourceFiles()
			if err != nil {
				a.Logger.Errorln(err)
			}

			arc, err := source.NewArchive(files, source.Zip)
			if err != nil {
				a.Logger.Errorln(err)
			}
			arc.ResourceName = res.Name
			archives[i] = arc

			a.Logger.Verbosef("%s: Checking if source exists\n", res.Name)

			exists, err := s3.Has(ctx, arc.FileName())
			if err != nil {
				a.Logger.Errorln(err)
			}
			if exists {
				a.Logger.Verbosef("%s: Source ok\n", res.Name)
				return
			}

			file := cache.Get(arc.FileName())
			if file != nil {
				a.Logger.Verbosef("%s: Source cached\n", res.Name)
			} else {
				a.Logger.Infof("%s: Building..\n", res.Name)
				time.Sleep(1000 * time.Millisecond) // Fake timer for now

				f, err := cache.Create(arc.FileName())
				if err != nil {
					a.Logger.Errorln(err)
				}

				if err := arc.Write(f); err != nil {
					a.Logger.Errorln(err)
				}
				if err := f.Sync(); err != nil {
					a.Logger.Errorln(err)
				}
				if _, err := f.Seek(0, 0); err != nil {
					a.Logger.Errorln(err)
				}

				file = f
			}

			a.Logger.Infof("%s: Uploading\n", res.Name)

			err = s3.Upload(ctx, arc.FileName(), file)
			if err != nil {
				a.Logger.Errorln(err)
			}

			a.Logger.Verbosef("%s: Upload complete\n", res.Name)
		}()
	}
	wg.Wait()

	a.Logger.Traceln("Generating CloudFormation template")

	sources := make(map[string]cloudformation.S3Location, len(srcs))
	for _, arc := range archives {
		sources[arc.ResourceName] = cloudformation.S3Location{
			Bucket: a.SourceS3Bucket,
			Key:    arc.FileName(),
		}
	}

	tmpl, diags := cloudformation.Generate(resources, sources)
	loader.PrintDiagnostics(os.Stderr, diags)
	if diags.HasErrors() {
		os.Exit(1)
	}

	return tmpl, 0
}
