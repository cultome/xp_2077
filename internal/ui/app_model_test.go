package ui

import (
	"math"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cultome/xp_2077/internal/domain"
	"github.com/cultome/xp_2077/internal/env"
	"github.com/cultome/xp_2077/internal/mock"
)

func TestAppRouteTransitions(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "token")

	m := NewAppModel(mock.NewRepository(2077), false)

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

	updated, _ = m.Update(extractionDoneMsg{err: nil})
	m = updated.(AppModel)

	if m.route != routeHome {
		t.Fatalf("expected routeHome after loading, got %v", m.route)
	}
	if len(m.users) == 0 {
		t.Fatal("expected leaderboard users to be loaded")
	}
}

func TestDetailToIssueNavigation(t *testing.T) {
	m := NewAppModel(mock.NewRepository(2077), false)
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

func TestSkipExtractBypassesEnvAndLoading(t *testing.T) {
	m := NewAppModel(mock.NewRepository(2077), true)

	for i := 0; i < splashFrames+2; i++ {
		updated, _ := m.Update(tickMsg(time.Now()))
		m = updated.(AppModel)
	}

	if m.route != routeHome {
		t.Fatalf("expected routeHome when skip extract is enabled, got %v", m.route)
	}
}

type dateRangeRepo struct{}

func (r dateRangeRepo) Leaderboard(dateRange domain.DateRange) ([]domain.UserXP, error) {
	jan := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if dateRange.Start.Equal(jan) {
		return []domain.UserXP{
			{Login: "a", XP: 10},
			{Login: "b", XP: 20},
			{Login: "c", XP: 30},
			{Login: "d", XP: 40},
			{Login: "e", XP: 50},
		}, nil
	}
	return []domain.UserXP{
		{Login: "a", XP: 100},
		{Login: "b", XP: 110},
		{Login: "c", XP: 120},
		{Login: "d", XP: 130},
		{Login: "e", XP: 200},
	}, nil
}

func (r dateRangeRepo) TasksForUser(login string, dateRange domain.DateRange) ([]domain.TaskXP, error) {
	return nil, nil
}

func (r dateRangeRepo) TaskByID(taskID string) (domain.TaskXP, error) {
	return domain.TaskXP{}, domain.ErrTaskNotFound
}

func TestComputeXPRangesWithoutOutstandingUpperBound(t *testing.T) {
	ranges := computeXPRanges([]domain.UserXP{
		{Login: "a", XP: 10},
		{Login: "b", XP: 20},
		{Login: "c", XP: 30},
		{Login: "d", XP: 40},
		{Login: "e", XP: 50},
	})

	if !ranges.Available {
		t.Fatal("expected ranges to be available")
	}
	if !approxEqual(ranges.Median, 30, 0.001) {
		t.Fatalf("expected median 30.0, got %.4f", ranges.Median)
	}
	if !approxEqual(ranges.StdDeviation, math.Sqrt(200), 0.001) {
		t.Fatalf("unexpected std deviation: %.4f", ranges.StdDeviation)
	}
	if !approxEqual(ranges.Normal.Low, 30-0.5*math.Sqrt(200), 0.001) {
		t.Fatalf("unexpected normal low: %.4f", ranges.Normal.Low)
	}
	if !approxEqual(ranges.Normal.High, ranges.Normal.Low+math.Sqrt(200), 0.001) {
		t.Fatalf("unexpected normal high: %.4f", ranges.Normal.High)
	}
	if !approxEqual(ranges.High.Low, 30+0.5*math.Sqrt(200), 0.001) {
		t.Fatalf("unexpected high low: %.4f", ranges.High.Low)
	}
	if !approxEqual(ranges.High.High, ranges.High.Low+1.5*math.Sqrt(200), 0.001) {
		t.Fatalf("unexpected high high: %.4f", ranges.High.High)
	}
	if !approxEqual(ranges.Outstanding.Low, 30+2*math.Sqrt(200), 0.001) {
		t.Fatalf("unexpected outstanding low: %.4f", ranges.Outstanding.Low)
	}
	if ranges.Outstanding.HasHigh {
		t.Fatalf("expected outstanding range without high bound, got %.4f", ranges.Outstanding.High)
	}
}

func TestComputeXPRangesWithOutstandingUpperBound(t *testing.T) {
	ranges := computeXPRanges([]domain.UserXP{
		{Login: "a", XP: 10},
		{Login: "b", XP: 20},
		{Login: "c", XP: 30},
		{Login: "d", XP: 40},
		{Login: "e", XP: 100},
	})

	if !ranges.Outstanding.HasHigh {
		t.Fatal("expected outstanding range to have upper bound")
	}
	if !approxEqual(ranges.Outstanding.High, 100, 0.001) {
		t.Fatalf("expected outstanding high 100.0, got %.4f", ranges.Outstanding.High)
	}
}

func TestApplyDateFilterRecomputesXPRanges(t *testing.T) {
	m := NewAppModel(dateRangeRepo{}, true)

	m.startInput.SetValue("2026-01-01")
	m.endInput.SetValue("2026-01-01")
	m.applyDateFilter()

	firstMedian := m.xpRanges.Median
	firstStdDev := m.xpRanges.StdDeviation

	m.startInput.SetValue("2026-02-01")
	m.endInput.SetValue("2026-02-01")
	m.applyDateFilter()

	if approxEqual(firstMedian, m.xpRanges.Median, 0.001) {
		t.Fatalf("expected median to change after date filter, stayed at %.4f", m.xpRanges.Median)
	}
	if approxEqual(firstStdDev, m.xpRanges.StdDeviation, 0.001) {
		t.Fatalf("expected std deviation to change after date filter, stayed at %.4f", m.xpRanges.StdDeviation)
	}
}

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) <= epsilon
}
