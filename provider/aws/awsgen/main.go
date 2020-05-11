package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Flags struct {
	Data   string
	Config string
	Output string
	CFSpec string
}

func main() {
	var f Flags
	flag.StringVar(&f.Data, "data", "botocore/botocore/data", "Botocore data directory")
	flag.StringVar(&f.Output, "out", "aws", "Output directory")
	flag.StringVar(&f.Config, "config", "config.yml", "Config file")
	flag.StringVar(&f.CFSpec, "cf", "https://d1uauaxba7bl26.cloudfront.net/latest/gzip/CloudFormationResourceSpecification.json", "CloudFormation spec file or url")
	flag.Parse()

	if err := run(f); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(flags Flags) error {
	cfg, err := LoadConfig(flags.Config)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cf, err := LoadCloudFormation(flags.CFSpec)
	if err != nil {
		return fmt.Errorf("load cloudformation spec: %w", err)
	}

	serviceDirs, err := subdirs(flags.Data)
	if err != nil {
		return fmt.Errorf("list service dirs: %w", err)
	}

	for _, svcDir := range serviceDirs {
		version, err := latest(filepath.Join(flags.Data, svcDir))
		if err != nil {
			return fmt.Errorf("get %s version: %w", svcDir, err)
		}
		svcFile := filepath.Join(flags.Data, svcDir, version, "service-2.json")
		f, err := os.Open(svcFile)
		if err != nil {
			return err
		}

		svcID, err := ParseServiceID(f)
		if err != nil {
			return fmt.Errorf("parse service id: %w", err)
		}
		svcCfg, ok := cfg.Services[svcID]
		if !ok {
			continue
		}
		if _, err := f.Seek(0, 0); err != nil {
			return err
		}

		svc, err := ParseService(f)
		if err != nil {
			return fmt.Errorf("load service: %w", err)
		}
		f.Close()

		pkg := PackageName(svc.Metadata.ServiceID)
		dir := filepath.Join(flags.Output, pkg)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		resources := ResolveResources(svc)
		if len(resources) == 0 {
			continue
		}

		for i, res := range resources {
			gen := &Generator{
				Package:        pkg,
				Service:        svc,
				ServiceConfig:  svcCfg,
				Resource:       res,
				ResourceConfig: svcCfg.Resources[res.Name],
				CloudFormation: cf.Resource(svc.Metadata.ServiceID, res.Name),
			}
			if i == 0 {
				pkgDoc, err := os.Create(filepath.Join(dir, "doc.go"))
				if err != nil {
					return err
				}
				gen.GeneratePkgDoc(pkgDoc)
				if err := pkgDoc.Close(); err != nil {
					return err
				}

				pkgReg, err := os.Create(filepath.Join(dir, "register.go"))
				if err != nil {
					return err
				}
				gen.GeneratePkgRegister(pkgReg, resources)
				if err := pkgReg.Close(); err != nil {
					return err
				}
			}
			f, err := os.Create(filepath.Join(dir, strings.ToLower(res.Name)+".go"))
			if err != nil {
				return err
			}
			gen.GenerateResource(f)
			if err := f.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}

func latest(serviceDir string) (string, error) {
	versions, err := subdirs(serviceDir)
	if err != nil {
		return "", fmt.Errorf("list versions: %w", err)
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions")
	}
	sort.Strings(versions)
	return versions[len(versions)-1], nil
}

func subdirs(dir string) ([]string, error) {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		out = append(out, info.Name())
	}
	return out, nil
}
