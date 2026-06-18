package gh

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
)

type SourceKind string

const (
	SourceProjectV2 SourceKind = "project_v2"
	SourceRepoIssue SourceKind = "repo_issue"
)

const specialTasksAlephPrefix = "[Special Tasks for Aleph] "

// ProjectFetchStats summarizes project cards that were not ingested as issues
// during a project fetch.
type ProjectFetchStats struct {
	// InaccessibleIssues counts ISSUE cards whose content the token could not
	// read (typically the issue lives in a repo the token lacks access to).
	// These are potential lost tasks worth surfacing to the user.
	InaccessibleIssues int
	// NonIssues counts cards intentionally ignored because they are not issues
	// (pull requests, draft issues, redacted items).
	NonIssues int
}

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
	IssueBody          string
	AuthorLogin        string
	AssigneeLogins     []string
	Labels             []string
	ProjectFields      map[string]string
	XPBase             *float64
	PlannedStartDate   *time.Time
	PlannedEndDate     *time.Time
	RealEndDate        *time.Time
	XPFinal            *float64
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
	Body       string     `json:"body"`
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
	xpBase, plannedStart, plannedEnd, realEnd, xpFinal := parseProjectXPFields(projectFields)

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
		IssueBody:          issue.Body,
		AuthorLogin:        authorLogin,
		AssigneeLogins:     assignees,
		Labels:             labels,
		ProjectFields:      projectFields,
		XPBase:             xpBase,
		PlannedStartDate:   plannedStart,
		PlannedEndDate:     plannedEnd,
		RealEndDate:        realEnd,
		XPFinal:            xpFinal,
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
	Body      string     `json:"body"`
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

func normalizeRepoIssue(issue repoIssueDTO, owner, repo string, projectFields map[string]string) NormalizedIssue {
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
	if projectFields == nil {
		projectFields = map[string]string{}
	}
	xpBase, plannedStart, plannedEnd, realEnd, xpFinal := parseRepoIssueXPFields(issue.Title, projectFields)

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
		IssueBody:          issue.Body,
		AuthorLogin:        authorLogin,
		AssigneeLogins:     assignees,
		Labels:             labels,
		ProjectFields:      projectFields,
		XPBase:             xpBase,
		PlannedStartDate:   plannedStart,
		PlannedEndDate:     plannedEnd,
		RealEndDate:        realEnd,
		XPFinal:            xpFinal,
		CreatedAt:          issue.CreatedAt,
		UpdatedAt:          issue.UpdatedAt,
		ClosedAt:           issue.ClosedAt,
	}
}

var fieldKeyNormalizer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n",
	"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ñ", "n",
)

func parseProjectXPFields(fields map[string]string) (xpBase *float64, plannedStart, plannedEnd, realEnd *time.Time, xpFinal *float64) {
	xpText, hasXP := getFirstFieldValue(fields, []string{"xp"})
	plannedStartText, hasPlannedStart := getFirstFieldValue(fields, []string{
		"fecha programada de inicio",
		"implementacion inicio",
	})
	plannedEndText, hasPlannedEnd := getFirstFieldValue(fields, []string{
		"fecha programada de fin",
		"implementacion fin",
	})
	realEndText, hasRealEnd := getFirstFieldValue(fields, []string{
		"fecha real de fin",
		"implementacion fin real",
	})
	if !hasXP || !hasPlannedStart || !hasPlannedEnd || !hasRealEnd {
		return nil, nil, nil, nil, nil
	}

	parsedXP, err := strconv.ParseFloat(strings.TrimSpace(xpText), 64)
	if err != nil {
		return nil, nil, nil, nil, nil
	}
	startDate, err := parseProjectDate(plannedStartText)
	if err != nil {
		return nil, nil, nil, nil, nil
	}
	endDate, err := parseProjectDate(plannedEndText)
	if err != nil {
		return nil, nil, nil, nil, nil
	}
	realDate, err := parseProjectDate(realEndText)
	if err != nil {
		return nil, nil, nil, nil, nil
	}

	durationDays := endDate.Sub(startDate).Hours() / 24
	if durationDays < 0 {
		// end before start is corrupt data; a same-day plan (duration 0) is valid.
		return nil, nil, nil, nil, nil
	}

	deltaDays := endDate.Sub(realDate).Hours() / 24
	// A zero-length plan has no ratio to scale by, so credit the base XP
	// instead of dropping the task entirely.
	deltaPct := 0.0
	if durationDays > 0 {
		deltaPct = math.Abs(deltaDays) / durationDays
	}
	finalXP := parsedXP
	if deltaDays > 0 {
		finalXP = parsedXP + (parsedXP * deltaPct)
	} else if deltaDays < 0 {
		finalXP = parsedXP - (parsedXP * deltaPct)
	}
	if finalXP < 0 {
		finalXP = 0
	}
	finalXP = math.Round(finalXP*10) / 10

	return floatPtr(parsedXP), timePtr(startDate), timePtr(endDate), timePtr(realDate), floatPtr(finalXP)
}

