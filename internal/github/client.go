package gh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	githubAPIBase = "https://api.github.com"
)

// graphQLError mirrors a single entry of a GraphQL `errors` array.
type graphQLError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Path    []any  `json:"path"`
}

// fatalGraphErrors returns only the errors that should abort the request.
//
// GitHub answers with partial `data` plus per-item field-level errors when the
// token can't read a particular item (e.g. FORBIDDEN on the `content` of a
// project card whose issue/PR lives in a repo the token lacks access to). Those
// nodes come back with null content and are skipped by the callers, so they must
// not fail the whole extraction. Errors scoped to an individual node/content are
// treated as non-fatal; anything else (auth fully broken, project not found,
// no path) is fatal.
func fatalGraphErrors(errs []graphQLError) []graphQLError {
	fatal := make([]graphQLError, 0, len(errs))
	for _, e := range errs {
		if isPerItemContentError(e) {
			continue
		}
		fatal = append(fatal, e)
	}
	return fatal
}

func isPerItemContentError(e graphQLError) bool {
	for _, segment := range e.Path {
		if s, ok := segment.(string); ok && (s == "content" || s == "nodes") {
			return true
		}
	}
	return false
}

type projectNodeKind int

const (
	nodeReadableIssue projectNodeKind = iota
	nodeInaccessibleIssue
	nodeNonIssue
)

// classifyProjectNode decides how a project card should be treated. A card is a
// readable issue when its content resolved to an Issue. Otherwise it is either
// an issue we could not read (content is null/forbidden but the card type is
// still ISSUE — e.g. the issue lives in a repo the token can't access) or a
// non-issue card (pull request, draft, redacted) that we intentionally ignore.
func classifyProjectNode(contentTypeName, nodeType string) projectNodeKind {
	if contentTypeName == "Issue" {
		return nodeReadableIssue
	}
	if strings.EqualFold(strings.TrimSpace(nodeType), "ISSUE") {
		return nodeInaccessibleIssue
	}
	return nodeNonIssue
}

type Client struct {
	httpClient *http.Client
	token      string
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      strings.TrimSpace(token),
	}
}

func (c *Client) FetchProjectV2Issues(ctx context.Context, org string, projectNumber int) ([]ProjectItemRawRecord, []NormalizedIssue, ProjectFetchStats, error) {
	return c.FetchProjectV2IssuesWithProgress(ctx, org, projectNumber, nil)
}

