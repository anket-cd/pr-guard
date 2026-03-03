package validator

import (
	"context"
	"fmt"
	"strings"
	"time"

	githubClient "github.com/anket-cd/pr-guard/internal/githubc"
	"github.com/google/go-github/v57/github"
)

type PRGuard struct {
	semanticValidator *SemanticPRValidator
	commitValidator   *ConventionalCommitValidator
	githubClient      *githubClient.Client
	config            *Config
}

func NewPRGuard(config *Config, client *githubClient.Client) *PRGuard {
	return &PRGuard{
		semanticValidator: NewSemanticPRValidator(config),
		commitValidator:   NewConventionalCommitValidator(config),
		githubClient:      client,
		config:            config,
	}
}

// ValidatePR performs complete PR validation
func (g *PRGuard) ValidatePR(ctx context.Context, owner, repo string, prNumber int) (*PRValidationReport, error) {
	startTime := time.Now()

	report := &PRValidationReport{
		PRNumber: prNumber,
		Results:  make([]ValidationResult, 0),
		Commits:  make([]CommitInfo, 0),
	}

	// Fetch PR details using the underlying github client
	pr, _, err := g.githubClient.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR: %v", err)
	}

	report.Title = pr.GetTitle()
	report.Description = pr.GetBody()
	report.Branch = pr.GetHead().GetRef()
	report.BaseBranch = pr.GetBase().GetRef()

	// Validate PR title
	titleResults := g.semanticValidator.ValidatePRTitle(report.Title)
	report.Results = append(report.Results, titleResults...)

	// Validate branch name
	branchResults := g.semanticValidator.ValidateBranchName(report.Branch)
	report.Results = append(report.Results, branchResults...)

	// Validate PR description
	descResults := g.semanticValidator.ValidatePRDescription(report.Description)
	report.Results = append(report.Results, descResults...)

	// Fetch and validate commits
	commits, err := g.fetchPRCommits(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commits: %v", err)
	}

	report.Commits = commits

	// Validate each commit
	for _, commit := range commits {
		if g.config.PR.Commits.IgnoreMerge && strings.HasPrefix(commit.Message, "Merge") {
			continue
		}

		commitResults := g.commitValidator.ValidateCommit(commit.Message)
		for _, result := range commitResults {
			result.Message = fmt.Sprintf("Commit %s: %s", commit.Hash[:7], result.Message)
			report.Results = append(report.Results, result)
		}
	}

	// Validate labels
	labelResults := g.validatePRLabels(pr.Labels)
	report.Results = append(report.Results, labelResults...)

	// Determine overall status
	report.Passed = g.hasNoErrors(report.Results)
	report.Duration = time.Since(startTime)

	return report, nil
}

