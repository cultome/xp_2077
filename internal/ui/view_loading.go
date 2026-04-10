package ui

import (
	"fmt"
)

func (m AppModel) viewLoading() string {
	title := m.headerLine("L04D1NG :: gh_t3l3m3try")
	width := max(30, m.width-22)
	progress := meter(width, m.pipeState.Progress)
	stageLine := "st4g3 " + m.pipeState.Label()
	detailLine := terminalTicker(m.frame, m.pipeState.Detail)

	if m.loadingErr != "" {
		stageLine = "st4g3 [x] extraction_failed"
		detailLine = m.loadingErr
	}
	lines := []string{
		title,
		"",
		m.styles.Subtle.Render("3xtr4ct10n p1p3l1n3 1n pr0gr3ss..."),
		"",
		fmt.Sprintf("%s %s", m.styles.Accent.Render(progress), m.styles.Accent.Render(fmt.Sprintf("%d%%", m.pipeState.Progress))),
		m.styles.Subtle.Render(stageLine),
		"",
		m.styles.Subtle.Render(detailLine),
		"",
		m.styles.Footer.Render("3xtr4ct -> n0rm4l1z3 -> c4lcXP   R: r3try (on error)"),
	}
	return m.screen(lines)
}
