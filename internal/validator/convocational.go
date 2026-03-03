package validator

import (
	"fmt"
	"regexp"
	"strings"
)

// ConventionalCommitValidator handles commit message validations
type ConventionalCommitValidator struct {
	config *Config
}

func NewConventionalCommitValidator(config *Config) *ConventionalCommitValidator {
	return &ConventionalCommitValidator{
		config: config,
	}
}

// ValidateCommit validates a single commit message
func (v *ConventionalCommitValidator) ValidateCommit(commitMsg string) []ValidationResult {
	var results []ValidationResult

	if commitMsg == "" {
		results = append(results, ValidationResult{
			Rule:       "commit-required",
			Level:      LevelError,
			Message:    "Commit message is required",
			Suggestion: "Please provide a commit message",
		})
		return results
	}

	// Parse commit message
	commitInfo := v.parseCommitMessage(commitMsg)

	// Check for fixup/squash commits
	if commitInfo.IsFixup {
		results = append(results, ValidationResult{
			Rule:       "commit-fixup",
			Level:      LevelWarning,
			Message:    "Fixup commit detected",
			Suggestion: "Consider squashing this commit before merging",
		})
		return results
	}

	if commitInfo.IsSquash {
		results = append(results, ValidationResult{
			Rule:       "commit-squash",
			Level:      LevelWarning,
			Message:    "Squash commit detected",
			Suggestion: "Consider squashing this commit before merging",
		})
		return results
	}

	// Validate commit format
	if !v.isConventionalFormat(commitMsg) && v.config.Commits.Conventional {
		results = append(results, ValidationResult{
			Rule:       "commit-format",
			Level:      LevelError,
			Message:    "Commit message must follow conventional commits format",
			Suggestion: v.getFormatSuggestion(),
		})
		return results
	}

	// Validate commit type
	if commitInfo.Type != "" && len(v.config.PR.Commits.AllowedTypes) > 0 {
		if !v.isValidType(commitInfo.Type) {
			results = append(results, ValidationResult{
				Rule:       "commit-type",
				Level:      LevelError,
				Message:    fmt.Sprintf("Invalid commit type: '%s'", commitInfo.Type),
				Suggestion: fmt.Sprintf("Allowed types: %s", strings.Join(v.config.PR.Commits.AllowedTypes, ", ")),
			})
		}
	}

	// Validate subject length
	if v.config.PR.Commits.SubjectLength.Max > 0 && len(commitInfo.Subject) > v.config.PR.Commits.SubjectLength.Max {
		results = append(results, ValidationResult{
			Rule:  "commit-subject-length",
			Level: LevelError,
			Message: fmt.Sprintf("Subject exceeds maximum length of %d characters",
				v.config.PR.Commits.SubjectLength.Max),
			Suggestion: fmt.Sprintf("Current: %d characters, max: %d",
				len(commitInfo.Subject), v.config.PR.Commits.SubjectLength.Max),
		})
	}

	if v.config.PR.Commits.SubjectLength.Min > 0 && len(commitInfo.Subject) < v.config.PR.Commits.SubjectLength.Min {
		results = append(results, ValidationResult{
			Rule:  "commit-subject-min-length",
			Level: LevelWarning,
			Message: fmt.Sprintf("Subject is too short (min: %d characters)",
				v.config.PR.Commits.SubjectLength.Min),
			Suggestion: fmt.Sprintf("Current: %d characters, min: %d",
				len(commitInfo.Subject), v.config.PR.Commits.SubjectLength.Min),
		})
	}

	// Validate scope if required
	if v.config.PR.Commits.RequireScope && commitInfo.Scope == "" && commitInfo.Type != "" {
		results = append(results, ValidationResult{
			Rule:       "commit-scope",
			Level:      LevelWarning,
			Message:    "Commit scope is recommended",
			Suggestion: "Add a scope to indicate which part of the codebase is affected\nExample: feat(api): add endpoint",
		})
	}

	// Check for breaking changes
	if commitInfo.IsBreaking {
		results = append(results, ValidationResult{
			Rule:       "commit-breaking",
			Level:      LevelInfo,
			Message:    "This commit introduces breaking changes",
			Suggestion: "Ensure breaking changes are documented in the commit body",
		})
	}

	return results
}

// ValidateCommits validates multiple commits
func (v *ConventionalCommitValidator) ValidateCommits(commits []CommitInfo) map[string][]ValidationResult {
	results := make(map[string][]ValidationResult)

	for _, commit := range commits {
		if v.config.PR.Commits.IgnoreMerge && strings.HasPrefix(commit.Message, "Merge") {
			continue
		}

		commitResults := v.ValidateCommit(commit.Message)
		if len(commitResults) > 0 {
			results[commit.Hash] = commitResults
		}
	}

	return results
}

func (v *ConventionalCommitValidator) parseCommitMessage(msg string) *CommitInfo {
	info := &CommitInfo{
		Message: msg,
	}

	lines := strings.Split(msg, "\n")
	if len(lines) == 0 {
		return info
	}

	// Parse header
	header := lines[0]

	// Check for fixup/squash
	if strings.HasPrefix(header, "fixup!") {
		info.IsFixup = true
		header = strings.TrimSpace(strings.TrimPrefix(header, "fixup!"))
	} else if strings.HasPrefix(header, "squash!") {
		info.IsSquash = true
		header = strings.TrimSpace(strings.TrimPrefix(header, "squash!"))
	}

	// Parse conventional commit format
	pattern := regexp.MustCompile(`^(\w+)(?:\(([\w\-]+)\))?(!)?: (.*)$`)
	matches := pattern.FindStringSubmatch(strings.TrimSpace(header))

	if len(matches) == 5 {
		info.Type = matches[1]
		info.Scope = matches[2]
		if matches[3] == "!" {
			info.IsBreaking = true
		}
		info.Subject = matches[4]
	} else {
		info.Subject = header
	}

	// Parse body and footer
	if len(lines) > 1 {
		bodyLines := []string{}
		footerLines := []string{}
		isFooter := false

		for i := 1; i < len(lines); i++ {
			line := lines[i]

			if line == "" {
				isFooter = true
				continue
			}

			if !isFooter {
				bodyLines = append(bodyLines, line)
			} else {
				footerLines = append(footerLines, line)
			}
		}

		info.Body = strings.Join(bodyLines, "\n")
		info.Footer = strings.Join(footerLines, "\n")

		// Check for breaking change in footer
		if strings.Contains(info.Footer, "BREAKING CHANGE:") {
			info.IsBreaking = true
		}
	}

	return info
}

func (v *ConventionalCommitValidator) isConventionalFormat(msg string) bool {
	pattern := regexp.MustCompile(`^(\w+)(?:\([\w\-]+\))?!?: .+`)
	firstLine := strings.Split(msg, "\n")[0]
	return pattern.MatchString(firstLine)
}

func (v *ConventionalCommitValidator) isValidType(commitType string) bool {
	for _, t := range v.config.PR.Commits.AllowedTypes {
		if t == commitType {
			return true
		}
	}
	return false
}

func (v *ConventionalCommitValidator) getFormatSuggestion() string {
	return `Commit message should be structured as follows:

<type>[optional scope]: <description>

[optional body]

[optional footer(s)]

Types:
- feat: A new feature
- fix: A bug fix
- docs: Documentation only changes
- style: Code style changes
- refactor: Code refactoring
- perf: Performance improvements
- test: Test updates
- chore: Maintenance tasks

Examples:
- feat: add new feature
- fix(api): resolve null pointer exception
- feat!: breaking change with exclamation
- docs: update README`
}
