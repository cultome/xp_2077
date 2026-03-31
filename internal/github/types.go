package gh

import (
	"encoding/json"
	"strings"
	"time"
)

type SourceKind string

const (
	SourceProjectV2 SourceKind = "project_v2"
	SourceRepoIssue SourceKind = "repo_issue"
)

type ProjectItemRawRecord struct {
	ProjectItemID      string
	IssueNodeID        string
	IssueNumber        int
	RepositoryFullName string
	UpdatedAt          time.Time
	RawPayload         []byte
}

type RepoIssueRawRecord struct {
	IssueNodeID        string
	IssueNumber        int
	RepositoryFullName string
	UpdatedAt          time.Time
	RawPayload         []byte
}

type NormalizedIssue struct {
	Source             SourceKind
	SourceRecordID     string
	IssueNodeID        string
	IssueNumber        int
	RepositoryOwner    string
	RepositoryName     string
	RepositoryFullName string
	Title              string
	State              string
	URL                string
	AuthorLogin        string
	AssigneeLogins     []string
	Labels             []string
	ProjectFields      map[string]string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ClosedAt           *time.Time
}

type projectIssueDTO struct {
	ID         string     `json:"id"`
	Number     int        `json:"number"`
	Title      string     `json:"title"`
	State      string     `json:"state"`
	URL        string     `json:"url"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	ClosedAt   *time.Time `json:"closedAt"`
	Repository struct {
		Name          string `json:"name"`
		NameWithOwner string `json:"nameWithOwner"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
}

type projectFieldValueDTO struct {
	Type           string   `json:"__typename"`
	Text           string   `json:"text"`
	Number         *float64 `json:"number"`
	Date           string   `json:"date"`
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	IterationTitle string   `json:"iterationTitle"`
	Field          *struct {
		Name string `json:"name"`
	} `json:"field"`
}

func normalizeProjectIssue(sourceRecordID string, issue projectIssueDTO, projectFields map[string]string) NormalizedIssue {
	assignees := make([]string, 0, len(issue.Assignees.Nodes))
	for _, node := range issue.Assignees.Nodes {
		if strings.TrimSpace(node.Login) != "" {
			assignees = append(assignees, node.Login)
		}
	}

	labels := make([]string, 0, len(issue.Labels.Nodes))
	for _, node := range issue.Labels.Nodes {
		if strings.TrimSpace(node.Name) != "" {
			labels = append(labels, node.Name)
		}
	}

	authorLogin := ""
	if issue.Author != nil {
		authorLogin = issue.Author.Login
	}

	return NormalizedIssue{
		Source:             SourceProjectV2,
		SourceRecordID:     sourceRecordID,
		IssueNodeID:        issue.ID,
		IssueNumber:        issue.Number,
		RepositoryOwner:    issue.Repository.Owner.Login,
		RepositoryName:     issue.Repository.Name,
		RepositoryFullName: issue.Repository.NameWithOwner,
		Title:              issue.Title,
		State:              strings.ToLower(issue.State),
		URL:                issue.URL,
		AuthorLogin:        authorLogin,
		AssigneeLogins:     assignees,
		Labels:             labels,
		ProjectFields:      projectFields,
		CreatedAt:          issue.CreatedAt,
		UpdatedAt:          issue.UpdatedAt,
		ClosedAt:           issue.ClosedAt,
	}
}

type repoIssueDTO struct {
	ID        int64      `json:"id"`
	NodeID    string     `json:"node_id"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	HTMLURL   string     `json:"html_url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	User      *struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	RepositoryURL string           `json:"repository_url"`
	PullRequest   *json.RawMessage `json:"pull_request"`
}

func (issue repoIssueDTO) isPullRequest() bool {
	return issue.PullRequest != nil
}

func normalizeRepoIssue(issue repoIssueDTO, owner, repo string) NormalizedIssue {
	assignees := make([]string, 0, len(issue.Assignees))
	for _, entry := range issue.Assignees {
		if strings.TrimSpace(entry.Login) != "" {
			assignees = append(assignees, entry.Login)
		}
	}

	labels := make([]string, 0, len(issue.Labels))
	for _, entry := range issue.Labels {
		if strings.TrimSpace(entry.Name) != "" {
			labels = append(labels, entry.Name)
		}
	}

	authorLogin := ""
	if issue.User != nil {
		authorLogin = issue.User.Login
	}

	return NormalizedIssue{
		Source:             SourceRepoIssue,
		SourceRecordID:     issue.NodeID,
		IssueNodeID:        issue.NodeID,
		IssueNumber:        issue.Number,
		RepositoryOwner:    owner,
		RepositoryName:     repo,
		RepositoryFullName: owner + "/" + repo,
		Title:              issue.Title,
		State:              strings.ToLower(issue.State),
		URL:                issue.HTMLURL,
		AuthorLogin:        authorLogin,
		AssigneeLogins:     assignees,
		Labels:             labels,
		ProjectFields:      map[string]string{},
		CreatedAt:          issue.CreatedAt,
		UpdatedAt:          issue.UpdatedAt,
		ClosedAt:           issue.ClosedAt,
	}
}
