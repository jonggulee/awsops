package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "github.com/jgulee/awsops/internal/aws"
)

type instancesLoadedMsg struct {
	instances []awsclient.Instance
	errs      []error
}

type mode int

const (
	modeNormal mode = iota
	modeSearch
)

type Model struct {
	table     table.Model
	spinner   spinner.Model
	search    textinput.Model
	instances []awsclient.Instance
	mode      mode
	loading   bool
	fetchErr  []error
	width     int
	height    int
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "type to search..."
	ti.CharLimit = 64

	return Model{
		spinner: s,
		search:  ti,
		loading: true,
		mode:    modeNormal,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchInstances())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.loading {
			m.table = buildTable(filterRows(m.instances, m.search.Value()), msg.Width, msg.Height)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case instancesLoadedMsg:
		m.loading = false
		m.fetchErr = msg.errs
		m.instances = msg.instances
		m.table = buildTable(toRows(m.instances), m.width, m.height)

	case tea.KeyMsg:
		switch m.mode {
		case modeNormal:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "/":
				m.mode = modeSearch
				m.search.Focus()
				return m, textinput.Blink
			case "esc":
				m.search.SetValue("")
				m.table = buildTable(toRows(m.instances), m.width, m.height)
				return m, nil
			}
		case modeSearch:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = modeNormal
				m.search.SetValue("")
				m.search.Blur()
				m.table = buildTable(toRows(m.instances), m.width, m.height)
				return m, nil
			case "enter":
				m.mode = modeNormal
				m.search.Blur()
				return m, nil
			}
		}
	}

	if !m.loading {
		var cmds []tea.Cmd
		var cmd tea.Cmd

		if m.mode == modeSearch {
			prevVal := m.search.Value()
			m.search, cmd = m.search.Update(msg)
			cmds = append(cmds, cmd)
			if m.search.Value() != prevVal {
				m.table = buildTable(filterRows(m.instances, m.search.Value()), m.width, m.height)
			}
		}

		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func buildTable(rows []table.Row, _ int, height int) table.Model {
	cols := []table.Column{
		{Title: "Profile", Width: 18},
		{Title: "Name", Width: 16},
		{Title: "Instance ID", Width: 20},
		{Title: "State", Width: 12},
		{Title: "Type", Width: 10},
		{Title: "Private IP", Width: 16},
		{Title: "Public IP", Width: 16},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height-6),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}

func filterRows(instances []awsclient.Instance, query string) []table.Row {
	if query == "" {
		return toRows(instances)
	}
	q := strings.ToLower(query)
	var filtered []awsclient.Instance
	for _, inst := range instances {
		if strings.Contains(strings.ToLower(inst.Profile), q) ||
			strings.Contains(strings.ToLower(inst.Name), q) ||
			strings.Contains(strings.ToLower(inst.InstanceID), q) ||
			strings.Contains(strings.ToLower(inst.State), q) ||
			strings.Contains(strings.ToLower(inst.PrivateIP), q) {
			filtered = append(filtered, inst)
		}
	}
	return toRows(filtered)
}

func toRows(instances []awsclient.Instance) []table.Row {
	rows := make([]table.Row, len(instances))
	for i, inst := range instances {
		rows[i] = table.Row{
			inst.Profile,
			inst.Name,
			inst.InstanceID,
			inst.State,
			inst.Type,
			inst.PrivateIP,
			inst.PublicIP,
		}
	}
	return rows
}
