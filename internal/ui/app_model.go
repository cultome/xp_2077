package ui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cultome/xp_2077/internal/domain"
	"github.com/cultome/xp_2077/internal/env"
	"github.com/cultome/xp_2077/internal/mock"
)

type route int

const (
	routeSplash route = iota
	routeEnvCheck
	routeLoading
	routeHome
	routeDetail
)

const splashFrames = 35

type envCheckedMsg struct {
	report env.Report
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
	pipeline    *mock.Pipeline
	pipeState   mock.PipelineState
	repo        *mock.Repository

	startInput textinput.Model
	endInput   textinput.Model
	focusIndex int
	homeErr    string
	dateRange  domain.DateRange
	users      []domain.UserXP
	userTable  table.Model

	detailUser  domain.UserXP
	detailTasks []domain.TaskXP
	detailTable table.Model
}

func NewAppModel() AppModel {
	start, _ := domain.ParseDate("2026-01-01")
	end, _ := domain.ParseDate("2026-04-30")

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
			{Title: "USER", Width: 24},
			{Title: "XP", Width: 8},
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
			{Title: "DESCRIPCION", Width: 22},
			{Title: "FECHA PLANEADA", Width: 14},
			{Title: "FECHA REAL", Width: 12},
			{Title: "PROYECTO", Width: 12},
			{Title: "ID", Width: 10},
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
		requiredEnv: []string{"GITHUB_TOKEN", "GITHUB_ORG"},
		pipeline:    mock.NewPipeline(),
		repo:        mock.NewRepository(2077),
		startInput:  startInput,
		endInput:    endInput,
		dateRange:   domain.DateRange{Start: start, End: end},
		userTable:   userTable,
		detailTable: detailTable,
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
				m.route = routeEnvCheck
				return m, tea.Batch(checkEnvCmd(m.requiredEnv), tickCmd())
			}
		case routeLoading:
			m.pipeState = m.pipeline.Tick()
			if m.pipeState.Done {
				m.refreshLeaderboard()
				m.route = routeHome
			}
		}
		return m, tickCmd()
	case envCheckedMsg:
		m.envReport = typed.report
		if !m.envReport.Missing {
			m.route = routeLoading
			m.pipeline.Reset()
			m.pipeState = m.pipeline.State()
		}
		return m, nil
	case tea.KeyMsg:
		if key.Matches(typed, m.keys.Quit) {
			return m, tea.Quit
		}
		switch m.route {
		case routeEnvCheck:
			return m.handleEnvCheckKeys(typed)
		case routeHome:
			return m.handleHomeKeys(typed)
		case routeDetail:
			return m.handleDetailKeys(typed)
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
		m.route = routeLoading
		m.pipeline.Reset()
		m.pipeState = m.pipeline.State()
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
	var cmd tea.Cmd
	m.detailTable, cmd = m.detailTable.Update(msg)
	return *m, cmd
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
	m.users = m.repo.Leaderboard(m.dateRange)
	rows := make([]table.Row, 0, len(m.users))
	for _, user := range m.users {
		rows = append(rows, table.Row{user.Login, strconv.Itoa(user.XP)})
	}
	m.userTable.SetRows(rows)
	if len(rows) > 0 && m.userTable.Cursor() >= len(rows) {
		m.userTable.SetCursor(0)
	}
}

func (m *AppModel) refreshDetail() {
	m.detailTasks = m.repo.TasksForUser(m.detailUser.Login, m.dateRange)
	rows := make([]table.Row, 0, len(m.detailTasks))
	for _, task := range m.detailTasks {
		rows = append(rows, table.Row{
			task.Description,
			task.PlannedDate.Format(domain.DateLayout),
			task.RealDate.Format(domain.DateLayout),
			task.Project,
			task.ID,
			strconv.Itoa(task.XP),
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
	m.userTable.SetHeight(max(6, m.height-12))
	m.detailTable.SetWidth(w)
	m.detailTable.SetHeight(max(6, m.height-10))
}

func checkEnvCmd(required []string) tea.Cmd {
	return func() tea.Msg {
		return envCheckedMsg{report: env.Check(required)}
	}
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
