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

// --- RDS table ---

// RDS: 1=Profile 2=Identifier 3=Engine 4=Class 5=Status 6=VpcID 7=Endpoint 8=Region
func buildRDSTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile, "Profile",    profileWidth),
		h(2, sortName,    "Identifier", 28),
		h(3, sortEngine,  "Engine",     20),
		h(4, sortDBClass, "Class",      16),
		h(5, sortState,   "Status",     14),
		h(6, sortVpcID,   "VPC ID",     22),
		h(7, sortNone,    "Endpoint",   48),
		h(8, sortRegion,  "Region",     18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterRDSData(dbs []awsclient.DBInstance, filters []string) []awsclient.DBInstance {
	if len(filters) == 0 {
		return dbs
	}
	var out []awsclient.DBInstance
	for _, db := range dbs {
		if matchAllWithTags(filters, db.Tags, db.Profile, db.DBInstanceID, db.Engine, db.EngineVersion, db.DBInstanceClass, db.Status, db.VpcID, db.Endpoint, db.Region) {
			out = append(out, db)
		}
	}
	return out
}

func rdsRows(dbs []awsclient.DBInstance) []table.Row {
	rows := make([]table.Row, len(dbs))
	for i, db := range dbs {
		engineVer := db.Engine + " " + db.EngineVersion
		rows[i] = table.Row{db.Profile, db.DBInstanceID, engineVer, db.DBInstanceClass, db.Status, db.VpcID, db.Endpoint, db.Region}
	}
	return rows
}

func (m *Model) sortedRDS() []awsclient.DBInstance {
	dbs := make([]awsclient.DBInstance, len(m.rdsInstances))
	copy(dbs, m.rdsInstances)
	if m.sortBy == sortNone {
		return dbs
	}
	sort.Slice(dbs, func(i, j int) bool {
		a, b := dbs[i], dbs[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortName:
			less = a.DBInstanceID < b.DBInstanceID
		case sortEngine:
			less = a.Engine < b.Engine
		case sortDBClass:
			less = a.DBInstanceClass < b.DBInstanceClass
		case sortState:
			less = a.Status < b.Status
		case sortVpcID:
			less = a.VpcID < b.VpcID
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return dbs
}

func (m *Model) selectedRDS_() *awsclient.DBInstance {
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.displayedRDS) {
		return nil
	}
	db := m.displayedRDS[cursor]
	return &db
}

func buildS3Table(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile,  "Profile",       profileWidth),
		h(2, sortName,     "Name",          40),
		h(3, sortRegion,   "Region",        18),
		h(4, sortVersioning,    "Versioning",    12),
		h(5, sortPublicAccess,  "Public Access", 14),
		h(6, sortCreateTime,  "Created",       12),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterS3Data(buckets []awsclient.S3Bucket, filters []string, regions []string) []awsclient.S3Bucket {
	regionSet := make(map[string]bool, len(regions))
	for _, r := range regions {
		regionSet[r] = true
	}
	var out []awsclient.S3Bucket
	for _, b := range buckets {
		if len(regionSet) > 0 && !regionSet[b.Region] {
			continue
		}
		if len(filters) > 0 && !matchAllWithTags(filters, b.Tags, b.Profile, b.Name, b.Region, b.VersioningStatus, b.PublicAccess) {
			continue
		}
		out = append(out, b)
	}
	return out
}

func s3Rows(buckets []awsclient.S3Bucket) []table.Row {
	rows := make([]table.Row, len(buckets))
	for i, b := range buckets {
		rows[i] = table.Row{b.Profile, b.Name, b.Region, b.VersioningStatus, b.PublicAccess, b.CreationDateStr()}
	}
	return rows
}

func (m *Model) sortedS3() []awsclient.S3Bucket {
	buckets := make([]awsclient.S3Bucket, len(m.s3Buckets))
	copy(buckets, m.s3Buckets)
	if m.sortBy == sortNone {
		return buckets
	}
	sort.Slice(buckets, func(i, j int) bool {
		a, b := buckets[i], buckets[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortName:
			less = a.Name < b.Name
		case sortRegion:
			less = a.Region < b.Region
		case sortCreateTime:
			less = a.CreationDate.Before(b.CreationDate)
		case sortVersioning:
			less = a.VersioningStatus < b.VersioningStatus
		case sortPublicAccess:
			less = a.PublicAccess < b.PublicAccess
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return buckets
}

func buildElastiCacheTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile, "Profile",   profileWidth),
		h(2, sortName,    "ID",        30),
		h(3, sortEngine,  "Engine",    12),
		h(4, sortType,    "Node Type", 20),
		h(5, sortState,   "Status",    14),
		h(6, sortNone,    "Nodes",      6),
		h(7, sortNone,    "Endpoint",  50),
		h(8, sortRegion,  "Region",    18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterElastiCacheData(clusters []awsclient.ElastiCacheCluster, filters []string, regions []string) []awsclient.ElastiCacheCluster {
	regionSet := make(map[string]bool, len(regions))
	for _, r := range regions {
		regionSet[r] = true
	}
	var out []awsclient.ElastiCacheCluster
	for _, c := range clusters {
		if len(regionSet) > 0 && !regionSet[c.Region] {
			continue
		}
		if len(filters) > 0 && !matchAllWithTags(filters, nil, c.Profile, c.ID, c.Engine, c.EngineVersion, c.NodeType, c.Status, c.Endpoint) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func elastiCacheRows(clusters []awsclient.ElastiCacheCluster) []table.Row {
	rows := make([]table.Row, len(clusters))
	for i, c := range clusters {
		endpoint := c.Endpoint
		if c.Port > 0 {
			endpoint = fmt.Sprintf("%s:%d", c.Endpoint, c.Port)
		}
		rows[i] = table.Row{c.Profile, c.ID, c.Engine + " " + c.EngineVersion, c.NodeType, c.Status, fmt.Sprintf("%d", c.NumNodes), endpoint, c.Region}
	}
	return rows
}

func (m *Model) sortedElastiCache() []awsclient.ElastiCacheCluster {
	clusters := make([]awsclient.ElastiCacheCluster, len(m.elastiCacheClusters))
	copy(clusters, m.elastiCacheClusters)
	if m.sortBy == sortNone {
		return clusters
	}
	sort.Slice(clusters, func(i, j int) bool {
		a, b := clusters[i], clusters[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortName:
			less = a.ID < b.ID
		case sortEngine:
			less = a.Engine < b.Engine
		case sortType:
			less = a.NodeType < b.NodeType
		case sortState:
			less = a.Status < b.Status
		case sortRegion:
			less = a.Region < b.Region
		default:
			return false
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return clusters
}

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
	case "eni":
		m.view = viewENI
	case "eks":
		m.view = viewEKS
	case "route53":
		m.view = viewRoute53
	case "elb":
		m.view = viewALB
	case "rds":
		m.view = viewRDS
	case "s3":
		m.view = viewS3
	case "redis", "elasticache":
		m.view = viewElastiCache
	case "profile", "account":
		m.view = viewAccount
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
	case viewENI:
		m.displayedENIs = filterENIData(m.sortedENIs(), m.filters)
		return buildENITable(rowsSliced(eniRows(m.displayedENIs), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewEKS:
		m.displayedEKS = filterEKSData(m.sortedEKSClusters(), m.filters)
		return buildEKSTable(rowsSliced(eksRows(m.displayedEKS), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewRoute53:
		m.displayedRoute53 = filterRoute53Data(m.sortedRoute53(), m.filters)
		return buildRoute53Table(rowsSliced(route53Rows(m.displayedRoute53), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewALB:
		m.displayedALBs = filterALBData(m.sortedALBs(), m.filters)
		return buildALBTable(rowsSliced(albRows(m.displayedALBs), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewRDS:
		m.displayedRDS = filterRDSData(m.sortedRDS(), m.filters)
		return buildRDSTable(rowsSliced(rdsRows(m.displayedRDS), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewS3:
		m.displayedS3 = filterS3Data(m.sortedS3(), m.filters, selectedRegionIDs(m.regions))
		return buildS3Table(rowsSliced(s3Rows(m.displayedS3), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewElastiCache:
		m.displayedElastiCache = filterElastiCacheData(m.sortedElastiCache(), m.filters, selectedRegionIDs(m.regions))
		return buildElastiCacheTable(rowsSliced(elastiCacheRows(m.displayedElastiCache), m.colOffset), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc, m.colOffset)
	case viewAccount:
		rows := accountRows(m.profileToAccount)
		return buildAccountTable(rows, m.height)
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
	case viewENI:
		return 9
	case viewEKS:
		return 6
	case viewRoute53:
		return 6
	case viewALB:
		return 6
	case viewRDS:
		return 7
	case viewS3:
		return 5
	case viewElastiCache:
		return 7
	case viewAccount:
		return 0
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
		if matchAllWithTags(filters, inst.Tags, inst.Profile, inst.Region, inst.Name, inst.InstanceID, inst.State, inst.Type, inst.PrivateIP, inst.VpcID, inst.SubnetID) {
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
		if matchAllWithTags(filters, v.Tags, v.Profile, v.Name, v.VpcID, v.CidrBlock, v.State, v.Region) {
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
		if matchAllWithTags(filters, s.Tags, s.Profile, s.Name, s.SubnetID, s.VpcID, s.CidrBlock, s.AvailabilityZone, s.Region) {
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

// parseTagFilter splits "key=value" into (key, value, true).
// Returns ("", "", false) if f does not contain "=".
func parseTagFilter(f string) (key, value string, ok bool) {
	idx := strings.Index(f, "=")
	if idx < 0 {
		return "", "", false
	}
	return f[:idx], f[idx+1:], true
}

// matchAllWithTags is like matchAll but also searches tags.
// Tokens in "key=value" format are matched against tag keys/values only.
// Plain text tokens are matched against both fields and tag keys/values.
func matchAllWithTags(filters []string, tags map[string]string, fields ...string) bool {
	for _, f := range filters {
		q := strings.ToLower(f)
		matched := false

		if key, val, isTag := parseTagFilter(q); isTag {
			// key=value 형식: 태그에서만 검색
			for k, v := range tags {
				if strings.Contains(strings.ToLower(k), key) &&
					(val == "" || strings.Contains(strings.ToLower(v), val)) {
					matched = true
					break
				}
			}
		} else {
			// 일반 텍스트: 필드 먼저 검색
			for _, field := range fields {
				if strings.Contains(strings.ToLower(field), q) {
					matched = true
					break
				}
			}
			// 필드에서 못 찾으면 태그 키/값에서도 검색
			if !matched {
				for k, v := range tags {
					if strings.Contains(strings.ToLower(k), q) ||
						strings.Contains(strings.ToLower(v), q) {
						matched = true
						break
					}
				}
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

// --- ENI table ---

func buildENITable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1,  sortProfile,       "Profile",     profileWidth),
		h(2,  sortENIID,         "ENI ID",      22),
		h(3,  sortName,          "Name",        20),
		h(4,  sortState,         "Status",      12),
		h(5,  sortInterfaceType, "Type",        16),
		h(6,  sortPrivateIP,     "Private IP",  16),
		h(7,  sortInstanceID,    "Instance ID", 20),
		h(8,  sortVpcID,         "VPC ID",      22),
		h(9,  sortSubnetID,      "Subnet ID",   24),
		h(10, sortRegion,        "Region",      18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterENIData(enis []awsclient.ENI, filters []string) []awsclient.ENI {
	if len(filters) == 0 {
		return enis
	}
	var out []awsclient.ENI
	for _, e := range enis {
		if matchAll(filters, e.Profile, e.ENIID, e.Name, e.Description, e.Status, e.InterfaceType, e.PrivateIP, e.InstanceID, e.VpcID, e.SubnetID, e.Region) {
			out = append(out, e)
		}
	}
	return out
}

func eniRows(enis []awsclient.ENI) []table.Row {
	rows := make([]table.Row, len(enis))
	for i, e := range enis {
		name := e.Name
		if name == "" {
			name = e.Description
		}
		rows[i] = table.Row{e.Profile, e.ENIID, name, e.Status, e.InterfaceType, e.PrivateIP, e.InstanceID, e.VpcID, e.SubnetID, e.Region}
	}
	return rows
}

// --- EKS table ---

func buildEKSTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile, "Profile",  profileWidth),
		h(2, sortName,    "Name",     28),
		h(3, sortState,   "Status",   14),
		h(4, sortVersion, "Version",  10),
		h(5, sortVpcID,   "VPC ID",   22),
		h(6, sortNone,    "Endpoint", 36),
		h(7, sortRegion,  "Region",   18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterEKSData(clusters []awsclient.EKSCluster, filters []string) []awsclient.EKSCluster {
	if len(filters) == 0 {
		return clusters
	}
	var out []awsclient.EKSCluster
	for _, c := range clusters {
		if matchAllWithTags(filters, c.Tags, c.Profile, c.Name, c.Status, c.Version, c.VpcID, c.Region) {
			out = append(out, c)
		}
	}
	return out
}

func eksRows(clusters []awsclient.EKSCluster) []table.Row {
	rows := make([]table.Row, len(clusters))
	for i, c := range clusters {
		rows[i] = table.Row{c.Profile, c.Name, c.Status, c.Version, c.VpcID, c.Endpoint, c.Region}
	}
	return rows
}

func (m *Model) sortedEKSClusters() []awsclient.EKSCluster {
	clusters := make([]awsclient.EKSCluster, len(m.eksClusters))
	copy(clusters, m.eksClusters)
	if m.sortBy == sortNone {
		return clusters
	}
	sort.Slice(clusters, func(i, j int) bool {
		a, b := clusters[i], clusters[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortName:
			less = a.Name < b.Name
		case sortState:
			less = a.Status < b.Status
		case sortVersion:
			less = a.Version < b.Version
		case sortVpcID:
			less = a.VpcID < b.VpcID
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return clusters
}

// --- ENI sort ---

func (m *Model) sortedENIs() []awsclient.ENI {
	enis := make([]awsclient.ENI, len(m.enis))
	copy(enis, m.enis)
	if m.sortBy == sortNone {
		return enis
	}
	sort.Slice(enis, func(i, j int) bool {
		a, b := enis[i], enis[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortENIID:
			less = a.ENIID < b.ENIID
		case sortName:
			less = a.Name < b.Name
		case sortState:
			less = a.Status < b.Status
		case sortInterfaceType:
			less = a.InterfaceType < b.InterfaceType
		case sortPrivateIP:
			less = a.PrivateIP < b.PrivateIP
		case sortInstanceID:
			less = a.InstanceID < b.InstanceID
		case sortVpcID:
			less = a.VpcID < b.VpcID
		case sortSubnetID:
			less = a.SubnetID < b.SubnetID
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return enis
}

// --- Route53 table ---

// Route53: 1=Profile 2=Zone 3=Name 4=Type 5=TTL 6=Value 7=ZoneType
func buildRoute53Table(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile,    "Profile",   profileWidth),
		h(2, sortZoneName,   "Zone",      28),
		h(3, sortName,       "Name",      36),
		h(4, sortRecordType, "Type",      8),
		h(5, sortTTL,        "TTL",       8),
		h(6, sortNone,       "Value",     40),
		h(7, sortZoneType,   "Zone Type", 10),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterRoute53Data(records []awsclient.Route53Record, filters []string) []awsclient.Route53Record {
	if len(filters) == 0 {
		return records
	}
	var out []awsclient.Route53Record
	for _, r := range records {
		if matchAll(filters, r.Profile, r.ZoneName, r.Name, r.Type, r.ZoneType, r.FirstValue()) {
			out = append(out, r)
		}
	}
	return out
}

func route53Rows(records []awsclient.Route53Record) []table.Row {
	rows := make([]table.Row, len(records))
	for i, r := range records {
		rows[i] = table.Row{r.Profile, r.ZoneName, r.Name, r.Type, r.TTLStr(), r.FirstValue(), r.ZoneType}
	}
	return rows
}

func (m *Model) sortedRoute53() []awsclient.Route53Record {
	records := make([]awsclient.Route53Record, len(m.route53Records))
	copy(records, m.route53Records)
	if m.sortBy == sortNone {
		return records
	}
	sort.Slice(records, func(i, j int) bool {
		a, b := records[i], records[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortZoneName:
			less = a.ZoneName < b.ZoneName
		case sortName:
			less = a.Name < b.Name
		case sortRecordType:
			less = a.Type < b.Type
		case sortTTL:
			less = a.TTL < b.TTL
		case sortZoneType:
			less = a.ZoneType < b.ZoneType
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return records
}

func (m *Model) selectedRoute53_() *awsclient.Route53Record {
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.displayedRoute53) {
		return nil
	}
	r := m.displayedRoute53[cursor]
	return &r
}

// --- ALB table ---

// ALB: 1=Profile 2=Name 3=Type 4=Scheme 5=State 6=VpcID 7=DNS Name 8=Region
func buildALBTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool, colOffset int) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	allCols := []table.Column{
		h(1, sortProfile, "Profile",  profileWidth),
		h(2, sortName,    "Name",     28),
		h(3, sortLBType,  "Type",     8),
		h(4, sortScheme,  "Scheme",   16),
		h(5, sortState,   "State",    12),
		h(6, sortVpcID,   "VPC ID",   22),
		h(7, sortNone,    "DNS Name", 52),
		h(8, sortRegion,  "Region",   18),
	}
	cols := allCols
	if colOffset > 0 && colOffset < len(allCols) {
		cols = allCols[colOffset:]
	}
	return newTable(cols, rows, height)
}

func filterALBData(lbs []awsclient.LoadBalancer, filters []string) []awsclient.LoadBalancer {
	if len(filters) == 0 {
		return lbs
	}
	var out []awsclient.LoadBalancer
	for _, lb := range lbs {
		if matchAllWithTags(filters, lb.Tags, lb.Profile, lb.Name, lb.LBType, lb.TypeShort(), lb.Scheme, lb.State, lb.VpcID, lb.DNSName, lb.Region) {
			out = append(out, lb)
		}
	}
	return out
}

func albRows(lbs []awsclient.LoadBalancer) []table.Row {
	rows := make([]table.Row, len(lbs))
	for i, lb := range lbs {
		rows[i] = table.Row{lb.Profile, lb.Name, lb.TypeShort(), lb.Scheme, lb.State, lb.VpcID, lb.DNSName, lb.Region}
	}
	return rows
}

func (m *Model) sortedALBs() []awsclient.LoadBalancer {
	lbs := make([]awsclient.LoadBalancer, len(m.loadBalancers))
	copy(lbs, m.loadBalancers)
	if m.sortBy == sortNone {
		return lbs
	}
	sort.Slice(lbs, func(i, j int) bool {
		a, b := lbs[i], lbs[j]
		var less bool
		switch m.sortBy {
		case sortProfile:
			less = a.Profile < b.Profile
		case sortName:
			less = a.Name < b.Name
		case sortLBType:
			less = a.LBType < b.LBType
		case sortScheme:
			less = a.Scheme < b.Scheme
		case sortState:
			less = a.State < b.State
		case sortVpcID:
			less = a.VpcID < b.VpcID
		case sortRegion:
			less = a.Region < b.Region
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
	return lbs
}

func (m *Model) selectedALB_() *awsclient.LoadBalancer {
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.displayedALBs) {
		return nil
	}
	lb := m.displayedALBs[cursor]
	return &lb
}

// --- Account table ---

func buildAccountTable(rows []table.Row, height int) table.Model {
	cols := []table.Column{
		{Title: "Profile", Width: 24},
		{Title: "Account ID", Width: 14},
	}
	return newTable(cols, rows, height)
}

func accountRows(profileToAccount map[string]string) []table.Row {
	type entry struct{ profile, accountID string }
	entries := make([]entry, 0, len(profileToAccount))
	for p, id := range profileToAccount {
		entries = append(entries, entry{p, id})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].profile < entries[j].profile })
	rows := make([]table.Row, len(entries))
	for i, e := range entries {
		rows[i] = table.Row{e.profile, e.accountID}
	}
	return rows
}
