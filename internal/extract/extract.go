package extract

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	gh "github.com/cultome/xp_2077/internal/github"
	"github.com/cultome/xp_2077/internal/store"
)

type Config struct {
	Token         string
	Owner         string
	Repo          string
	ProjectNumber int
	OutputDB      string
}

func ConfigFromEnv() Config {
	owner := firstNonEmpty(os.Getenv("GITHUB_OWNER"), os.Getenv("GITHUB_ORG"))
	repo := strings.TrimSpace(os.Getenv("GITHUB_REPO"))
	project := envInt("GITHUB_PROJECT_NUMBER", 0)
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	outputDB := firstNonEmpty(os.Getenv("OUTPUT_DB"), "./tmp/github_extract.db")

	if repoOwner, repoName, ok := splitRepoRef(repo); ok {
		owner = repoOwner
		repo = repoName
	}

	return Config{
		Token:         token,
		Owner:         owner,
		Repo:          repo,
		ProjectNumber: project,
		OutputDB:      strings.TrimSpace(outputDB),
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("token is required (GITHUB_TOKEN)")
	}
	if strings.TrimSpace(c.Owner) == "" {
		return fmt.Errorf("owner is required (GITHUB_OWNER or GITHUB_ORG)")
	}
	if strings.TrimSpace(c.Repo) == "" {
		return fmt.Errorf("repo is required (GITHUB_REPO)")
	}
	if c.ProjectNumber <= 0 {
		return fmt.Errorf("project number must be > 0 (GITHUB_PROJECT_NUMBER)")
	}
	if strings.TrimSpace(c.OutputDB) == "" {
		return fmt.Errorf("db path is required (OUTPUT_DB)")
	}
	return nil
}

type Result struct {
	ProjectRaw []gh.ProjectItemRawRecord
	RepoRaw    []gh.RepoIssueRawRecord
	Normalized []gh.NormalizedIssue
	Counts     store.SummaryCounts
}

type State struct {
	StageName string
	StageIdx  int
	StageMax  int
	Progress  int
	Detail    string
	Done      bool
}

func (s State) Label() string {
	return fmt.Sprintf("[%d/%d] %s", s.StageIdx, s.StageMax, s.StageName)
}

type Tracker struct {
	mu    sync.RWMutex
	state State
}

func NewTracker() *Tracker {
	return &Tracker{
		state: State{
			StageName: "init",
			StageIdx:  1,
			StageMax:  6,
			Progress:  0,
			Detail:    "preparing extraction",
			Done:      false,
		},
	}
}

func (t *Tracker) Update(next State) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = next
}

func (t *Tracker) State() State {
	if t == nil {
		return State{}
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}

func Run(ctx context.Context, cfg Config, tracker *Tracker) (Result, error) {
	if err := cfg.Validate(); err != nil {
		return Result{}, err
	}
	update := func(state State) {
		if tracker != nil {
			tracker.Update(state)
		}
	}

	stageMax := 6
	client := gh.NewClient(cfg.Token)
	update(State{StageName: "fetch_project", StageIdx: 1, StageMax: stageMax, Progress: 2, Detail: "fetching project items"})

	projectRaw, projectIssues, err := client.FetchProjectV2IssuesWithProgress(ctx, cfg.Owner, cfg.ProjectNumber, func(page int) {
		progress := boundedProgress(2, 40, page)
		update(State{
			StageName: "fetch_project",
			StageIdx:  1,
			StageMax:  stageMax,
			Progress:  progress,
			Detail:    fmt.Sprintf("project page %d", page),
		})
	})
	if err != nil {
		return Result{}, fmt.Errorf("failed to fetch project issues: %w", err)
	}

	update(State{StageName: "fetch_repo", StageIdx: 2, StageMax: stageMax, Progress: 42, Detail: "fetching repository issues"})
	repoRaw, repoIssues, err := client.FetchRepoIssuesWithProgress(ctx, cfg.Owner, cfg.Repo, func(page int) {
		progress := boundedProgress(42, 72, page)
		update(State{
			StageName: "fetch_repo",
			StageIdx:  2,
			StageMax:  stageMax,
			Progress:  progress,
			Detail:    fmt.Sprintf("repo page %d", page),
		})
	})
	if err != nil {
		return Result{}, fmt.Errorf("failed to fetch repo issues: %w", err)
	}

	allNormalized := make([]gh.NormalizedIssue, 0, len(projectIssues)+len(repoIssues))
	allNormalized = append(allNormalized, projectIssues...)
	allNormalized = append(allNormalized, repoIssues...)

	db, err := store.OpenSQLite(cfg.OutputDB)
	if err != nil {
		return Result{}, fmt.Errorf("failed to open sqlite store: %w", err)
	}
	defer db.Close()

	update(State{StageName: "apply_schema", StageIdx: 3, StageMax: stageMax, Progress: 76, Detail: "applying sqlite schema"})
	if err := db.ApplySchema(ctx); err != nil {
		return Result{}, fmt.Errorf("failed to apply sqlite schema: %w", err)
	}

	update(State{StageName: "persist_project_raw", StageIdx: 4, StageMax: stageMax, Progress: 82, Detail: "persisting project payloads"})
	if err := db.UpsertProjectRaw(ctx, projectRaw); err != nil {
		return Result{}, fmt.Errorf("failed to persist project raw records: %w", err)
	}

	update(State{StageName: "persist_repo_raw", StageIdx: 5, StageMax: stageMax, Progress: 88, Detail: "persisting repo payloads"})
	if err := db.UpsertRepoRaw(ctx, repoRaw); err != nil {
		return Result{}, fmt.Errorf("failed to persist repo raw records: %w", err)
	}

	update(State{StageName: "persist_normalized", StageIdx: 6, StageMax: stageMax, Progress: 94, Detail: "persisting normalized issues"})
	if err := db.UpsertNormalized(ctx, allNormalized); err != nil {
		return Result{}, fmt.Errorf("failed to persist normalized records: %w", err)
	}

	counts, err := db.Counts(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("failed to read sqlite counts: %w", err)
	}

	update(State{
		StageName: "complete",
		StageIdx:  stageMax,
		StageMax:  stageMax,
		Progress:  100,
		Detail:    "extraction complete",
		Done:      true,
	})

	return Result{
		ProjectRaw: projectRaw,
		RepoRaw:    repoRaw,
		Normalized: allNormalized,
		Counts:     counts,
	}, nil
}

func boundedProgress(start, end, pagesCompleted int) int {
	if pagesCompleted <= 0 {
		return start
	}
	span := end - start
	if span <= 0 {
		return end
	}
	// This grows with actual fetched pages and converges near end.
	portion := float64(pagesCompleted) / float64(pagesCompleted+2)
	value := start + int(math.Round(float64(span)*portion))
	if value >= end {
		return end - 1
	}
	return value
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
