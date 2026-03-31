package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	gh "github.com/cultome/xp_2077/internal/github"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteStore) ApplySchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpsertProjectRaw(ctx context.Context, records []gh.ProjectItemRawRecord) error {
	const stmt = `
INSERT INTO project_items_raw (
	project_item_id, issue_node_id, issue_number, repository_full_name, updated_at, payload_json, fetched_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(project_item_id) DO UPDATE SET
	issue_node_id = excluded.issue_node_id,
	issue_number = excluded.issue_number,
	repository_full_name = excluded.repository_full_name,
	updated_at = excluded.updated_at,
	payload_json = excluded.payload_json,
	fetched_at = excluded.fetched_at
`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin project raw tx: %w", err)
	}
	defer rollbackQuietly(tx)

	q, err := tx.PrepareContext(ctx, stmt)
	if err != nil {
		return fmt.Errorf("prepare project raw upsert: %w", err)
	}
	defer q.Close()

	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	for _, record := range records {
		if _, err := q.ExecContext(
			ctx,
			record.ProjectItemID,
			nullIfEmpty(record.IssueNodeID),
			record.IssueNumber,
			nullIfEmpty(record.RepositoryFullName),
			timeStringOrNil(record.UpdatedAt),
			string(record.RawPayload),
			fetchedAt,
		); err != nil {
			return fmt.Errorf("upsert project raw record: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit project raw tx: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpsertRepoRaw(ctx context.Context, records []gh.RepoIssueRawRecord) error {
	const stmt = `
INSERT INTO repo_issues_raw (
	issue_node_id, issue_number, repository_full_name, updated_at, payload_json, fetched_at
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(issue_node_id) DO UPDATE SET
	issue_number = excluded.issue_number,
	repository_full_name = excluded.repository_full_name,
	updated_at = excluded.updated_at,
	payload_json = excluded.payload_json,
	fetched_at = excluded.fetched_at
`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin repo raw tx: %w", err)
	}
	defer rollbackQuietly(tx)

	q, err := tx.PrepareContext(ctx, stmt)
	if err != nil {
		return fmt.Errorf("prepare repo raw upsert: %w", err)
	}
	defer q.Close()

	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	for _, record := range records {
		if _, err := q.ExecContext(
			ctx,
			record.IssueNodeID,
			record.IssueNumber,
			record.RepositoryFullName,
			timeStringOrNil(record.UpdatedAt),
			string(record.RawPayload),
			fetchedAt,
		); err != nil {
			return fmt.Errorf("upsert repo raw record: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit repo raw tx: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpsertNormalized(ctx context.Context, records []gh.NormalizedIssue) error {
	const stmt = `
INSERT INTO issues_normalized (
	source, source_record_id, issue_node_id, issue_number, repository_owner, repository_name, repository_full_name,
	title, state, url, author_login, assignees_json, labels_json, project_fields_json, created_at, updated_at, closed_at, fetched_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(source, source_record_id) DO UPDATE SET
	issue_node_id = excluded.issue_node_id,
	issue_number = excluded.issue_number,
	repository_owner = excluded.repository_owner,
	repository_name = excluded.repository_name,
	repository_full_name = excluded.repository_full_name,
	title = excluded.title,
	state = excluded.state,
	url = excluded.url,
	author_login = excluded.author_login,
	assignees_json = excluded.assignees_json,
	labels_json = excluded.labels_json,
	project_fields_json = excluded.project_fields_json,
	created_at = excluded.created_at,
	updated_at = excluded.updated_at,
	closed_at = excluded.closed_at,
	fetched_at = excluded.fetched_at
`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin normalized tx: %w", err)
	}
	defer rollbackQuietly(tx)

	q, err := tx.PrepareContext(ctx, stmt)
	if err != nil {
		return fmt.Errorf("prepare normalized upsert: %w", err)
	}
	defer q.Close()

	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	for _, record := range records {
		assignees, err := json.Marshal(record.AssigneeLogins)
		if err != nil {
			return fmt.Errorf("marshal assignees: %w", err)
		}
		labels, err := json.Marshal(record.Labels)
		if err != nil {
			return fmt.Errorf("marshal labels: %w", err)
		}
		projectFields, err := json.Marshal(record.ProjectFields)
		if err != nil {
			return fmt.Errorf("marshal project fields: %w", err)
		}

		if _, err := q.ExecContext(
			ctx,
			string(record.Source),
			record.SourceRecordID,
			nullIfEmpty(record.IssueNodeID),
			record.IssueNumber,
			nullIfEmpty(record.RepositoryOwner),
			nullIfEmpty(record.RepositoryName),
			nullIfEmpty(record.RepositoryFullName),
			nullIfEmpty(record.Title),
			nullIfEmpty(record.State),
			nullIfEmpty(record.URL),
			nullIfEmpty(record.AuthorLogin),
			string(assignees),
			string(labels),
			string(projectFields),
			timeStringOrNil(record.CreatedAt),
			timeStringOrNil(record.UpdatedAt),
			timePtrStringOrNil(record.ClosedAt),
			fetchedAt,
		); err != nil {
			return fmt.Errorf("upsert normalized record: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit normalized tx: %w", err)
	}
	return nil
}

type SummaryCounts struct {
	ProjectRawCount int
	RepoRawCount    int
	NormalizedCount int
}

func (s *SQLiteStore) Counts(ctx context.Context) (SummaryCounts, error) {
	countOne := func(query string) (int, error) {
		var n int
		if err := s.db.QueryRowContext(ctx, query).Scan(&n); err != nil {
			return 0, err
		}
		return n, nil
	}

	projectCount, err := countOne(`SELECT COUNT(*) FROM project_items_raw`)
	if err != nil {
		return SummaryCounts{}, fmt.Errorf("count project_items_raw: %w", err)
	}
	repoCount, err := countOne(`SELECT COUNT(*) FROM repo_issues_raw`)
	if err != nil {
		return SummaryCounts{}, fmt.Errorf("count repo_issues_raw: %w", err)
	}
	normalizedCount, err := countOne(`SELECT COUNT(*) FROM issues_normalized`)
	if err != nil {
		return SummaryCounts{}, fmt.Errorf("count issues_normalized: %w", err)
	}

	return SummaryCounts{
		ProjectRawCount: projectCount,
		RepoRawCount:    repoCount,
		NormalizedCount: normalizedCount,
	}, nil
}

func rollbackQuietly(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func timeStringOrNil(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func timePtrStringOrNil(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}
