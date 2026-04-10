package ui

import (
	"fmt"
	"strings"

	"github.com/cultome/xp_2077/internal/domain"
)

func (m AppModel) viewIssueDetail() string {
	title := m.headerLine("1SSU3_D3741L")
	task := m.issueTask
	stateBadge := m.styles.Success.Render("[0P3N]")
	if strings.EqualFold(task.IssueState, "closed") {
		stateBadge = m.styles.Error.Render("[CL0S3D]")
	}

	repoIssue := fmt.Sprintf("%s #%d", task.Project, task.IssueNumber)
	meta := fmt.Sprintf(
		"%s %s by %s · created %s",
		stateBadge,
		repoIssue,
		m.styles.Accent.Render(task.IssueAuthorLogin),
		task.IssueCreatedAt.Format(domain.DateLayout),
	)

	closedLine := "closed: -"
	if task.IssueClosedAt != nil {
		closedLine = "closed: " + task.IssueClosedAt.Format(domain.DateLayout)
	}

	lines := []string{
		title,
		"",
		m.styles.Accent.Render(task.Description),
		meta,
		"",
		m.styles.Panel.Render(strings.Join([]string{
			fmt.Sprintf("st4t3: %s", strings.ToUpper(task.IssueState)),
			fmt.Sprintf("author: %s", task.IssueAuthorLogin),
			fmt.Sprintf("assignees: %s", renderList(task.IssueAssigneeLogins)),
			fmt.Sprintf("labels: %s", renderList(task.IssueLabels)),
			fmt.Sprintf("planned: %s", task.PlannedDate.Format(domain.DateLayout)),
			fmt.Sprintf("actual: %s", task.RealDate.Format(domain.DateLayout)),
			fmt.Sprintf("updated: %s", task.IssueUpdatedAt.Format(domain.DateLayout)),
			closedLine,
			fmt.Sprintf("url: %s", task.IssueURL),
		}, "\n")),
		"",
		m.styles.Subtle.Render("B0DY"),
		m.styles.Panel.Render(task.IssueBody),
		"",
		m.styles.Footer.Render("ESC:b4ck t0 t4sks  Q:qu1t"),
		m.styles.Subtle.Render(terminalTicker(m.frame, "gh issue view")),
	}

	return m.screen(lines)
}

func renderList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}
