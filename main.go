package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cultome/xp_2077/internal/store"
	"github.com/cultome/xp_2077/internal/ui"
)

func main() {
	skipExtract := flag.Bool("skip-extract", false, "Skip GitHub extraction and load only existing SQLite data")
	flag.Parse()

	dbPath := strings.TrimSpace(os.Getenv("OUTPUT_DB"))
	if dbPath == "" {
		dbPath = "./tmp/github_extract.db"
	}

	repo, err := store.OpenSQLite(dbPath)
	if err != nil {
		log.Fatalf("could not open sqlite repository: %v", err)
	}
	defer repo.Close()
	if err := repo.ApplySchema(context.Background()); err != nil {
		log.Fatalf("could not apply sqlite schema: %v", err)
	}

	program := tea.NewProgram(ui.NewAppModel(repo, *skipExtract), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		log.Fatalf("could not start program: %v", err)
	}
}
