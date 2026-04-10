package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/cultome/xp_2077/internal/domain"
)

func (m AppModel) viewIssueDetail() string {
	contentLines := m.issueDetailContentLines()
	visible := m.issueDetailVisibleHeight()
	maxScroll := m.issueDetailMaxScroll()
	scroll := m.issueScroll
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}

	end := scroll + visible
	if end > len(contentLines) {
		end = len(contentLines)
	}
	window := contentLines[scroll:end]
	scrollHint := "UP/DOWN:scroll"
	if maxScroll == 0 {
		scrollHint = "UP/DOWN:no-scroll"
	}

	lines := []string{
		m.headerLine("1SSU3_D3741L"),
		"",
	}
	lines = append(lines, window...)
	lines = append(lines, "")
	lines = append(lines, m.styles.Footer.Render(fmt.Sprintf("%s  ESC:b4ck t0 t4sks  Q:qu1t", scrollHint)))
	lines = append(lines, m.styles.Subtle.Render(terminalTicker(m.frame, "gh issue view")))

	return m.screen(lines)
}

func (m AppModel) issueDetailContentLines() []string {
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

	lines := []string{
		m.styles.Accent.Render(task.Description),
		meta,
		"",
		m.styles.Panel.Render(strings.Join([]string{
			fmt.Sprintf("st4t3: %s", strings.ToUpper(task.IssueState)),
			fmt.Sprintf("xp(base): %s", formatOptionalXP(task.XPBase)),
			fmt.Sprintf("author: %s", task.IssueAuthorLogin),
			fmt.Sprintf("assignees: %s", renderList(task.IssueAssigneeLogins)),
			fmt.Sprintf("labels: %s", renderList(task.IssueLabels)),
			fmt.Sprintf("Implementacion Inicio: %s", formatDateOrDash(task.PlannedDate)),
			fmt.Sprintf("Implementacion Fin: %s", formatDateOrDash(task.PlannedEndDate)),
			fmt.Sprintf("Implementacion Fin Real: %s", formatDateOrDash(task.RealDate)),
			fmt.Sprintf("url: %s", task.IssueURL),
		}, "\n")),
		"",
		m.styles.Subtle.Render("B0DY"),
		m.styles.Panel.Render(task.IssueBody),
	}

	return strings.Split(strings.Join(lines, "\n"), "\n")
}

func renderList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func formatOptionalXP(value *float64) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f", *value)
}

func formatDateOrDash(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format(domain.DateLayout)
}
