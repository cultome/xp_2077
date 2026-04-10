package env

import "testing"

func TestCheckRequirements(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "abc")
	t.Setenv("ANOTHER_REQUIRED_VAR", "")

	report := Check([]string{"GITHUB_TOKEN", "ANOTHER_REQUIRED_VAR"})
	if !report.Missing {
		t.Fatal("expected missing requirements")
	}
	if len(report.Statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(report.Statuses))
	}
	if !report.Statuses[0].Present {
		t.Fatal("expected first variable to be present")
	}
	if report.Statuses[1].Present {
		t.Fatal("expected second variable to be missing")
	}
}
