package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/cultome/xp_2077/internal/domain"
	gh "github.com/cultome/xp_2077/internal/github"
)

func TestApplySchemaMigratesIssuesNormalizedColumns(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "migration.db")
	store, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS issues_normalized (
	source TEXT NOT NULL,
	source_record_id TEXT NOT NULL,
	issue_node_id TEXT,
	issue_number INTEGER,
	repository_owner TEXT,
	repository_name TEXT,
	repository_full_name TEXT,
	title TEXT,
	state TEXT,
	url TEXT,
	author_login TEXT,
	created_at TEXT,
	updated_at TEXT,
	closed_at TEXT,
	project_fields_json TEXT NOT NULL,
	assignees_json TEXT NOT NULL,
	labels_json TEXT NOT NULL,
	fetched_at TEXT NOT NULL,
	PRIMARY KEY (source, source_record_id)
)`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	if err := store.ApplySchema(ctx); err != nil {
		t.Fatalf("apply schema with migration: %v", err)
	}

	cols, err := store.tableColumnSet(ctx, "issues_normalized")
	if err != nil {
		t.Fatalf("read columns: %v", err)
	}
	for _, required := range []string{"issue_body", "xp_base", "planned_start_date", "planned_end_date", "real_end_date", "xp_final"} {
		if _, ok := cols[required]; !ok {
			t.Fatalf("expected migrated column %q", required)
		}
	}
}

func TestSQLiteRepositoryReturnsLeaderboardAndTasks(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "repo.db")
	store, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.ApplySchema(ctx); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	start := mustDate(t, "2026-01-01")
	end := mustDate(t, "2026-01-11")
	real := mustDate(t, "2026-01-09")
	xpBase := 100.0
	xpFinal := 120.0
	createdAt := time.Date(2025, 12, 30, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC)

	records := []gh.NormalizedIssue{
		{
			Source:             gh.SourceProjectV2,
			SourceRecordID:     "PI_1",
			IssueNodeID:        "I_1",
			IssueNumber:        101,
			RepositoryOwner:    "cultome",
			RepositoryName:     "xp_2077",
			RepositoryFullName: "cultome/xp_2077",
			Title:              "Implement extraction projection",
			State:              "closed",
			URL:                "https://github.com/cultome/xp_2077/issues/101",
			IssueBody:          "Detailed issue body",
			AuthorLogin:        "owner_user",
			AssigneeLogins:     []string{"alice"},
			Labels:             []string{"backend", "high-priority"},
			ProjectFields:      map[string]string{"XP": "100"},
			XPBase:             &xpBase,
			PlannedStartDate:   &start,
			PlannedEndDate:     &end,
			RealEndDate:        &real,
			XPFinal:            &xpFinal,
			CreatedAt:          createdAt,
			UpdatedAt:          updatedAt,
			ClosedAt:           &updatedAt,
		},
	}
	if err := store.UpsertNormalized(ctx, records); err != nil {
		t.Fatalf("upsert normalized: %v", err)
	}

	dateRange := domain.DateRange{
		Start: mustDate(t, "2026-01-01"),
		End:   mustDate(t, "2026-01-31"),
	}

	leaderboard, err := store.Leaderboard(dateRange)
	if err != nil {
		t.Fatalf("leaderboard query: %v", err)
	}
	if len(leaderboard) != 1 {
		t.Fatalf("expected one leaderboard row, got %d", len(leaderboard))
	}
	if leaderboard[0].Login != "alice" || leaderboard[0].XP != 120.0 {
		t.Fatalf("unexpected leaderboard row: %+v", leaderboard[0])
	}

	tasks, err := store.TasksForUser("alice", dateRange)
	if err != nil {
		t.Fatalf("tasks query: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one task row, got %d", len(tasks))
	}
	if tasks[0].IssueBody == "" {
		t.Fatal("expected issue body to be populated")
	}
	if tasks[0].XPBase == nil || *tasks[0].XPBase != 100.0 {
		t.Fatalf("expected xp base 100.0, got %+v", tasks[0].XPBase)
	}
	if got, want := tasks[0].PlannedDate.Format(domain.DateLayout), "2026-01-01"; got != want {
		t.Fatalf("expected planned start date %s, got %s", want, got)
	}

	task, err := store.TaskByID("I_1")
	if err != nil {
		t.Fatalf("task by id: %v", err)
	}
	if task.ID != "I_1" || task.XP != 120.0 {
		t.Fatalf("unexpected task by id: %+v", task)
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(domain.DateLayout, value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return parsed
}
