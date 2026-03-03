package validator

import (
	"time"
)

// ValidationLevel defines the severity of validation rules
type ValidationLevel string

const (
	LevelError   ValidationLevel = "error"
	LevelWarning ValidationLevel = "warning"
	LevelInfo    ValidationLevel = "info"
)

// CommitType represents a conventional commit type
type CommitType struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Scope       bool   `json:"scope"`
	Breaking    bool   `json:"breaking"`
}

// ValidationResult represents a single validation check result
type ValidationResult struct {
	Rule       string          `json:"rule"`
	Level      ValidationLevel `json:"level"`
	Message    string          `json:"message"`
	Suggestion string          `json:"suggestion,omitempty"`
	LineNumber int             `json:"line_number,omitempty"`
}

// CommitInfo represents information about a commit
type CommitInfo struct {
	Hash       string    `json:"hash"`
	Message    string    `json:"message"`
	Author     string    `json:"author"`
	Date       time.Time `json:"date"`
	Type       string    `json:"type"`
	Scope      string    `json:"scope"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	Footer     string    `json:"footer"`
	IsBreaking bool      `json:"is_breaking"`
	IsFixup    bool      `json:"is_fixup"`
	IsSquash   bool      `json:"is_squash"`
}

// PRValidationReport represents the complete PR validation report
type PRValidationReport struct {
	PRNumber    int                `json:"pr_number"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Branch      string             `json:"branch"`
	BaseBranch  string             `json:"base_branch"`
	Commits     []CommitInfo       `json:"commits"`
	Results     []ValidationResult `json:"results"`
	Passed      bool               `json:"passed"`
	Duration    time.Duration      `json:"duration_ms"`
}

// SubjectLengthConfig represents subject length validation configuration
type SubjectLengthConfig struct {
	Max int `yaml:"max" mapstructure:"max"`
	Min int `yaml:"min" mapstructure:"min"`
}

// TitleConfig represents PR title validation configuration
type TitleConfig struct {
	Conventional bool     `yaml:"conventional" mapstructure:"conventional"`
	Types        []string `yaml:"types" mapstructure:"types"`
	Scopes       []string `yaml:"scopes" mapstructure:"scopes"`
	MaxLength    int      `yaml:"max_length" mapstructure:"max_length"`
	MinLength    int      `yaml:"min_length" mapstructure:"min_length"`
	Pattern      string   `yaml:"pattern" mapstructure:"pattern"`
}

// DescriptionConfig represents PR description validation configuration
type DescriptionConfig struct {
	Required      bool     `yaml:"required" mapstructure:"required"`
	MinLength     int      `yaml:"min_length" mapstructure:"min_length"`
	CheckTemplate bool     `yaml:"check_template" mapstructure:"check_template"`
	Keywords      []string `yaml:"keywords" mapstructure:"keywords"`
}

// BranchConfig represents branch validation configuration
type BranchConfig struct {
	Pattern       string   `yaml:"pattern" mapstructure:"pattern"`
	AllowedPrefix []string `yaml:"allowed_prefixes" mapstructure:"allowed_prefixes"`
}

// PRCommitsConfig represents commits validation configuration within PR
type PRCommitsConfig struct {
	ValidateAll   bool                `yaml:"validate_all" mapstructure:"validate_all"`
	IgnoreMerge   bool                `yaml:"ignore_merge" mapstructure:"ignore_merge"`
	AllowedTypes  []string            `yaml:"allowed_types" mapstructure:"allowed_types"`
	RequireScope  bool                `yaml:"require_scope" mapstructure:"require_scope"`
	SubjectLength SubjectLengthConfig `yaml:"subject_length" mapstructure:"subject_length"`
}

// LabelsConfig represents PR labels validation configuration
type LabelsConfig struct {
	Require []string `yaml:"require" mapstructure:"require"`
	Forbid  []string `yaml:"forbid" mapstructure:"forbid"`
}

// PRConfig represents PR validation configuration
type PRConfig struct {
	Title       TitleConfig       `yaml:"title" mapstructure:"title"`
	Description DescriptionConfig `yaml:"description" mapstructure:"description"`
	Branch      BranchConfig      `yaml:"branch" mapstructure:"branch"`
	Commits     PRCommitsConfig   `yaml:"commits" mapstructure:"commits"`
	Labels      LabelsConfig      `yaml:"labels" mapstructure:"labels"`
}

// CommitsConfig represents global commits validation configuration
type CommitsConfig struct {
	Conventional    bool                `yaml:"conventional" mapstructure:"conventional"`
	Types           []CommitType        `yaml:"types" mapstructure:"types"`
	Scopes          []string            `yaml:"scopes" mapstructure:"scopes"`
	CheckFooter     bool                `yaml:"check_footer" mapstructure:"check_footer"`
	BreakingPattern string              `yaml:"breaking_pattern" mapstructure:"breaking_pattern"`
	SubjectLength   SubjectLengthConfig `yaml:"subject_length" mapstructure:"subject_length"`
	BodyLength      SubjectLengthConfig `yaml:"body_length" mapstructure:"body_length"`
}

// Config represents the complete validation configuration
type Config struct {
	PR      PRConfig      `yaml:"pr" mapstructure:"pr"`
	Commits CommitsConfig `yaml:"commits" mapstructure:"commits"`
}
