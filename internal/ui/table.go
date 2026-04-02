package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	awsclient "github.com/jgulee/awsops/internal/aws"
)

// --- sort helpers ---

func (m *Model) sortedInstances() []awsclient.Instance {
	instances := make([]awsclient.Instance, len(m.instances))
	copy(instances, m.instances)
	if m.sortBy == sortNone {
		return instances
	}
	sort.Slice(instances, func(i, j int) bool {
		a, b := instances[i], instances[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortName:
			less = a.Name < b.Name
		case sortState:
			less = a.State < b.State
		case sortType:
			less = a.Type < b.Type
		case sortPrivateIP:
			less = a.PrivateIP < b.PrivateIP
		case sortPublicIP:
			less = a.PublicIP < b.PublicIP
		case sortVpcID:
			less = a.VpcID < b.VpcID
		case sortSubnetID:
			less = a.SubnetID < b.SubnetID
		case sortInstanceID:
			less = a.InstanceID < b.InstanceID
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return instances
}

func (m *Model) sortedGroups() []awsclient.SecurityGroup {
	groups := make([]awsclient.SecurityGroup, len(m.groups))
	copy(groups, m.groups)
	if m.sortBy == sortNone {
		return groups
	}
	sort.Slice(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortName:
			less = a.Name < b.Name
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return groups
}

func (m *Model) sortedVPCs() []awsclient.VPC {
	vpcs := make([]awsclient.VPC, len(m.vpcs))
	copy(vpcs, m.vpcs)
	if m.sortBy == sortNone {
		return vpcs
	}
	sort.Slice(vpcs, func(i, j int) bool {
		a, b := vpcs[i], vpcs[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortName:
			less = a.Name < b.Name
		case sortVpcID:
			less = a.VpcID < b.VpcID
		case sortCidr:
			less = a.CidrBlock < b.CidrBlock
		case sortState:
			less = a.State < b.State
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return vpcs
}

func (m *Model) sortedSubnets() []awsclient.Subnet {
	subnets := make([]awsclient.Subnet, len(m.subnets))
	copy(subnets, m.subnets)
	if m.sortBy == sortNone {
		return subnets
	}
	sort.Slice(subnets, func(i, j int) bool {
		a, b := subnets[i], subnets[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortName:
			less = a.Name < b.Name
		case sortInstanceID: // SubnetID
			less = a.SubnetID < b.SubnetID
		case sortVpcID:
			less = a.VpcID < b.VpcID
		case sortCidr:
			less = a.CidrBlock < b.CidrBlock
		case sortAZ:
			less = a.AvailabilityZone < b.AvailabilityZone
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return subnets
}

func (m *Model) sortedAttachments() []awsclient.TGWAttachment {
	atts := make([]awsclient.TGWAttachment, len(m.tgwAttachments))
	copy(atts, m.tgwAttachments)
	if m.sortBy == sortNone {
		return atts
	}
	sort.Slice(atts, func(i, j int) bool {
		a, b := atts[i], atts[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortTgwID:
			less = a.TgwID < b.TgwID
		case sortAttachmentID:
			less = a.AttachmentID < b.AttachmentID
		case sortResourceType:
			less = a.ResourceType < b.ResourceType
		case sortResourceID:
			less = a.ResourceID < b.ResourceID
		case sortOwnerID:
			less = a.ResourceOwnerID < b.ResourceOwnerID
		case sortState:
			less = a.State < b.State
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return atts
}

func (m *Model) sortedCerts() []awsclient.Certificate {
	certs := make([]awsclient.Certificate, len(m.certs))
	copy(certs, m.certs)
	if m.sortBy == sortNone {
		return certs
	}
	sort.Slice(certs, func(i, j int) bool {
		a, b := certs[i], certs[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortDomainName:
			less = a.DomainName < b.DomainName
		case sortCertStatus:
			less = a.Status < b.Status
		case sortType:
			less = a.Type < b.Type
		case sortExpiry:
			less = a.NotAfter.Before(b.NotAfter)
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return certs
}

// --- view command & table dispatch ---

func (m *Model) applyCommand(cmd string) {
	switch strings.TrimSpace(strings.ToLower(cmd)) {
	case "ec2":
		m.view = viewEC2
	case "sg":
		m.view = viewSG
	case "vpc":
		m.view = viewVPC
	case "subnet":
		m.view = viewSubnet
	case "tgw":
		m.view = viewTGW
	case "acm":
		m.view = viewACM
	}
	m.sortBy = sortNone
	m.filters = nil
	m.colOffset = 0
	m.input.SetValue("")
	m.table = m.buildCurrentTable()
}

func (m *Model) buildCurrentTable() table.Model {
	switch m.view {
	case viewSG:
		m.displayedGroups = filterSGData(m.sortedGroups(), m.filters)
		return buildSGTable(rowsSliced(sgRows(m.displayedGroups), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewVPC:
		m.displayedVPCs = filterVPCData(m.sortedVPCs(), m.filters)
		return buildVPCTable(rowsSliced(vpcRows(m.displayedVPCs), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewSubnet:
		m.displayedSubnets = filterSubnetData(m.sortedSubnets(), m.filters)
		return buildSubnetTable(rowsSliced(subnetRows(m.displayedSubnets), m.colOffset), m.height, m.maxProfileWidth(), m.width, m.sortBy, m.sortAsc, m.colOffset)
	case viewTGW:
		m.displayedAttachments = filterTGWData(m.sortedAttachments(), m.filters)
		return buildTGWTable(rowsSliced(tgwRows(m.displayedAttachments, m.accountToProfile), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewACM:
		m.displayedCerts = filterCertData(m.sortedCerts(), m.filters)
		return buildACMTable(rowsSliced(certRows(m.displayedCerts), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	default:
		m.displayedInstances = filterEC2Data(m.sortedInstances(), m.filters)
		return buildEC2Table(rowsSliced(ec2Rows(m.displayedInstances), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	}
}

// rowsSliced returns rows with each row's values starting from colOffset.
func rowsSliced(rows []table.Row, colOffset int) []table.Row {
	if colOffset <= 0 {
		return rows
	}
	result := make([]table.Row, len(rows))
	for i, row := range rows {
		if colOffset < len(row) {
			result[i] = row[colOffset:]
		} else {
			result[i] = table.Row{}
		}
	}
	return result
}

// maxColOffset returns the maximum colOffset for the current view (always keep at least 1 column).
func (m *Model) maxColOffset() int {
	switch m.view {
	case viewEC2:
		return 9
	case viewSG:
		return 5
	case viewVPC:
		return 5
	case viewSubnet:
		return 6
	case viewTGW:
		return 8
	case viewACM:
		return 5
	}
	return 0
}

func (m *Model) maxProfileWidth() int {
	const minWidth = 10
	const maxWidth = 30
	max := minWidth
	for _, inst := range m.instances {
		if len(inst.Profile) > max {
			max = len(inst.Profile)
		}
	}
	for _, sg := range m.groups {
		if len(sg.Profile) > max {
			max = len(sg.Profile)
		}
	}
	if max > maxWidth {
		max = maxWidth
	}
	return max + 1 // 여백 1칸
}

// --- EC2 table ---

func buildEC2Table(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1,  sortProfile,    "Profile",     profileWidth),
		h(2,  sortName,       "Name",        16),
		h(3,  sortInstanceID, "Instance ID", 20),
		h(4,  sortState,      "State",       12),
		h(5,  sortType,       "Type",        10),
		h(6,  sortPrivateIP,  "Private IP",  16),
		h(7,  sortPublicIP,   "Public IP",   16),
		h(8,  sortVpcID,      "VPC ID",      22),
		h(9,  sortSubnetID,   "Subnet ID",   24),
		h(10, sortRegion,     "Region",      18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterEC2Data(instances []awsclient.Instance, filters []string) []awsclient.Instance {
	if len(filters) == 0 {
		return instances
	}
	var out []awsclient.Instance
	for _, inst := range instances {
		if matchAll(filters, inst.Profile, inst.Region, inst.Name, inst.InstanceID, inst.State, inst.Type, inst.PrivateIP, inst.VpcID, inst.SubnetID) {
			out = append(out, inst)
		}
	}
	return out
}

func ec2Rows(instances []awsclient.Instance) []table.Row {
	rows := make([]table.Row, len(instances))
	for i, inst := range instances {
		rows[i] = table.Row{inst.Profile, inst.Name, inst.InstanceID, inst.State, inst.Type, inst.PrivateIP, inst.PublicIP, inst.VpcID, inst.SubnetID, inst.Region}
	}
	return rows
}

// --- SG table ---

func buildSGTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile, "Profile",     profileWidth),
		h(2, sortName,    "Name",        22),
		h(3, sortGroupID, "Group ID",    22),
		h(4, sortVpcID,   "VPC ID",      22),
		h(5, sortNone,    "Description", 36),
		h(6, sortRegion,  "Region",      18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func colTitle(n int, col sortCol, title string, sortBy sortCol, sortAsc bool) string {
	prefix := fmt.Sprintf("%d:", n)
	if col != sortNone && col == sortBy {
		arrow := "↑"
		if !sortAsc {
			arrow = "↓"
		}
		return prefix + title + " " + arrow
	}
	return prefix + title
}

func filterSGData(groups []awsclient.SecurityGroup, filters []string) []awsclient.SecurityGroup {
	if len(filters) == 0 {
		return groups
	}
	var out []awsclient.SecurityGroup
	for _, sg := range groups {
		if matchAll(filters, sg.Profile, sg.Region, sg.Name, sg.GroupID, sg.VpcID, sg.Description) {
			out = append(out, sg)
		}
	}
	return out
}

func sgRows(groups []awsclient.SecurityGroup) []table.Row {
	rows := make([]table.Row, len(groups))
	for i, sg := range groups {
		rows[i] = table.Row{sg.Profile, sg.Name, sg.GroupID, sg.VpcID, sg.Description, sg.Region}
	}
	return rows
}

// --- VPC table ---

func buildVPCTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile, "Profile", profileWidth),
		h(2, sortName,    "Name",    20),
		h(3, sortVpcID,   "VPC ID",  22),
		h(4, sortCidr,    "CIDR",    18),
		h(5, sortState,   "State",   12),
		h(6, sortRegion,  "Region",  18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterVPCData(vpcs []awsclient.VPC, filters []string) []awsclient.VPC {
	if len(filters) == 0 {
		return vpcs
	}
	var out []awsclient.VPC
	for _, v := range vpcs {
		if matchAll(filters, v.Profile, v.Name, v.VpcID, v.CidrBlock, v.State, v.Region) {
			out = append(out, v)
		}
	}
	return out
}

func vpcRows(vpcs []awsclient.VPC) []table.Row {
	rows := make([]table.Row, len(vpcs))
	for i, v := range vpcs {
		rows[i] = table.Row{v.Profile, v.Name, v.VpcID, v.CidrBlock, v.State, v.Region}
	}
	return rows
}

// --- Subnet table ---

func buildSubnetTable(rows []table.Row, height, profileWidth, termWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	const fixedCols = 24 + 22 + 18 + 20 + 18 // SubnetID+VpcID+CIDR+AZ+Region
	const padding = 16
	const minNameWidth = 18
	nameWidth := termWidth - profileWidth - fixedCols - padding
	if nameWidth < minNameWidth {
		nameWidth = minNameWidth
	}
	allCols := []table.Column{
		h(1, sortProfile,    "Profile",   profileWidth),
		h(2, sortName,       "Name",      nameWidth),
		h(3, sortInstanceID, "Subnet ID", 24),
		h(4, sortVpcID,      "VPC ID",    22),
		h(5, sortCidr,       "CIDR",      18),
		h(6, sortAZ,         "AZ",        20),
		h(7, sortRegion,     "Region",    18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterSubnetData(subnets []awsclient.Subnet, filters []string) []awsclient.Subnet {
	if len(filters) == 0 {
		return subnets
	}
	var out []awsclient.Subnet
	for _, s := range subnets {
		if matchAll(filters, s.Profile, s.Name, s.SubnetID, s.VpcID, s.CidrBlock, s.AvailabilityZone, s.Region) {
			out = append(out, s)
		}
	}
	return out
}

func subnetRows(subnets []awsclient.Subnet) []table.Row {
	rows := make([]table.Row, len(subnets))
	for i, s := range subnets {
		rows[i] = table.Row{s.Profile, s.Name, s.SubnetID, s.VpcID, s.CidrBlock, s.AvailabilityZone, s.Region}
	}
	return rows
}

// matchAll returns true if every filter term matches at least one of the fields.
func matchAll(filters []string, fields ...string) bool {
	for _, f := range filters {
		q := strings.ToLower(f)
		matched := false
		for _, field := range fields {
			if strings.Contains(strings.ToLower(field), q) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// --- TGW table ---

func buildTGWTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile,      "Profile",      profileWidth),
		h(2, sortTgwID,        "TGW ID",       22),
		h(3, sortAttachmentID, "Attachment ID", 26),
		h(4, sortResourceType, "Type",          10),
		h(5, sortResourceID,   "Resource ID",   22),
		h(6, sortOwnerID,      "Owner",         20),
		h(7, sortOwnerID,      "TGW Owner",     20),
		h(8, sortState,        "State",         14),
		h(9, sortRegion,       "Region",        18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterTGWData(atts []awsclient.TGWAttachment, filters []string) []awsclient.TGWAttachment {
	if len(filters) == 0 {
		return atts
	}
	var out []awsclient.TGWAttachment
	for _, a := range atts {
		if matchAll(filters, a.Profile, a.TgwID, a.AttachmentID, a.ResourceType, a.ResourceID,
			a.ResourceOwnerID, a.TgwOwnerID, a.State, a.Region) {
			out = append(out, a)
		}
	}
	return out
}

func tgwRows(atts []awsclient.TGWAttachment, accountToProfile map[string]string) []table.Row {
	rows := make([]table.Row, len(atts))
	for i, a := range atts {
		ownerDisplay := a.ResourceOwnerID
		if profile, ok := accountToProfile[a.ResourceOwnerID]; ok {
			ownerDisplay = profile
		}
		tgwOwnerDisplay := a.TgwOwnerID
		if profile, ok := accountToProfile[a.TgwOwnerID]; ok {
			tgwOwnerDisplay = profile
		}
		rows[i] = table.Row{
			a.Profile,
			a.TgwID,
			a.AttachmentID,
			a.ResourceType,
			a.ResourceID,
			ownerDisplay,
			tgwOwnerDisplay,
			a.State,
			a.Region,
		}
	}
	return rows
}

// --- ACM table ---

func buildACMTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile,    "Profile",     profileWidth),
		h(2, sortDomainName, "Domain Name", 36),
		h(3, sortCertStatus, "Status",      20),
		h(4, sortType,       "Type",        16),
		h(5, sortExpiry,     "Expiry",      14),
		h(6, sortRegion,     "Region",      18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterCertData(certs []awsclient.Certificate, filters []string) []awsclient.Certificate {
	if len(filters) == 0 {
		return certs
	}
	var out []awsclient.Certificate
	for _, c := range certs {
		if matchAll(filters, c.Profile, c.DomainName, c.Status, c.Type, c.Region) {
			out = append(out, c)
		}
	}
	return out
}

func certRows(certs []awsclient.Certificate) []table.Row {
	rows := make([]table.Row, len(certs))
	for i, c := range certs {
		rows[i] = table.Row{c.Profile, c.DomainName, c.Status, c.Type, c.ExpiryStr(), c.Region}
	}
	return rows
}

// --- shared table builder ---

func newTable(cols []table.Column, rows []table.Row, height int) table.Model {
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(height-6),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	return t
}
