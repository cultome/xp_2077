package ui

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cultome/xp_2077/internal/domain"
	"github.com/cultome/xp_2077/internal/env"
	"github.com/cultome/xp_2077/internal/extract"
	"github.com/cultome/xp_2077/internal/mock"
)

type route int

const (
	routeSplash route = iota
	routeEnvCheck
	routeLoading
	routeHome
	routeDetail
	routeIssueDetail
)

const splashFrames = 35

type envCheckedMsg struct {
	report env.Report
}

type extractionDoneMsg struct {
	err    error
	result extract.Result
}

type xpRange struct {
	Low     float64
	High    float64
	HasHigh bool
}

type xpRanges struct {
	Available    bool
	Normal       xpRange
	High         xpRange
	Outstanding  xpRange
	Median       float64
	StdDeviation float64
}

type AppModel struct {
	route route

	width  int
	height int
	frame  int

	keys   keyMap
	styles styles

	requiredEnv []string
	envReport   env.Report
	extractCfg  extract.Config
	tracker     *extract.Tracker
	pipeState   extract.State
	loadingErr  string
	repo        domain.Repository

	startInput textinput.Model
	endInput   textinput.Model
	focusIndex int
	homeErr    string
	dateRange  domain.DateRange
	users      []domain.UserXP
	xpRanges   xpRanges
	userTable  table.Model

	detailUser  domain.UserXP
	detailTasks []domain.TaskXP
	detailTable table.Model
	issueTask   domain.TaskXP
	issueScroll int
	skipExtract bool

	extractionRan     bool
	skippedIssueCards int // project ISSUE cards skipped because the repo wasn't accessible
	skippedOtherCards int // PR/draft/redacted cards intentionally ignored
}

func NewAppModel(repo domain.Repository, skipExtract bool) AppModel {
	if repo == nil {
		repo = mock.NewRepository(2077)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := today
	start := today.AddDate(-1, 0, 0)

	startInput := textinput.New()
	startInput.Placeholder = domain.DateLayout
	startInput.SetValue(start.Format(domain.DateLayout))
	startInput.CharLimit = 10
	startInput.Width = 12

	endInput := textinput.New()
	endInput.Placeholder = domain.DateLayout
	endInput.SetValue(end.Format(domain.DateLayout))
	endInput.CharLimit = 10
	endInput.Width = 12

	userTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "US3R", Width: 24},
			{Title: "XP", Width: 8},
			{Title: "1SSU3S", Width: 8},
			{Title: "4VG D(+/-)", Width: 12},
		}),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	tableStyles := table.DefaultStyles()
	tableStyles.Header = tableStyles.Header.Foreground(lipgloss.Color("#FFB347")).Bold(true)
	tableStyles.Selected = tableStyles.Selected.Foreground(lipgloss.Color("#0B0804")).Background(lipgloss.Color("#FF8C00")).Bold(true)
	userTable.SetStyles(tableStyles)

	detailTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "D3SCR1PC10N", Width: 22},
			{Title: "PL4N D4T3", Width: 14},
			{Title: "1MPL-F1N", Width: 14},
			{Title: "R34L D4T3", Width: 12},
			{Title: "D3LT4 D4YS", Width: 11},
			{Title: "PR0Y3CT0", Width: 24},
			{Title: "XP", Width: 6},
		}),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(11),
	)
	detailTable.SetStyles(tableStyles)

	m := AppModel{
		route:       routeSplash,
		keys:        newKeyMap(),
		styles:      newStyles(),
		requiredEnv: []string{"GITHUB_TOKEN"},
		extractCfg:  extract.ConfigFromEnv(),
		repo:        repo,
		startInput:  startInput,
		endInput:    endInput,
		dateRange:   domain.DateRange{Start: start, End: end},
		userTable:   userTable,
		detailTable: detailTable,
		skipExtract: skipExtract,
	}
	m.refreshInputFocus()
	m.refreshLeaderboard()
	return m
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(tickCmd())
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.resizeTables()
		return m, nil
	case tickMsg:
		m.frame++
		switch m.route {
		case routeSplash:
			if m.frame > splashFrames {
				if m.skipExtract {
					m.route = routeHome
					return m, tickCmd()
				}
				m.route = routeEnvCheck
				return m, tea.Batch(checkEnvCmd(m.requiredEnv), tickCmd())
			}
		case routeLoading:
			if m.tracker != nil {
				m.pipeState = m.tracker.State()
			}
		}
		return m, tickCmd()
	case envCheckedMsg:
		m.envReport = typed.report
		if !m.envReport.Missing {
			m.startLoading()
			return m, tea.Batch(runExtractionCmd(m.extractCfg, m.tracker), tickCmd())
		}
		return m, nil
	case extractionDoneMsg:
		if typed.err != nil {
			m.loadingErr = typed.err.Error()
			return m, nil
		}
		m.loadingErr = ""
		if m.tracker != nil {
			m.pipeState = m.tracker.State()
		}
		m.extractionRan = true
		m.skippedIssueCards = typed.result.ProjectStats.InaccessibleIssues
		m.skippedOtherCards = typed.result.ProjectStats.NonIssues
		m.refreshLeaderboard()
		m.route = routeHome
		return m, nil
	case tea.KeyMsg:
		if key.Matches(typed, m.keys.Quit) {
			return m, tea.Quit
		}
		switch m.route {
		case routeEnvCheck:
			return m.handleEnvCheckKeys(typed)
		case routeLoading:
			return m.handleLoadingKeys(typed)
		case routeHome:
			return m.handleHomeKeys(typed)
		case routeDetail:
			return m.handleDetailKeys(typed)
		case routeIssueDetail:
			return m.handleIssueDetailKeys(typed)
		default:
			return m, nil
		}
	default:
		return m, nil
	}
}

