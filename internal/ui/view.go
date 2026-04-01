package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// --- styles ---

var (
	// 상단 헤더 바 (보라색 배경)
	headerBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	headerAppStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("226")).
			Bold(true)

	headerDimStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("189"))

	// 브레드크럼 바 (어두운 배경)
	crumbBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("255")).
			PaddingLeft(1).
			PaddingRight(1)

	crumbActiveStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("226")).
				Bold(true)

	crumbFilterStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("214"))

	// 하단 힌트 바
	hintBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("245")).
			PaddingLeft(1).
			PaddingRight(1)

	hintKeyStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("255")).
			Bold(true)

	// 입력 줄
	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			PaddingLeft(1)

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			PaddingLeft(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			PaddingLeft(1)
)

func (m Model) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Fetching AWS resources...", m.spinner.View())
	}

	if m.screen == screenDetail {
		return applyScroll(m.currentDetailContent(), m.detailScroll, m.height)
	}

	if m.screen == screenRegion {
		return renderRegionScreen(m.regions, m.regionCursor, m.width)
	}

	if m.screen == screenConnectivity {
		return applyScroll(renderConnectivityScreen(m), m.detailScroll, m.height)
	}

	var sections []string

	sections = append(sections, m.renderHeaderBar())
	sections = append(sections, m.renderCrumbBar())

	if m.mode == modeSearch || m.mode == modeCommand {
		sections = append(sections, m.renderInputLine())
	}

	if len(m.fetchErr) > 0 {
		sections = append(sections, errStyle.Render(fmt.Sprintf("⚠ %d profile(s) failed to load", len(m.fetchErr))))
	}

	sections = append(sections, m.table.View())
	sections = append(sections, m.renderHintBar())

	return strings.Join(sections, "\n")
}

func (m Model) renderHeaderBar() string {
	left := headerAppStyle.Render("awsops")

	regionIDs := selectedRegionIDs(m.regions)
	profileCount := fmt.Sprintf("%d profiles  %s", m.profileCount(), strings.Join(regionIDs, ", "))
	right := headerDimStyle.Render(profileCount)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	content := left + strings.Repeat(" ", gap) + right
	return headerBarStyle.Width(m.width).Render(content)
}

func (m Model) renderCrumbBar() string {
	// "ec2" 또는 "sg" 현재 뷰
	view := crumbActiveStyle.Render(viewNames[m.view])

	// 필터 표시
	filterPart := ""
	if len(m.filters) > 0 {
		filterPart = crumbBarStyle.Render(" › ") +
			crumbFilterStyle.Render("["+strings.Join(m.filters, " & ")+"]")
	}

	// 정렬 표시
	sortPart := ""
	if m.sortBy != sortNone {
		arrow := " ↑"
		if !m.sortAsc {
			arrow = " ↓"
		}
		sortPart = crumbBarStyle.Render("  ") +
			crumbFilterStyle.Render(sortColNames[m.sortBy]+arrow)
	}

	// 행 수
	rowCount := crumbBarStyle.Render(fmt.Sprintf("  (%d)", len(m.table.Rows())))

	content := view + filterPart + sortPart + rowCount
	padding := m.width - lipgloss.Width(content) - 1
	if padding < 0 {
		padding = 0
	}
	return crumbBarStyle.Width(m.width).Render(
		lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(content),
	)
}

func (m Model) renderInputLine() string {
	switch m.mode {
	case modeSearch:
		prefix := ""
		if len(m.filters) > 0 {
			prefix = "[" + strings.Join(m.filters, " & ") + "] & "
		}
		return inputStyle.Render("/ " + prefix + m.input.View())
	case modeCommand:
		inputLine := inputStyle.Render(": " + m.input.View())

		activeOpt := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).PaddingLeft(1).PaddingRight(1)
		dimOpt    := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(1).PaddingRight(1)
		sep       := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("│")

		views := []viewType{viewEC2, viewSG, viewVPC, viewSubnet, viewTGW}
		var parts []string
		for _, v := range views {
			name := viewNames[v]
			if v == m.view {
				parts = append(parts, activeOpt.Render(name))
			} else {
				parts = append(parts, dimOpt.Render(name))
			}
		}
		hintLine := "  " + strings.Join(parts, sep)
		return inputLine + "\n" + hintLine
	}
	return ""
}

func (m Model) renderHintBar() string {
	var hints []string
	hints = []string{
		hintItem("/", "Search"),
		hintItem(":", "View"),
		hintItem("d", "Describe"),
		hintItem("1-0", "Sort"),
		hintItem("r", "Refresh"),
		hintItem("R", "Regions"),
		hintItem("esc", "Clear"),
		hintItem("↑/↓", "Navigate"),
		hintItem("q", "Quit"),
	}
	if m.view == viewVPC {
		hints = append(hints, hintItem("c", "Check"))
	}
	content := strings.Join(hints, hintBarStyle.Render("  "))
	return hintBarStyle.Width(m.width).Render(content)
}

func (m Model) currentDetailContent() string {
	switch {
	case m.selectedSG != nil:
		return renderSGDetail(m.selectedSG)
	case m.selectedVPC != nil:
		return renderVPCDetail(m.selectedVPC)
	case m.selectedSubnet != nil:
		return renderSubnetDetail(m.selectedSubnet)
	case m.selectedTGWAtt != nil:
		return renderTGWAttDetail(m.selectedTGWAtt, m.tgwAssociations, m.tgwRoutes, m.tgwAttachments, m.accountToProfile, m.width)
	default:
		return renderDetail(m.selectedInst)
	}
}

func (m Model) detailMaxScroll() int {
	lines := strings.Count(m.currentDetailContent(), "\n") + 1
	max := lines - m.height
	if max < 0 {
		return 0
	}
	return max
}

// applyScroll slices a multi-line string to fit within the terminal height,
// starting from the given scroll offset. Prevents scrolling past the last line.
func applyScroll(content string, scroll, height int) string {
	lines := strings.Split(content, "\n")
	maxScroll := len(lines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[scroll:end], "\n")
}

func hintItem(key, action string) string {
	return hintKeyStyle.Render("<"+key+">") + hintBarStyle.Render(" "+action)
}

func (m Model) profileCount() int {
	seen := map[string]struct{}{}
	for _, inst := range m.instances {
		seen[inst.Profile] = struct{}{}
	}
	for _, sg := range m.groups {
		seen[sg.Profile] = struct{}{}
	}
	return len(seen)
}
