package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type regionEntry struct {
	id       string
	label    string
	selected bool
}

var allRegions = []regionEntry{
	{id: "ap-northeast-2", label: "ap-northeast-2  Seoul"},
	{id: "ap-northeast-1", label: "ap-northeast-1  Tokyo"},
	{id: "ap-southeast-1", label: "ap-southeast-1  Singapore"},
	{id: "ap-southeast-2", label: "ap-southeast-2  Sydney"},
	{id: "us-east-1", label: "us-east-1       N. Virginia"},
	{id: "us-east-2", label: "us-east-2       Ohio"},
	{id: "us-west-1", label: "us-west-1       N. California"},
	{id: "us-west-2", label: "us-west-2       Oregon"},
	{id: "eu-west-1", label: "eu-west-1       Ireland"},
	{id: "eu-central-1", label: "eu-central-1    Frankfurt"},
	{id: "eu-north-1", label: "eu-north-1      Stockholm"},
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
	regionSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	regionCursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	regionNormalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	regionHintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(1)
)

func renderRegionScreen(entries []regionEntry, cursor int, width int) string {
	var b strings.Builder

	title := regionTitleStyle.Render("Select Regions")
	padding := width - lipgloss.Width(title)
	if padding < 0 {
		padding = 0
	}
	b.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("57")).Width(width).Render(title) + "\n\n")

	for i, r := range entries {
		check := "  "
		nameStyle := regionNormalStyle
		if r.selected {
			check = "✓ "
			nameStyle = regionSelectedStyle
		}

		line := fmt.Sprintf("%s%s", check, nameStyle.Render(r.label))

		if i == cursor {
			line = regionCursorStyle.Render(fmt.Sprintf(" %s ", line))
		} else {
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + regionHintStyle.Render("<space> toggle  <enter> apply  <esc> cancel"))
	_ = padding
	return b.String()
}