func (m AppModel) View() string {
	content := ""
	switch m.route {
	case routeSplash:
		content = m.viewSplash()
	case routeEnvCheck:
		content = m.viewEnvCheck()
	case routeLoading:
		content = m.viewLoading()
	case routeDetail:
		content = m.viewDetail()
	case routeIssueDetail:
		content = m.viewIssueDetail()
	default:
		content = m.viewHome()
	}
	return m.renderFull(content)
}

func (m *AppModel) handleEnvCheckKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Retry) {
		return *m, checkEnvCmd(m.requiredEnv)
	}
	if key.Matches(msg, m.keys.Enter) && !m.envReport.Missing {
		m.startLoading()
		return *m, tea.Batch(runExtractionCmd(m.extractCfg, m.tracker), tickCmd())
	}
	return *m, nil
}

func (m *AppModel) handleLoadingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Retry) && m.loadingErr != "" {
		m.startLoading()
		return *m, tea.Batch(runExtractionCmd(m.extractCfg, m.tracker), tickCmd())
	}
	return *m, nil
}

func (m *AppModel) handleHomeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Tab):
		m.focusIndex = (m.focusIndex + 1) % 3
		m.refreshInputFocus()
		return *m, nil
	case key.Matches(msg, m.keys.Enter):
		if m.focusIndex < 2 {
			m.applyDateFilter()
			return *m, nil
		}
		if len(m.users) == 0 {
			return *m, nil
		}
		idx := m.userTable.Cursor()
		if idx < 0 || idx >= len(m.users) {
			return *m, nil
		}
		m.detailUser = m.users[idx]
		m.refreshDetail()
		m.route = routeDetail
		return *m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.applyDateFilter()
		return *m, nil
	case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
		if m.focusIndex == 2 {
			var cmd tea.Cmd
			m.userTable, cmd = m.userTable.Update(msg)
			return *m, cmd
		}
	}

	var cmd tea.Cmd
	if m.focusIndex == 0 {
		m.startInput, cmd = m.startInput.Update(msg)
		return *m, cmd
	}
	if m.focusIndex == 1 {
		m.endInput, cmd = m.endInput.Update(msg)
		return *m, cmd
	}
	m.userTable, cmd = m.userTable.Update(msg)
	return *m, cmd
}

