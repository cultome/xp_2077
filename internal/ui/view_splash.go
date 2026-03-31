package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m AppModel) viewSplash() string {
	title := m.headerLine("Aleph OS :: cold_boot")
	progress := (m.frame * 100) / splashFrames
	if progress > 100 {
		progress = 100
	}

	subsystems := []string{
		"kernel_matrix",
		"mem_weave",
		"i0_bus",
		"net_stack",
		"gfx_terminal",
		"telemetry_daemon",
	}
	revealed := (m.frame * len(subsystems)) / splashFrames
	if revealed < 1 {
		revealed = 1
	}
	if revealed > len(subsystems) {
		revealed = len(subsystems)
	}

	lines := []string{
		title,
		"",
		centeredLogo(m.width, m.styles.Accent),
		"",
		centeredText(m.width, m.styles.Accent, "A L E P H  O S"),
		centeredText(m.width, m.styles.Subtle, "0p3r4t1ng syst3m // b00t s3qu3nc3"),
		"",
		m.styles.Subtle.Render("subsystem init status:"),
	}

	for idx, name := range subsystems {
		if idx >= revealed {
			break
		}
		status := "[BOOT]"
		if idx < revealed-1 || progress == 100 {
			status = "[OK]"
		}
		lines = append(lines, m.styles.Accent.Render(status)+" "+name)
	}

	lines = append(lines, "")
	lines = append(lines, m.styles.Subtle.Render(terminalTicker(m.frame, "bootlog stream")))
	lines = append(lines, "")
	lines = append(lines, m.styles.Footer.Render("sync> handoff t0 env_ch3ck"))

	return m.screen(lines)
}

func centeredLogo(width int, style lipgloss.Style) string {
	logo := []string{
		"                     /\\",
		"                    /  \\",
		"                   / /\\ \\",
		"             /\\   / /__\\ \\   /\\",
		"            /  \\ / /____\\ \\ /  \\",
		"           / /\\ V /______\\ V /\\ \\",
		"          / /  \\_/  /\\    \\_/  \\ \\",
		"         /_/ ALEPH__/  \\__OS__\\_\\",
		"            [ kernel :: signal :: memory ]",
	}
	block := style.Render(strings.Join(logo, "\n"))
	return lipgloss.PlaceHorizontal(max(20, width-4), lipgloss.Center, block)
}

func centeredText(width int, style lipgloss.Style, text string) string {
	return lipgloss.PlaceHorizontal(max(20, width-4), lipgloss.Center, style.Render(text))
}