func parseRepoIssueXPFields(title string, fields map[string]string) (xpBase *float64, plannedStart, plannedEnd, realEnd *time.Time, xpFinal *float64) {
	if !hasSpecialTasksAlephPrefix(title) {
		return nil, nil, nil, nil, nil
	}
	if !isDoneFromProjectFields(fields) {
		return nil, nil, nil, nil, nil
	}
	storyPointsText, hasStoryPoints := getFirstFieldValue(fields, []string{"story points"})
	priorityText, hasPriority := getFirstFieldValue(fields, []string{"priority"})
	dueDateText, hasDueDate := getFirstFieldValue(fields, []string{"due date"})
	if !hasStoryPoints || !hasPriority || !hasDueDate {
		return nil, nil, nil, nil, nil
	}
	parsedBase, err := strconv.ParseFloat(strings.TrimSpace(storyPointsText), 64)
	if err != nil {
		return nil, nil, nil, nil, nil
	}
	multiplier, ok := priorityMultiplier(priorityText)
	if !ok {
		return nil, nil, nil, nil, nil
	}
	dueDate, err := parseProjectDate(dueDateText)
	if err != nil {
		return nil, nil, nil, nil, nil
	}
	finalXP := math.Round((parsedBase*multiplier)*10) / 10
	return floatPtr(parsedBase), timePtr(dueDate), timePtr(dueDate), timePtr(dueDate), floatPtr(finalXP)
}

func hasSpecialTasksAlephPrefix(title string) bool {
	return strings.HasPrefix(strings.TrimSpace(title), specialTasksAlephPrefix)
}

func isDoneFromProjectFields(fields map[string]string) bool {
	status, ok := getFirstFieldValue(fields, []string{"status"})
	if !ok {
		return false
	}
	return normalizeFieldName(status) == "done"
}

func priorityMultiplier(priority string) (float64, bool) {
	switch strings.ToUpper(strings.TrimSpace(priority)) {
	case "P0":
		return 2, true
	case "P1":
		return 1.5, true
	case "P2":
		return 1, true
	default:
		return 0, false
	}
}

func parseProjectDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func getFieldValue(fields map[string]string, expectedName string) (string, bool) {
	want := normalizeFieldName(expectedName)
	for key, value := range fields {
		if normalizeFieldName(key) == want {
			return value, true
		}
	}
	return "", false
}

func getFirstFieldValue(fields map[string]string, expectedNames []string) (string, bool) {
	for _, expectedName := range expectedNames {
		if value, ok := getFieldValue(fields, expectedName); ok {
			return value, true
		}
	}
	return "", false
}

func normalizeFieldName(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(fieldKeyNormalizer.Replace(strings.TrimSpace(value)))), " ")
}

func floatPtr(value float64) *float64 {
	v := value
	return &v
}

func timePtr(value time.Time) *time.Time {
	v := value
	return &v
}
