package validator

import (
	"regexp"
	"strings"
)

type ValidationRule struct {
	Name        string
	Pattern     *regexp.Regexp
	Description string
	Validate    func(string) bool
}

type CommitRules struct {
	Rules []ValidationRule
}

func NewCommitRules() *CommitRules {
	return &CommitRules{
		Rules: []ValidationRule{
			{
				Name:        "Conventional Commit",
				Pattern:     regexp.MustCompile(`^(feat|fix|docs|style|refactor|test|chore|perf|ci|build|revert)(\(.+\))?: .+$`),
				Description: "Commit must follow conventional commit format",
				Validate: func(msg string) bool {
					return validateConventionalCommit(msg)
				},
			},
			{
				Name:        "Length Limit",
				Description: "Commit subject must be under 50 characters",
				Validate: func(msg string) bool {
					return len(strings.Split(msg, "\n")[0]) <= 50
				},
			},
			{
				Name:        "No Trailing Period",
				Description: "Commit subject should not end with a period",
				Validate: func(msg string) bool {
					subject := strings.Split(msg, "\n")[0]
					return !strings.HasSuffix(subject, ".")
				},
			},
		},
	}
}

func validateConventionalCommit(msg string) bool {
	pattern := regexp.MustCompile(`^(feat|fix|docs|style|refactor|test|chore|perf|ci|build|revert)(\(.+\))?: .+$`)
	firstLine := strings.Split(msg, "\n")[0]
	return pattern.MatchString(firstLine)
}

func (cr *CommitRules) ValidateCommitMessage(msg string) []string {
	var violations []string

	for _, rule := range cr.Rules {
		if !rule.Validate(msg) {
			violations = append(violations, rule.Description)
		}
	}

	return violations
}
