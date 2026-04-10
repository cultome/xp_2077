package gh

import "testing"

func TestParseProjectXPFieldsEarlyFinish(t *testing.T) {
	fields := map[string]string{
		"XP":                         "100",
		"fecha programada de inicio": "2026-01-01",
		"fecha programada de fin":    "2026-01-11",
		"fecha real de fin":          "2026-01-09",
	}

	xpBase, start, end, real, xpFinal := parseProjectXPFields(fields)
	if xpBase == nil || start == nil || end == nil || real == nil || xpFinal == nil {
		t.Fatal("expected all xp fields to be parsed")
	}
	if got, want := *xpFinal, 120.0; got != want {
		t.Fatalf("expected xp final %.1f, got %.1f", want, got)
	}
}

func TestParseProjectXPFieldsLateFinishClamp(t *testing.T) {
	fields := map[string]string{
		"XP":                         "50",
		"fecha programada de inicio": "2026-01-01",
		"fecha programada de fin":    "2026-01-11",
		"fecha real de fin":          "2026-01-31",
	}

	_, _, _, _, xpFinal := parseProjectXPFields(fields)
	if xpFinal == nil {
		t.Fatal("expected xp final to be computed")
	}
	if got, want := *xpFinal, 0.0; got != want {
		t.Fatalf("expected clamped xp final %.1f, got %.1f", want, got)
	}
}

func TestParseProjectXPFieldsRounding(t *testing.T) {
	fields := map[string]string{
		"XP":                         "80",
		"fecha programada de inicio": "2026-01-01",
		"fecha programada de fin":    "2026-01-07",
		"fecha real de fin":          "2026-01-06",
	}

	_, _, _, _, xpFinal := parseProjectXPFields(fields)
	if xpFinal == nil {
		t.Fatal("expected xp final to be computed")
	}
	if got, want := *xpFinal, 93.3; got != want {
		t.Fatalf("expected rounded xp final %.1f, got %.1f", want, got)
	}
}

func TestParseProjectXPFieldsMissingData(t *testing.T) {
	fields := map[string]string{
		"XP":                         "100",
		"fecha programada de inicio": "2026-01-01",
		"fecha programada de fin":    "2026-01-07",
	}

	xpBase, start, end, real, xpFinal := parseProjectXPFields(fields)
	if xpBase != nil || start != nil || end != nil || real != nil || xpFinal != nil {
		t.Fatal("expected nil values when any required field is missing")
	}
}

func TestParseProjectXPFieldsSupportsImplementationAliases(t *testing.T) {
	fields := map[string]string{
		"XP":                      "100",
		"Implementacion Inicio":   "2026-01-01",
		"Implementacion Fin":      "2026-01-11",
		"Implementacion Fin Real": "2026-01-09",
	}

	_, _, _, _, xpFinal := parseProjectXPFields(fields)
	if xpFinal == nil {
		t.Fatal("expected xp final for implementation alias fields")
	}
	if got, want := *xpFinal, 120.0; got != want {
		t.Fatalf("expected xp final %.1f, got %.1f", want, got)
	}
}

func TestParseRepoIssueXPFieldsAppliesSpecialRules(t *testing.T) {
	fields := map[string]string{
		"Status":       "Done",
		"Story Points": "8",
		"Priority":     "P1",
		"Due Date":     "2026-04-10",
	}
	xpBase, start, end, real, xpFinal := parseRepoIssueXPFields("[Special Tasks for Aleph] Sample", fields)
	if xpBase == nil || start == nil || end == nil || real == nil || xpFinal == nil {
		t.Fatal("expected all values for valid repo issue special task")
	}
	if got, want := *xpBase, 8.0; got != want {
		t.Fatalf("expected xp base %.1f, got %.1f", want, got)
	}
	if got, want := *xpFinal, 12.0; got != want {
		t.Fatalf("expected xp final %.1f, got %.1f", want, got)
	}
	if got, want := start.Format("2006-01-02"), "2026-04-10"; got != want {
		t.Fatalf("expected due date mapping %s, got %s", want, got)
	}
	if got, want := end.Format("2006-01-02"), "2026-04-10"; got != want {
		t.Fatalf("expected due date mapping %s, got %s", want, got)
	}
	if got, want := real.Format("2006-01-02"), "2026-04-10"; got != want {
		t.Fatalf("expected due date mapping %s, got %s", want, got)
	}
}

func TestParseRepoIssueXPFieldsRequiresDoneStatus(t *testing.T) {
	fields := map[string]string{
		"Status":       "In Progress",
		"Story Points": "8",
		"Priority":     "P1",
		"Due Date":     "2026-04-10",
	}
	xpBase, start, end, real, xpFinal := parseRepoIssueXPFields("[Special Tasks for Aleph] Sample", fields)
	if xpBase != nil || start != nil || end != nil || real != nil || xpFinal != nil {
		t.Fatal("expected nil values when status is not done")
	}
}

func TestParseRepoIssueXPFieldsRequiresPrefixAndKnownPriority(t *testing.T) {
	fields := map[string]string{
		"Status":       "Done",
		"Story Points": "5",
		"Priority":     "P9",
		"Due Date":     "2026-04-10",
	}
	xpBase, start, end, real, xpFinal := parseRepoIssueXPFields("Regular issue", fields)
	if xpBase != nil || start != nil || end != nil || real != nil || xpFinal != nil {
		t.Fatal("expected nil values when title prefix does not match")
	}
	xpBase, start, end, real, xpFinal = parseRepoIssueXPFields("[Special Tasks for Aleph] Sample", fields)
	if xpBase != nil || start != nil || end != nil || real != nil || xpFinal != nil {
		t.Fatal("expected nil values for unknown priority")
	}
}
