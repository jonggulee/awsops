package ui

import (
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
	{id: "us-east-1", label: "us-east-1       N. Virginia", group: "United States"},
	{id: "us-east-2", label: "us-east-2       Ohio", group: "United States"},
	{id: "us-west-1", label: "us-west-1       N. California", group: "United States"},
	{id: "us-west-2", label: "us-west-2       Oregon", group: "United States"},
	{id: "ap-northeast-2", label: "ap-northeast-2  Seoul", group: "Asia Pacific"},
	{id: "ap-northeast-1", label: "ap-northeast-1  Tokyo", group: "Asia Pacific"},
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

func regionsChanged(current, prev []regionEntry) bool {
	if len(current) != len(prev) {
		return true
	}
	for i := range current {
		if current[i].selected != prev[i].selected {
			return true
		}
	}
	return false
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
	regionTitleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")).Background(lipgloss.Color("57")).PaddingLeft(1).PaddingRight(1)
	regionGroupStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Bold(true).PaddingLeft(1)
	regionSelectedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	regionCursorArrowStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	regionCursorLabelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	regionNormalStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	regionHintStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(1)
)

func renderRegionScreenWithErr(entries []regionEntry, cursor int, width int, showErr, showConfirm bool) string {
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

		arrow := "  "
		if i == cursor {
			arrow = regionCursorArrowStyle.Render("> ")
			if !r.selected {
				nameStyle = regionCursorLabelStyle
			}
		}

		b.WriteString("    " + arrow + check + nameStyle.Render(r.label) + "\n")
	}

	if showConfirm {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).PaddingLeft(1).Render("Discard changes?  y  yes    n / esc  no"))
	} else if showErr {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).PaddingLeft(1).Render("⚠ Select at least one region"))
	}
	b.WriteString("\n" + regionHintStyle.Render("<space> toggle  <a> all  <n> none  <enter> apply (save)  <esc/q> cancel (discard)"))
	return b.String()
}
