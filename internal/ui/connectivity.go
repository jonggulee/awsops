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
	pickerAZStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
)

func renderConnectivityScreen(m Model) string {
	if m.connectivityResult != nil {
		return renderConnectivityResult(m)
	}
	return renderConnectivityPicker(m)
}

func renderConnectivityPicker(m Model) string {
	if m.connectivitySelectedRoute == nil {
		return renderRoutePicker(m)
	}
	return renderSubnetPicker(m)
}

// renderRoutePicker shows phase 1: TGW routes from the source subnet's route table.
func renderRoutePicker(m Model) string {
	var header strings.Builder

	header.WriteString(connTitleStyle.Render("Connectivity Check  ›  Select Route") + "\n\n")

	if src := m.connectivitySrcSubnet; src != nil {
		vpcName := src.VpcID
		for _, v := range m.vpcs {
			if v.VpcID == src.VpcID && v.Name != "" {
				vpcName = v.Name
				break
			}
		}
		header.WriteString(connSrcStyle.Render("  From: ") +
			valueStyle.Render(orDash(src.Name)) + "  " +
			connDetailStyle.Render(fmt.Sprintf("(%s)  %s  %s  vpc: %s  %s",
				src.Profile, src.SubnetID, src.CidrBlock, vpcName, src.AvailabilityZone)) + "\n")
	}

	header.WriteString("\n")
	header.WriteString(inputStyle.Render("/ " + m.input.View()))
	header.WriteString("\n\n")

	const cidrW = 20
	const gwW = 22
	const rtW = 22
	const typeW = 10
	const cursorW = 2
	sepW := cidrW + gwW + rtW + typeW + cursorW

	header.WriteString("  " +
		pickerDimStyle.Render(fmt.Sprintf("  %-*s %-*s %-*s %s",
			cidrW, "Destination CIDR", gwW, "Via (TGW)", rtW, "Route Table", "Type")) + "\n")
	header.WriteString("  " + pickerDimStyle.Render(strings.Repeat("─", sepW)) + "\n")

	headerStr := header.String()

	headerLines := strings.Count(headerStr, "\n")
	const footerLines = 2
	visibleItems := m.height - headerLines - footerLines
	if visibleItems < 1 {
		visibleItems = 1
	}

	routes := m.connectivityPickerRoutes()

	listOffset := 0
	if m.connectivityCursor >= visibleItems {
		listOffset = m.connectivityCursor - visibleItems + 1
	}

	var b strings.Builder
	b.WriteString(headerStr)

	if len(routes) == 0 {
		b.WriteString(connDetailStyle.Render("  No TGW routes found in route table") + "\n")
	} else {
		end := listOffset + visibleItems
		if end > len(routes) {
			end = len(routes)
		}
		cidrStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Width(cidrW)
		gwStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("180")).Width(gwW)
		rtStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(rtW)

		for i := listOffset; i < end; i++ {
			r := routes[i]
			cursor := "  "
			cs := cidrStyle
			if i == m.connectivityCursor {
				cursor = pickerCursorStyle.Render("▶ ")
				cs = cidrStyle.Foreground(lipgloss.Color("226")).Bold(true)
			}
			tableNote := "main"
			if r.IsExplicit {
				tableNote = "explicit"
			}
			b.WriteString(cursor +
				cs.Render(r.DestinationCIDR) +
				gwStyle.Render(r.GatewayID) +
				rtStyle.Render(r.RouteTableID) +
				pickerDimStyle.Render(tableNote) + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("↑/↓/pgup/pgdown: navigate  enter: select route  esc: back"))
	return b.String()
}

// renderSubnetPicker shows phase 2: destination subnets covered by the selected route's CIDR.
func renderSubnetPicker(m Model) string {
	var header strings.Builder

	header.WriteString(connTitleStyle.Render("Connectivity Check  ›  Select Destination Subnet") + "\n\n")

	if src := m.connectivitySrcSubnet; src != nil {
		vpcName := src.VpcID
		for _, v := range m.vpcs {
			if v.VpcID == src.VpcID && v.Name != "" {
				vpcName = v.Name
				break
			}
		}
		header.WriteString(connSrcStyle.Render("  From: ") +
			valueStyle.Render(orDash(src.Name)) + "  " +
			connDetailStyle.Render(fmt.Sprintf("(%s)  %s  vpc: %s", src.Profile, src.SubnetID, vpcName)) + "\n")
	}

	if sel := m.connectivitySelectedRoute; sel != nil {
		tableNote := "main"
		if sel.IsExplicit {
			tableNote = "explicit"
		}
		header.WriteString(connSrcStyle.Render("  Route: ") +
			valueStyle.Render(sel.DestinationCIDR) + "  " +
			connDetailStyle.Render(fmt.Sprintf("→ %s  [%s: %s]", sel.GatewayID, tableNote, sel.RouteTableID)) + "\n")
	}

	header.WriteString("\n")
	header.WriteString(inputStyle.Render("/ " + m.input.View()))
	header.WriteString("\n\n")

	pw := m.maxProfileWidth()
	const subnetIDW = 24
	const colGap = 2 // SubnetID ↔ CIDR 사이 여백
	const cidrW = 18
	const azW = 20
	const cursorW = 2
	const margin = 4
	nameW := m.width - pw - subnetIDW - colGap - cidrW - azW - cursorW - margin
	if nameW < 14 {
		nameW = 14
	}

	profStyle    := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Width(pw)
	profStyleSel := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Width(pw)
	dynName      := pickerNameStyle.Width(nameW)
	dynID        := pickerIDStyle.Width(subnetIDW)
	sepW := pw + nameW + subnetIDW + colGap + cidrW + azW + cursorW

	header.WriteString("  " +
		pickerDimStyle.Render(fmt.Sprintf("  %-*s %-*s %-*s %*s%-*s %s",
			pw, "Profile", nameW, "Name", subnetIDW, "Subnet ID", colGap, "", cidrW, "CIDR", "AZ")) + "\n")
	header.WriteString("  " + pickerDimStyle.Render(strings.Repeat("─", sepW)) + "\n")

	headerStr := header.String()

	headerLines := strings.Count(headerStr, "\n")
	const footerLines = 2
	visibleItems := m.height - headerLines - footerLines
	if visibleItems < 1 {
		visibleItems = 1
	}

	subnets := m.connectivityPickerSubnets()

	listOffset := 0
	if m.connectivityCursor >= visibleItems {
		listOffset = m.connectivityCursor - visibleItems + 1
	}

	var b strings.Builder
	b.WriteString(headerStr)

	if len(subnets) == 0 {
		b.WriteString(connDetailStyle.Render("  No subnets found in this CIDR range") + "\n")
	} else {
		end := listOffset + visibleItems
		if end > len(subnets) {
			end = len(subnets)
		}
		for i := listOffset; i < end; i++ {
			s := subnets[i]
			cursor := "  "
			ps        := profStyle
			nameStyle := dynName
			idStyle   := dynID

			if i == m.connectivityCursor {
				cursor    = pickerCursorStyle.Render("▶ ")
				ps        = profStyleSel
				nameStyle = dynName.Foreground(lipgloss.Color("226"))
				idStyle   = dynID.Foreground(lipgloss.Color("226"))
			}

			b.WriteString(cursor +
				ps.Render(s.Profile) +
				nameStyle.Render(orDash(s.Name)) +
				idStyle.Render(s.SubnetID) +
				strings.Repeat(" ", colGap) +
				pickerCIDRStyle.Render(fmt.Sprintf("%-*s", cidrW, s.CidrBlock)) +
				pickerAZStyle.Render(s.AvailabilityZone) + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("↑/↓/pgup/pgdown: navigate  enter: check  esc: back to routes"))
	return b.String()
}

func renderConnectivityResult(m Model) string {
	var b strings.Builder
	res := m.connectivityResult

	b.WriteString(connTitleStyle.Render("Connectivity Check › Result") + "\n\n")

	// 소스 → 목적지 요약 (서브넷 기준)
	srcName := res.SrcSubnetID
	srcVpcName := res.SrcVpcID
	if m.connectivitySrcSubnet != nil {
		if m.connectivitySrcSubnet.Name != "" {
			srcName = m.connectivitySrcSubnet.Name
		}
		for _, v := range m.vpcs {
			if v.VpcID == res.SrcVpcID {
				if v.Name != "" {
					srcVpcName = v.Name
				}
				break
			}
		}
	}

	dstName := res.DstSubnetID
	dstVpcName := res.DstVpcID
	for _, s := range m.subnets {
		if s.SubnetID == res.DstSubnetID {
			if s.Name != "" {
				dstName = s.Name
			}
			break
		}
	}
	for _, v := range m.vpcs {
		if v.VpcID == res.DstVpcID {
			if v.Name != "" {
				dstVpcName = v.Name
			}
			break
		}
	}

	b.WriteString(connSrcStyle.Render("  From: ") +
		valueStyle.Render(srcName) +
		connDetailStyle.Render(fmt.Sprintf("  %s    vpc: %s  %s", res.SrcSubnetID, srcVpcName, res.SrcVpcCIDR)) + "\n")
	b.WriteString(connSrcStyle.Render("  To:   ") +
		valueStyle.Render(dstName) +
		connDetailStyle.Render(fmt.Sprintf("  %s    vpc: %s  %s", res.DstSubnetID, dstVpcName, res.DstVpcCIDR)) + "\n\n")

	// 단계별 결과
	maxLabelLen := 0
	for _, step := range res.Steps {
		l := len(fmt.Sprintf("Step %d  %s", step.Step, step.Description))
		if l > maxLabelLen {
			maxLabelLen = l
		}
	}
	labelW := maxLabelLen + 2
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
