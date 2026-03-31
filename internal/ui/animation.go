package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(85*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func pulseGlyph(frame int) string {
	frames := []string{"◢", "◣", "◤", "◥"}
	return frames[frame%len(frames)]
}

func terminalTicker(frame int, label string) string {
	cursor := "_"
	if frame%2 == 0 {
		cursor = " "
	}
	return "> " + label + cursor
}

func meter(width, progress int) string {
	if width < 10 {
		width = 10
	}
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	filled := (inner * progress) / 100
	if filled > inner {
		filled = inner
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", inner-filled) + "]"
}
