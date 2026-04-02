package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	awsclient "github.com/jgulee/awsops/internal/aws"
)

var (
	detailTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).
				MarginBottom(1)
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).
			MarginTop(1)
	labelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(20)
	valueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	tagStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("180"))
	nameTagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114")) // 이름 힌트: 연두색
)

func renderDetail(inst *awsclient.Instance, vpcName, subnetName string, detailCursor int, hasHistory bool) string {
	if inst == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("EC2 › %s", nameOrID(inst))) + "\n")

	// General
	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", inst.Profile))
	b.WriteString(row("Instance ID", inst.InstanceID))
	b.WriteString(row("Name", orDash(inst.Name)))
	b.WriteString(row("State", coloredState(inst.State)))
	b.WriteString(row("Instance Type", inst.Type))
	b.WriteString(row("Launch Time", inst.LaunchTimeStr()))

	// Network
	b.WriteString(sectionStyle.Render("Network") + "\n")
	b.WriteString(row("Private IP", orDash(inst.PrivateIP)))
	b.WriteString(row("Public IP", orDash(inst.PublicIP)))
	b.WriteString(rowMaybeActive("VPC ID", withName(inst.VpcID, vpcName), detailCursor == 0))
	b.WriteString(rowMaybeActive("Subnet ID", withName(inst.SubnetID, subnetName), detailCursor == 1))
	b.WriteString(row("Availability Zone", orDash(inst.AvailabilityZone)))

	// Security Groups
	b.WriteString(sectionStyle.Render(fmt.Sprintf("Security Groups (%d)", len(inst.SecurityGroups))) + "\n")
	if len(inst.SecurityGroups) == 0 {
		b.WriteString(row("", "-"))
	} else {
		for i, sg := range inst.SecurityGroups {
			label := fmt.Sprintf("SG %d", i+1)
			b.WriteString(rowMaybeActive(label, withName(sg.ID, sg.Name), detailCursor == 2+i))
		}
	}

	// Configuration
	b.WriteString(sectionStyle.Render("Configuration") + "\n")
	b.WriteString(row("AMI ID", orDash(inst.AMIID)))
	b.WriteString(row("Key Name", orDash(inst.KeyName)))

	// Tags
	b.WriteString(sectionStyle.Render("Tags") + "\n")
	b.WriteString(renderTags(inst.Tags))

	var hint string
	switch {
	case detailCursor >= 0:
		hint = "esc  deselect    enter  open detail    j/k  scroll"
	case hasHistory:
		hint = "esc  back ◀    ↑/↓  select field    j/k  scroll"
	default:
		hint = "esc / q  back to list    ↑/↓  select field    j/k  scroll"
	}
	b.WriteString("\n" + helpStyle.Render(hint))

	return b.String()
}

func row(label, value string) string {
	return "  " + labelStyle.Render(label) + valueStyle.Render(value) + "\n"
}

// rowMaybeActive renders a row with a ▶ cursor indicator when active is true.
func rowMaybeActive(label, value string, active bool) string {
	if active {
		cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
		activeLabelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Width(20).Bold(true)
		return cursorStyle.Render("▶ ") + activeLabelStyle.Render(label) + value + "\n"
	}
	return row(label, value)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// withName renders an AWS resource ID with an optional name hint: "vpc-xxx  [my-vpc]"
func withName(id, name string) string {
	if id == "" {
		return "-"
	}
	if name != "" && name != id {
		return valueStyle.Render(id) + "  " + nameTagStyle.Render("["+name+"]")
	}
	return id
}

func nameOrID(inst *awsclient.Instance) string {
	if inst.Name != "" {
		return inst.Name
	}
	return inst.InstanceID
}

func renderSGDetail(sg *awsclient.SecurityGroup, vpcName string, sgNames map[string]string, enis []awsclient.ENI) string {
	if sg == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("SG › %s", sg.Name)) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", sg.Profile))
	b.WriteString(row("Group ID", sg.GroupID))
	b.WriteString(row("Name", sg.Name))
	b.WriteString(row("Description", orDash(sg.Description)))
	b.WriteString(row("VPC ID", withName(sg.VpcID, vpcName)))
	b.WriteString(row("Region", sg.Region))

	inbound := filterRules(sg.Rules, "inbound")
	outbound := filterRules(sg.Rules, "outbound")

	b.WriteString(sectionStyle.Render(fmt.Sprintf("Inbound Rules (%d)", len(inbound))) + "\n")
	if len(inbound) == 0 {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		b.WriteString(renderRules(inbound, sgNames))
	}

	b.WriteString(sectionStyle.Render(fmt.Sprintf("Outbound Rules (%d)", len(outbound))) + "\n")
	if len(outbound) == 0 {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		b.WriteString(renderRules(outbound, sgNames))
	}

	// Associated Resources (ENIs)
	b.WriteString(sectionStyle.Render(fmt.Sprintf("Associated Resources (%d)", len(enis))) + "\n")
	if len(enis) == 0 {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		b.WriteString(renderENIs(enis))
	}

	b.WriteString("\n" + helpStyle.Render("esc / q  back to list"))
	return b.String()
}

