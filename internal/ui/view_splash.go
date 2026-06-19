package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m AppModel) viewSplash() string {
	title := m.headerLine("BOOT_SEQUENCE: COLD_START_INITIATED")
	progress := (m.frame * 100) / splashFrames
	if progress > 100 {
		progress = 100
	}

	subsystems := []string{
		"KERNEL.MATRIX",
		"MEM *WEAVE",
		"I0.BUS ",
		"NET.STACK ",
		"GFX.TERM ",
		"TELEM.DAEMON",
	}
	subsystemTags := []string{
		"[LOAD]",
		"[BOND]",
		"[SYNC]",
		"[VOID]",
		"[SHOW]",
		"[SPY ]",
	}
	// Uneven durations (in frames) per subsystem to feel more realistic.
	stageFrames := []int{4, 7, 3, 8, 5, 8}
	doneCount, activeIdx, stageElapsed, stageTotal := bootStageProgress(m.frame, stageFrames)

	// Logo starts heavily corrupted and resolves as the boot progresses.
	glitchAmt := 0
	if progress < 100 {
		glitchAmt = (100-progress)/12 + 1
	}

	lines := []string{
		title,
		"",
		centeredLogo(m.width, m.styles.Accent, glitchAmt, m.frame),
		"",
		centeredText(m.width, m.styles.Subtle, decryptReveal("0p3r4t1ng syst3m // b00t s3qu3nc3", progress)),
		"",
		m.styles.Subtle.Render("V_4.0.2-STABLE"),
	}

	visible := doneCount + 1
	if visible < 1 {
		visible = 1
	}
	if visible > len(subsystems) {
		visible = len(subsystems)
	}

	for idx := 0; idx < visible; idx++ {
		line := ""
		switch {
		case idx < doneCount:
			line = renderSubsystemLine(subsystems[idx], subsystemTags[idx], subsystemDotsWidth, true)
		case idx == activeIdx:
			dots := stageDots(stageElapsed, stageTotal, subsystemDotsWidth)
			line = renderSubsystemLine(subsystems[idx], subsystemTags[idx], dots, false)
		}
		lines = append(lines, m.styles.Accent.Render(line))
	}
	for i := visible; i < len(subsystems); i++ {
		lines = append(lines, "")
	}
	lines = append(lines, "")
	lines = append(lines, "")
	lines = append(lines, splashCopyright(m))

	return m.screen(lines)
}

func centeredLogo(width int, style lipgloss.Style, glitchAmt, frame int) string {
	logo := []string{
		"██╗    ██╗███████╗ █████╗ ██╗   ██╗███████╗██████╗ ",
		"██║    ██║██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗",
		"██║ █╗ ██║█████╗  ███████║██║   ██║█████╗  ██████╔╝",
		"██║███╗██║██╔══╝  ██╔══██║╚██╗ ██╔╝██╔══╝  ██╔══██╗",
		"╚███╔███╔╝███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║",
		" ╚══╝╚══╝ ╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝",
	}
	if glitchAmt > 0 {
		for i := range logo {
			logo[i] = glitch(logo[i], frame+i, glitchAmt)
		}
	}
	block := style.Render(strings.Join(logo, "\n"))
	return lipgloss.PlaceHorizontal(max(20, width-4), lipgloss.Center, block)
}

func centeredText(width int, style lipgloss.Style, text string) string {
	return lipgloss.PlaceHorizontal(max(20, width-4), lipgloss.Center, style.Render(text))
}

func splashCopyright(m AppModel) string {
	return centeredText(
		m.width,
		m.styles.Subtle,
		"© 2077 // ΔLΞPH CØRP // 4LL R1GHT5 R353RV3D",
	)
}

const subsystemDotsWidth = 24
const subsystemNameWidth = 13

func bootStageProgress(frame int, stageFrames []int) (doneCount, activeIdx, stageElapsed, stageTotal int) {
	if frame < 0 {
		return 0, 0, 0, stageFrames[0]
	}
	elapsed := frame
	done := 0
	for idx, stage := range stageFrames {
		if elapsed >= stage {
			elapsed -= stage
			done++
			continue
		}
		return done, idx, elapsed, stage
	}
	last := len(stageFrames) - 1
	return len(stageFrames), last, stageFrames[last], stageFrames[last]
}

func stageDots(elapsed, total, width int) int {
	if total <= 0 || width <= 0 {
		return 0
	}
	dots := ((elapsed + 1) * width) / total
	if dots < 1 {
		dots = 1
	}
	if dots > width {
		dots = width
	}
	return dots
}

func renderSubsystemLine(name, doneTag string, dotsCount int, done bool) string {
	if dotsCount < 0 {
		dotsCount = 0
	}
	if dotsCount > subsystemDotsWidth {
		dotsCount = subsystemDotsWidth
	}
	dots := strings.Repeat(".", dotsCount)
	pad := strings.Repeat(" ", subsystemDotsWidth-dotsCount)
	tag := "      "
	if done {
		tag = doneTag
	}
	return fmt.Sprintf("> [ %-*s ] %s%s %s", subsystemNameWidth, name, dots, pad, tag)
}
