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
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(20)
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	tagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("180"))
)

func renderDetail(inst *awsclient.Instance) string {
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
	b.WriteString(row("VPC ID", orDash(inst.VpcID)))
	b.WriteString(row("Subnet ID", orDash(inst.SubnetID)))
	b.WriteString(row("Availability Zone", orDash(inst.AvailabilityZone)))

	// Configuration
	b.WriteString(sectionStyle.Render("Configuration") + "\n")
	b.WriteString(row("AMI ID", orDash(inst.AMIID)))
	b.WriteString(row("Key Name", orDash(inst.KeyName)))

	// Tags
	b.WriteString(sectionStyle.Render("Tags") + "\n")
	if len(inst.Tags) == 0 {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		keys := make([]string, 0, len(inst.Tags))
		for k := range inst.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// 키 최대 길이 계산 (최대 36자 제한)
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

		for _, k := range keys {
			displayKey := k
			if len(k) > maxTagKeyWidth {
				displayKey = k[:maxTagKeyWidth-1] + "…"
			}
			b.WriteString("  " + tagLabelStyle.Render(displayKey) + valueStyle.Render(inst.Tags[k]) + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("esc / q  back to list"))

	return b.String()
}

func row(label, value string) string {
	return "  " + labelStyle.Render(label) + valueStyle.Render(value) + "\n"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func nameOrID(inst *awsclient.Instance) string {
	if inst.Name != "" {
		return inst.Name
	}
	return inst.InstanceID
}

func renderSGDetail(sg *awsclient.SecurityGroup) string {
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
	b.WriteString(row("VPC ID", orDash(sg.VpcID)))
	b.WriteString(row("Region", sg.Region))

	inbound := filterRules(sg.Rules, "inbound")
	outbound := filterRules(sg.Rules, "outbound")

	b.WriteString(sectionStyle.Render(fmt.Sprintf("Inbound Rules (%d)", len(inbound))) + "\n")
	if len(inbound) == 0 {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		b.WriteString(renderRules(inbound))
	}

	b.WriteString(sectionStyle.Render(fmt.Sprintf("Outbound Rules (%d)", len(outbound))) + "\n")
	if len(outbound) == 0 {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		b.WriteString(renderRules(outbound))
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

func renderRules(rules []awsclient.SGRule) string {
	ruleProtoStyle  := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Width(8)
	rulePortStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Width(12)
	ruleSourceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("180"))

	var b strings.Builder
	for _, r := range rules {
		b.WriteString("  " +
			ruleProtoStyle.Render(r.ProtocolStr()) +
			rulePortStyle.Render(r.PortRange()) +
			ruleSourceStyle.Render(r.Source) + "\n")
	}
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

