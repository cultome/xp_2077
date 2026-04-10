package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cultome/xp_2077/internal/domain"
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
	if err := s.ensureIssuesNormalizedColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureIssuesNormalizedIndexes(ctx); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) ensureIssuesNormalizedColumns(ctx context.Context) error {
	definitions := map[string]string{
		"issue_body":         "TEXT",
		"xp_base":            "REAL",
		"planned_start_date": "TEXT",
		"planned_end_date":   "TEXT",
		"real_end_date":      "TEXT",
		"xp_final":           "REAL",
	}

	existing, err := s.tableColumnSet(ctx, "issues_normalized")
	if err != nil {
		return fmt.Errorf("inspect issues_normalized columns: %w", err)
	}

	for column, sqlType := range definitions {
		if _, ok := existing[column]; ok {
			continue
		}
		query := fmt.Sprintf("ALTER TABLE issues_normalized ADD COLUMN %s %s", column, sqlType)
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("add column %s to issues_normalized: %w", column, err)
		}
	}
	return nil
}

func (s *SQLiteStore) ensureIssuesNormalizedIndexes(ctx context.Context) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_issues_norm_source ON issues_normalized(source)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_norm_real_end_date ON issues_normalized(real_end_date)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_norm_updated_at ON issues_normalized(updated_at)`,
	}
	for _, query := range indexes {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("create issues_normalized indexes: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) tableColumnSet(ctx context.Context, table string) (map[string]struct{}, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := map[string]struct{}{}
	for rows.Next() {
		var (
			cid       int
			name      string
			valueType string
			notNull   int
			defaultV  any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &valueType, &notNull, &defaultV, &pk); err != nil {
			return nil, err
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
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
	title, state, url, issue_body, author_login, assignees_json, labels_json, project_fields_json,
	xp_base, planned_start_date, planned_end_date, real_end_date, xp_final,
	created_at, updated_at, closed_at, fetched_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(source, source_record_id) DO UPDATE SET
	issue_node_id = excluded.issue_node_id,
	issue_number = excluded.issue_number,
	repository_owner = excluded.repository_owner,
	repository_name = excluded.repository_name,
	repository_full_name = excluded.repository_full_name,
	title = excluded.title,
	state = excluded.state,
	url = excluded.url,
	issue_body = excluded.issue_body,
	author_login = excluded.author_login,
	assignees_json = excluded.assignees_json,
	labels_json = excluded.labels_json,
	project_fields_json = excluded.project_fields_json,
	xp_base = excluded.xp_base,
	planned_start_date = excluded.planned_start_date,
	planned_end_date = excluded.planned_end_date,
	real_end_date = excluded.real_end_date,
	xp_final = excluded.xp_final,
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
			nullIfEmpty(record.IssueBody),
			nullIfEmpty(record.AuthorLogin),
			string(assignees),
			string(labels),
			string(projectFields),
			floatPtrOrNil(record.XPBase),
			timePtrDateOrNil(record.PlannedStartDate),
			timePtrDateOrNil(record.PlannedEndDate),
			timePtrDateOrNil(record.RealEndDate),
			floatPtrOrNil(record.XPFinal),
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

func (s *SQLiteStore) Leaderboard(dateRange domain.DateRange) ([]domain.UserXP, error) {
	const query = `
WITH task_rows AS (
	SELECT
		json_each.value AS login,
		i.xp_final AS xp_final,
		julianday(i.real_end_date) - julianday(i.planned_end_date) AS delay_days
	FROM issues_normalized i, json_each(i.assignees_json)
	WHERE i.source = 'project_v2'
	  AND i.xp_final IS NOT NULL
	  AND i.real_end_date IS NOT NULL
	  AND i.real_end_date >= ?
	  AND i.real_end_date <= ?
)
SELECT
	login,
	COALESCE(SUM(xp_final), 0) AS total_xp,
	COUNT(*) AS ticket_count,
	COALESCE(AVG(delay_days), 0) AS avg_delay_days
