package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cultome/xp_2077/internal/extract"
)

func main() {
	config := loadConfig()
	extractConfig := extract.Config{
		Token:         config.Token,
		Owner:         config.Owner,
		Repo:          config.Repo,
		ProjectNumber: config.ProjectNumber,
		OutputDB:      config.OutputDB,
	}
	if err := extractConfig.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	result, err := extract.Run(context.Background(), extractConfig, nil)
	if err != nil {
		log.Fatalf("extraction failed: %v", err)
	}

	printSummary(config.OutputDB, result)
}

type Config struct {
	Token         string
	Owner         string
	Repo          string
	ProjectNumber int
	OutputDB      string
}

func loadConfig() Config {
	defaultOwner := extract.DefaultOwner
	defaultRepo := extract.DefaultRepo
	defaultProject := extract.DefaultProjectNumber
	defaultToken := os.Getenv("GITHUB_TOKEN")
	defaultDB := firstNonEmpty(os.Getenv("OUTPUT_DB"), "./tmp/github_extract.db")

	owner := flag.String("owner", defaultOwner, "GitHub owner or org login")
	repo := flag.String("repo", defaultRepo, "GitHub repository name (without owner)")
	project := flag.Int("project", defaultProject, "GitHub Project v2 number")
	token := flag.String("token", defaultToken, "GitHub token (or GITHUB_TOKEN env var)")
	outputDB := flag.String("db", defaultDB, "SQLite output database path")

	flag.Parse()
	cfg := Config{
		Token:         strings.TrimSpace(*token),
		Owner:         strings.TrimSpace(*owner),
		Repo:          strings.TrimSpace(*repo),
		ProjectNumber: *project,
		OutputDB:      strings.TrimSpace(*outputDB),
	}
	if repoOwner, repoName, ok := splitRepoRef(cfg.Repo); ok {
		// If repo is passed as owner/repo, trust that pair.
		cfg.Owner = repoOwner
		cfg.Repo = repoName
	}
	return cfg
}

func printSummary(path string, result extract.Result) {
	projectRaw := result.ProjectRaw
	repoRaw := result.RepoRaw
	all := result.Normalized
	counts := result.Counts
	duplicateByNodeID := map[string]int{}
	var (
		minUpdated       time.Time
		maxUpdated       time.Time
		xpComputedCount  int
		issueBodyPresent int
	)

	for _, issue := range all {
		if issue.IssueNodeID != "" {
			duplicateByNodeID[issue.IssueNodeID]++
		}
		if !issue.UpdatedAt.IsZero() {
			if minUpdated.IsZero() || issue.UpdatedAt.Before(minUpdated) {
				minUpdated = issue.UpdatedAt
			}
			if maxUpdated.IsZero() || issue.UpdatedAt.After(maxUpdated) {
				maxUpdated = issue.UpdatedAt
			}
		}
		if issue.XPFinal != nil {
			xpComputedCount++
		}
		if strings.TrimSpace(issue.IssueBody) != "" {
			issueBodyPresent++
		}
	}

	duplicates := 0
	for _, n := range duplicateByNodeID {
		if n > 1 {
			duplicates++
		}
	}

	fmt.Println()
	fmt.Println("Extraction summary")
	fmt.Printf("- Output DB: %s\n", path)
	fmt.Printf("- Fetched Project v2 raw records: %d\n", len(projectRaw))
	fmt.Printf("- Fetched repo raw records: %d\n", len(repoRaw))
	fmt.Printf("- Fetched normalized records: %d\n", len(all))
	fmt.Printf("- SQLite project_items_raw count: %d\n", counts.ProjectRawCount)
	fmt.Printf("- SQLite repo_issues_raw count: %d\n", counts.RepoRawCount)
	fmt.Printf("- SQLite issues_normalized count: %d\n", counts.NormalizedCount)
	fmt.Printf("- Normalized records with computed xp_final: %d\n", xpComputedCount)
	fmt.Printf("- Normalized records with issue body: %d\n", issueBodyPresent)
	fmt.Printf("- Duplicate issue_node_id entries across sources: %d\n", duplicates)
	if minUpdated.IsZero() || maxUpdated.IsZero() {
		fmt.Println("- UpdatedAt range: N/A")
	} else {
		fmt.Printf("- UpdatedAt range: %s .. %s\n", minUpdated.UTC().Format(time.RFC3339), maxUpdated.UTC().Format(time.RFC3339))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func splitRepoRef(value string) (owner string, repo string, ok bool) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 {
		return "", "", false
	}
	owner = strings.TrimSpace(parts[0])
	repo = strings.TrimSpace(parts[1])
	if owner == "" || repo == "" {
		return "", "", false
	}
	return owner, repo, true
}