func (m *AppModel) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.route = routeHome
		return *m, nil
	}
	if key.Matches(msg, m.keys.Enter) {
		idx := m.detailTable.Cursor()
		if idx >= 0 && idx < len(m.detailTasks) {
			m.issueTask = m.detailTasks[idx]
			m.issueScroll = 0
			m.route = routeIssueDetail
		}
		return *m, nil
	}
	var cmd tea.Cmd
	m.detailTable, cmd = m.detailTable.Update(msg)
	return *m, cmd
}

func (m *AppModel) handleIssueDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.route = routeDetail
		m.issueScroll = 0
		return *m, nil
	}
	if key.Matches(msg, m.keys.Up) {
		if m.issueScroll > 0 {
			m.issueScroll--
		}
		return *m, nil
	}
	if key.Matches(msg, m.keys.Down) {
		maxScroll := m.issueDetailMaxScroll()
		if m.issueScroll < maxScroll {
			m.issueScroll++
		}
	}
	return *m, nil
}

func (m AppModel) issueDetailVisibleHeight() int {
	if m.height <= 0 {
		return 14
	}
	return max(8, m.height-14)
}

func (m AppModel) issueDetailMaxScroll() int {
	total := len(m.issueDetailContentLines())
	visible := m.issueDetailVisibleHeight()
	if total <= visible {
		return 0
	}
	return total - visible
}

func (m *AppModel) refreshInputFocus() {
	m.startInput.Blur()
	m.endInput.Blur()
	if m.focusIndex == 0 {
		m.startInput.Focus()
	} else if m.focusIndex == 1 {
		m.endInput.Focus()
	}
}

func (m *AppModel) applyDateFilter() {
	parsed, err := domain.ParseDateRange(m.startInput.Value(), m.endInput.Value())
	if err != nil {
		m.homeErr = "D4T3-R4NG3 ERR: usa YYYY-MM-DD y start <= end."
		return
	}
	m.homeErr = ""
	m.dateRange = parsed
	m.refreshLeaderboard()
}

func (m *AppModel) refreshLeaderboard() {
	users, err := m.repo.Leaderboard(m.dateRange)
	if err != nil {
		m.users = nil
		m.xpRanges = xpRanges{}
		m.homeErr = "D4T4 ERR: no fue posible cargar leaderboard."
		m.userTable.SetRows([]table.Row{})
		return
	}
	m.users = users
	m.xpRanges = computeXPRanges(m.users)
	rows := make([]table.Row, 0, len(m.users))
	for _, user := range m.users {
		rows = append(rows, table.Row{
			user.Login,
			fmt.Sprintf("%.1f", user.XP),
			fmt.Sprintf("%d", user.TicketCount),
			fmt.Sprintf("%+.1f", user.AvgDelayDays),
		})
	}
	m.userTable.SetRows(rows)
	if len(rows) > 0 && m.userTable.Cursor() >= len(rows) {
		m.userTable.SetCursor(0)
	}
}

func computeXPRanges(users []domain.UserXP) xpRanges {
	if len(users) == 0 {
		return xpRanges{}
	}

	xpValues := make([]float64, 0, len(users))
	for _, user := range users {
		xpValues = append(xpValues, user.XP)
	}
	sort.Float64s(xpValues)

	median := median(xpValues)
	stdDev := populationStdDev(xpValues)

	normalLow := median - 0.5*stdDev
	normalHigh := normalLow + stdDev
	highLow := median + 0.5*stdDev
	highHigh := highLow + 1.5*stdDev
	outstandingLow := median + 2*stdDev

	outstanding := xpRange{
		Low: outstandingLow,
	}
	for i := len(xpValues) - 1; i >= 0; i-- {
		if xpValues[i] > outstandingLow {
			outstanding.High = xpValues[i]
			outstanding.HasHigh = true
			break
		}
	}

	return xpRanges{
		Available: true,
		Normal: xpRange{
			Low:     normalLow,
			High:    normalHigh,
			HasHigh: true,
		},
		High: xpRange{
			Low:     highLow,
			High:    highHigh,
			HasHigh: true,
		},
		Outstanding:  outstanding,
		Median:       median,
		StdDeviation: stdDev,
	}
}

