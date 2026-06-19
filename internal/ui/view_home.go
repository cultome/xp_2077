package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m AppModel) viewHome() string {
	title := m.headerLine("XP_L34D3RB04RD")
	filterLine := strings.Join([]string{
		m.styles.Subtle.Render("fr0m"),
		m.startInput.View(),
		m.styles.Subtle.Render("t0"),
		m.endInput.View(),
		m.styles.Subtle.Render("  f1ltr0"),
		m.filterInput.View(),
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
		lines = append(lines, m.styles.Subtle.Render("TAB c4mb1a f0c0 · ENTER 4pl1c4/4br3"))
	}

	lines = append(lines, "")
	lines = append(lines, m.styles.Accent.Render("xp r4ng3s"))
	lines = append(lines, m.viewXPRanges())
	lines = append(lines, "")
	lines = append(lines, m.styles.Panel.Render(m.renderLeaderboard()))
	lines = append(lines, "")
	lines = append(lines, m.styles.Footer.Render("TAB:f0cus  ENTER:0k  UP/DOWN:n4v  S:s0rt  P:r4ng3  ^E:r3-3xtr43r  ESC:b4ck  Q:qu1t"))
	if summary := m.extractionSummaryLine(); summary != "" {
		lines = append(lines, summary)
	}
	lines = append(lines, m.styles.Subtle.Render(terminalTicker(m.frame, "r34dy")))
	return m.screen(lines)
}

// renderLeaderboard draws the leaderboard as custom rows with per-user neon XP
// bars (colored by tier), highlighting the row under the cursor. The bubbles
// table is kept only for cursor/scroll state.
func (m AppModel) renderLeaderboard() string {
	if len(m.users) == 0 {
		return m.styles.Subtle.Render("// N0 D4T4 1N R4NG3 //")
	}

	maxXP := 0.0
	for _, u := range m.users {
		if u.XP > maxXP {
			maxXP = u.XP
		}
	}

	cursor := m.userTable.Cursor()
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(m.users) {
		cursor = len(m.users) - 1
	}

	capacity := 10
	if m.height > 0 {
		capacity = max(5, m.height-20)
	}
	start := 0
	if len(m.users) > capacity {
		start = cursor - capacity/2
		if start < 0 {
			start = 0
		}
		if start+capacity > len(m.users) {
			start = len(m.users) - capacity
		}
	}
	end := start + capacity
	if end > len(m.users) {
		end = len(m.users)
	}

	const barW = 16
	header := m.styles.Subtle.Render(fmt.Sprintf("   %-3s %-16s %8s  %-*s %7s %8s", "#", "US3R", "XP", barW, "L3V3L", "1SSU3S", "D(+/-)"))
	rows := []string{header}
	for i := start; i < end; i++ {
		u := m.users[i]
		bar := xpBar(u.XP, maxXP, barW, m.barStyleFor(u.XP), m.styles.BarEmpty)
		left := fmt.Sprintf("%-3d %-16s %8.1f ", i+1, truncate(u.Login, 16), u.XP)
		right := fmt.Sprintf(" %7d %+8.1f", u.TicketCount, u.AvgDelayDays)

		textStyle := m.styles.Footer
		marker := "  "
		if i == cursor {
			textStyle = m.styles.Accent
			marker = m.styles.Accent.Render("▶ ")
		}
		rows = append(rows, marker+textStyle.Render(left)+bar+textStyle.Render(right))
	}
	if end < len(m.users) || start > 0 {
		rows = append(rows, m.styles.Subtle.Render(fmt.Sprintf("   .. %d-%d / %d ..", start+1, end, len(m.users))))
	}
	return strings.Join(rows, "\n")
}

func (m AppModel) barStyleFor(xp float64) lipgloss.Style {
	if m.xpRanges.Available {
		if xp >= m.xpRanges.Outstanding.Low {
			return m.styles.BarBreaker
		}
		if xp >= m.xpRanges.High.Low {
			return m.styles.BarOverclocked
		}
	}
	return m.styles.BarStandard
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

// extractionSummaryLine reports cards that the last extraction omitted. Issue
// cards skipped for lack of repo access are flagged as a warning (potential lost
// tasks); non-issue cards (PRs/drafts) are reported as a neutral note.
func (m AppModel) extractionSummaryLine() string {
	if !m.extractionRan {
		return ""
	}
	if m.skippedIssueCards > 0 {
		return m.styles.Error.Render(fmt.Sprintf(
			"[!] %d c4rd(s) d3 1ssu3 0m1t1d4s (s1n 4cc3s0 4l r3p0)  +%d n0-1ssu3",
			m.skippedIssueCards, m.skippedOtherCards,
		))
	}
	return m.styles.Subtle.Render(fmt.Sprintf(
		"3xtr: 0 c4rds d3 1ssu3 0m1t1d4s  (%d n0-1ssu3 f1ltr4d4s)",
		m.skippedOtherCards,
	))
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