func (c *Client) FetchProjectV2IssuesWithProgress(ctx context.Context, org string, projectNumber int, onPageFetched func(page int)) ([]ProjectItemRawRecord, []NormalizedIssue, ProjectFetchStats, error) {
	var stats ProjectFetchStats
	if strings.TrimSpace(org) == "" {
		return nil, nil, stats, fmt.Errorf("organization is required")
	}
	if projectNumber <= 0 {
		return nil, nil, stats, fmt.Errorf("project number must be > 0")
	}

	const query = `
query ProjectItems($org: String!, $number: Int!, $cursor: String) {
  organization(login: $org) {
    projectV2(number: $number) {
      items(first: 50, after: $cursor) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          type
          updatedAt
          fieldValues(first: 50) {
            nodes {
              __typename
              ... on ProjectV2ItemFieldTextValue {
                text
                field { ... on ProjectV2FieldCommon { name } }
              }
              ... on ProjectV2ItemFieldNumberValue {
                number
                field { ... on ProjectV2FieldCommon { name } }
              }
              ... on ProjectV2ItemFieldDateValue {
                date
                field { ... on ProjectV2FieldCommon { name } }
              }
              ... on ProjectV2ItemFieldSingleSelectValue {
                name
                field { ... on ProjectV2FieldCommon { name } }
              }
              ... on ProjectV2ItemFieldIterationValue {
                title
                field { ... on ProjectV2FieldCommon { name } }
              }
            }
          }
          content {
            __typename
            ... on Issue {
              id
              number
              title
              body
              state
              url
              createdAt
              updatedAt
              closedAt
              repository {
                name
                nameWithOwner
                owner { login }
              }
              author { login }
              assignees(first: 20) { nodes { login } }
              labels(first: 50) { nodes { name } }
            }
          }
        }
      }
    }
  }
}
`

	type graphNode struct {
		ID          string    `json:"id"`
		Type        string    `json:"type"`
		UpdatedAt   time.Time `json:"updatedAt"`
		FieldValues struct {
			Nodes []projectFieldValueDTO `json:"nodes"`
		} `json:"fieldValues"`
		Content struct {
			TypeName string `json:"__typename"`
			projectIssueDTO
		} `json:"content"`
	}

	type graphResponse struct {
		Data struct {
			Organization struct {
				ProjectV2 struct {
					Items struct {
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
						Nodes []graphNode `json:"nodes"`
					} `json:"items"`
				} `json:"projectV2"`
			} `json:"organization"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}

	rawRecords := make([]ProjectItemRawRecord, 0, 128)
	normalized := make([]NormalizedIssue, 0, 128)
	cursor := ""

	page := 1
	for {
		variables := map[string]any{
			"org":    org,
			"number": projectNumber,
			"cursor": nil,
		}
		if cursor != "" {
			variables["cursor"] = cursor
		}

		body, err := json.Marshal(map[string]any{
			"query":     query,
			"variables": variables,
		})
		if err != nil {
			return nil, nil, stats, fmt.Errorf("marshal graphql request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubAPIBase+"/graphql", bytes.NewReader(body))
		if err != nil {
			return nil, nil, stats, fmt.Errorf("build graphql request: %w", err)
		}
		c.setCommonHeaders(req)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, stats, fmt.Errorf("execute graphql request: %w", err)
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, nil, stats, fmt.Errorf("read graphql response: %w", readErr)
		}
		if closeErr != nil {
			return nil, nil, stats, fmt.Errorf("close graphql response body: %w", closeErr)
		}
		if resp.StatusCode >= 300 {
			return nil, nil, stats, fmt.Errorf("graphql request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
		}

		var parsed graphResponse
		if err := json.Unmarshal(responseBody, &parsed); err != nil {
			return nil, nil, stats, fmt.Errorf("decode graphql response: %w", err)
		}
		if fatal := fatalGraphErrors(parsed.Errors); len(fatal) > 0 {
			return nil, nil, stats, fmt.Errorf("graphql error: %s", fatal[0].Message)
		}

		items := parsed.Data.Organization.ProjectV2.Items
		for _, node := range items.Nodes {
			switch classifyProjectNode(node.Content.TypeName, node.Type) {
			case nodeInaccessibleIssue:
				stats.InaccessibleIssues++
				continue
			case nodeNonIssue:
				stats.NonIssues++
				continue
			}
			payload, err := json.Marshal(node)
			if err != nil {
				return nil, nil, stats, fmt.Errorf("marshal project item payload: %w", err)
			}

			projectFields := mapProjectFields(node.FieldValues.Nodes)
			issue := normalizeProjectIssue(node.ID, node.Content.projectIssueDTO, projectFields)
			rawRecords = append(rawRecords, ProjectItemRawRecord{
				ProjectItemID:      node.ID,
				IssueNodeID:        issue.IssueNodeID,
				IssueNumber:        issue.IssueNumber,
				RepositoryFullName: issue.RepositoryFullName,
				UpdatedAt:          node.UpdatedAt,
				RawPayload:         payload,
			})
			normalized = append(normalized, issue)
		}

		if onPageFetched != nil {
			onPageFetched(page)
		}
		if !items.PageInfo.HasNextPage {
			break
		}
		cursor = items.PageInfo.EndCursor
		page++
	}

	return rawRecords, normalized, stats, nil
}

func (c *Client) FetchRepoIssues(ctx context.Context, owner, repo string) ([]RepoIssueRawRecord, []NormalizedIssue, error) {
	return c.FetchRepoIssuesWithProgress(ctx, owner, repo, nil)
}

func (c *Client) FetchRepoIssuesWithProgress(ctx context.Context, owner, repo string, onPageFetched func(page int)) ([]RepoIssueRawRecord, []NormalizedIssue, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, nil, fmt.Errorf("owner and repo are required")
	}

	rawRecords := make([]RepoIssueRawRecord, 0, 128)
	normalized := make([]NormalizedIssue, 0, 128)
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("%s/repos/%s/%s/issues?state=all&per_page=%d&page=%d", githubAPIBase, owner, repo, perPage, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("build repo issues request: %w", err)
		}
		c.setCommonHeaders(req)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, fmt.Errorf("execute repo issues request: %w", err)
		}
		responseBody, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, nil, fmt.Errorf("read repo issues response: %w", readErr)
		}
		if closeErr != nil {
			return nil, nil, fmt.Errorf("close repo issues response body: %w", closeErr)
		}
		if resp.StatusCode >= 300 {
			if resp.StatusCode == http.StatusNotFound {
				return nil, nil, fmt.Errorf(
					"repo issues request failed (404) for %s/%s: verify owner/repo and token access (private repos may return 404 without permissions). response: %s",
					owner, repo, strings.TrimSpace(string(responseBody)),
				)
			}
			return nil, nil, fmt.Errorf("repo issues request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
		}

		var items []repoIssueDTO
		if err := json.Unmarshal(responseBody, &items); err != nil {
			return nil, nil, fmt.Errorf("decode repo issues response: %w", err)
		}

		issueNodeIDs := make([]string, 0, len(items))
		for _, issueDTO := range items {
			if issueDTO.isPullRequest() {
				continue
			}
			if strings.TrimSpace(issueDTO.NodeID) != "" {
				issueNodeIDs = append(issueNodeIDs, strings.TrimSpace(issueDTO.NodeID))
			}
		}
		projectFieldsByIssueNodeID, err := c.fetchProjectFieldsForIssueNodeIDs(ctx, issueNodeIDs)
		if err != nil {
			return nil, nil, err
		}

		for _, issueDTO := range items {
			if issueDTO.isPullRequest() {
				continue
			}
			payload, err := json.Marshal(issueDTO)
			if err != nil {
				return nil, nil, fmt.Errorf("marshal repo issue payload: %w", err)
			}
			n := normalizeRepoIssue(issueDTO, owner, repo, projectFieldsByIssueNodeID[issueDTO.NodeID])
			rawRecords = append(rawRecords, RepoIssueRawRecord{
				IssueNodeID:        n.IssueNodeID,
				IssueNumber:        n.IssueNumber,
				RepositoryFullName: n.RepositoryFullName,
				UpdatedAt:          n.UpdatedAt,
				RawPayload:         payload,
			})
			normalized = append(normalized, n)
		}

		if onPageFetched != nil {
			onPageFetched(page)
		}
		if len(items) < perPage {
			break
		}
		page++
	}

	return rawRecords, normalized, nil
}

func (c *Client) fetchProjectFieldsForIssueNodeIDs(ctx context.Context, issueNodeIDs []string) (map[string]map[string]string, error) {
	if len(issueNodeIDs) == 0 {
		return map[string]map[string]string{}, nil
	}
	const query = `
query IssueProjectFields($ids: [ID!]!) {
  nodes(ids: $ids) {
    __typename
    ... on Issue {
      id
      projectItems(first: 20) {
        nodes {
          fieldValues(first: 50) {
            nodes {
              __typename
              ... on ProjectV2ItemFieldTextValue {
                text
                field { ... on ProjectV2FieldCommon { name } }
              }
              ... on ProjectV2ItemFieldNumberValue {
                number
                field { ... on ProjectV2FieldCommon { name } }
              }
              ... on ProjectV2ItemFieldDateValue {
                date
                field { ... on ProjectV2FieldCommon { name } }
              }
              ... on ProjectV2ItemFieldSingleSelectValue {
                name
                field { ... on ProjectV2FieldCommon { name } }
              }
              ... on ProjectV2ItemFieldIterationValue {
                title
                field { ... on ProjectV2FieldCommon { name } }
              }
            }
          }
        }
      }
    }
  }
}
`
	requestBody, err := json.Marshal(map[string]any{
		"query": query,
		"variables": map[string]any{
			"ids": issueNodeIDs,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal issue project fields graphql request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubAPIBase+"/graphql", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("build issue project fields graphql request: %w", err)
	}
	c.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute issue project fields graphql request: %w", err)
	}
	responseBody, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read issue project fields graphql response: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close issue project fields graphql response body: %w", closeErr)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("issue project fields graphql request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	type graphResponse struct {
		Data struct {
			Nodes []struct {
				TypeName     string `json:"__typename"`
				ID           string `json:"id"`
				ProjectItems struct {
					Nodes []struct {
						FieldValues struct {
							Nodes []projectFieldValueDTO `json:"nodes"`
						} `json:"fieldValues"`
					} `json:"nodes"`
				} `json:"projectItems"`
			} `json:"nodes"`
		} `json:"data"`
		Errors []graphQLError `json:"errors"`
	}
	var parsed graphResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode issue project fields graphql response: %w", err)
	}
	if fatal := fatalGraphErrors(parsed.Errors); len(fatal) > 0 {
		return nil, fmt.Errorf("issue project fields graphql error: %s", fatal[0].Message)
	}

	result := make(map[string]map[string]string, len(parsed.Data.Nodes))
	for _, node := range parsed.Data.Nodes {
		if node.TypeName != "Issue" || strings.TrimSpace(node.ID) == "" {
			continue
		}
		fields := map[string]string{}
		for _, projectItem := range node.ProjectItems.Nodes {
			itemFields := mapProjectFields(projectItem.FieldValues.Nodes)
			for key, value := range itemFields {
				if _, exists := fields[key]; !exists {
					fields[key] = value
				}
			}
		}
		result[node.ID] = fields
	}
	return result, nil
}

func (c *Client) setCommonHeaders(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "xp_2077_github_extract")
}

func mapProjectFields(values []projectFieldValueDTO) map[string]string {
	fields := make(map[string]string, len(values))
	for _, value := range values {
		if value.Field == nil || strings.TrimSpace(value.Field.Name) == "" {
			continue
		}

		switch value.Type {
		case "ProjectV2ItemFieldTextValue":
			fields[value.Field.Name] = value.Text
		case "ProjectV2ItemFieldNumberValue":
			if value.Number != nil {
				fields[value.Field.Name] = strconv.FormatFloat(*value.Number, 'f', -1, 64)
			}
		case "ProjectV2ItemFieldDateValue":
			fields[value.Field.Name] = value.Date
		case "ProjectV2ItemFieldSingleSelectValue":
			fields[value.Field.Name] = value.Name
		case "ProjectV2ItemFieldIterationValue":
			if strings.TrimSpace(value.IterationTitle) != "" {
				fields[value.Field.Name] = value.IterationTitle
			} else {
				fields[value.Field.Name] = value.Title
			}
		}
	}
	return fields
}
