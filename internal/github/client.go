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

func (c *Client) FetchProjectV2Issues(ctx context.Context, org string, projectNumber int) ([]ProjectItemRawRecord, []NormalizedIssue, error) {
	if strings.TrimSpace(org) == "" {
		return nil, nil, fmt.Errorf("organization is required")
	}
	if projectNumber <= 0 {
		return nil, nil, fmt.Errorf("project number must be > 0")
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
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	rawRecords := make([]ProjectItemRawRecord, 0, 128)
	normalized := make([]NormalizedIssue, 0, 128)
	cursor := ""

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
			return nil, nil, fmt.Errorf("marshal graphql request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubAPIBase+"/graphql", bytes.NewReader(body))
		if err != nil {
			return nil, nil, fmt.Errorf("build graphql request: %w", err)
		}
		c.setCommonHeaders(req)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, fmt.Errorf("execute graphql request: %w", err)
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, nil, fmt.Errorf("read graphql response: %w", readErr)
		}
		if closeErr != nil {
			return nil, nil, fmt.Errorf("close graphql response body: %w", closeErr)
		}
		if resp.StatusCode >= 300 {
			return nil, nil, fmt.Errorf("graphql request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
		}

		var parsed graphResponse
		if err := json.Unmarshal(responseBody, &parsed); err != nil {
			return nil, nil, fmt.Errorf("decode graphql response: %w", err)
		}
		if len(parsed.Errors) > 0 {
			return nil, nil, fmt.Errorf("graphql error: %s", parsed.Errors[0].Message)
		}

		items := parsed.Data.Organization.ProjectV2.Items
		for _, node := range items.Nodes {
			if node.Content.TypeName != "Issue" {
				continue
			}
			payload, err := json.Marshal(node)
			if err != nil {
				return nil, nil, fmt.Errorf("marshal project item payload: %w", err)
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

		if !items.PageInfo.HasNextPage {
			break
		}
		cursor = items.PageInfo.EndCursor
	}

	return rawRecords, normalized, nil
}

func (c *Client) FetchRepoIssues(ctx context.Context, owner, repo string) ([]RepoIssueRawRecord, []NormalizedIssue, error) {
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

		for _, issueDTO := range items {
			if issueDTO.isPullRequest() {
				continue
			}
			payload, err := json.Marshal(issueDTO)
			if err != nil {
				return nil, nil, fmt.Errorf("marshal repo issue payload: %w", err)
			}
			n := normalizeRepoIssue(issueDTO, owner, repo)
			rawRecords = append(rawRecords, RepoIssueRawRecord{
				IssueNodeID:        n.IssueNodeID,
				IssueNumber:        n.IssueNumber,
				RepositoryFullName: n.RepositoryFullName,
				UpdatedAt:          n.UpdatedAt,
				RawPayload:         payload,
			})
			normalized = append(normalized, n)
		}

		if len(items) < perPage {
			break
		}
		page++
	}

	return rawRecords, normalized, nil
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
