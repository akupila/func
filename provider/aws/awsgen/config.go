package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Services map[string]ServiceConfig
}

type ServiceConfig struct {
	PkgDoc    string
	Resources map[string]ResourceConfig
}

type ResourceConfig struct {
	Name    string
	Doc     string
	Fields  map[string]FieldConfig
	Methods []string
	Output  string
}

type FieldConfig struct {
	Name           string
	CloudFormation string
	Doc            string
	Input          string
	NoInput        bool
}

func LoadConfig(file string) (*Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	dec := yaml.NewDecoder(f)
	dec.SetStrict(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &cfg, nil
}
