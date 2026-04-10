package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cultome/xp_2077/internal/env"
)

func TestAppRouteTransitions(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token")
	t.Setenv("GITHUB_ORG", "org")

	m := NewAppModel()

	for i := 0; i < splashFrames+2; i++ {
		updated, _ := m.Update(tickMsg(time.Now()))
		m = updated.(AppModel)
	}

	if m.route != routeEnvCheck {
		t.Fatalf("expected routeEnvCheck, got %v", m.route)
	}
	updated, _ := m.Update(envCheckedMsg{report: env.Check(m.requiredEnv)})
	m = updated.(AppModel)
	if m.route != routeLoading {
		t.Fatalf("expected routeLoading, got %v", m.route)
	}

	for i := 0; i < 40 && m.route != routeHome; i++ {
		updated, _ := m.Update(tickMsg(time.Now()))
		m = updated.(AppModel)
	}

	if m.route != routeHome {
		t.Fatalf("expected routeHome after loading, got %v", m.route)
	}
	if len(m.users) == 0 {
		t.Fatal("expected leaderboard users to be loaded")
	}
}

func TestDetailToIssueNavigation(t *testing.T) {
	m := NewAppModel()
	m.route = routeHome
	m.focusIndex = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AppModel)
	if m.route != routeDetail {
		t.Fatalf("expected routeDetail, got %v", m.route)
	}
	if len(m.detailTasks) == 0 {
		t.Fatal("expected detail tasks to be loaded")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(AppModel)
	if m.route != routeIssueDetail {
		t.Fatalf("expected routeIssueDetail, got %v", m.route)
	}
	if m.issueTask.ID == "" {
		t.Fatal("expected selected issue task to be set")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(AppModel)
	if m.route != routeDetail {
		t.Fatalf("expected routeDetail after esc, got %v", m.route)
	}
}
