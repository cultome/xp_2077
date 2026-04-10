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
	labels := []string{"bug", "enhancement", "documentation", "high-priority", "frontend", "backend", "infra"}
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	tasks := make(map[string][]domain.TaskXP, len(users))
	for _, user := range users {
		userTasks := make([]domain.TaskXP, 0, 24)
		for i := range 24 {
			offsetDays := rnd.Intn(120)
			planned := base.AddDate(0, 0, offsetDays)
			real := planned.AddDate(0, 0, rnd.Intn(4)-1)
			xp := 15 + rnd.Intn(90)
			xpBase := float64(xp)
			issueNumber := 1000 + rnd.Intn(9000)
			issueState := "open"
			var issueClosedAt *time.Time
			if rnd.Intn(100) < 40 {
				issueState = "closed"
				closed := real.AddDate(0, 0, rnd.Intn(3))
				issueClosedAt = &closed
			}
			issueLabels := []string{
				labels[rnd.Intn(len(labels))],
				labels[rnd.Intn(len(labels))],
			}
			assignees := []string{user}
			if rnd.Intn(100) < 35 {
				assignees = append(assignees, users[rnd.Intn(len(users))])
			}
			userTasks = append(userTasks, domain.TaskXP{
				Description:         fmt.Sprintf("Task %02d pipeline tuning", i+1),
				PlannedDate:         planned,
				RealDate:            real,
				Project:             projects[rnd.Intn(len(projects))],
				ID:                  fmt.Sprintf("%s-%03d", user[:4], i+1),
				XP:                  float64(xp),
				XPBase:              &xpBase,
				IssueNumber:         issueNumber,
				IssueState:          issueState,
				IssueURL:            fmt.Sprintf("https://github.com/cultome/%s/issues/%d", "xp_2077", issueNumber),
				IssueAuthorLogin:    user,
				IssueAssigneeLogins: assignees,
				IssueLabels:         issueLabels,
				IssueBody:           fmt.Sprintf("### Context\nTune flow for %s.\n\n### Acceptance criteria\n- Keep XP stable\n- Improve reliability for batch %02d\n", projects[rnd.Intn(len(projects))], i+1),
				IssueCreatedAt:      planned.AddDate(0, 0, -2),
				IssueUpdatedAt:      real,
				IssueClosedAt:       issueClosedAt,
			})
		}
		tasks[user] = userTasks
	}

	return &Repository{users: users, tasks: tasks}
}

func (r *Repository) Leaderboard(dateRange domain.DateRange) ([]domain.UserXP, error) {
	result := make([]domain.UserXP, 0, len(r.users))
	for _, user := range r.users {
		total := 0.0
		ticketCount := 0
		delayDaysTotal := 0.0
		for _, task := range r.tasks[user] {
			if dateRange.Contains(task.RealDate) {
				total += task.XP
				ticketCount++
				delayDaysTotal += task.RealDate.Sub(task.PlannedDate).Hours() / 24
			}
		}
		avgDelayDays := 0.0
		if ticketCount > 0 {
			avgDelayDays = delayDaysTotal / float64(ticketCount)
		}
		result = append(result, domain.UserXP{
			Login:        user,
			XP:           total,
			TicketCount:  ticketCount,
			AvgDelayDays: avgDelayDays,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].XP == result[j].XP {
			return result[i].Login < result[j].Login
		}
		return result[i].XP > result[j].XP
	})
	return result, nil
}

func (r *Repository) TasksForUser(login string, dateRange domain.DateRange) ([]domain.TaskXP, error) {
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
	return result, nil
}

func (r *Repository) TaskByID(taskID string) (domain.TaskXP, error) {
	for _, tasks := range r.tasks {
		for _, task := range tasks {
			if task.ID == taskID {
				return task, nil
			}
		}
	}
	return domain.TaskXP{}, fmt.Errorf("%w: id=%s", domain.ErrTaskNotFound, taskID)
}
