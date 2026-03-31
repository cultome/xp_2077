package domain

import "testing"

func TestParseDateRangeValid(t *testing.T) {
	r, err := ParseDateRange("2026-01-01", "2026-02-01")
	if err != nil {
		t.Fatalf("expected valid range, got error: %v", err)
	}
	if r.Start.After(r.End) {
		t.Fatalf("invalid range produced: %+v", r)
	}
}

func TestParseDateRangeInvalidOrder(t *testing.T) {
	_, err := ParseDateRange("2026-02-01", "2026-01-01")
	if err == nil {
		t.Fatal("expected error for invalid date order")
	}
	if err != ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}
