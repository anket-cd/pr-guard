package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub struct {
		Token string `yaml:"token"`
	} `yaml:"github"`

	Validation struct {
		PR struct {
			Enabled     bool              `yaml:"enabled"`
			Title       TitleConfig       `yaml:"title"`
			Description DescriptionConfig `yaml:"description"`
			Branch      BranchConfig      `yaml:"branch"`
			Labels      LabelsConfig      `yaml:"labels"`
		} `yaml:"pr"`

		Commits struct {
			Enabled       bool         `yaml:"enabled"`
			Conventional  bool         `yaml:"conventional"`
			Types         []TypeConfig `yaml:"types"`
			SubjectLength LengthConfig `yaml:"subject_length"`
			BodyLength    LengthConfig `yaml:"body_length"`
		} `yaml:"commits"`
	} `yaml:"validation"`
}

type TitleConfig struct {
	Pattern       string   `yaml:"pattern"`
	MaxLength     int      `yaml:"max_length"`
	MinLength     int      `yaml:"min_length"`
	AllowedTypes  []string `yaml:"allowed_types"`
	AllowedScopes []string `yaml:"allowed_scopes"`
}

type DescriptionConfig struct {
	Required      bool `yaml:"required"`
	MinLength     int  `yaml:"min_length"`
	MaxLength     int  `yaml:"max_length"`
	CheckTemplate bool `yaml:"check_template"`
}

type BranchConfig struct {
	Pattern       string   `yaml:"pattern"`
	AllowedPrefix []string `yaml:"allowed_prefixes"`
}

type LabelsConfig struct {
	Require []string `yaml:"require"`
	Forbid  []string `yaml:"forbid"`
}

type TypeConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Breaking    bool   `yaml:"breaking"`
}

type LengthConfig struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}
