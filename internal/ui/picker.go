package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// resourceItem represents a navigable resource type in the command picker.
type resourceItem struct {
	name  string // display name  ("instances", "security-groups", ...)
	cmd   string // applyCommand key ("ec2", "sg", ...)
	group string // category header ("Compute", "Network", "Security")
}

var pickerResources = []resourceItem{
	{"Instances",          "ec2",    "EC2"},
	{"Security Groups",    "sg",     "EC2"},
	{"Network Interfaces", "eni",    "EC2"},
	{"Your VPCs",          "vpc",    "VPC"},
	{"Subnets",            "subnet", "VPC"},
	{"Transit Gateways",   "tgw",    "VPC"},
	{"Clusters",           "eks",    "EKS"},
	{"Certificates",       "acm",    "ACM"},
}

// viewBreadcrumb maps a view to its "Service > Resource" label for the crumb bar.
var viewBreadcrumb = map[viewType]string{
	viewEC2:    "EC2  ›  Instances",
	viewSG:     "EC2  ›  Security Groups",
	viewENI:    "EC2  ›  Network Interfaces",
	viewVPC:    "VPC  ›  Your VPCs",
	viewSubnet: "VPC  ›  Subnets",
	viewTGW:    "VPC  ›  Transit Gateways",
	viewEKS:    "EKS  ›  Clusters",
	viewACM:    "ACM  ›  Certificates",
}

var (
	navGroupStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Bold(true)
	navItemStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	navPickerCursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	navActiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))  // 현재 뷰
	navInputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).PaddingLeft(1)
	navCmdStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	navCmdActiveStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	navNoMatchStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(4)
)

// filteredPickerItems returns items matching the query (empty = all).
func filteredPickerItems(query string) []resourceItem {
	if query == "" {
		return pickerResources
	}
	q := strings.ToLower(query)
	var out []resourceItem
	for _, r := range pickerResources {
		if strings.Contains(strings.ToLower(r.name), q) ||
			strings.Contains(r.cmd, q) ||
			strings.Contains(strings.ToLower(r.group), q) {
			out = append(out, r)
		}
	}
	return out
}

// renderResourcePicker renders the bottom-up command picker.
func renderResourcePicker(m Model) string {
	query := m.input.Value()
	items := filteredPickerItems(query)

	// available lines = total height - header(1) - crumb(1) - input(1) - hint(1)
	availableLines := m.height - 4
	if availableLines < 1 {
		availableLines = 1
	}

	var listLines []string

	if len(items) == 0 {
		listLines = append(listLines, navNoMatchStyle.Render("No matching resources"))
	} else {
		lastGroup := ""
		for i, item := range items {
			// 그룹 헤더
			if item.group != lastGroup {
				if lastGroup != "" {
					listLines = append(listLines, "")
				}
				listLines = append(listLines, "  "+navGroupStyle.Render(item.group))
				lastGroup = item.group
			}

			// 커서 & 현재 뷰 여부 판단
			isCursor  := i == m.commandCursor
			isCurrent := item.cmd == viewNames[m.view]

			arrow := "  "
			var label, cmd string
			switch {
			case isCursor && isCurrent:
				arrow = navPickerCursorStyle.Render("> ")
				label = navActiveStyle.Bold(true).Render(item.name)
				cmd   = navCmdActiveStyle.Render("  " + item.cmd)
			case isCursor:
				arrow = navPickerCursorStyle.Render("> ")
				label = navPickerCursorStyle.Render(item.name)
				cmd   = navCmdActiveStyle.Render("  " + item.cmd)
			case isCurrent:
				arrow = "  "
				label = navActiveStyle.Render(item.name)
				cmd   = navCmdActiveStyle.Render("  " + item.cmd)
			default:
				arrow = "  "
				label = navItemStyle.Render(item.name)
				cmd   = navCmdStyle.Render("  " + item.cmd)
			}
			listLines = append(listLines, "    "+arrow+label+cmd)
		}
	}

	var b strings.Builder

	// 입력창 (상단)
	b.WriteString(navInputStyle.Render(": "+m.input.View()) + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("─", m.width)) + "\n")

	// 리스트
	for _, line := range listLines {
		b.WriteString(line + "\n")
	}

	// 남은 공간 채우기 (테이블 영역 유지)
	contentHeight := len(listLines) + 2 // input + separator
	for i := contentHeight; i < availableLines; i++ {
		b.WriteString("\n")
	}

	return b.String()
}
