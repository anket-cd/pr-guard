package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anket-cd/pr-guard/internal/githubc"
	"github.com/anket-cd/pr-guard/internal/validator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	owner       string
	repo        string
	prNumber    int
	commitRange string
	format      string
	token       string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "pr-guard",
		Short: "PR Guard - Comprehensive PR and commit validation",
		Long: `PR Guard combines semantic PR validation with conventional commits
to ensure high quality pull requests and commit messages.`,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .pr-guard.yaml)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", "output format: text, json, markdown")

	validatePRCmd := &cobra.Command{
		Use:   "validate-pr",
		Short: "Validate a pull request",
		RunE:  validatePR,
	}

	validatePRCmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	validatePRCmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	validatePRCmd.Flags().IntVar(&prNumber, "pr", 0, "PR number")
	validatePRCmd.Flags().StringVar(&token, "token", "", "GitHub token (or set GITHUB_TOKEN env)")
	validatePRCmd.MarkFlagRequired("owner")
	validatePRCmd.MarkFlagRequired("repo")
	validatePRCmd.MarkFlagRequired("pr")

	validateCommitCmd := &cobra.Command{
		Use:   "validate-commit [message]",
		Short: "Validate a single commit message",
		Args:  cobra.ExactArgs(1),
		RunE:  validateCommit,
	}

	validateCommitsCmd := &cobra.Command{
		Use:   "validate-commits [range]",
		Short: "Validate commits in a range",
		Args:  cobra.ExactArgs(1),
		RunE:  validateCommits,
	}

	validateLocalCmd := &cobra.Command{
		Use:   "validate-local",
		Short: "Validate local commit (for git hooks)",
		RunE:  validateLocal,
	}

	validateBranchCmd := &cobra.Command{
		Use:   "validate-branch [branch]",
		Short: "Validate branch name",
		Args:  cobra.ExactArgs(1),
		RunE:  validateBranch,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("PR Guard v1.0.0")
		},
	}

	rootCmd.AddCommand(validatePRCmd)
	rootCmd.AddCommand(validateCommitCmd)
	rootCmd.AddCommand(validateCommitsCmd)
	rootCmd.AddCommand(validateLocalCmd)
	rootCmd.AddCommand(validateBranchCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadConfig() (*validator.Config, error) {
	v := viper.New()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName(".pr-guard")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.pr-guard")
	}

	v.SetConfigType("yaml")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		// Return default config if no config file found
		return getDefaultConfig(), nil
	}

	var config validator.Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}

func getDefaultConfig() *validator.Config {
	cfg := &validator.Config{}

	cfg.PR.Title.Conventional = true
	cfg.PR.Title.Types = []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "chore"}
	cfg.PR.Title.MaxLength = 100
	cfg.PR.Title.MinLength = 10

	cfg.PR.Description.Required = true
	cfg.PR.Description.MinLength = 20

	cfg.PR.Branch.AllowedPrefix = []string{"feature/", "bugfix/", "hotfix/", "release/", "chore/"}

	cfg.PR.Commits.ValidateAll = true
	cfg.PR.Commits.IgnoreMerge = true
	cfg.PR.Commits.AllowedTypes = []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "chore"}
	cfg.PR.Commits.SubjectLength.Max = 50
	cfg.PR.Commits.SubjectLength.Min = 3

	cfg.Commits.Conventional = true
	cfg.Commits.CheckFooter = true
	cfg.Commits.BreakingPattern = "BREAKING CHANGE:"

	return cfg
}

func validatePR(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Get token from flag or environment
	githubToken := token
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}
	if githubToken == "" {
		return fmt.Errorf("GitHub token required. Set GITHUB_TOKEN environment variable or use --token flag")
	}

	client := githubc.NewClient(githubToken)
	guard := validator.NewPRGuard(cfg, client)

	report, err := guard.ValidatePR(context.Background(), owner, repo, prNumber)
	if err != nil {
		return err
	}

	// Output based on format
	switch format {
	case "json":
		output, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(output))
	case "markdown":
		fmt.Print(guard.GenerateMarkdownReport(report))
	default:
		printTextReport(report)
	}

	if !report.Passed {
		os.Exit(1)
	}

	return nil
}

func validateCommit(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	commitValidator := validator.NewConventionalCommitValidator(cfg)
	results := commitValidator.ValidateCommit(args[0])

	if format == "json" {
		output, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(output))
	} else {
		printCommitResults(args[0], results)
	}

	for _, r := range results {
		if r.Level == validator.LevelError {
			os.Exit(1)
		}
	}

	return nil
}

func validateCommits(cmd *cobra.Command, args []string) error {
	// This would need git integration
	fmt.Println("Commit range validation not yet implemented")
	return nil
}

