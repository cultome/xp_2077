package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cultome/xp_2077/internal/ui"
)

func main() {
	program := tea.NewProgram(ui.NewAppModel(), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		log.Fatalf("could not start program: %v", err)
	}
}
