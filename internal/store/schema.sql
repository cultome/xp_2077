CREATE TABLE IF NOT EXISTS project_items_raw (
    project_item_id TEXT PRIMARY KEY,
    issue_node_id TEXT,
    issue_number INTEGER,
    repository_full_name TEXT,
    updated_at TEXT,
    payload_json TEXT NOT NULL,
    fetched_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS repo_issues_raw (
    issue_node_id TEXT PRIMARY KEY,
    issue_number INTEGER NOT NULL,
    repository_full_name TEXT NOT NULL,
    updated_at TEXT,
    payload_json TEXT NOT NULL,
    fetched_at TEXT NOT NULL
);

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
    assignees_json TEXT NOT NULL,
    labels_json TEXT NOT NULL,
    project_fields_json TEXT NOT NULL,
    created_at TEXT,
    updated_at TEXT,
    closed_at TEXT,
    fetched_at TEXT NOT NULL,
    PRIMARY KEY (source, source_record_id)
);