func filterRules(rules []awsclient.SGRule, direction string) []awsclient.SGRule {
	var filtered []awsclient.SGRule
	for _, r := range rules {
		if r.Direction == direction {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func renderRules(rules []awsclient.SGRule, sgNames map[string]string) string {
	ruleProtoStyle  := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Width(8)
	rulePortStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Width(12)
	ruleSourceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("180"))

	var b strings.Builder
	for _, r := range rules {
		source := ruleSourceStyle.Render(r.Source)
		if strings.HasPrefix(r.Source, "sg-") {
			if name, ok := sgNames[r.Source]; ok && name != "" {
				source = ruleSourceStyle.Render(r.Source) + "  " + nameTagStyle.Render("["+name+"]")
			}
		}
		b.WriteString("  " +
			ruleProtoStyle.Render(r.ProtocolStr()) +
			rulePortStyle.Render(r.PortRange()) +
			source + "\n")
	}
	return b.String()
}

func renderENIs(enis []awsclient.ENI) string {
	eniIDStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Width(24)
	eniMetaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	eniInstStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	var b strings.Builder
	for _, e := range enis {
		instPart := "-"
		if e.InstanceID != "" {
			instPart = e.InstanceID
		}
		desc := e.Description
		if desc == "" {
			desc = e.InterfaceType
		}
		meta := fmt.Sprintf("%s  %s  %s", instPart, e.PrivateIP, e.Status)
		b.WriteString("  " +
			eniIDStyle.Render(e.ENIID) +
			eniInstStyle.Render(meta))
		if desc != "" {
			b.WriteString("  " + eniMetaStyle.Render("("+desc+")"))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderVPCDetail(vpc *awsclient.VPC) string {
	if vpc == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("VPC › %s", orDash(vpc.Name))) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", vpc.Profile))
	b.WriteString(row("VPC ID", vpc.VpcID))
	b.WriteString(row("Name", orDash(vpc.Name)))
	b.WriteString(row("CIDR Block", vpc.CidrBlock))
	b.WriteString(row("State", vpc.State))
	b.WriteString(row("Default", fmt.Sprintf("%v", vpc.IsDefault)))
	b.WriteString(row("Region", vpc.Region))

	b.WriteString(sectionStyle.Render("Tags") + "\n")
	b.WriteString(renderTags(vpc.Tags))

	b.WriteString("\n" + helpStyle.Render("esc / q  back to list"))
	return b.String()
}

func renderSubnetDetail(subnet *awsclient.Subnet) string {
	if subnet == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("Subnet › %s", orDash(subnet.Name))) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", subnet.Profile))
	b.WriteString(row("Subnet ID", subnet.SubnetID))
	b.WriteString(row("Name", orDash(subnet.Name)))
	b.WriteString(row("VPC ID", subnet.VpcID))
	b.WriteString(row("CIDR Block", subnet.CidrBlock))
	b.WriteString(row("Availability Zone", subnet.AvailabilityZone))
	b.WriteString(row("Available IPs", fmt.Sprintf("%d", subnet.AvailableIPs)))
	b.WriteString(row("Default", fmt.Sprintf("%v", subnet.IsDefault)))
	b.WriteString(row("Region", subnet.Region))

	b.WriteString(sectionStyle.Render("Tags") + "\n")
	b.WriteString(renderTags(subnet.Tags))

	b.WriteString("\n" + helpStyle.Render("esc / q  back to list"))
	return b.String()
}

func renderTags(tags map[string]string) string {
	if len(tags) == 0 {
		return "  " + tagStyle.Render("-") + "\n"
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxKeyLen := 0
	for _, k := range keys {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}
	const maxTagKeyWidth = 36
	if maxKeyLen > maxTagKeyWidth {
		maxKeyLen = maxTagKeyWidth
	}
	tagLabelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(maxKeyLen + 2)

	var b strings.Builder
	for _, k := range keys {
		displayKey := k
		if len(k) > maxTagKeyWidth {
			displayKey = k[:maxTagKeyWidth-1] + "…"
		}
		b.WriteString("  " + tagLabelStyle.Render(displayKey) + valueStyle.Render(tags[k]) + "\n")
	}
	return b.String()
}

func renderTGWAttDetail(
	att *awsclient.TGWAttachment,
	associations []awsclient.TGWAssociation,
	routes []awsclient.TGWRoute,
	allAtts []awsclient.TGWAttachment,
	accountToProfile map[string]string,
	termWidth int,
) string {
	if att == nil {
		return ""
	}

	resolve := func(accountID string) string {
		if p, ok := accountToProfile[accountID]; ok {
			return fmt.Sprintf("%s (%s)", p, accountID)
		}
		return accountID
	}

	var b strings.Builder
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("TGW › %s", att.AttachmentID)) + "\n")

	b.WriteString(sectionStyle.Render("Attachment") + "\n")
	b.WriteString(row("Profile", att.Profile))
	b.WriteString(row("Attachment ID", att.AttachmentID))
	b.WriteString(row("TGW ID", att.TgwID))
	b.WriteString(row("TGW Owner", resolve(att.TgwOwnerID)))
	b.WriteString(row("Resource Type", att.ResourceType))
	b.WriteString(row("Resource ID", att.ResourceID))
	b.WriteString(row("Resource Owner", resolve(att.ResourceOwnerID)))
	b.WriteString(row("State", att.State))
	b.WriteString(row("Region", att.Region))

	// Associated route table
	b.WriteString(sectionStyle.Render("Route Table Association") + "\n")
	assocRTID := ""
	for _, a := range associations {
		if a.AttachmentID == att.AttachmentID {
			assocRTID = a.RouteTableID
			b.WriteString(row("Route Table", a.RouteTableID))
			b.WriteString(row("State", a.State))
			break
		}
	}
	if assocRTID == "" {
		b.WriteString("  " + tagStyle.Render("No route table association found") + "\n")
	}

	// Routes in the associated route table
	b.WriteString(sectionStyle.Render("Routes (reachable destinations)") + "\n")
	if assocRTID == "" {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		// Build attachment lookup: attachmentID → resourceID
		attResource := make(map[string]string)
		attOwner := make(map[string]string)
		for _, a := range allAtts {
			attResource[a.AttachmentID] = a.ResourceID
			attOwner[a.AttachmentID] = a.ResourceOwnerID
		}

		const cidrW, typeW, stateW, indent = 20, 12, 10, 4
		nextW := termWidth - cidrW - typeW - stateW - indent
		if nextW < 20 {
			nextW = 20
		}
		sepW := cidrW + nextW + typeW + stateW

		cidrStyle  := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Width(cidrW)
		nextStyle  := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Width(nextW)
		typeStyle  := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(typeW)
		stateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("180"))

		// Header
		b.WriteString("  " +
			cidrStyle.Render("Destination") +
			nextStyle.Render("Next Hop [Account]") +
			typeStyle.Render("Type") +
			stateStyle.Render("State") + "\n")
		b.WriteString("  " + strings.Repeat("─", sepW) + "\n")

		count := 0
		for _, r := range routes {
			if r.RouteTableID != assocRTID {
				continue
			}
			nextHop := r.ResourceID
			if r.AttachmentID != "" {
				// 맵에서 리소스 ID 조회 (크로스 계정 attachment는 없을 수 있음)
				if res, ok := attResource[r.AttachmentID]; ok && res != "" {
					nextHop = res
				}
				// 리소스 ID도 없으면 attachment ID 자체를 표시
				if nextHop == "" {
					nextHop = r.AttachmentID
				}
				// 소유 계정: 프로필명 우선, 없으면 account ID 그대로 표시
				if owner, ok := attOwner[r.AttachmentID]; ok && owner != "" {
					if profile, ok2 := accountToProfile[owner]; ok2 {
						nextHop = fmt.Sprintf("%s [%s]", nextHop, profile)
					} else {
						nextHop = fmt.Sprintf("%s [%s]", nextHop, owner)
					}
				}
			}
			if nextHop == "" {
				nextHop = "-"
			}
			stateColor := stateStyle
			if r.State == "blackhole" {
				stateColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
			} else if r.State == "active" {
				stateColor = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
			}
			b.WriteString("  " +
				cidrStyle.Render(r.DestinationCIDR) +
				nextStyle.Render(nextHop) +
				typeStyle.Render(r.RouteType) +
				stateColor.Render(r.State) + "\n")
			count++
		}
		if count == 0 {
			b.WriteString("  " + tagStyle.Render("No routes found") + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("esc / q  back to list"))
	return b.String()
}

// coloredState is used in the detail view only.
func coloredState(state string) string {
	switch state {
	case "stopped":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(state)
	case "pending", "stopping":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(state)
	default:
		return state
	}
}

