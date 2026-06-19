package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	AppFrame     lipgloss.Style
	Header       lipgloss.Style
	Subtle       lipgloss.Style
	Error        lipgloss.Style
	Success      lipgloss.Style
	Accent       lipgloss.Style
	Link         lipgloss.Style
	Panel        lipgloss.Style
	FocusedInput lipgloss.Style
	BlurInput    lipgloss.Style
	Footer       lipgloss.Style

	// XP bar tiers (filled segments), reused by the leaderboard.
	BarStandard    lipgloss.Style
	BarOverclocked lipgloss.Style
	BarBreaker     lipgloss.Style
	BarEmpty       lipgloss.Style
}

func newStyles() styles {
	bg := "#0B0804"
	amber := "#FF8C00"
	amberBright := "#FFB347"
	amberDim := "#A9681F"
	amberDeep := "#4A2C08"

	// Neon accents layered on the amber base for semantics only.
	neonRed := "#FF2A6D"
	neonGreen := "#00FF9C"
	neonCyan := "#00F0FF"

	return styles{
		AppFrame: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(amber)).
			Background(lipgloss.Color(bg)).
			Padding(0, 1),
		Header: lipgloss.NewStyle().
			Foreground(lipgloss.Color(amberBright)).
			Bold(true),
		Subtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(amberDim)),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(neonRed)).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color(neonGreen)).
			Bold(true),
		Accent: lipgloss.NewStyle().
			Foreground(lipgloss.Color(amber)).
			Bold(true),
		Link: lipgloss.NewStyle().
			Foreground(lipgloss.Color(neonCyan)),
		Panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(amberDeep)).
			Padding(0, 1),
		FocusedInput: lipgloss.NewStyle().
			Foreground(lipgloss.Color(amberBright)).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(amber)).
			Padding(0, 1),
		BlurInput: lipgloss.NewStyle().
			Foreground(lipgloss.Color(amberDim)).
			Border(lipgloss.HiddenBorder()).
			Padding(0, 1),
		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color(amber)),

		BarStandard:    lipgloss.NewStyle().Foreground(lipgloss.Color(amber)),
		BarOverclocked: lipgloss.NewStyle().Foreground(lipgloss.Color(neonCyan)),
		BarBreaker:     lipgloss.NewStyle().Foreground(lipgloss.Color(neonGreen)).Bold(true),
		BarEmpty:       lipgloss.NewStyle().Foreground(lipgloss.Color(amberDeep)),
	}
}
