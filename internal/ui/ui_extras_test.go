package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cultome/xp_2077/internal/domain"
	"github.com/cultome/xp_2077/internal/mock"
)

func TestViewsRenderWithoutPanic(t *testing.T) {
	m := NewAppModel(mock.NewRepository(2077), true)
	m.width = 120
	m.height = 40
	m.resizeTables()

	routes := []route{routeSplash, routeEnvCheck, routeLoading, routeHome, routeDetail, routeIssueDetail}
	for _, r := range routes {
		m.route = r
		if r == routeDetail || r == routeIssueDetail {
			m.detailUser = domain.UserXP{Login: "akcervantes"}
			m.refreshDetail()
			if len(m.detailTasks) > 0 {
				m.issueTask = m.detailTasks[0]
			}
		}
		out := m.View()
		if strings.TrimSpace(out) == "" {
			t.Fatalf("route %d rendered empty", r)
		}
	}
}

func TestHomeViewShowsHUDAndLeaderboard(t *testing.T) {
	m := NewAppModel(mock.NewRepository(2077), true)
	m.width = 120
	m.height = 40
	m.resizeTables()
	m.route = routeHome
	out := m.View()
	if !strings.Contains(out, "0NL1N3") {
		t.Error("home view missing HUD online marker")
	}
	if !strings.Contains(out, "L3V3L") {
		t.Error("home view missing leaderboard bar header")
	}
	if !strings.Contains(out, "█") && !strings.Contains(out, "░") {
		t.Error("home view missing XP bar glyphs")
	}
}

func TestXPFillCount(t *testing.T) {
	cases := []struct {
		value, max  float64
		width, want int
	}{
		{0, 100, 10, 0},
		{100, 100, 10, 10},
		{50, 100, 10, 5},
		{200, 100, 10, 10}, // clamp
		{1, 1000, 10, 1},   // nonzero shows at least one cell
		{-5, 100, 10, 0},
		{50, 0, 10, 0}, // max 0 => ratio 0
	}
	for _, c := range cases {
		if got := xpFillCount(c.value, c.max, c.width); got != c.want {
			t.Errorf("xpFillCount(%v,%v,%d)=%d want %d", c.value, c.max, c.width, got, c.want)
		}
	}
}

func TestXPBarVisibleWidth(t *testing.T) {
	style := lipgloss.NewStyle()
	bar := xpBar(40, 100, 16, style, style)
	if w := lipgloss.Width(bar); w != 16 {
		t.Fatalf("xpBar visible width = %d, want 16", w)
	}
}

func TestGlitchDeterministicAndShape(t *testing.T) {
	const s = "XP L34D3RB04RD"
	a := glitch(s, 7, 3)
	b := glitch(s, 7, 3)
	if a != b {
		t.Fatalf("glitch not deterministic for same frame: %q vs %q", a, b)
	}
	if len([]rune(a)) != len([]rune(s)) {
		t.Fatalf("glitch changed length: %d vs %d", len([]rune(a)), len([]rune(s)))
	}
	if glitch(s, 7, 0) != s {
		t.Fatal("glitch with intensity 0 should be a no-op")
	}
	// spaces preserved
	g := glitch("A B C", 3, 10)
	r := []rune(g)
	if r[1] != ' ' || r[3] != ' ' {
		t.Fatalf("glitch corrupted spaces: %q", g)
	}
}

func TestDecryptReveal(t *testing.T) {
	const s = "B00T S3QU3NC3"
	if decryptReveal(s, 100) != s {
		t.Fatal("progress 100 must return the original text")
	}
	d1 := decryptReveal(s, 50)
	d2 := decryptReveal(s, 50)
	if d1 != d2 {
		t.Fatal("decryptReveal not deterministic for same progress")
	}
	zero := decryptReveal(s, 0)
	if len([]rune(zero)) != len([]rune(s)) {
		t.Fatal("decryptReveal changed length")
	}
	// space at index 4 preserved
	if []rune(zero)[4] != ' ' {
		t.Fatalf("decryptReveal corrupted a space: %q", zero)
	}
}

func TestSortUsers(t *testing.T) {
	base := []domain.UserXP{
		{Login: "carol", XP: 50, TicketCount: 9, AvgDelayDays: -2},
		{Login: "alice", XP: 200, TicketCount: 3, AvgDelayDays: 5},
		{Login: "bob", XP: 120, TicketCount: 1, AvgDelayDays: 1},
	}
	clone := func() []domain.UserXP { return append([]domain.UserXP(nil), base...) }

	xp := clone()
	sortUsers(xp, sortXPDesc)
	if xp[0].Login != "alice" {
		t.Errorf("XP desc: got %s", xp[0].Login)
	}
	tk := clone()
	sortUsers(tk, sortTicketsDesc)
	if tk[0].Login != "carol" {
		t.Errorf("tickets desc: got %s", tk[0].Login)
	}
	dl := clone()
	sortUsers(dl, sortDelayAsc)
	if dl[0].Login != "carol" {
		t.Errorf("delay asc: got %s", dl[0].Login)
	}
	nm := clone()
	sortUsers(nm, sortLoginAsc)
	if nm[0].Login != "alice" {
		t.Errorf("login asc: got %s", nm[0].Login)
	}
}

func TestPresetRange(t *testing.T) {
	now := time.Date(2026, 6, 18, 13, 30, 0, 0, time.UTC)
	today := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)

	r0, l0 := presetRange(0, now)
	if l0 != "7D" || !r0.Start.Equal(today.AddDate(0, 0, -7)) || !r0.End.Equal(today) {
		t.Errorf("7D preset wrong: %+v %s", r0, l0)
	}
	r3, l3 := presetRange(3, now)
	if l3 != "Q" || !r3.Start.Equal(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("quarter preset wrong: %+v %s", r3, l3)
	}
	r4, l4 := presetRange(4, now)
	if l4 != "YTD" || !r4.Start.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("YTD preset wrong: %+v %s", r4, l4)
	}
	// wraps around
	if _, l5 := presetRange(5, now); l5 != "7D" {
		t.Errorf("preset idx 5 should wrap to 7D, got %s", l5)
	}
}

func TestBrowserCommand(t *testing.T) {
	cases := map[string]struct {
		name string
		args []string
	}{
		"linux":   {"xdg-open", []string{"https://x"}},
		"darwin":  {"open", []string{"https://x"}},
		"windows": {"rundll32", []string{"url.dll,FileProtocolHandler", "https://x"}},
	}
	for goos, want := range cases {
		name, args := browserCommand(goos, "https://x")
		if name != want.name || strings.Join(args, " ") != strings.Join(want.args, " ") {
			t.Errorf("browserCommand(%s) = %s %v, want %s %v", goos, name, args, want.name, want.args)
		}
	}
}