func (g *PRGuard) fetchPRCommits(ctx context.Context, owner, repo string, prNumber int) ([]CommitInfo, error) {
	var allCommits []*github.RepositoryCommit
	opts := &github.ListOptions{PerPage: 100}

	for {
		commits, resp, err := g.githubClient.PullRequests.ListCommits(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return nil, err
		}
		allCommits = append(allCommits, commits...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	var commitInfos []CommitInfo
	for _, c := range allCommits {
		info := CommitInfo{
			Hash:    c.GetSHA(),
			Message: c.GetCommit().GetMessage(),
			Author:  c.GetCommit().GetAuthor().GetName(),
			Date:    c.GetCommit().GetAuthor().GetDate().Time,
		}

		// Parse commit info
		parsed := g.commitValidator.parseCommitMessage(info.Message)
		info.Type = parsed.Type
		info.Scope = parsed.Scope
		info.Subject = parsed.Subject
		info.Body = parsed.Body
		info.Footer = parsed.Footer
		info.IsBreaking = parsed.IsBreaking
		info.IsFixup = parsed.IsFixup
		info.IsSquash = parsed.IsSquash

		commitInfos = append(commitInfos, info)
	}

	return commitInfos, nil
}

func (g *PRGuard) validatePRLabels(labels []*github.Label) []ValidationResult {
	var results []ValidationResult

	if len(g.config.PR.Labels.Require) == 0 && len(g.config.PR.Labels.Forbid) == 0 {
		return results
	}

	existingLabels := make(map[string]bool)
	for _, label := range labels {
		existingLabels[label.GetName()] = true
	}

	// Check required labels
	for _, required := range g.config.PR.Labels.Require {
		if !existingLabels[required] {
			results = append(results, ValidationResult{
				Rule:       "pr-labels-required",
				Level:      LevelError,
				Message:    fmt.Sprintf("Missing required label: '%s'", required),
				Suggestion: fmt.Sprintf("Add the '%s' label to this PR", required),
			})
		}
	}

	// Check forbidden labels
	for _, forbidden := range g.config.PR.Labels.Forbid {
		if existingLabels[forbidden] {
			results = append(results, ValidationResult{
				Rule:       "pr-labels-forbidden",
				Level:      LevelError,
				Message:    fmt.Sprintf("Forbidden label found: '%s'", forbidden),
				Suggestion: fmt.Sprintf("Remove the '%s' label from this PR", forbidden),
			})
		}
	}

	return results
}

func (g *PRGuard) hasNoErrors(results []ValidationResult) bool {
	for _, r := range results {
		if r.Level == LevelError {
			return false
		}
	}
	return true
}

// GenerateMarkdownReport creates a formatted markdown report
func (g *PRGuard) GenerateMarkdownReport(report *PRValidationReport) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# 🔒 PR Guard Validation Report\n\n")

	// Status
	if report.Passed {
		sb.WriteString("## ✅ Validation Passed\n\n")
	} else {
		sb.WriteString("## ❌ Validation Failed\n\n")
	}

	// Summary table
	sb.WriteString("### Summary\n")
	sb.WriteString("| Check | Status | Details |\n")
	sb.WriteString("|-------|--------|---------|\n")

	// Count results by type
	var errors, warnings, infos int
	for _, r := range report.Results {
		switch r.Level {
		case LevelError:
			errors++
		case LevelWarning:
			warnings++
		case LevelInfo:
			infos++
		}
	}

	sb.WriteString(fmt.Sprintf("| **PR #%d** | `%s` | ", report.PRNumber, report.Branch))
	sb.WriteString(fmt.Sprintf("%d commits |\n", len(report.Commits)))
	sb.WriteString(fmt.Sprintf("| **Errors** | ❌ %d | Need to fix |\n", errors))
	sb.WriteString(fmt.Sprintf("| **Warnings** | ⚠️ %d | Should address |\n", warnings))
	sb.WriteString(fmt.Sprintf("| **Info** | ℹ️ %d | For your information |\n", infos))
	sb.WriteString(fmt.Sprintf("| **Duration** | ⏱️ %.2fs | |\n\n", report.Duration.Seconds()))

	// PR Details
	sb.WriteString("### 📋 PR Details\n")
	sb.WriteString(fmt.Sprintf("- **Title:** `%s`\n", report.Title))
	sb.WriteString(fmt.Sprintf("- **Branch:** `%s` → `%s`\n", report.Branch, report.BaseBranch))
	if len(report.Description) > 100 {
		sb.WriteString(fmt.Sprintf("- **Description:** %s...\n", report.Description[:100]))
	} else {
		sb.WriteString(fmt.Sprintf("- **Description:** %s\n", report.Description))
	}
	sb.WriteString("\n")

	// Errors
	if errors > 0 {
		sb.WriteString("### 🚫 Errors (Must Fix)\n")
		for _, r := range report.Results {
			if r.Level == LevelError {
				sb.WriteString(fmt.Sprintf("- **%s:** %s\n", r.Rule, r.Message))
				if r.Suggestion != "" {
					sb.WriteString(fmt.Sprintf("  - 💡 %s\n", r.Suggestion))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Warnings
	if warnings > 0 {
		sb.WriteString("### ⚠️ Warnings (Should Address)\n")
		for _, r := range report.Results {
			if r.Level == LevelWarning {
				sb.WriteString(fmt.Sprintf("- **%s:** %s\n", r.Rule, r.Message))
				if r.Suggestion != "" {
					sb.WriteString(fmt.Sprintf("  - 💡 %s\n", r.Suggestion))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Info
	if infos > 0 {
		sb.WriteString("### ℹ️ Information\n")
		for _, r := range report.Results {
			if r.Level == LevelInfo {
				sb.WriteString(fmt.Sprintf("- **%s:** %s\n", r.Rule, r.Message))
				if r.Suggestion != "" {
					sb.WriteString(fmt.Sprintf("  - 💡 %s\n", r.Suggestion))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Commits overview
	if len(report.Commits) > 0 {
		sb.WriteString("### 📝 Commits\n")
		sb.WriteString("| Hash | Message | Status |\n")
		sb.WriteString("|------|---------|--------|\n")

		for _, c := range report.Commits {
			status := "✅"
			for _, r := range report.Results {
				if strings.Contains(r.Message, c.Hash[:7]) && r.Level == LevelError {
					status = "❌"
					break
				} else if strings.Contains(r.Message, c.Hash[:7]) && r.Level == LevelWarning {
					status = "⚠️"
				}
			}

			shortMsg := c.Subject
			if len(shortMsg) > 50 {
				shortMsg = shortMsg[:47] + "..."
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", c.Hash[:7], shortMsg, status))
		}
		sb.WriteString("\n")
	}

	// Next steps
	if !report.Passed {
		sb.WriteString("### 🔧 Next Steps\n")
		sb.WriteString("1. Fix all errors listed above\n")
		sb.WriteString("2. Address warnings if applicable\n")
		sb.WriteString("3. Push new commits to update this PR\n")
		sb.WriteString("4. Add required labels\n\n")
	}

	// Footer
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("<sub>Report generated by PR Guard in %.2f seconds</sub>\n", report.Duration.Seconds()))

	return sb.String()
}
