package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// SemanticPRValidator handles PR-specific validations
type SemanticPRValidator struct {
	config *Config
}

func NewSemanticPRValidator(config *Config) *SemanticPRValidator {
	return &SemanticPRValidator{
		config: config,
	}
}

// ValidatePRTitle validates PR title against semantic rules
func (v *SemanticPRValidator) ValidatePRTitle(title string) []ValidationResult {
	var results []ValidationResult

	if title == "" {
		results = append(results, ValidationResult{
			Rule:       "pr-title-required",
			Level:      LevelError,
			Message:    "PR title is required",
			Suggestion: "Please provide a title for this PR",
		})
		return results
	}

	// Check if title follows conventional commit format
	titlePattern := regexp.MustCompile(`^(\w+)(?:\(([\w\-]+)\))?!?: (.+)$`)
	matches := titlePattern.FindStringSubmatch(title)

	if matches == nil && v.config.PR.Title.Conventional {
		results = append(results, ValidationResult{
			Rule:       "pr-title-format",
			Level:      LevelError,
			Message:    "PR title must follow conventional commit format",
			Suggestion: "Use format: <type>[optional scope]: <description>\nExample: feat(api): add new endpoint",
		})
		return results
	}

	if matches != nil {
		commitType := matches[1]
		scope := matches[2]
		description := matches[3]

		// Validate commit type
		if len(v.config.PR.Title.Types) > 0 && !v.isValidType(commitType) {
			results = append(results, ValidationResult{
				Rule:       "pr-title-type",
				Level:      LevelError,
				Message:    fmt.Sprintf("Invalid commit type: '%s'", commitType),
				Suggestion: fmt.Sprintf("Allowed types: %s", strings.Join(v.config.PR.Title.Types, ", ")),
			})
		}

		// Validate scope if provided
		if scope != "" && len(v.config.PR.Title.Scopes) > 0 && !v.isValidScope(scope) {
			results = append(results, ValidationResult{
				Rule:       "pr-title-scope",
				Level:      LevelWarning,
				Message:    fmt.Sprintf("Unrecognized scope: '%s'", scope),
				Suggestion: fmt.Sprintf("Common scopes: %s", strings.Join(v.config.PR.Title.Scopes, ", ")),
			})
		}

		// Check description
		if len(description) < 3 {
			results = append(results, ValidationResult{
				Rule:       "pr-title-description",
				Level:      LevelWarning,
				Message:    "PR title description is too short",
				Suggestion: "Provide a more descriptive title",
			})
		}
	}

	// Validate title length
	if v.config.PR.Title.MaxLength > 0 && len(title) > v.config.PR.Title.MaxLength {
		results = append(results, ValidationResult{
			Rule:       "pr-title-length",
			Level:      LevelError,
			Message:    fmt.Sprintf("Title exceeds maximum length of %d characters", v.config.PR.Title.MaxLength),
			Suggestion: fmt.Sprintf("Current: %d characters, max: %d", len(title), v.config.PR.Title.MaxLength),
		})
	}

	if v.config.PR.Title.MinLength > 0 && len(title) < v.config.PR.Title.MinLength {
		results = append(results, ValidationResult{
			Rule:       "pr-title-min-length",
			Level:      LevelWarning,
			Message:    fmt.Sprintf("Title is shorter than minimum length of %d characters", v.config.PR.Title.MinLength),
			Suggestion: fmt.Sprintf("Current: %d characters, min: %d", len(title), v.config.PR.Title.MinLength),
		})
	}

	return results
}

// ValidateBranchName validates branch naming conventions
func (v *SemanticPRValidator) ValidateBranchName(branch string) []ValidationResult {
	var results []ValidationResult

	if branch == "" {
		results = append(results, ValidationResult{
			Rule:       "branch-required",
			Level:      LevelError,
			Message:    "Branch name is required",
			Suggestion: "Please provide a branch name",
		})
		return results
	}

	// Check branch prefix
	if len(v.config.PR.Branch.AllowedPrefix) > 0 {
		valid := false
		for _, prefix := range v.config.PR.Branch.AllowedPrefix {
			if strings.HasPrefix(branch, prefix) {
				valid = true
				break
			}
		}

		if !valid {
			results = append(results, ValidationResult{
				Rule:    "branch-prefix",
				Level:   LevelError,
				Message: "Branch name doesn't follow naming convention",
				Suggestion: fmt.Sprintf("Branch should start with one of: %s",
					strings.Join(v.config.PR.Branch.AllowedPrefix, ", ")),
			})
		}
	}

	// Check branch pattern
	if v.config.PR.Branch.Pattern != "" {
		pattern := regexp.MustCompile(v.config.PR.Branch.Pattern)
		if !pattern.MatchString(branch) {
			results = append(results, ValidationResult{
				Rule:       "branch-pattern",
				Level:      LevelError,
				Message:    "Branch name doesn't match required pattern",
				Suggestion: fmt.Sprintf("Pattern: %s\nExample: feature/add-login", v.config.PR.Branch.Pattern),
			})
		}
	}

	return results
}

// ValidatePRDescription validates PR description content
func (v *SemanticPRValidator) ValidatePRDescription(description string) []ValidationResult {
	var results []ValidationResult

	if v.config.PR.Description.Required && description == "" {
		results = append(results, ValidationResult{
			Rule:       "pr-description-required",
			Level:      LevelError,
			Message:    "PR description is required",
			Suggestion: "Please provide a description for this PR",
		})
		return results
	}

	if v.config.PR.Description.MinLength > 0 && len(description) < v.config.PR.Description.MinLength {
		results = append(results, ValidationResult{
			Rule:  "pr-description-length",
			Level: LevelWarning,
			Message: fmt.Sprintf("PR description is too short (min: %d characters)",
				v.config.PR.Description.MinLength),
			Suggestion: "Provide more context about the changes",
		})
	}

	// Check for required keywords
	if len(v.config.PR.Description.Keywords) > 0 {
		missingKeywords := []string{}
		for _, keyword := range v.config.PR.Description.Keywords {
			if !strings.Contains(strings.ToLower(description), strings.ToLower(keyword)) {
				missingKeywords = append(missingKeywords, keyword)
			}
		}

		if len(missingKeywords) > 0 {
			results = append(results, ValidationResult{
				Rule:       "pr-description-keywords",
				Level:      LevelInfo,
				Message:    "Consider adding these keywords to your description",
				Suggestion: fmt.Sprintf("Add: %s", strings.Join(missingKeywords, ", ")),
			})
		}
	}

	return results
}

func (v *SemanticPRValidator) isValidType(commitType string) bool {
	for _, t := range v.config.PR.Title.Types {
		if t == commitType {
			return true
		}
	}
	return false
}

func (v *SemanticPRValidator) isValidScope(scope string) bool {
	for _, s := range v.config.PR.Title.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
