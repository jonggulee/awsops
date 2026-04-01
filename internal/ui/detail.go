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
		for _, k := range keys {
			b.WriteString(row(k, inst.Tags[k]))
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

func coloredState(state string) string {
	switch state {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(state)
	case "stopped":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(state)
	case "pending", "stopping":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(state)
	default:
		return state
	}
}