FROM task_rows
GROUP BY login
ORDER BY total_xp DESC, login ASC
`
	rows, err := s.db.QueryContext(ctxBackground(), query, dateRange.Start.Format(domain.DateLayout), dateRange.End.Format(domain.DateLayout))
	if err != nil {
		return nil, fmt.Errorf("query leaderboard: %w", err)
	}
	defer rows.Close()

	users := make([]domain.UserXP, 0, 64)
	for rows.Next() {
		var u domain.UserXP
		if err := rows.Scan(&u.Login, &u.XP, &u.TicketCount, &u.AvgDelayDays); err != nil {
			return nil, fmt.Errorf("scan leaderboard row: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate leaderboard rows: %w", err)
	}
	return users, nil
}

func (s *SQLiteStore) TasksForUser(login string, dateRange domain.DateRange) ([]domain.TaskXP, error) {
	const query = `
SELECT
	i.title,
	i.planned_start_date,
	i.planned_end_date,
	i.real_end_date,
	i.repository_full_name,
	COALESCE(i.issue_node_id, i.source || ':' || i.source_record_id) AS task_id,
	i.xp_final,
	i.xp_base,
	COALESCE(i.issue_number, 0),
	COALESCE(i.state, ''),
	COALESCE(i.url, ''),
	COALESCE(i.author_login, ''),
	i.assignees_json,
	i.labels_json,
	COALESCE(i.issue_body, ''),
	COALESCE(i.created_at, ''),
	COALESCE(i.updated_at, ''),
	i.closed_at
FROM issues_normalized i
WHERE i.source = 'project_v2'
  AND i.xp_final IS NOT NULL
  AND i.real_end_date IS NOT NULL
  AND i.real_end_date >= ?
  AND i.real_end_date <= ?
  AND EXISTS (
    SELECT 1
    FROM json_each(i.assignees_json) a
    WHERE a.value = ?
  )
ORDER BY i.real_end_date ASC, i.issue_number ASC
`
	rows, err := s.db.QueryContext(ctxBackground(), query, dateRange.Start.Format(domain.DateLayout), dateRange.End.Format(domain.DateLayout), strings.TrimSpace(login))
	if err != nil {
		return nil, fmt.Errorf("query tasks for user: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.TaskXP, 0, 64)
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks rows: %w", err)
	}
	return tasks, nil
}

func (s *SQLiteStore) TaskByID(taskID string) (domain.TaskXP, error) {
	const query = `
SELECT
	i.title,
	i.planned_start_date,
	i.planned_end_date,
	i.real_end_date,
	i.repository_full_name,
	COALESCE(i.issue_node_id, i.source || ':' || i.source_record_id) AS task_id,
	i.xp_final,
	i.xp_base,
	COALESCE(i.issue_number, 0),
	COALESCE(i.state, ''),
	COALESCE(i.url, ''),
	COALESCE(i.author_login, ''),
	i.assignees_json,
	i.labels_json,
	COALESCE(i.issue_body, ''),
	COALESCE(i.created_at, ''),
	COALESCE(i.updated_at, ''),
	i.closed_at
