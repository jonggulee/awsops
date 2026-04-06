package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

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

// profileWithAccount renders a profile name with a dimmed account ID hint.
func profileWithAccount(profile string, profileToAccount map[string]string) string {
	id, ok := profileToAccount[profile]
	if !ok || id == "" {
		return profile
	}
	return profile + "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("["+id+"]")
}

// detailHintBar renders a styled hint bar (same visual as main hintBar) for detail screens.
func detailHintBar(width int, items ...string) string {
	content := strings.Join(items, hintBarStyle.Render("  "))
	return "\n" + hintBarStyle.Width(width).Render(content)
}

func renderDetail(inst *awsclient.Instance, vpcName, subnetName string, detailCursor int, hasHistory bool, typeSpecs map[string]awsclient.InstanceTypeSpec, profileToAccount map[string]string, width int) string {
	if inst == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("EC2 › %s", nameOrID(inst))) + "\n")

	// General
	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", profileWithAccount(inst.Profile, profileToAccount)))
	b.WriteString(row("Instance ID", inst.InstanceID))
	b.WriteString(row("Name", orDash(inst.Name)))
	b.WriteString(row("State", coloredState(inst.State)))
	instTypeVal := inst.Type
	if spec, ok := typeSpecs[inst.Type]; ok {
		mem := spec.MemoryGiB
		var memStr string
		if mem == float64(int(mem)) {
			memStr = fmt.Sprintf("%d GiB", int(mem))
		} else {
			memStr = fmt.Sprintf("%.1f GiB", mem)
		}
		hint := fmt.Sprintf("%d vCPU, %s", spec.VCPU, memStr)
		instTypeVal += "  " + nameTagStyle.Render("["+hint+"]")
	}
	b.WriteString(row("Instance Type", instTypeVal))
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
	b.WriteString(renderTags(inst.Tags, width))

	var hint string
	switch {
	case detailCursor >= 0:
		hint = detailHintBar(width, hintItem("esc", "Deselect"), hintItem("enter", "Open"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	case hasHistory:
		hint = detailHintBar(width, hintItem("esc", "Back ◀"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	default:
		hint = detailHintBar(width, hintItem("esc/q", "Back to list"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	}
	b.WriteString(hint)

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

func renderSGDetail(sg *awsclient.SecurityGroup, vpcName string, sgNames map[string]string, enis []awsclient.ENI, profileToAccount map[string]string, width int) string {
	if sg == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("SG › %s", sg.Name)) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", profileWithAccount(sg.Profile, profileToAccount)))
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

	b.WriteString(detailHintBar(width, hintItem("esc/q", "Back to list")))
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

func renderVPCDetail(vpc *awsclient.VPC, profileToAccount map[string]string, width int) string {
	if vpc == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("VPC › %s", orDash(vpc.Name))) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", profileWithAccount(vpc.Profile, profileToAccount)))
	b.WriteString(row("VPC ID", vpc.VpcID))
	b.WriteString(row("Name", orDash(vpc.Name)))
	b.WriteString(row("CIDR Block", vpc.CidrBlock))
	b.WriteString(row("State", vpc.State))
	b.WriteString(row("Default", fmt.Sprintf("%v", vpc.IsDefault)))
	b.WriteString(row("Region", vpc.Region))

	b.WriteString(sectionStyle.Render("Tags") + "\n")
	b.WriteString(renderTags(vpc.Tags, width))

	b.WriteString(detailHintBar(width, hintItem("esc/q", "Back to list")))
	return b.String()
}

func renderSubnetDetail(subnet *awsclient.Subnet, profileToAccount map[string]string, width int) string {
	if subnet == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("Subnet › %s", orDash(subnet.Name))) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", profileWithAccount(subnet.Profile, profileToAccount)))
	b.WriteString(row("Subnet ID", subnet.SubnetID))
	b.WriteString(row("Name", orDash(subnet.Name)))
	b.WriteString(row("VPC ID", subnet.VpcID))
	b.WriteString(row("CIDR Block", subnet.CidrBlock))
	b.WriteString(row("Availability Zone", subnet.AvailabilityZone))
	b.WriteString(row("Available IPs", fmt.Sprintf("%d", subnet.AvailableIPs)))
	b.WriteString(row("Default", fmt.Sprintf("%v", subnet.IsDefault)))
	b.WriteString(row("Region", subnet.Region))

	b.WriteString(sectionStyle.Render("Tags") + "\n")
	b.WriteString(renderTags(subnet.Tags, width))

	b.WriteString(detailHintBar(width, hintItem("esc/q", "Back to list")))
	return b.String()
}

func renderENIDetail(eni *awsclient.ENI, vpcName, subnetName string, sgNames map[string]string, profileToAccount map[string]string, width int) string {
	if eni == nil {
		return ""
	}
	var b strings.Builder

	title := eni.ENIID
	if eni.Name != "" {
		title = eni.Name + "  " + nameTagStyle.Render("["+eni.ENIID+"]")
	}
	b.WriteString(detailTitleStyle.Render("EC2  ›  Network Interfaces  ›  "+title) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile",    profileWithAccount(eni.Profile, profileToAccount)))
	b.WriteString(row("ENI ID",     eni.ENIID))
	if eni.Name != "" {
		b.WriteString(row("Name", eni.Name))
	}
	b.WriteString(row("Description", orDash(eni.Description)))
	b.WriteString(row("Status",      eni.Status))
	b.WriteString(row("Type",        eni.InterfaceType))
	b.WriteString(row("Region",      eni.Region))

	b.WriteString(sectionStyle.Render("Network") + "\n")
	b.WriteString(row("Private IP", orDash(eni.PrivateIP)))
	b.WriteString(row("VPC ID",     withName(eni.VpcID, vpcName)))
	b.WriteString(row("Subnet ID",  withName(eni.SubnetID, subnetName)))

	b.WriteString(sectionStyle.Render("Attached Resource") + "\n")
	if eni.InstanceID != "" {
		b.WriteString(row("Instance ID", eni.InstanceID))
	} else {
		b.WriteString("  " + tagStyle.Render("Not attached") + "\n")
	}

	b.WriteString(sectionStyle.Render(fmt.Sprintf("Security Groups (%d)", len(eni.SecurityGroupIDs))) + "\n")
	for _, sgID := range eni.SecurityGroupIDs {
		name := sgNames[sgID]
		b.WriteString("  " + valueStyle.Render(withName(sgID, name)) + "\n")
	}

	b.WriteString(detailHintBar(width, hintItem("esc/q", "Back to list")))
	return b.String()
}

func renderCertDetail(cert *awsclient.Certificate, profileToAccount map[string]string, width int) string {
	if cert == nil {
		return ""
	}
	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("ACM › %s", cert.DomainName)) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", profileWithAccount(cert.Profile, profileToAccount)))
	b.WriteString(row("Domain Name", cert.DomainName))
	b.WriteString(row("ARN", cert.ARN))
	b.WriteString(row("Status", coloredCertStatus(cert.Status)))
	b.WriteString(row("Type", cert.Type))
	b.WriteString(row("Key Algorithm", orDash(cert.KeyAlgorithm)))
	b.WriteString(row("Region", cert.Region))

	b.WriteString(sectionStyle.Render("Validity") + "\n")
	notBefore := "-"
	if !cert.NotBefore.IsZero() {
		notBefore = cert.NotBefore.In(time.Local).Format("2006-01-02")
	}
	b.WriteString(row("Not Before", notBefore))
	b.WriteString(row("Not After (Expiry)", cert.ExpiryStr()))

	b.WriteString(sectionStyle.Render(fmt.Sprintf("Subject Alternative Names (%d)", len(cert.SANs))) + "\n")
	if len(cert.SANs) == 0 {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		for _, san := range cert.SANs {
			b.WriteString("  " + valueStyle.Render(san) + "\n")
		}
	}

	b.WriteString(sectionStyle.Render(fmt.Sprintf("In Use By (%d)", len(cert.InUseBy))) + "\n")
	if len(cert.InUseBy) == 0 {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		for _, arn := range cert.InUseBy {
			b.WriteString("  " + valueStyle.Render(arn) + "\n")
		}
	}

	b.WriteString(detailHintBar(width, hintItem("esc/q", "Back to list")))
	return b.String()
}

func coloredCertStatus(status string) string {
	switch status {
	case "ISSUED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(status)
	case "EXPIRED", "REVOKED", "FAILED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(status)
	case "PENDING_VALIDATION":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render(status)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(status)
	}
}

func renderTags(tags map[string]string, width int) string {
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

	// 2(indent) + (maxKeyLen+2)(label) + value 로 width를 초과하지 않도록 동적으로 자름
	maxValLen := width - 2 - (maxKeyLen + 2)
	if maxValLen < 20 {
		maxValLen = 20
	}

	var b strings.Builder
	for _, k := range keys {
		displayKey := k
		if len(k) > maxTagKeyWidth {
			displayKey = k[:maxTagKeyWidth-1] + "…"
		}
		val := tags[k]
		if len(val) > maxValLen {
			val = val[:maxValLen-3] + "..."
		}
		b.WriteString("  " + tagLabelStyle.Render(displayKey) + valueStyle.Render(val) + "\n")
	}
	return b.String()
}

func renderTGWAttDetail(
	att *awsclient.TGWAttachment,
	associations []awsclient.TGWAssociation,
	routes []awsclient.TGWRoute,
	allAtts []awsclient.TGWAttachment,
	accountToProfile map[string]string,
	profileToAccount map[string]string,
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
	b.WriteString(row("Profile", profileWithAccount(att.Profile, profileToAccount)))
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

	b.WriteString(detailHintBar(termWidth, hintItem("esc/q", "Back to list")))
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


func renderEKSDetail(cluster *awsclient.EKSCluster, vpcName string, subnetNames map[string]string, sgNames map[string]string, detailCursor int, hasHistory bool, profileToAccount map[string]string, width int) string {
	if cluster == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("EKS › %s", cluster.Name)) + "\n")

	// General
	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", profileWithAccount(cluster.Profile, profileToAccount)))
	b.WriteString(row("Name", cluster.Name))
	b.WriteString(row("Status", coloredEKSStatus(cluster.Status)))
	b.WriteString(row("Version", cluster.Version))
	b.WriteString(row("Platform Version", orDash(cluster.PlatformVersion)))
	b.WriteString(row("Created At", cluster.CreatedAtStr()))

	// Network
	b.WriteString(sectionStyle.Render("Network") + "\n")
	b.WriteString(row("VPC ID", withName(cluster.VpcID, vpcName)))
	b.WriteString(row("Cluster SG", orDash(cluster.ClusterSecurityGroupID)))

	accessParts := []string{}
	if cluster.PublicAccess {
		accessParts = append(accessParts, "Public")
	}
	if cluster.PrivateAccess {
		accessParts = append(accessParts, "Private")
	}
	access := "-"
	if len(accessParts) > 0 {
		access = strings.Join(accessParts, " + ")
	}
	b.WriteString(row("API Access", access))
	b.WriteString(row("Endpoint", orDash(cluster.Endpoint)))

	nodeCount := len(cluster.Nodes)

	if len(cluster.SubnetIDs) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Subnets (%d)", len(cluster.SubnetIDs))) + "\n")
		for i, id := range cluster.SubnetIDs {
			idx := nodeCount + i
			b.WriteString(rowMaybeActive(fmt.Sprintf("Subnet %d", i+1), withName(id, subnetNames[id]), detailCursor == idx))
		}
	}

	if len(cluster.SecurityGroupIDs) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Security Groups (%d)", len(cluster.SecurityGroupIDs))) + "\n")
		for i, id := range cluster.SecurityGroupIDs {
			idx := nodeCount + len(cluster.SubnetIDs) + i
			name := sgNames[id]
			b.WriteString(rowMaybeActive("SG "+fmt.Sprintf("%d", i+1), withName(id, name), detailCursor == idx))
		}
	}

	// IAM
	b.WriteString(sectionStyle.Render("IAM") + "\n")
	b.WriteString(row("Role ARN", orDash(cluster.RoleARN)))

	// Nodes (EC2 instances)
	b.WriteString(sectionStyle.Render(fmt.Sprintf("Nodes (%d)", len(cluster.Nodes))) + "\n")
	if len(cluster.Nodes) == 0 {
		b.WriteString("  " + tagStyle.Render("No nodes found") + "\n")
	} else {
		// 노드 커서: detailCursor가 노드 범위일 때만 활성화
		nodeCursor := -1
		if detailCursor >= 0 && detailCursor < nodeCount {
			nodeCursor = detailCursor
		}
		b.WriteString(renderNodeTable(cluster.Nodes, nodeCursor))
	}

	// Node Groups
	b.WriteString(sectionStyle.Render(fmt.Sprintf("Node Groups (%d)", len(cluster.Nodegroups))) + "\n")
	if len(cluster.Nodegroups) == 0 {
		b.WriteString("  " + tagStyle.Render("No node groups") + "\n")
	} else {
		for _, ng := range cluster.Nodegroups {
			b.WriteString(renderNodegroupBlock(ng))
		}
	}

	// Tags
	b.WriteString(sectionStyle.Render("Tags") + "\n")
	b.WriteString(renderTags(cluster.Tags, width))

	var hint string
	switch {
	case detailCursor >= 0:
		hint = detailHintBar(width, hintItem("esc", "Deselect"), hintItem("enter", "Open"), hintItem("↑/↓", "Navigate"))
	case hasHistory:
		hint = detailHintBar(width, hintItem("esc", "Back ◀"), hintItem("↑/↓", "Navigate"))
	default:
		hint = detailHintBar(width, hintItem("esc/q", "Back to list"), hintItem("↑/↓", "Navigate"))
	}
	b.WriteString(hint)
	return b.String()
}

func renderNodeTable(nodes []awsclient.EKSNode, cursor int) string {
	var b strings.Builder

	const (
		wID    = 22
		wName  = 20
		wState = 12
		wType  = 14
		wIP    = 16
		wAZ    = 20
		wNG    = 22
	)

	hStyle      := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	vStyle      := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57"))
	sepStr      := "  " + strings.Repeat("─", wID+wName+wState+wType+wIP+wAZ+wNG) + "\n"

	col := func(s string, w int) string {
		if len(s) >= w {
			return s[:w-1] + " "
		}
		return s + strings.Repeat(" ", w-len(s))
	}

	// 헤더
	b.WriteString("  " +
		hStyle.Render(col("Instance ID", wID)) +
		hStyle.Render(col("Name", wName)) +
		hStyle.Render(col("State", wState)) +
		hStyle.Render(col("Type", wType)) +
		hStyle.Render(col("Private IP", wIP)) +
		hStyle.Render(col("AZ", wAZ)) +
		hStyle.Render(col("Node Group", wNG)) + "\n")
	b.WriteString(sepStr)

	for i, node := range nodes {
		stateStr := node.State
		stateColor := lipgloss.Color("255")
		switch node.State {
		case "running":
			stateColor = lipgloss.Color("10")
		case "stopped", "terminated":
			stateColor = lipgloss.Color("9")
		case "pending", "stopping":
			stateColor = lipgloss.Color("11")
		}

		if i == cursor {
			// 커서 행: 배경 하이라이트 (테이블과 동일한 보라색)
			cs := cursorStyle
			b.WriteString("> " +
				cs.Render(col(node.InstanceID, wID)) +
				cs.Render(col(orDash(node.Name), wName)) +
				cs.Foreground(stateColor).Render(col(stateStr, wState)) +
				cs.Foreground(lipgloss.Color("229")).Render(col(node.InstanceType, wType)) +
				cs.Render(col(orDash(node.PrivateIP), wIP)) +
				cs.Render(col(node.AvailabilityZone, wAZ)) +
				cs.Render(col(orDash(node.NodegroupName), wNG)) + "\n")
		} else {
			stateRendered := lipgloss.NewStyle().Foreground(stateColor).Render(col(stateStr, wState))
			b.WriteString("  " +
				vStyle.Render(col(node.InstanceID, wID)) +
				vStyle.Render(col(orDash(node.Name), wName)) +
				stateRendered +
				vStyle.Render(col(node.InstanceType, wType)) +
				vStyle.Render(col(orDash(node.PrivateIP), wIP)) +
				vStyle.Render(col(node.AvailabilityZone, wAZ)) +
				vStyle.Render(col(orDash(node.NodegroupName), wNG)) + "\n")
		}
	}
	b.WriteString("\n")
	return b.String()
}

func renderNodegroupBlock(ng awsclient.EKSNodegroup) string {
	var b strings.Builder

	ngNameStyle  := lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Bold(true)
	ngLabelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(18).PaddingLeft(2)
	ngValStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	ngRow := func(label, val string) string {
		return ngLabelStyle.Render(label) + ngValStyle.Render(val) + "\n"
	}

	b.WriteString("  " + ngNameStyle.Render("▸ "+ng.Name) + "  " +
		coloredEKSStatus(ng.Status) + "\n")
	b.WriteString(ngRow("Version",    ng.Version))
	b.WriteString(ngRow("Capacity",   ng.CapacityType))
	b.WriteString(ngRow("Instance",   strings.Join(ng.InstanceTypes, ", ")))
	b.WriteString(ngRow("AMI Type",   ng.AMIType))
	b.WriteString(ngRow("Disk (GB)",  fmt.Sprintf("%d", ng.DiskSize)))
	b.WriteString(ngRow("Scaling",    fmt.Sprintf("desired %d  (min %d – max %d)", ng.DesiredSize, ng.MinSize, ng.MaxSize)))
	b.WriteString(ngRow("Created At", ng.CreatedAtStr()))
	b.WriteString("\n")

	return b.String()
}

func renderALBDetail(lb *awsclient.LoadBalancer, vpcName string, sgNames map[string]string, listeners []awsclient.Listener, spinnerView string, detailCursor int, hasHistory bool, profileToAccount map[string]string, width int) string {
	if lb == nil {
		return ""
	}
	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("%s  ›  %s", lb.TypeShort(), lb.Name)) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", profileWithAccount(lb.Profile, profileToAccount)))
	b.WriteString(row("Name",    lb.Name))
	b.WriteString(row("Type",    lb.TypeShort()))
	b.WriteString(row("Scheme",  lb.Scheme))
	b.WriteString(row("State",   coloredALBState(lb.State)))
	b.WriteString(row("Region",  lb.Region))
	b.WriteString(row("ARN",     lb.ARN))

	b.WriteString(sectionStyle.Render("Network") + "\n")
	b.WriteString(row("VPC ID",   withName(lb.VpcID, vpcName)))
	b.WriteString(row("DNS Name", lb.DNSName))

	// Listeners 섹션 (lazy loaded)
	if listeners == nil {
		b.WriteString(sectionStyle.Render("Listeners") + "\n")
		b.WriteString("  " + spinnerView + " Loading...\n")
	} else if len(listeners) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Listeners (%d)", len(listeners))) + "\n")
		for i, li := range listeners {
			label := fmt.Sprintf("Listener %d", i+1)
			b.WriteString(rowMaybeActive(label, li.Title(), detailCursor == i))
		}
	}

	if len(lb.AvailabilityZones) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Availability Zones (%d)", len(lb.AvailabilityZones))) + "\n")
		for _, az := range lb.AvailabilityZones {
			b.WriteString("  " + valueStyle.Render(az) + "\n")
		}
	}

	listenerCount := len(listeners)
	if len(lb.SecurityGroupIDs) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Security Groups (%d)", len(lb.SecurityGroupIDs))) + "\n")
		for i, sgID := range lb.SecurityGroupIDs {
			label := fmt.Sprintf("SG %d", i+1)
			b.WriteString(rowMaybeActive(label, withName(sgID, sgNames[sgID]), detailCursor == listenerCount+i))
		}
	}

	if len(lb.Tags) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Tags (%d)", len(lb.Tags))) + "\n")
		b.WriteString(renderTags(lb.Tags, width))
	}

	var hint string
	switch {
	case detailCursor >= 0:
		hint = detailHintBar(width, hintItem("esc", "Deselect"), hintItem("enter", "Open"), hintItem("m", "Resource Map"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	case hasHistory:
		hint = detailHintBar(width, hintItem("esc", "Back ◀"), hintItem("m", "Resource Map"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	default:
		hint = detailHintBar(width, hintItem("esc/q", "Back to list"), hintItem("m", "Resource Map"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	}
	b.WriteString(hint)
	return b.String()
}

func renderListenerDetail(li *awsclient.Listener, rules []awsclient.ListenerRule, spinnerView string, tgNames map[string]string, detailCursor int, hasHistory bool, width int) string {
	if li == nil {
		return ""
	}
	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("Listener  ›  %s", li.Title())) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Protocol",   li.Protocol))
	b.WriteString(row("Port",       fmt.Sprintf("%d", li.Port)))
	if li.SSLPolicy != "" {
		b.WriteString(row("SSL Policy", li.SSLPolicy))
	}

	if len(li.CertARNs) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Certificates (%d)", len(li.CertARNs))) + "\n")
		for _, arn := range li.CertARNs {
			b.WriteString("  " + valueStyle.Render(arn) + "\n")
		}
	}

	if li.IsALB() {
		// ALB: rules (lazy loaded)
		if rules == nil {
			b.WriteString(sectionStyle.Render("Rules") + "\n")
			b.WriteString("  " + spinnerView + " Loading...\n")
		} else {
			b.WriteString(sectionStyle.Render(fmt.Sprintf("Rules (%d)", len(rules))) + "\n")
			for i, r := range rules {
				label := fmt.Sprintf("Rule %d", i+1)
				if r.IsDefault {
					label = "Default"
				}
				summary := ruleSummary(r, tgNames)
				b.WriteString(rowMaybeActive(label, summary, detailCursor == i))
			}
		}
	} else {
		// NLB: default action
		b.WriteString(sectionStyle.Render("Default Action") + "\n")
		for i, a := range li.DefaultActions {
			if a.Type == "forward" && a.TargetGroupARN != "" {
				name := tgNames[a.TargetGroupARN]
				label := "Target Group"
				b.WriteString(rowMaybeActive(label, withName(a.TargetGroupARN, name), detailCursor == i))
			}
		}
	}

	var hint string
	switch {
	case detailCursor >= 0:
		hint = detailHintBar(width, hintItem("esc", "Deselect"), hintItem("enter", "Open"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	case hasHistory:
		hint = detailHintBar(width, hintItem("esc", "Back ◀"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	default:
		hint = detailHintBar(width, hintItem("esc/q", "Back"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	}
	b.WriteString(hint)
	return b.String()
}

func ruleSummary(r awsclient.ListenerRule, tgNames map[string]string) string {
	var parts []string
	for _, c := range r.Conditions {
		if len(c.Values) > 0 {
			parts = append(parts, c.Field+": "+strings.Join(c.Values, ", "))
		}
	}
	condStr := strings.Join(parts, "  |  ")
	tgARNs := r.ForwardTGARNs()
	var actionStr string
	if len(tgARNs) > 0 {
		name := tgNames[tgARNs[0]]
		if name != "" {
			actionStr = "→ " + name
		} else {
			actionStr = "→ " + tgARNs[0]
		}
	} else {
		for _, a := range r.Actions {
			if a.Type == "redirect" {
				actionStr = "↪ " + a.RedirectCode + " " + a.RedirectTarget
				break
			} else if a.Type == "fixed-response" {
				actionStr = "▪ fixed-response"
				break
			}
		}
	}
	if condStr != "" && actionStr != "" {
		return condStr + "  " + nameTagStyle.Render(actionStr)
	}
	if actionStr != "" {
		return nameTagStyle.Render(actionStr)
	}
	return condStr
}

func renderRuleDetail(rule *awsclient.ListenerRule, tgNames map[string]string, detailCursor int, hasHistory bool, width int) string {
	if rule == nil {
		return ""
	}
	var b strings.Builder

	priority := rule.Priority
	if rule.IsDefault {
		priority = "default"
	}
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("Rule  ›  Priority %s", priority)) + "\n")

	if len(rule.Conditions) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Conditions (%d)", len(rule.Conditions))) + "\n")
		for _, c := range rule.Conditions {
			b.WriteString(row(c.Field, strings.Join(c.Values, ", ")))
		}
	} else {
		b.WriteString(sectionStyle.Render("Conditions") + "\n")
		b.WriteString("  " + valueStyle.Render("(default rule — matches all requests)") + "\n")
	}

	b.WriteString(sectionStyle.Render("Actions") + "\n")
	tgARNs := rule.ForwardTGARNs()
	for i, tgARN := range tgARNs {
		name := tgNames[tgARN]
		b.WriteString(rowMaybeActive("Target Group", withName(tgARN, name), detailCursor == i))
	}
	for _, a := range rule.Actions {
		switch a.Type {
		case "redirect":
			b.WriteString(row("Redirect", fmt.Sprintf("%s  %s", a.RedirectCode, a.RedirectTarget)))
		case "fixed-response":
			b.WriteString(row("Fixed Response", ""))
		}
	}

	var hint string
	switch {
	case detailCursor >= 0:
		hint = detailHintBar(width, hintItem("esc", "Deselect"), hintItem("enter", "Open"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	case hasHistory:
		hint = detailHintBar(width, hintItem("esc", "Back ◀"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	default:
		hint = detailHintBar(width, hintItem("esc/q", "Back"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	}
	b.WriteString(hint)
	return b.String()
}

func renderTargetGroupDetail(tg *awsclient.TargetGroup, vpcName string, targets []awsclient.TargetEntry, spinnerView string, lookupInstanceName func(string) string, lookupNodeByIP func(string) (string, string), hasHistory bool, width int) string {
	if tg == nil {
		return ""
	}
	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("Target Group  ›  %s", tg.Name)) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Name",        tg.Name))
	b.WriteString(row("Protocol",    tg.Protocol))
	if tg.Port > 0 {
		b.WriteString(row("Port",    fmt.Sprintf("%d", tg.Port)))
	}
	b.WriteString(row("Target Type", tg.TargetType))
	b.WriteString(row("VPC ID",      withName(tg.VpcID, vpcName)))

	hc := tg.HealthCheck
	b.WriteString(sectionStyle.Render("Health Check") + "\n")
	b.WriteString(row("Protocol",            hc.Protocol))
	if hc.Path != "" {
		b.WriteString(row("Path",            hc.Path))
	}
	b.WriteString(row("Port",                hc.Port))
	b.WriteString(row("Healthy threshold",   fmt.Sprintf("%d", hc.HealthyThreshold)))
	b.WriteString(row("Unhealthy threshold", fmt.Sprintf("%d", hc.UnhealthyThreshold)))
	if hc.TimeoutSeconds > 0 {
		b.WriteString(row("Timeout",         fmt.Sprintf("%ds", hc.TimeoutSeconds)))
	}
	if hc.IntervalSeconds > 0 {
		b.WriteString(row("Interval",        fmt.Sprintf("%ds", hc.IntervalSeconds)))
	}

	if targets == nil {
		b.WriteString(sectionStyle.Render("Targets") + "\n")
		b.WriteString("  " + spinnerView + " Loading...\n")
	} else {
		healthy, unhealthy, other := 0, 0, 0
		for _, t := range targets {
			switch t.State {
			case "healthy":
				healthy++
			case "unhealthy":
				unhealthy++
			default:
				other++
			}
		}
		title := fmt.Sprintf("Targets (%d)", len(targets))
		if len(targets) > 0 {
			title = fmt.Sprintf("Targets (%d  ●%d ○%d)", len(targets), healthy, unhealthy+other)
		}
		b.WriteString(sectionStyle.Render(title) + "\n")
		for _, t := range targets {
			state := coloredTargetState(t.State)
			addr := t.ID
			if t.Port > 0 {
				addr = fmt.Sprintf("%s:%d", t.ID, t.Port)
			}
			var hint string
			switch tg.TargetType {
			case "instance":
				// 인스턴스 ID → 이름 직접 조회
				if name := lookupInstanceName(t.ID); name != "" {
					hint = nameTagStyle.Render("[" + name + "]")
				}
			case "ip":
				// IP → ENI secondary IP 역추적 → 노드 확인
				if instID, name := lookupNodeByIP(t.ID); instID != "" {
					node := instID
					if name != "" {
						node = name + " (" + instID + ")"
					}
					hint = nameTagStyle.Render("[node: " + node + "]")
				}
			}
			line := fmt.Sprintf("%-42s", addr)
			if hint != "" {
				line += "  " + hint
			}
			line += fmt.Sprintf("  %-28s  %s", state, t.AZ)
			if t.Description != "" {
				line += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(t.Description)
			}
			b.WriteString("  " + line + "\n")
		}
	}

	var hint string
	if hasHistory {
		hint = detailHintBar(width, hintItem("esc", "Back ◀"), hintItem("j/k", "Scroll"))
	} else {
		hint = detailHintBar(width, hintItem("esc/q", "Back"), hintItem("j/k", "Scroll"))
	}
	b.WriteString(hint)
	return b.String()
}

func coloredTargetState(state string) string {
	switch state {
	case "healthy":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("● " + state)
	case "unhealthy":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("○ " + state)
	case "draining":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("◌ " + state)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("- " + state)
	}
}

func coloredALBState(state string) string {
	switch state {
	case "active":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(state)
	case "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(state)
	case "provisioning":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(state)
	default:
		return state
	}
}

func renderRoute53Detail(rec *awsclient.Route53Record, detailCursor int, aliasLinked bool, width int) string {
	if rec == nil {
		return ""
	}
	var b strings.Builder

	title := rec.Name
	if title == "" {
		title = rec.ZoneName
	}
	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("Route 53  ›  %s  ›  %s", rec.ZoneName, title)) + "\n")

	b.WriteString(sectionStyle.Render("Record") + "\n")
	b.WriteString(row("Profile",   rec.Profile))
	b.WriteString(row("Name",      rec.Name))
	b.WriteString(row("Type",      rec.Type))
	b.WriteString(row("TTL",       rec.TTLStr()))
	b.WriteString(row("Zone",      rec.ZoneName))
	b.WriteString(row("Zone Type", rec.ZoneType))
	b.WriteString(row("Zone ID",   rec.ZoneID))

	if rec.AliasTarget != "" {
		b.WriteString(sectionStyle.Render("Alias Target") + "\n")
		if aliasLinked {
			// ALB와 매칭됨 → 커서로 선택해서 Enter로 진입 가능
			b.WriteString(rowMaybeActive("DNS Name", valueStyle.Render(rec.AliasTarget)+"  "+nameTagStyle.Render("[→ ALB]"), detailCursor == 0))
		} else {
			b.WriteString(row("DNS Name", rec.AliasTarget))
		}
	} else {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Values (%d)", len(rec.Values))) + "\n")
		if len(rec.Values) == 0 {
			b.WriteString("  " + tagStyle.Render("-") + "\n")
		} else {
			for _, v := range rec.Values {
				b.WriteString("  " + valueStyle.Render(v) + "\n")
			}
		}
	}

	var hint string
	switch {
	case aliasLinked && detailCursor >= 0:
		hint = detailHintBar(width, hintItem("esc", "Deselect"), hintItem("enter", "Open ALB"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	case aliasLinked:
		hint = detailHintBar(width, hintItem("esc/q", "Back"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	default:
		hint = detailHintBar(width, hintItem("esc/q", "Back to list"))
	}
	b.WriteString(hint)
	return b.String()
}

func renderRDSDetail(db *awsclient.DBInstance, vpcName string, subnetNames map[string]string, sgNames map[string]string, enis []awsclient.ENI, primaryENIID string, spinnerView string, detailCursor int, hasHistory bool, profileToAccount map[string]string, width int) string {
	if db == nil {
		return ""
	}
	var b strings.Builder

	b.WriteString(detailTitleStyle.Render(fmt.Sprintf("RDS  ›  %s", db.DBInstanceID)) + "\n")

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile",     profileWithAccount(db.Profile, profileToAccount)))
	b.WriteString(row("Identifier",  db.DBInstanceID))
	if db.Name != "" {
		b.WriteString(row("Name", db.Name))
	}
	b.WriteString(row("Status",      coloredRDSStatus(db.Status)))
	b.WriteString(row("Engine",      db.Engine+" "+db.EngineVersion))
	b.WriteString(row("Class",       db.DBInstanceClass))
	b.WriteString(row("Multi-AZ",    fmt.Sprintf("%v", db.MultiAZ)))
	b.WriteString(row("Region",      db.Region))
	b.WriteString(row("Created",     db.CreateTimeStr()))

	b.WriteString(sectionStyle.Render("Storage") + "\n")
	b.WriteString(row("Storage Type",      db.StorageType))
	b.WriteString(row("Allocated Storage", fmt.Sprintf("%d GiB", db.AllocatedStorage)))

	b.WriteString(sectionStyle.Render("Network") + "\n")
	b.WriteString(row("VPC ID",            withName(db.VpcID, vpcName)))
	b.WriteString(row("Subnet Group",      orDash(db.SubnetGroupName)))
	b.WriteString(row("Availability Zone", orDash(db.AvailabilityZone)))
	endpoint := orDash(db.Endpoint)
	if db.Endpoint != "" && db.Port > 0 {
		endpoint = fmt.Sprintf("%s:%d", db.Endpoint, db.Port)
	}
	b.WriteString(row("Endpoint", endpoint))

	subnetCount := len(db.SubnetIDs)
	if subnetCount > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Subnets (%d)", subnetCount)) + "\n")
		for i, sid := range db.SubnetIDs {
			label := fmt.Sprintf("Subnet %d", i+1)
			b.WriteString(rowMaybeActive(label, withName(sid, subnetNames[sid]), detailCursor == i))
		}
	}

	if len(db.SecurityGroupIDs) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Security Groups (%d)", len(db.SecurityGroupIDs))) + "\n")
		for i, sgID := range db.SecurityGroupIDs {
			label := fmt.Sprintf("SG %d", i+1)
			b.WriteString(rowMaybeActive(label, withName(sgID, sgNames[sgID]), detailCursor == subnetCount+i))
		}
	}

	// Network Interfaces
	b.WriteString(sectionStyle.Render("Network Interfaces") + "\n")
	if enis == nil {
		b.WriteString("  " + spinnerView + " Loading...\n")
	} else if len(enis) == 0 {
		b.WriteString("  " + tagStyle.Render("-") + "\n")
	} else {
		sgCount := len(db.SecurityGroupIDs)
		for i, e := range enis {
			ip := e.PrivateIP
			if len(e.PrivateIPs) > 1 {
				ip = strings.Join(e.PrivateIPs, ", ")
			}
			role := mapSepStyle.Render("standby")
			if e.ENIID == primaryENIID {
				role = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("primary")
			}
			val := valueStyle.Render(e.ENIID) +
				"  " + lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render(ip) +
				"  " + mapSepStyle.Render(e.Status) +
				"  " + mapSepStyle.Render(orDash(e.AvailabilityZone)) +
				"  " + role
			b.WriteString(rowMaybeActive(fmt.Sprintf("ENI %d", i+1), val, detailCursor == subnetCount+sgCount+i))
		}
	}

	if len(db.Tags) > 0 {
		b.WriteString(sectionStyle.Render(fmt.Sprintf("Tags (%d)", len(db.Tags))) + "\n")
		b.WriteString(renderTags(db.Tags, width))
	}

	var hint string
	switch {
	case detailCursor >= 0:
		hint = detailHintBar(width, hintItem("esc", "Deselect"), hintItem("enter", "Open"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	case hasHistory:
		hint = detailHintBar(width, hintItem("esc", "Back ◀"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	default:
		hint = detailHintBar(width, hintItem("esc/q", "Back to list"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	}
	b.WriteString(hint)
	return b.String()
}

func renderElastiCacheDetail(ec *awsclient.ElastiCacheCluster, sgNameMap map[string]string, subnetIDs []string, subnetNames map[string]string, spinnerView string, detailCursor int, hasHistory bool, profileToAccount map[string]string, width int) string {
	var b strings.Builder

	subnetCount := len(subnetIDs)
	sgCount := len(ec.SecurityGroupIDs)

	b.WriteString(sectionStyle.Render("General") + "\n")
	b.WriteString(row("Profile", profileWithAccount(ec.Profile, profileToAccount)))
	b.WriteString(row("ID", ec.ID))
	b.WriteString(row("Engine", ec.Engine+" "+ec.EngineVersion))
	b.WriteString(row("Node Type", ec.NodeType))
	b.WriteString(row("Status", coloredElastiCacheStatus(ec.Status)))
	b.WriteString(row("Nodes", fmt.Sprintf("%d", ec.NumNodes)))
	b.WriteString(row("Multi-AZ", ec.MultiAZ))
	b.WriteString(row("Region", ec.Region))

	b.WriteString("\n" + sectionStyle.Render("Network") + "\n")
	endpoint := ec.Endpoint
	if ec.Port > 0 {
		endpoint = fmt.Sprintf("%s:%d", ec.Endpoint, ec.Port)
	}
	b.WriteString(row("Endpoint", orDash(endpoint)))

	// Subnet Group (lazy loaded)
	switch {
	case subnetIDs == nil:
		b.WriteString("\n" + sectionStyle.Render("Subnet Group") + "\n")
		b.WriteString(row("Name", orDash(ec.SubnetGroupName)))
		b.WriteString("  " + spinnerView + "\n")
	case len(subnetIDs) == 0:
		b.WriteString("\n" + sectionStyle.Render("Subnet Group") + "\n")
		b.WriteString(row("Name", orDash(ec.SubnetGroupName)))
	default:
		b.WriteString("\n" + sectionStyle.Render(fmt.Sprintf("Subnet Group  (%d subnets)", subnetCount)) + "\n")
		b.WriteString(row("Name", orDash(ec.SubnetGroupName)))
		for i, subnetID := range subnetIDs {
			name := subnetNames[subnetID]
			val := valueStyle.Render(subnetID)
			if name != "" {
				val += "  " + mapSepStyle.Render(name)
			}
			b.WriteString(rowMaybeActive("Subnet", val, detailCursor == i))
		}
	}

	// Security Groups
	if sgCount > 0 {
		b.WriteString("\n" + sectionStyle.Render(fmt.Sprintf("Security Groups (%d)", sgCount)) + "\n")
		for i, sgID := range ec.SecurityGroupIDs {
			name := sgNameMap[sgID]
			val := valueStyle.Render(sgID)
			if name != "" {
				val += "  " + mapSepStyle.Render(name)
			}
			b.WriteString(rowMaybeActive("SG", val, detailCursor == subnetCount+i))
		}
	}

	var hint string
	switch {
	case detailCursor >= 0:
		hint = detailHintBar(width, hintItem("esc", "Deselect"), hintItem("enter", "Open"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	case hasHistory:
		hint = detailHintBar(width, hintItem("esc", "Back ◀"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	default:
		hint = detailHintBar(width, hintItem("esc/q", "Back to list"), hintItem("↑/↓", "Navigate"), hintItem("j/k", "Scroll"))
	}
	b.WriteString(hint)
	return b.String()
}

func coloredElastiCacheStatus(status string) string {
	switch status {
	case "available":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(status)
	case "creating", "modifying", "snapshotting", "rebooting cluster nodes":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(status)
	case "deleting", "incompatible-parameters", "incompatible-network", "restore-failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(status)
	default:
		return status
	}
}

func renderS3Detail(b *awsclient.S3Bucket, tags map[string]string, spinnerView string, hasHistory bool, profileToAccount map[string]string, width int) string {
	var sb strings.Builder

	sb.WriteString(sectionStyle.Render("General") + "\n")
	sb.WriteString(row("Profile", profileWithAccount(b.Profile, profileToAccount)))
	sb.WriteString(row("Name", b.Name))
	sb.WriteString(row("Region", orDash(b.Region)))
	sb.WriteString(row("Created", b.CreationDateStr()))

	sb.WriteString("\n" + sectionStyle.Render("Access") + "\n")
	pubColor := lipgloss.Color("9")
	if b.PublicAccess == "Blocked" {
		pubColor = lipgloss.Color("10")
	}
	sb.WriteString(row("Public Access", lipgloss.NewStyle().Foreground(pubColor).Render(b.PublicAccess)))

	sb.WriteString("\n" + sectionStyle.Render("Versioning") + "\n")
	verColor := lipgloss.Color("10")
	if b.VersioningStatus != "Enabled" {
		verColor = lipgloss.Color("243")
	}
	sb.WriteString(row("Versioning", lipgloss.NewStyle().Foreground(verColor).Render(b.VersioningStatus)))

	sb.WriteString("\n" + sectionStyle.Render("Tags") + "\n")
	switch {
	case tags == nil:
		sb.WriteString("  " + spinnerView + "\n")
	case len(tags) == 0:
		sb.WriteString(row("", "-"))
	default:
		sb.WriteString(renderTags(tags, width))
	}

	var hint string
	if hasHistory {
		hint = detailHintBar(width, hintItem("esc", "Back ◀"), hintItem("j/k", "Scroll"))
	} else {
		hint = detailHintBar(width, hintItem("esc/q", "Back to list"), hintItem("j/k", "Scroll"))
	}
	sb.WriteString(hint)
	return sb.String()
}

func coloredRDSStatus(status string) string {
	switch status {
	case "available":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(status)
	case "stopped":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(status)
	case "starting", "stopping", "rebooting", "modifying", "upgrading", "creating", "backing-up":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(status)
	default:
		return status
	}
}

func coloredEKSStatus(status string) string {
	switch status {
	case "ACTIVE":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(status)
	case "CREATING", "UPDATING":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(status)
	case "DELETING", "FAILED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(status)
	default:
		return status
	}
}
