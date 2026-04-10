package mock

import (
	"math"
	"testing"
	"time"

	"github.com/cultome/xp_2077/internal/domain"
)

func TestLeaderboardIncludesTicketMetrics(t *testing.T) {
	repo := NewRepository(2077)
	dateRange := domain.DateRange{
		Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	leaderboard := repo.Leaderboard(dateRange)
	if len(leaderboard) == 0 {
		t.Fatal("expected leaderboard with users")
	}

	for _, user := range leaderboard {
		tasks := repo.TasksForUser(user.Login, dateRange)
		if user.TicketCount != len(tasks) {
			t.Fatalf("expected ticket count %d for %s, got %d", len(tasks), user.Login, user.TicketCount)
		}

		expectedAvg := 0.0
		if len(tasks) > 0 {
			delayTotal := 0.0
			for _, task := range tasks {
				delayTotal += task.RealDate.Sub(task.PlannedDate).Hours() / 24
			}
			expectedAvg = delayTotal / float64(len(tasks))
		}

		if math.Abs(user.AvgDelayDays-expectedAvg) > 1e-9 {
			t.Fatalf("expected avg delay %.10f for %s, got %.10f", expectedAvg, user.Login, user.AvgDelayDays)
		}
	}
}