FROM issues_normalized i
WHERE COALESCE(i.issue_node_id, i.source || ':' || i.source_record_id) = ?
LIMIT 1
`
	row := s.db.QueryRowContext(ctxBackground(), query, strings.TrimSpace(taskID))
	task, err := scanTaskScanner(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TaskXP{}, fmt.Errorf("%w: id=%s", domain.ErrTaskNotFound, taskID)
		}
		return domain.TaskXP{}, err
	}
	return task, nil
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

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTaskRow(rows *sql.Rows) (domain.TaskXP, error) {
	return scanTaskScanner(rows)
}

func scanTaskScanner(scanner rowScanner) (domain.TaskXP, error) {
	var (
		title         string
		plannedDate   string
		plannedEnd    string
		realDate      string
		project       string
		taskID        string
		xp            float64
		xpBase        sql.NullFloat64
		issueNumber   int
		issueState    string
		issueURL      string
		issueAuthor   string
		assigneesJSON string
		labelsJSON    string
		issueBody     string
		createdAtText string
		updatedAtText string
		closedAtText  sql.NullString
	)
	if err := scanner.Scan(
		&title,
		&plannedDate,
		&plannedEnd,
		&realDate,
		&project,
		&taskID,
		&xp,
		&xpBase,
		&issueNumber,
		&issueState,
		&issueURL,
		&issueAuthor,
		&assigneesJSON,
		&labelsJSON,
		&issueBody,
		&createdAtText,
		&updatedAtText,
		&closedAtText,
	); err != nil {
		return domain.TaskXP{}, err
	}

	plannedTime, err := time.Parse(domain.DateLayout, plannedDate)
	if err != nil {
		return domain.TaskXP{}, fmt.Errorf("parse planned_end_date %q: %w", plannedDate, err)
	}
	realTime, err := time.Parse(domain.DateLayout, realDate)
	if err != nil {
		return domain.TaskXP{}, fmt.Errorf("parse real_end_date %q: %w", realDate, err)
	}
	plannedEndTime, err := time.Parse(domain.DateLayout, plannedEnd)
	if err != nil {
		return domain.TaskXP{}, fmt.Errorf("parse planned_end_date %q: %w", plannedEnd, err)
	}
	createdAt, err := parseRFC3339(createdAtText)
	if err != nil {
		return domain.TaskXP{}, fmt.Errorf("parse created_at %q: %w", createdAtText, err)
	}
	updatedAt, err := parseRFC3339(updatedAtText)
	if err != nil {
		return domain.TaskXP{}, fmt.Errorf("parse updated_at %q: %w", updatedAtText, err)
	}

	var closedAt *time.Time
	if closedAtText.Valid && strings.TrimSpace(closedAtText.String) != "" {
		parsed, err := parseRFC3339(closedAtText.String)
		if err != nil {
			return domain.TaskXP{}, fmt.Errorf("parse closed_at %q: %w", closedAtText.String, err)
		}
		closedAt = &parsed
	}

	assignees, err := decodeStringSlice(assigneesJSON)
	if err != nil {
		return domain.TaskXP{}, fmt.Errorf("decode assignees_json: %w", err)
	}
	labels, err := decodeStringSlice(labelsJSON)
	if err != nil {
		return domain.TaskXP{}, fmt.Errorf("decode labels_json: %w", err)
	}

	return domain.TaskXP{
		Description:         title,
		PlannedDate:         plannedTime,
		PlannedEndDate:      plannedEndTime,
		RealDate:            realTime,
		Project:             project,
		ID:                  taskID,
		XP:                  xp,
		XPBase:              nullFloatToPtr(xpBase),
		IssueNumber:         issueNumber,
		IssueState:          issueState,
		IssueURL:            issueURL,
		IssueAuthorLogin:    issueAuthor,
		IssueAssigneeLogins: assignees,
		IssueLabels:         labels,
		IssueBody:           issueBody,
		IssueCreatedAt:      createdAt,
		IssueUpdatedAt:      updatedAt,
		IssueClosedAt:       closedAt,
	}, nil
}

func parseRFC3339(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.TrimSpace(value))
}

func decodeStringSlice(value string) ([]string, error) {
	var raw []string
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return nil, err
	}
	clean := make([]string, 0, len(raw))
	for _, entry := range raw {
		entry = strings.TrimSpace(entry)
		if entry != "" {
			clean = append(clean, entry)
		}
	}
	return clean, nil
}

func ctxBackground() context.Context {
	return context.Background()
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

func timePtrDateOrNil(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC().Format(domain.DateLayout)
}

func floatPtrOrNil(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullFloatToPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}
