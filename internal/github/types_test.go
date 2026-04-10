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
