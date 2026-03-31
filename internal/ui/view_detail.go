package ui

import (
	"fmt"

	"github.com/cultome/xp_2077/internal/domain"
)

func (m AppModel) viewDetail() string {
	title := m.headerLine("US3R_D3741L")
	meta := fmt.Sprintf(
		"%s  |  r4ng3 %s .. %s",
		m.styles.Accent.Render(m.detailUser.Login),
		m.dateRange.Start.Format(domain.DateLayout),
		m.dateRange.End.Format(domain.DateLayout),
	)

	lines := []string{
		title,
		"",
		meta,
		"",
		m.styles.Panel.Render(m.detailTable.View()),
		"",
		m.styles.Footer.Render("UP/DOWN:n4v  ESC:b4ck  Q:qu1t"),
		m.styles.Subtle.Render(terminalTicker(m.frame, "us3r log op3n")),
	}
	return m.screen(lines)
}
