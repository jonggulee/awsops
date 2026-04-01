package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).PaddingLeft(1)
	searchStyle = lipgloss.NewStyle().PaddingLeft(1)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).PaddingLeft(1)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(1)
)

func (m Model) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching AWS resources...", m.spinner.View())
	}

	title := titleStyle.Render("awsops — EC2 Instances")

	var searchLine string
	if m.mode == modeSearch {
		searchLine = searchStyle.Render("/ " + m.search.View())
	} else if m.search.Value() != "" {
		searchLine = searchStyle.Render("/ " + m.search.Value() + " (esc: clear)")
	} else {
		searchLine = helpStyle.Render("/ search  ↑/↓ navigate  q quit")
	}

	errMsg := ""
	if len(m.fetchErr) > 0 {
		errMsg = "\n" + errStyle.Render(fmt.Sprintf("⚠ %d profile(s) failed to load", len(m.fetchErr)))
	}

	return fmt.Sprintf("%s\n%s%s\n%s", title, searchLine, errMsg, m.table.View())
}
