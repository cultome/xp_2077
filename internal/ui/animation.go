package ui

import (
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

var glitchGlyphs = []rune("▓▒░#%&@/\\<>*=+░▒")

// xpFillCount returns how many of `width` cells should be filled for value/max.
// A non-zero value always yields at least one cell so small scores stay visible.
func xpFillCount(value, max float64, width int) int {
	if width < 1 {
		width = 1
	}
	if max <= 0 {
		return 0
	}
	ratio := value / max
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	n := int(ratio*float64(width) + 0.5)
	if n > width {
		n = width
	}
	if value > 0 && n == 0 {
		n = 1
	}
	return n
}

// xpBar renders a proportional block bar (`█…░`) of the given width, coloring the
// filled portion with `filled` and the remainder with `empty`.
func xpBar(value, max float64, width int, filled, empty lipgloss.Style) string {
	if width < 1 {
		width = 1
	}
	n := xpFillCount(value, max, width)
	return filled.Render(strings.Repeat("█", n)) + empty.Render(strings.Repeat("░", width-n))
}

// glitch corrupts up to `intensity` non-space characters of text with cyberpunk
// glyphs. The corruption is seeded by `frame`, so a given frame always renders
// identically (no flicker between re-renders of the same frame).
func glitch(text string, frame, intensity int) string {
	if intensity <= 0 || text == "" {
		return text
	}
	runes := []rune(text)
	positions := make([]int, 0, len(runes))
	for i, ch := range runes {
		if ch != ' ' {
			positions = append(positions, i)
		}
	}
	if len(positions) == 0 {
		return text
	}
	r := rand.New(rand.NewSource(int64(frame)))
	count := intensity
	if count > len(positions) {
		count = len(positions)
	}
	r.Shuffle(len(positions), func(i, j int) { positions[i], positions[j] = positions[j], positions[i] })
	for k := 0; k < count; k++ {
		runes[positions[k]] = glitchGlyphs[r.Intn(len(glitchGlyphs))]
	}
	return string(runes)
}

// decryptReveal reveals text left-to-right as progress goes 0->100: the revealed
// prefix is clear and the rest is scrambled. progress>=100 returns text intact.
func decryptReveal(text string, progress int) string {
	if progress >= 100 {
		return text
	}
	if progress < 0 {
		progress = 0
	}
	runes := []rune(text)
	revealed := len(runes) * progress / 100
	r := rand.New(rand.NewSource(int64(progress) + 1))
	for i := revealed; i < len(runes); i++ {
		if runes[i] == ' ' {
			continue
		}
		runes[i] = glitchGlyphs[r.Intn(len(glitchGlyphs))]
	}
	return string(runes)
}

// scanline renders a faint horizontal rule with a single bright cell that sweeps
// across with `frame`, evoking a CRT scan. Used as a subtle separator.
func scanline(width, frame int, base, sweep lipgloss.Style) string {
	if width < 1 {
		width = 1
	}
	pos := frame % width
	left := strings.Repeat("·", pos)
	right := strings.Repeat("·", width-pos-1)
	return base.Render(left) + sweep.Render("▀") + base.Render(right)
}

// hudClock returns a small animated indicator for the HUD status bar.
func hudClock(frame int) string {
	glyphs := []string{"▰▱▱", "▱▰▱", "▱▱▰", "▱▰▱"}
	return glyphs[frame%len(glyphs)]
}
