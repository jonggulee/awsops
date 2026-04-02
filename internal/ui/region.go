package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type regionEntry struct {
	id       string
	label    string
	group    string
	selected bool
}

var allRegions = []regionEntry{
	{id: "ap-northeast-2", label: "ap-northeast-2  Seoul",        group: "Asia Pacific"},
	{id: "ap-northeast-1", label: "ap-northeast-1  Tokyo",        group: "Asia Pacific"},
	{id: "us-east-1",      label: "us-east-1       N. Virginia",  group: "United States"},
	{id: "us-east-2",      label: "us-east-2       Ohio",         group: "United States"},
	{id: "us-west-1",      label: "us-west-1       N. California", group: "United States"},
	{id: "us-west-2",      label: "us-west-2       Oregon",       group: "United States"},
}

func defaultRegions() []regionEntry {
	entries := make([]regionEntry, len(allRegions))
	copy(entries, allRegions)
	for i, r := range entries {
		if r.id == "ap-northeast-2" {
			entries[i].selected = true
		}
	}
	return entries
}

func selectedRegionIDs(entries []regionEntry) []string {
	var ids []string
	for _, r := range entries {
		if r.selected {
			ids = append(ids, r.id)
		}
	}
	return ids
}

var (
	regionTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")).Background(lipgloss.Color("57")).PaddingLeft(1).PaddingRight(1)
	regionGroupStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Bold(true).PaddingLeft(1)
	regionSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	regionCursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	regionNormalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	regionHintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(1)
)

func renderRegionScreen(entries []regionEntry, cursor int, width int) string {
	var b strings.Builder

	title := regionTitleStyle.Render("Select Regions")
	b.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("57")).Width(width).Render(title) + "\n\n")

	lastGroup := ""
	for i, r := range entries {
		if r.group != lastGroup {
			if lastGroup != "" {
				b.WriteString("\n")
			}
			b.WriteString(regionGroupStyle.Render("▸ "+r.group) + "\n")
			lastGroup = r.group
		}

		check := "  "
		nameStyle := regionNormalStyle
		if r.selected {
			check = "✓ "
			nameStyle = regionSelectedStyle
		}

		line := fmt.Sprintf("%s%s", check, nameStyle.Render(r.label))

		if i == cursor {
			line = regionCursorStyle.Render(fmt.Sprintf("  %s ", line))
		} else {
			line = "    " + line
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + regionHintStyle.Render("<space> toggle  <a> all  <n> none  <enter> apply  <esc> cancel"))
	return b.String()
}
