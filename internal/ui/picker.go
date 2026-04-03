package ui

import (
	"fmt"
	"sort"
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
	{"Instances", "ec2", "EC2"},
	{"Security Groups", "sg", "EC2"},
	{"Network Interfaces", "eni", "EC2"},
	{"Load Balancers", "elb", "EC2"},
	{"Your VPCs", "vpc", "VPC"},
	{"Subnets", "subnet", "VPC"},
	{"Transit Gateways", "tgw", "VPC"},
	{"Clusters", "eks", "Amazon Elastic Kubernetes Service"},
	{"DB Instances", "rds", "Aurora and RDS"},
	{"Certificates", "acm", "AWS Certificate Manager"},
	{"Records", "route53", "Route 53"},
}

// viewBreadcrumb maps a view to its "Service > Resource" label for the crumb bar.
var viewBreadcrumb = map[viewType]string{
	viewEC2:     "EC2  ›  Instances",
	viewSG:      "EC2  ›  Security Groups",
	viewENI:     "EC2  ›  Network Interfaces",
	viewVPC:     "VPC  ›  Your VPCs",
	viewSubnet:  "VPC  ›  Subnets",
	viewTGW:     "VPC  ›  Transit Gateways",
	viewEKS:     "Amazon Elastic Kubernetes Service  ›  Clusters",
	viewRDS:     "Aurora and RDS  ›  DB Instances",
	viewACM:     "AWS Certificate Manager  ›  Certificates",
	viewRoute53: "Route 53  ›  Records",
	viewALB:     "EC2  ›  Load Balancers",
}

var (
	navGroupStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Bold(true)
	navItemStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	navPickerCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	navActiveStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("78")) // 현재 뷰
	navInputStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).PaddingLeft(1)
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

// collectTagKeys returns sorted unique tag keys from the current view's resources.
func collectTagKeys(m Model) []string {
	seen := map[string]struct{}{}
	addKeys := func(tags map[string]string) {
		for k := range tags {
			seen[k] = struct{}{}
		}
	}
	switch m.view {
	case viewEC2:
		for _, inst := range m.instances {
			addKeys(inst.Tags)
		}
	case viewVPC:
		for _, v := range m.vpcs {
			addKeys(v.Tags)
		}
	case viewSubnet:
		for _, s := range m.subnets {
			addKeys(s.Tags)
		}
	case viewEKS:
		for _, c := range m.eksClusters {
			addKeys(c.Tags)
		}
	case viewALB:
		for _, lb := range m.loadBalancers {
			addKeys(lb.Tags)
		}
	case viewRDS:
		for _, db := range m.rdsInstances {
			addKeys(db.Tags)
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// collectTagValues returns sorted unique non-empty values for the given tag key in the current view.
func collectTagValues(m Model, key string) []string {
	seen := map[string]struct{}{}
	addVal := func(tags map[string]string) {
		if v, ok := tags[key]; ok && v != "" {
			seen[v] = struct{}{}
		}
	}
	switch m.view {
	case viewEC2:
		for _, inst := range m.instances {
			addVal(inst.Tags)
		}
	case viewVPC:
		for _, v := range m.vpcs {
			addVal(v.Tags)
		}
	case viewSubnet:
		for _, s := range m.subnets {
			addVal(s.Tags)
		}
	case viewEKS:
		for _, c := range m.eksClusters {
			addVal(c.Tags)
		}
	case viewALB:
		for _, lb := range m.loadBalancers {
			addVal(lb.Tags)
		}
	case viewRDS:
		for _, db := range m.rdsInstances {
			addVal(db.Tags)
		}
	}
	vals := make([]string, 0, len(seen))
	for v := range seen {
		vals = append(vals, v)
	}
	sort.Strings(vals)
	return vals
}

// renderTagPicker renders the 2-step tag key→value picker.
func renderTagPicker(m Model) string {
	query := strings.ToLower(m.input.Value())

	// header(1) + crumb(1) + input(1) + sep(1) + hint(1) = 5
	availableLines := m.height - 5
	if availableLines < 1 {
		availableLines = 1
	}

	var b strings.Builder
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("─", m.width))

	var items []string
	var header string

	if m.tagPickerStep == 0 {
		for _, k := range collectTagKeys(m) {
			if query == "" || strings.Contains(strings.ToLower(k), query) {
				items = append(items, k)
			}
		}
		header = navInputStyle.Render(": tag  ›  key    "+m.input.View()) + "\n" + sep
	} else {
		items = append(items, "(any)")
		for _, v := range collectTagValues(m, m.tagPickerKey) {
			if query == "" || strings.Contains(strings.ToLower(v), query) {
				items = append(items, v)
			}
		}
		keyLabel := navActiveStyle.Render(m.tagPickerKey)
		header = navInputStyle.Render(": tag  ›  "+keyLabel+"  =    "+m.input.View()) + "\n" + sep
	}

	b.WriteString(header + "\n")

	if len(items) == 0 {
		msg := "No tags found in current view"
		if query != "" {
			msg = "No matching items"
		}
		b.WriteString(navNoMatchStyle.Render(msg) + "\n")
		return b.String()
	}

	// 커서가 항상 화면 안에 오도록 슬라이딩 윈도우 계산
	start := m.tagPickerCursor - availableLines/2
	if start < 0 {
		start = 0
	}
	end := start + availableLines
	if end > len(items) {
		end = len(items)
		start = end - availableLines
		if start < 0 {
			start = 0
		}
	}

	// 위에 숨겨진 항목 수 표시
	if start > 0 {
		b.WriteString(navCmdStyle.Render(fmt.Sprintf("      ▲ %d more", start)) + "\n")
	}

	for i := start; i < end; i++ {
		v := items[i]
		var label string
		if v == "(any)" {
			label = navCmdActiveStyle.Render("(any)") + navCmdStyle.Render("  — match any resource with this tag key")
		} else {
			label = navItemStyle.Render(v)
		}
		if i == m.tagPickerCursor {
			b.WriteString("    " + navPickerCursorStyle.Render("> ") + label + "\n")
		} else {
			b.WriteString("      " + label + "\n")
		}
	}

	// 아래에 숨겨진 항목 수 표시
	if end < len(items) {
		b.WriteString(navCmdStyle.Render(fmt.Sprintf("      ▼ %d more", len(items)-end)) + "\n")
	}

	return b.String()
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
			isCursor := i == m.commandCursor
			isCurrent := item.cmd == viewNames[m.view]

			arrow := "  "
			var label, cmd string
			switch {
			case isCursor && isCurrent:
				arrow = navPickerCursorStyle.Render("> ")
				label = navActiveStyle.Bold(true).Render(item.name)
				cmd = navCmdActiveStyle.Render("  " + item.cmd)
			case isCursor:
				arrow = navPickerCursorStyle.Render("> ")
				label = navPickerCursorStyle.Render(item.name)
				cmd = navCmdActiveStyle.Render("  " + item.cmd)
			case isCurrent:
				arrow = "  "
				label = navActiveStyle.Render(item.name)
				cmd = navCmdActiveStyle.Render("  " + item.cmd)
			default:
				arrow = "  "
				label = navItemStyle.Render(item.name)
				cmd = navCmdStyle.Render("  " + item.cmd)
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
