package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gh "github.com/cultome/xp_2077/internal/github"
	"github.com/cultome/xp_2077/internal/store"
)

func main() {
	config := loadConfig()
	if err := config.validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	ctx := context.Background()
	client := gh.NewClient(config.Token)

	fmt.Printf("Fetching Project v2 items (%s #%d)\n", config.Owner, config.ProjectNumber)
	projectRaw, projectIssues, err := client.FetchProjectV2Issues(ctx, config.Owner, config.ProjectNumber)
	if err != nil {
		log.Fatalf("failed to fetch project issues: %v", err)
	}

	fmt.Printf("Fetching repository issues (%s/%s)\n", config.Owner, config.Repo)
	repoRaw, repoIssues, err := client.FetchRepoIssues(ctx, config.Owner, config.Repo)
	if err != nil {
		log.Fatalf("failed to fetch repo issues: %v", err)
	}

	allNormalized := make([]gh.NormalizedIssue, 0, len(projectIssues)+len(repoIssues))
	allNormalized = append(allNormalized, projectIssues...)
	allNormalized = append(allNormalized, repoIssues...)

	db, err := store.OpenSQLite(config.OutputDB)
	if err != nil {
		log.Fatalf("failed to open sqlite store: %v", err)
	}
	defer db.Close()

	if err := db.ApplySchema(ctx); err != nil {
		log.Fatalf("failed to apply sqlite schema: %v", err)
	}
	if err := db.UpsertProjectRaw(ctx, projectRaw); err != nil {
		log.Fatalf("failed to persist project raw records: %v", err)
	}
	if err := db.UpsertRepoRaw(ctx, repoRaw); err != nil {
		log.Fatalf("failed to persist repo raw records: %v", err)
	}
	if err := db.UpsertNormalized(ctx, allNormalized); err != nil {
		log.Fatalf("failed to persist normalized records: %v", err)
	}

	counts, err := db.Counts(ctx)
	if err != nil {
		log.Fatalf("failed to read sqlite counts: %v", err)
	}

	printSummary(config.OutputDB, projectRaw, repoRaw, allNormalized, counts)
}

type Config struct {
	Token         string
	Owner         string
	Repo          string
	ProjectNumber int
	OutputDB      string
}

func loadConfig() Config {
	defaultOwner := firstNonEmpty(os.Getenv("GITHUB_OWNER"), os.Getenv("GITHUB_ORG"))
	defaultRepo := os.Getenv("GITHUB_REPO")
	defaultToken := os.Getenv("GITHUB_TOKEN")
	defaultDB := firstNonEmpty(os.Getenv("OUTPUT_DB"), "./tmp/github_extract.db")

	owner := flag.String("owner", defaultOwner, "GitHub owner or org login")
	repo := flag.String("repo", defaultRepo, "GitHub repository name (without owner)")
	project := flag.Int("project", envInt("GITHUB_PROJECT_NUMBER", 0), "GitHub Project v2 number")
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

func (c Config) validate() error {
	if c.Token == "" {
		return fmt.Errorf("token is required (flag -token or GITHUB_TOKEN)")
	}
	if c.Owner == "" {
		return fmt.Errorf("owner is required (flag -owner or GITHUB_OWNER/GITHUB_ORG)")
	}
	if c.Repo == "" {
		return fmt.Errorf("repo is required (flag -repo or GITHUB_REPO)")
	}
	if c.ProjectNumber <= 0 {
		return fmt.Errorf("project must be > 0 (flag -project or GITHUB_PROJECT_NUMBER)")
	}
	if c.OutputDB == "" {
		return fmt.Errorf("db path is required")
	}
	return nil
}

func printSummary(path string, projectRaw []gh.ProjectItemRawRecord, repoRaw []gh.RepoIssueRawRecord, all []gh.NormalizedIssue, counts store.SummaryCounts) {
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

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(raw, "%d", &parsed); err != nil {
		return fallback
	}
	return parsed
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
