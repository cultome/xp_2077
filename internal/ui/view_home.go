package ui

import (
	"strings"
)

func (m AppModel) viewHome() string {
	title := m.headerLine("XP_L34D3RB04RD")
	filterLine := strings.Join([]string{
		m.styles.Subtle.Render("fr0m"),
		m.startInput.View(),
		m.styles.Subtle.Render("t0"),
		m.endInput.View(),
	}, " ")

	lines := []string{
		title,
		"",
		m.styles.Accent.Render("d4t3 r4ng3 f1lt3r"),
		filterLine,
	}

	if m.homeErr != "" {
		lines = append(lines, m.styles.Error.Render(m.homeErr))
	} else {
		lines = append(lines, m.styles.Subtle.Render("ENTER: 4pply r4ng3 / 0p3n us3r"))
	}

	lines = append(lines, "")
	lines = append(lines, m.styles.Panel.Render(m.userTable.View()))
	lines = append(lines, "")
	lines = append(lines, m.styles.Footer.Render("TAB:f0cus  ENTER:0k  UP/DOWN:n4v  ESC:b4ck  Q:qu1t"))
	lines = append(lines, m.styles.Subtle.Render(terminalTicker(m.frame, "r34dy")))
	return m.screen(lines)
}
