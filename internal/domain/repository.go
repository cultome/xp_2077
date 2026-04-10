package domain

import "errors"

var ErrTaskNotFound = errors.New("task not found")

// Repository provides UI-ready leaderboard and task views.
type Repository interface {
	Leaderboard(dateRange DateRange) ([]UserXP, error)
	TasksForUser(login string, dateRange DateRange) ([]TaskXP, error)
	TaskByID(taskID string) (TaskXP, error)
}
