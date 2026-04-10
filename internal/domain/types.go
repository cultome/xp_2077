package domain

import (
	"errors"
	"time"
)

const DateLayout = "2006-01-02"

var ErrInvalidDateRange = errors.New("start date must be before or equal to end date")

type DateRange struct {
	Start time.Time
	End   time.Time
}

type UserXP struct {
	Login        string
	XP           float64
	TicketCount  int
	AvgDelayDays float64
}

type TaskXP struct {
	Description         string
	PlannedDate         time.Time
	PlannedEndDate      time.Time
	RealDate            time.Time
	Project             string
	ID                  string
	XP                  float64
	XPBase              *float64
	IssueNumber         int
	IssueState          string
	IssueURL            string
	IssueAuthorLogin    string
	IssueAssigneeLogins []string
	IssueLabels         []string
	IssueBody           string
	IssueCreatedAt      time.Time
	IssueUpdatedAt      time.Time
	IssueClosedAt       *time.Time
}

func ParseDate(value string) (time.Time, error) {
	return time.Parse(DateLayout, value)
}

func ParseDateRange(start, end string) (DateRange, error) {
	startDate, err := ParseDate(start)
	if err != nil {
		return DateRange{}, err
	}
	endDate, err := ParseDate(end)
	if err != nil {
		return DateRange{}, err
	}
	if startDate.After(endDate) {
		return DateRange{}, ErrInvalidDateRange
	}
	return DateRange{Start: startDate, End: endDate}, nil
}

func (r DateRange) Contains(t time.Time) bool {
	return (t.Equal(r.Start) || t.After(r.Start)) && (t.Equal(r.End) || t.Before(r.End))
}