func validateLocal(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	commitValidator := validator.NewConventionalCommitValidator(cfg)

	// Read commit message from file (for git hooks)
	if len(os.Args) > 2 {
		commitMsgFile := os.Args[2]
		msg, err := os.ReadFile(commitMsgFile)
		if err != nil {
			return fmt.Errorf("failed to read commit message: %v", err)
		}

		results := commitValidator.ValidateCommit(string(msg))
		printCommitResults(string(msg), results)

		for _, r := range results {
			if r.Level == validator.LevelError {
				os.Exit(1)
			}
		}
	}

	return nil
}

func validateBranch(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	semanticValidator := validator.NewSemanticPRValidator(cfg)
	results := semanticValidator.ValidateBranchName(args[0])

	if format == "json" {
		output, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(output))
	} else {
		printBranchResults(args[0], results)
	}

	for _, r := range results {
		if r.Level == validator.LevelError {
			os.Exit(1)
		}
	}

	return nil
}

func printTextReport(report *validator.PRValidationReport) {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("\n%s PR Guard Validation Report %s\n", cyan("╔"), cyan("╗"))
	fmt.Printf("%s PR #%d: %s %s\n", cyan("║"), report.PRNumber, report.Title, cyan("║"))
	fmt.Printf("%s%s\n", cyan("╚"), cyan("══════════════════════════════════════════════════════════════╝"))

	if report.Passed {
		fmt.Printf("\n%s %s\n", green("✅"), green("VALIDATION PASSED"))
	} else {
		fmt.Printf("\n%s %s\n", red("❌"), red("VALIDATION FAILED"))
	}

	fmt.Printf("\n%s Summary\n", cyan("📊"))
	fmt.Printf("  Branch: %s → %s\n", report.Branch, report.BaseBranch)
	fmt.Printf("  Commits: %d\n", len(report.Commits))
	fmt.Printf("  Duration: %.2fs\n", report.Duration.Seconds())

	var errors, warnings, infos int
	for _, r := range report.Results {
		switch r.Level {
		case validator.LevelError:
			errors++
		case validator.LevelWarning:
			warnings++
		case validator.LevelInfo:
			infos++
		}
	}

	fmt.Printf("\n%s Results\n", cyan("📋"))
	if errors > 0 {
		fmt.Printf("  %s Errors: %d\n", red("❌"), errors)
	}
	if warnings > 0 {
		fmt.Printf("  %s Warnings: %d\n", yellow("⚠️"), warnings)
	}
	if infos > 0 {
		fmt.Printf("  %s Info: %d\n", cyan("ℹ️"), infos)
	}

	if errors > 0 {
		fmt.Printf("\n%s Errors (Must Fix)\n", red("🚫"))
		for _, r := range report.Results {
			if r.Level == validator.LevelError {
				fmt.Printf("  • %s\n", r.Message)
				if r.Suggestion != "" {
					fmt.Printf("    %s %s\n", cyan("💡"), r.Suggestion)
				}
			}
		}
	}

	if warnings > 0 {
		fmt.Printf("\n%s Warnings (Should Address)\n", yellow("⚠️"))
		for _, r := range report.Results {
			if r.Level == validator.LevelWarning {
				fmt.Printf("  • %s\n", r.Message)
				if r.Suggestion != "" {
					fmt.Printf("    %s %s\n", cyan("💡"), r.Suggestion)
				}
			}
		}
	}

	fmt.Println()
}

func printCommitResults(commit string, results []validator.ValidationResult) {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("\nCommit: %s\n", commit)

	var errors, warnings int
	for _, r := range results {
		switch r.Level {
		case validator.LevelError:
			errors++
		case validator.LevelWarning:
			warnings++
		}
	}

	if errors == 0 && warnings == 0 {
		fmt.Printf("%s Commit message is valid\n", green("✅"))
		return
	}

	if errors > 0 {
		fmt.Printf("\n%s Errors:\n", red("❌"))
		for _, r := range results {
			if r.Level == validator.LevelError {
				fmt.Printf("  • %s\n", r.Message)
				if r.Suggestion != "" {
					fmt.Printf("    %s %s\n", cyan("💡"), r.Suggestion)
				}
			}
		}
	}

	if warnings > 0 {
		fmt.Printf("\n%s Warnings:\n", yellow("⚠️"))
		for _, r := range results {
			if r.Level == validator.LevelWarning {
				fmt.Printf("  • %s\n", r.Message)
				if r.Suggestion != "" {
					fmt.Printf("    %s %s\n", cyan("💡"), r.Suggestion)
				}
			}
		}
	}

	fmt.Println()
}

func printBranchResults(branch string, results []validator.ValidationResult) {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("\nBranch: %s\n", branch)

	var errors int
	for _, r := range results {
		if r.Level == validator.LevelError {
			errors++
		}
	}

	if errors == 0 {
		fmt.Printf("%s Branch name is valid\n", green("✅"))
		return
	}

	fmt.Printf("\n%s Errors:\n", red("❌"))
	for _, r := range results {
		if r.Level == validator.LevelError {
			fmt.Printf("  • %s\n", r.Message)
			if r.Suggestion != "" {
				fmt.Printf("    %s %s\n", cyan("💡"), r.Suggestion)
			}
		}
	}

	fmt.Println()
}
