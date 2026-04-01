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

// --- messages ---

type instancesLoadedMsg struct {
	instances []awsclient.Instance
	errs      []error
}

type sgLoadedMsg struct {
	groups []awsclient.SecurityGroup
	errs   []error
}

// --- modes & views ---

type inputMode int

const (
	modeNormal  inputMode = iota
	modeSearch            // / 검색
	modeCommand           // : 커맨드
)

type viewType int

const (
	viewEC2 viewType = iota
	viewSG
)

type screen int

const (
	screenTable  screen = iota
	screenDetail        // d 눌렀을 때 상세 화면
)

var viewNames = map[viewType]string{
	viewEC2: "ec2",
	viewSG:  "sg",
}

// --- model ---

type Model struct {
	table          table.Model
	spinner        spinner.Model
	input          textinput.Model
	mode           inputMode
	view           viewType
	screen         screen
	selectedInst   *awsclient.Instance
	filters        []string
	instances      []awsclient.Instance
	groups         []awsclient.SecurityGroup
	fetchErr       []error
	loading        bool
	width          int
	height         int
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.CharLimit = 64

	return Model{
		spinner: s,
		input:   ti,
		loading: true,
		mode:    modeNormal,
		view:    viewEC2,
		screen:  screenTable,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchInstances(), fetchSecurityGroups())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case instancesLoadedMsg:
		m.instances = msg.instances
		m.fetchErr = append(m.fetchErr, msg.errs...)
		m.loading = len(m.groups) == 0 && len(m.instances) == 0
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case sgLoadedMsg:
		m.groups = msg.groups
		m.fetchErr = append(m.fetchErr, msg.errs...)
		m.loading = len(m.groups) == 0 && len(m.instances) == 0
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case tea.KeyMsg:
		// 디테일 화면에서는 esc/q 로만 뒤로
		if m.screen == screenDetail {
			switch msg.String() {
			case "esc", "q", "ctrl+c":
				if msg.String() == "ctrl+c" {
					return m, tea.Quit
				}
				m.screen = screenTable
				m.selectedInst = nil
			}
			return m, nil
		}

		switch m.mode {
		case modeNormal:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "d":
				if m.view == viewEC2 {
					inst := m.selectedInstance()
					if inst != nil {
						m.selectedInst = inst
						m.screen = screenDetail
					}
				}
				return m, nil
			case "/":
				m.mode = modeSearch
				m.input.Placeholder = "type to search..."
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case ":":
				m.mode = modeCommand
				m.input.Placeholder = "ec2 / sg"
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case "esc":
				m.filters = nil
				m.input.SetValue("")
				m.table = m.buildCurrentTable()
				return m, nil
			}

		case modeSearch:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = modeNormal
				m.filters = nil
				m.input.SetValue("")
				m.input.Blur()
				m.table = m.buildCurrentTable()
				return m, nil
			case "enter":
				if v := strings.TrimSpace(m.input.Value()); v != "" {
					m.filters = append(m.filters, v)
				}
				m.mode = modeNormal
				m.input.SetValue("")
				m.input.Blur()
				m.table = m.buildCurrentTable()
				return m, nil
			}

		case modeCommand:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = modeNormal
				m.input.SetValue("")
				m.input.Blur()
				return m, nil
			case "enter":
				m.mode = modeNormal
				m.input.Blur()
				m.applyCommand(m.input.Value())
				m.input.SetValue("")
				return m, nil
			}
		}
	}

	if !m.loading {
		var cmds []tea.Cmd
		var cmd tea.Cmd

		if m.mode == modeSearch || m.mode == modeCommand {
			prevVal := m.input.Value()
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
			if m.mode == modeSearch && m.input.Value() != prevVal {
				m.table = m.buildCurrentTable()
			}
		}

		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// selectedInstance returns the Instance matching the currently highlighted table row.
func (m *Model) selectedInstance() *awsclient.Instance {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	id := row[2] // Instance ID column
	for i := range m.instances {
		if m.instances[i].InstanceID == id {
			return &m.instances[i]
		}
	}
	return nil
}

func (m *Model) applyCommand(cmd string) {
	switch strings.TrimSpace(strings.ToLower(cmd)) {
	case "ec2":
		m.view = viewEC2
		m.table = m.buildCurrentTable()
	case "sg":
		m.view = viewSG
		m.table = m.buildCurrentTable()
	}
}

func (m *Model) buildCurrentTable() table.Model {
	switch m.view {
	case viewSG:
		return buildSGTable(filterSGRows(m.groups, m.filters), m.height)
	default:
		return buildEC2Table(filterEC2Rows(m.instances, m.filters), m.height)
	}
}

// --- EC2 table ---

func buildEC2Table(rows []table.Row, height int) table.Model {
	cols := []table.Column{
		{Title: "Profile", Width: 18},
		{Title: "Name", Width: 16},
		{Title: "Instance ID", Width: 20},
		{Title: "State", Width: 12},
		{Title: "Type", Width: 10},
		{Title: "Private IP", Width: 16},
		{Title: "Public IP", Width: 16},
	}
	return newTable(cols, rows, height)
}

func filterEC2Rows(instances []awsclient.Instance, filters []string) []table.Row {
	if len(filters) == 0 {
		return ec2Rows(instances)
	}
	var filtered []awsclient.Instance
	for _, inst := range instances {
		if matchAll(filters, inst.Profile, inst.Name, inst.InstanceID, inst.State, inst.Type, inst.PrivateIP) {
			filtered = append(filtered, inst)
		}
	}
	return ec2Rows(filtered)
}

func ec2Rows(instances []awsclient.Instance) []table.Row {
	rows := make([]table.Row, len(instances))
	for i, inst := range instances {
		rows[i] = table.Row{inst.Profile, inst.Name, inst.InstanceID, inst.State, inst.Type, inst.PrivateIP, inst.PublicIP}
	}
	return rows
}

// --- SG table ---

func buildSGTable(rows []table.Row, height int) table.Model {
	cols := []table.Column{
		{Title: "Profile", Width: 18},
		{Title: "Name", Width: 24},
		{Title: "Group ID", Width: 22},
		{Title: "VPC ID", Width: 22},
		{Title: "Description", Width: 40},
	}
	return newTable(cols, rows, height)
}

func filterSGRows(groups []awsclient.SecurityGroup, filters []string) []table.Row {
	if len(filters) == 0 {
		return sgRows(groups)
	}
	var filtered []awsclient.SecurityGroup
	for _, sg := range groups {
		if matchAll(filters, sg.Profile, sg.Name, sg.GroupID, sg.VpcID, sg.Description) {
			filtered = append(filtered, sg)
		}
	}
	return sgRows(filtered)
}

func sgRows(groups []awsclient.SecurityGroup) []table.Row {
	rows := make([]table.Row, len(groups))
	for i, sg := range groups {
		rows[i] = table.Row{sg.Profile, sg.Name, sg.GroupID, sg.VpcID, sg.Description}
	}
	return rows
}

// matchAll returns true if every filter term matches at least one of the fields.
func matchAll(filters []string, fields ...string) bool {
	for _, f := range filters {
		q := strings.ToLower(f)
		matched := false
		for _, field := range fields {
			if strings.Contains(strings.ToLower(field), q) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// --- shared table builder ---

func newTable(cols []table.Column, rows []table.Row, height int) table.Model {
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
