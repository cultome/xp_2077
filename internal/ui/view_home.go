package ui

import (
	"fmt"
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
	lines = append(lines, m.styles.Accent.Render("xp r4ng3s"))
	lines = append(lines, m.viewXPRanges())
	lines = append(lines, "")
	lines = append(lines, m.styles.Panel.Render(m.userTable.View()))
	lines = append(lines, "")
	lines = append(lines, m.styles.Footer.Render("TAB:f0cus  ENTER:0k  UP/DOWN:n4v  ESC:b4ck  Q:qu1t"))
	lines = append(lines, m.styles.Subtle.Render(terminalTicker(m.frame, "r34dy")))
	return m.screen(lines)
}

func (m AppModel) viewXPRanges() string {
	if !m.xpRanges.Available {
		return m.styles.Subtle.Render("sin d4t0s p4r4 c4lcul4r r4ng0s.")
	}

	normal := m.formatRangeLine("ST4ND4RD", m.xpRanges.Normal)
	high := m.formatRangeLine("0V3RCL0CK3D", m.xpRanges.High)
	outstanding := m.formatRangeLine("SYST3M BR34K3R", m.xpRanges.Outstanding)
	return strings.Join([]string{normal, high, outstanding}, "\n")
}

func (m AppModel) formatRangeLine(label string, r xpRange) string {
	if !r.HasHigh {
		return m.styles.Subtle.Render(label + ": d3sd3 " + formatXP(r.Low))
	}
	return m.styles.Subtle.Render(label + ": " + formatXP(r.Low) + " - " + formatXP(r.High))
}

func formatXP(value float64) string {
	return fmt.Sprintf("%.1f", value)
}
