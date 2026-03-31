package mock

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/cultome/xp_2077/internal/domain"
)

type Repository struct {
	users []string
	tasks map[string][]domain.TaskXP
}

func NewRepository(seed int64) *Repository {
	rnd := rand.New(rand.NewSource(seed))
	users := []string{
		"neon_ghost", "byte_samurai", "quantum_rat", "glitch_priest",
		"synth_monk", "orbital_witch", "void_hacker", "pixel_ronin",
		"acid_loop", "infra_ninja", "chrome_shade", "turbo_nomad",
	}
	projects := []string{"CYBER-OPS", "NEON-API", "OMEGA-CORE", "EDGE-NET", "PIXEL-DRIVE"}
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	tasks := make(map[string][]domain.TaskXP, len(users))
	for _, user := range users {
		userTasks := make([]domain.TaskXP, 0, 24)
		for i := range 24 {
			offsetDays := rnd.Intn(120)
			planned := base.AddDate(0, 0, offsetDays)
			real := planned.AddDate(0, 0, rnd.Intn(4)-1)
			xp := 15 + rnd.Intn(90)
			userTasks = append(userTasks, domain.TaskXP{
				Description: fmt.Sprintf("Task %02d pipeline tuning", i+1),
				PlannedDate: planned,
				RealDate:    real,
				Project:     projects[rnd.Intn(len(projects))],
				ID:          fmt.Sprintf("%s-%03d", user[:4], i+1),
				XP:          xp,
			})
		}
		tasks[user] = userTasks
	}

	return &Repository{users: users, tasks: tasks}
}

func (r *Repository) Leaderboard(dateRange domain.DateRange) []domain.UserXP {
	result := make([]domain.UserXP, 0, len(r.users))
	for _, user := range r.users {
		total := 0
		for _, task := range r.tasks[user] {
			if dateRange.Contains(task.RealDate) {
				total += task.XP
			}
		}
		result = append(result, domain.UserXP{Login: user, XP: total})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].XP == result[j].XP {
			return result[i].Login < result[j].Login
		}
		return result[i].XP > result[j].XP
	})
	return result
}

func (r *Repository) TasksForUser(login string, dateRange domain.DateRange) []domain.TaskXP {
	src := r.tasks[login]
	result := make([]domain.TaskXP, 0, len(src))
	for _, task := range src {
		if dateRange.Contains(task.RealDate) {
			result = append(result, task)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].RealDate.Before(result[j].RealDate)
	})
	return result
}