func median(values []float64) float64 {
	n := len(values)
	mid := n / 2
	if n%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] + values[mid]) / 2
}

func populationStdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))

	sum := 0.0
	for _, value := range values {
		delta := value - mean
		sum += delta * delta
	}
	return math.Sqrt(sum / float64(len(values)))
}

func (m *AppModel) refreshDetail() {
	tasks, err := m.repo.TasksForUser(m.detailUser.Login, m.dateRange)
	if err != nil {
		m.detailTasks = nil
		m.detailTable.SetRows([]table.Row{})
		if m.homeErr == "" {
			m.homeErr = "D4T4 ERR: no fue posible cargar detalle."
		}
		return
	}
	m.detailTasks = tasks
	rows := make([]table.Row, 0, len(m.detailTasks))
	for _, task := range m.detailTasks {
		deltaDays := int(task.RealDate.Sub(task.PlannedEndDate).Hours() / 24)
		deltaDaysValue := ""
		if deltaDays != 0 {
			deltaDaysValue = fmt.Sprintf("%+d", deltaDays)
		}
		rows = append(rows, table.Row{
			task.Description,
			task.PlannedDate.Format(domain.DateLayout),
			task.PlannedEndDate.Format(domain.DateLayout),
			task.RealDate.Format(domain.DateLayout),
			deltaDaysValue,
			fmt.Sprintf("%s#%d", task.Project, task.IssueNumber),
			fmt.Sprintf("%.1f", task.XP),
		})
	}
	m.detailTable.SetRows(rows)
	if len(rows) > 0 {
		m.detailTable.SetCursor(0)
	}
}

func (m *AppModel) resizeTables() {
	if m.width < 40 || m.height < 12 {
		return
	}
	w := m.width - 8
	if w < 32 {
		w = 32
	}
	m.userTable.SetWidth(w)
	m.userTable.SetHeight(max(6, m.height-16))
	m.detailTable.SetWidth(w)
	m.detailTable.SetHeight(max(6, m.height-10))
}

func checkEnvCmd(required []string) tea.Cmd {
	return func() tea.Msg {
		report := env.Check(required)
		return envCheckedMsg{report: report}
	}
}

func runExtractionCmd(cfg extract.Config, tracker *extract.Tracker) tea.Cmd {
	return func() tea.Msg {
		res, err := extract.Run(context.Background(), cfg, tracker)
		return extractionDoneMsg{err: err, result: res}
	}
}

func (m *AppModel) startLoading() {
	m.extractCfg = extract.ConfigFromEnv()
	m.route = routeLoading
	m.loadingErr = ""
	m.tracker = extract.NewTracker()
	m.pipeState = m.tracker.State()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m AppModel) headerLine(title string) string {
	return m.styles.Header.Render(fmt.Sprintf("%s %s", pulseGlyph(m.frame), title))
}

func (m AppModel) renderFull(content string) string {
	if m.width <= 0 || m.height <= 0 {
		return content
	}
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func (m AppModel) screen(lines []string) string {
	body := ""
	if len(lines) > 0 {
		body = lines[0]
		for i := 1; i < len(lines); i++ {
			body += "\n" + lines[i]
		}
	}
	if m.width <= 0 || m.height <= 0 {
		return m.styles.AppFrame.Render(body)
	}
	return m.styles.AppFrame.
		Width(max(20, m.width-2)).
		Height(max(8, m.height-2)).
		Render(body)
}
