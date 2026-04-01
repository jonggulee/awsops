package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	awsclient "github.com/jgulee/awsops/internal/aws"
)

var (
	connTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	connSrcStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	connOKStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	connFailStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	connUnknStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	connStepLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(38)
	connDetailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	pickerCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	pickerDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	pickerNameStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Width(24)
	pickerIDStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(24)
	pickerCIDRStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("180"))
)

func renderConnectivityScreen(m Model) string {
	if m.connectivityResult != nil {
		return renderConnectivityResult(m)
	}
	return renderConnectivityPicker(m)
}

func renderConnectivityPicker(m Model) string {
	var b strings.Builder

	b.WriteString(connTitleStyle.Render("Connectivity Check") + "\n\n")

	// 소스 VPC 정보
	if src := m.connectivitySrcVPC; src != nil {
		b.WriteString(connSrcStyle.Render("  From: ") +
			valueStyle.Render(orDash(src.Name)) + "  " +
			connDetailStyle.Render(fmt.Sprintf("(%s)  %s  %s", src.Profile, src.VpcID, src.CidrBlock)) + "\n")
	}

	// 필터 입력
	b.WriteString("\n")
	b.WriteString(inputStyle.Render("/ " + m.input.View()))
	b.WriteString("\n\n")

	// 동적 profile 너비
	pw := m.maxProfileWidth()
	profStyle     := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Width(pw)
	profStyleSel  := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Width(pw)
	sepW := pw + 24 + 24 + 16 + 2

	// VPC 목록 헤더
	b.WriteString("  " +
		pickerDimStyle.Render(fmt.Sprintf("  %-*s %-24s %-24s %s", pw, "Profile", "Name", "VPC ID", "CIDR")) + "\n")
	b.WriteString("  " + pickerDimStyle.Render(strings.Repeat("─", sepW)) + "\n")

	vpcs := m.connectivityPickerVPCs()
	if len(vpcs) == 0 {
		b.WriteString(connDetailStyle.Render("  No VPCs found") + "\n")
	} else {
		for i, v := range vpcs {
			cursor := "  "
			ps      := profStyle
			nameStyle := pickerNameStyle
			idStyle   := pickerIDStyle

			if i == m.connectivityCursor {
				cursor    = pickerCursorStyle.Render("▶ ")
				ps        = profStyleSel
				nameStyle = pickerNameStyle.Foreground(lipgloss.Color("226"))
				idStyle   = pickerIDStyle.Foreground(lipgloss.Color("226"))
			}

			b.WriteString(cursor +
				ps.Render(v.Profile) +
				nameStyle.Render(orDash(v.Name)) +
				idStyle.Render(v.VpcID) +
				pickerCIDRStyle.Render(v.CidrBlock) + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("↑/↓: navigate  enter: check  esc: back"))
	return b.String()
}

func renderConnectivityResult(m Model) string {
	var b strings.Builder
	res := m.connectivityResult

	b.WriteString(connTitleStyle.Render("Connectivity Check › Result") + "\n\n")

	// 소스 → 목적지 요약
	srcName := res.SrcVpcID
	if m.connectivitySrcVPC != nil {
		srcName = fmt.Sprintf("%s (%s)", orDash(m.connectivitySrcVPC.Name), m.connectivitySrcVPC.Profile)
	}
	dstName := res.DstVpcID
	for _, v := range m.vpcs {
		if v.VpcID == res.DstVpcID {
			dstName = fmt.Sprintf("%s (%s)", orDash(v.Name), v.Profile)
			break
		}
	}

	b.WriteString(connSrcStyle.Render("  From: ") + valueStyle.Render(srcName) +
		connDetailStyle.Render("  "+res.SrcCIDR) + "\n")
	b.WriteString(connSrcStyle.Render("  To:   ") + valueStyle.Render(dstName) +
		connDetailStyle.Render("  "+res.DstCIDR) + "\n\n")

	// 단계별 결과
	// label 너비를 실제 step description 중 가장 긴 것 기준으로 동적 계산
	maxLabelLen := 0
	for _, step := range res.Steps {
		l := len(fmt.Sprintf("Step %d  %s", step.Step, step.Description))
		if l > maxLabelLen {
			maxLabelLen = l
		}
	}
	labelW := maxLabelLen + 2 // 여유 2칸
	dynLabel := connStepLabel.Width(labelW)
	// "  " + "✓" + "  " = 5자 고정
	fixedW := 5 + labelW
	detailW := m.width - fixedW
	if detailW < 20 {
		detailW = 20
	}

	for _, step := range res.Steps {
		icon, iconStyle := stepIcon(step.Status)
		label := dynLabel.Render(fmt.Sprintf("Step %d  %s", step.Step, step.Description))
		prefix := "  " + iconStyle.Render(icon) + "  " + label

		if len(step.Detail) <= detailW {
			b.WriteString(prefix + connDetailStyle.Render(step.Detail) + "\n")
		} else {
			b.WriteString(prefix + "\n")
			b.WriteString(strings.Repeat(" ", fixedW) + connDetailStyle.Render(step.Detail) + "\n")
		}
	}

	// 최종 결과
	b.WriteString("\n  " + strings.Repeat("─", 64) + "\n  ")

	hasUnknown := false
	for _, s := range res.Steps {
		if s.Status == awsclient.StatusUnknown {
			hasUnknown = true
			break
		}
	}

	if res.Reachable {
		b.WriteString(connOKStyle.Render("✓ REACHABLE"))
	} else if hasUnknown {
		b.WriteString(connUnknStyle.Render("? UNCERTAIN") +
			connDetailStyle.Render("  (some route tables not accessible)"))
	} else {
		b.WriteString(connFailStyle.Render("✗ NOT REACHABLE"))
	}
	b.WriteString("\n")

	b.WriteString("\n" + helpStyle.Render("↑/↓/pgup/pgdown: scroll  b/esc: back to picker"))
	return b.String()
}

func stepIcon(status awsclient.CheckStatus) (string, lipgloss.Style) {
	switch status {
	case awsclient.StatusOK:
		return "✓", connOKStyle
	case awsclient.StatusFail:
		return "✗", connFailStyle
	default:
		return "?", connUnknStyle
	}
}
