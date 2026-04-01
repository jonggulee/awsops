package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsclient "github.com/jgulee/awsops/internal/aws"
)

// --- messages ---

type instancesLoadedMsg struct {
	instances []awsclient.Instance
	errs      []error
}

type sgLoadedMsg struct {
	groups []awsclient.SecurityGroup
	errs   []error
}

type vpcsLoadedMsg struct {
	vpcs    []awsclient.VPC
	subnets []awsclient.Subnet
	errs    []error
}

type tgwLoadedMsg struct {
	gateways     []awsclient.TransitGateway
	attachments  []awsclient.TGWAttachment
	routeTables  []awsclient.TGWRouteTable
	routes       []awsclient.TGWRoute
	associations []awsclient.TGWAssociation
	errs         []error
}

type accountIDsLoadedMsg struct {
	profileToAccount map[string]string
	accountToProfile map[string]string
}

type routeTablesLoadedMsg struct {
	tables []awsclient.VPCRouteTable
	errs   []error
}

// --- modes & views ---

type inputMode int

const (
	modeNormal  inputMode = iota
	modeSearch            // / 검색
	modeCommand           // : 커맨드
)

type viewType int

const (
	viewEC2    viewType = iota
	viewSG
	viewVPC
	viewSubnet
	viewTGW
)

type screen int

const (
	screenTable        screen = iota
	screenDetail              // d 눌렀을 때 상세 화면
	screenRegion              // R 눌렀을 때 리전 선택 화면
	screenConnectivity        // c 눌렀을 때 연결 체크 화면
)

var viewNames = map[viewType]string{
	viewEC2:    "ec2",
	viewSG:     "sg",
	viewVPC:    "vpc",
	viewSubnet: "subnet",
	viewTGW:    "tgw",
}

// --- sort ---

type sortCol int

const (
	sortNone sortCol = iota
	sortProfile
	sortName
	sortInstanceID
	sortState
	sortType
	sortPrivateIP
	sortPublicIP
	sortGroupID
	sortVpcID
	sortSubnetID
	sortCidr
	sortAZ
	sortRegion
	sortTgwID
	sortAttachmentID
	sortResourceID
	sortOwnerID
	sortResourceType
)

// EC2:    1=Profile 2=Name 3=InstanceID 4=State 5=Type 6=PrivateIP 7=PublicIP 8=VpcID 9=SubnetID 10=Region
var ec2SortCols = []sortCol{sortProfile, sortName, sortInstanceID, sortState, sortType, sortPrivateIP, sortPublicIP, sortVpcID, sortSubnetID, sortRegion}

// SG:     1=Profile 2=Name 3=GroupID 4=VpcID 5=- 6=Region
var sgSortCols = []sortCol{sortProfile, sortName, sortGroupID, sortVpcID, sortNone, sortRegion}

// VPC:    1=Profile 2=Name 3=VpcID 4=CIDR 5=State 6=Region
var vpcSortCols = []sortCol{sortProfile, sortName, sortVpcID, sortCidr, sortState, sortRegion}

// Subnet: 1=Profile 2=Name 3=SubnetID 4=VpcID 5=CIDR 6=AZ 7=Region
var subnetSortCols = []sortCol{sortProfile, sortName, sortInstanceID, sortVpcID, sortCidr, sortAZ, sortRegion}

// TGW:    1=Profile 2=TgwID 3=AttachmentID 4=Type 5=ResourceID 6=Owner 7=TgwOwner 8=State 9=Region
var tgwSortCols = []sortCol{sortProfile, sortTgwID, sortAttachmentID, sortResourceType, sortResourceID, sortOwnerID, sortOwnerID, sortState, sortRegion}

var sortColNames = map[sortCol]string{
	sortProfile:    "Profile",
	sortName:       "Name",
	sortInstanceID: "Instance ID",
	sortState:      "State",
	sortType:       "Type",
	sortPrivateIP:  "Private IP",
	sortPublicIP:   "Public IP",
	sortGroupID:    "Group ID",
	sortVpcID:      "VPC ID",
	sortSubnetID:   "Subnet ID",
	sortCidr:         "CIDR",
	sortAZ:           "AZ",
	sortRegion:       "Region",
	sortTgwID:        "TGW ID",
	sortAttachmentID: "Attachment ID",
	sortResourceID:   "Resource ID",
	sortOwnerID:      "Owner",
	sortResourceType: "Type",
}

// --- model ---

type Model struct {
	table          table.Model
	spinner        spinner.Model
	input          textinput.Model
	mode           inputMode
	view           viewType
	screen         screen
	selectedInst   *awsclient.Instance
	selectedSG     *awsclient.SecurityGroup
	selectedVPC    *awsclient.VPC
	selectedSubnet *awsclient.Subnet
	filters        []string
	instances        []awsclient.Instance
	groups           []awsclient.SecurityGroup
	vpcs             []awsclient.VPC
	subnets          []awsclient.Subnet
	tgws             []awsclient.TransitGateway
	tgwAttachments   []awsclient.TGWAttachment
	tgwRouteTables   []awsclient.TGWRouteTable
	tgwRoutes        []awsclient.TGWRoute
	tgwAssociations  []awsclient.TGWAssociation
	selectedTGWAtt      *awsclient.TGWAttachment
	connectivitySrcVPC  *awsclient.VPC
	connectivityResult  *awsclient.ConnectivityResult
	connectivityCursor  int
	routeTables         []awsclient.VPCRouteTable
	profileToAccount    map[string]string // profile → accountID
	accountToProfile    map[string]string // accountID → profile
	fetchErr         []error
	loading        bool
	width          int
	height         int
	regions        []regionEntry
	regionCursor   int
	sortBy         sortCol
	sortAsc        bool
	detailScroll   int
}

func New() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.CharLimit = 64

	return Model{
		spinner: s,
		input:   ti,
		loading: true,
		mode:    modeNormal,
		view:    viewEC2,
		screen:  screenTable,
		regions: defaultRegions(),
	}
}

func (m Model) Init() tea.Cmd {
	regions := []string{awsclient.DefaultRegion}
	return tea.Batch(
		m.spinner.Tick,
		fetchInstances(),
		fetchSecurityGroups(),
		fetchVPCsWithRegions(regions),
		fetchTGWsWithRegions(regions),
		fetchRouteTablesWithRegions(regions),
		fetchAccountIDs(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case instancesLoadedMsg:
		m.instances = msg.instances
		m.fetchErr = append(m.fetchErr, msg.errs...)
		m.loading = len(m.groups) == 0 && len(m.instances) == 0
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case sgLoadedMsg:
		m.groups = msg.groups
		m.fetchErr = append(m.fetchErr, msg.errs...)
		m.loading = len(m.groups) == 0 && len(m.instances) == 0
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case vpcsLoadedMsg:
		m.vpcs = msg.vpcs
		m.subnets = msg.subnets
		m.fetchErr = append(m.fetchErr, msg.errs...)
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case tgwLoadedMsg:
		m.tgws = msg.gateways
		m.tgwAttachments = msg.attachments
		m.tgwRouteTables = msg.routeTables
		m.tgwRoutes = msg.routes
		m.tgwAssociations = msg.associations
		m.fetchErr = append(m.fetchErr, msg.errs...)
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case routeTablesLoadedMsg:
		m.routeTables = msg.tables
		m.fetchErr = append(m.fetchErr, msg.errs...)

	case accountIDsLoadedMsg:
		m.profileToAccount = msg.profileToAccount
		m.accountToProfile = msg.accountToProfile

	case tea.KeyMsg:
		// 리전 선택 화면
		if m.screen == screenRegion {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.screen = screenTable
			case "up", "k":
				if m.regionCursor > 0 {
					m.regionCursor--
				}
			case "down", "j":
				if m.regionCursor < len(m.regions)-1 {
					m.regionCursor++
				}
			case " ":
				m.regions[m.regionCursor].selected = !m.regions[m.regionCursor].selected
			case "enter":
				ids := selectedRegionIDs(m.regions)
				if len(ids) == 0 {
					// 최소 하나는 선택 강제
					m.regions[m.regionCursor].selected = true
					ids = selectedRegionIDs(m.regions)
				}
				m.screen = screenTable
				m.loading = true
				m.instances = nil
				m.groups = nil
				m.vpcs = nil
				m.subnets = nil
				m.tgws = nil
				m.tgwAttachments = nil
				m.tgwRouteTables = nil
				m.tgwRoutes = nil
				m.tgwAssociations = nil
				m.routeTables = nil
				m.fetchErr = nil
				return m, tea.Batch(m.spinner.Tick, fetchInstancesWithRegions(ids), fetchSGWithRegions(ids), fetchVPCsWithRegions(ids), fetchTGWsWithRegions(ids), fetchRouteTablesWithRegions(ids))
			}
			return m, nil
		}

		// 연결 체크 화면
		if m.screen == screenConnectivity {
			// 결과 화면: 스크롤 + esc로 피커로 돌아가기
			if m.connectivityResult != nil {
				pageSize := m.height / 2
				if pageSize < 1 {
					pageSize = 1
				}
				switch msg.String() {
				case "ctrl+c":
					return m, tea.Quit
				case "esc", "b":
					m.connectivityResult = nil
					m.detailScroll = 0
				case "up", "k":
					if m.detailScroll > 0 {
						m.detailScroll--
					}
				case "down", "j":
					if m.detailScroll < m.detailMaxScroll() {
						m.detailScroll++
					}
				case "pgup":
					m.detailScroll -= pageSize
					if m.detailScroll < 0 {
						m.detailScroll = 0
					}
				case "pgdown":
					m.detailScroll += pageSize
					if max := m.detailMaxScroll(); m.detailScroll > max {
						m.detailScroll = max
					}
				}
				return m, nil
			}

			// 피커 화면: 필터 입력 + ↑↓ 네비게이션
			vpcs := m.connectivityPickerVPCs()
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.screen = screenTable
				m.connectivitySrcVPC = nil
				m.connectivityCursor = 0
				m.input.SetValue("")
			case "up", "k":
				if m.connectivityCursor > 0 {
					m.connectivityCursor--
				}
			case "down", "j":
				if m.connectivityCursor < len(vpcs)-1 {
					m.connectivityCursor++
				}
			case "enter":
				if m.connectivityCursor < len(vpcs) && m.connectivitySrcVPC != nil {
					dst := vpcs[m.connectivityCursor]
					res := awsclient.CheckConnectivity(
						m.connectivitySrcVPC.VpcID,
						dst.VpcID,
						m.tgwAttachments,
						m.tgwAssociations,
						m.tgwRoutes,
						m.vpcs,
						m.subnets,
						m.routeTables,
						m.accountToProfile,
					)
					m.connectivityResult = &res
					m.detailScroll = 0
				}
			default:
				prev := m.input.Value()
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				if m.input.Value() != prev {
					m.connectivityCursor = 0 // 필터 변경 시 커서 초기화
				}
				return m, cmd
			}
			return m, nil
		}

		// 디테일 화면에서는 스크롤 + esc/q 로 뒤로
		if m.screen == screenDetail {
			pageSize := m.height / 2
			if pageSize < 1 {
				pageSize = 1
			}
			switch msg.String() {
			case "esc", "q", "ctrl+c":
				if msg.String() == "ctrl+c" {
					return m, tea.Quit
				}
				m.screen = screenTable
				m.selectedInst = nil
				m.selectedTGWAtt = nil
				m.detailScroll = 0
			case "up", "k":
				if m.detailScroll > 0 {
					m.detailScroll--
				}
			case "down", "j":
				if m.detailScroll < m.detailMaxScroll() {
					m.detailScroll++
				}
			case "pgup":
				m.detailScroll -= pageSize
				if m.detailScroll < 0 {
					m.detailScroll = 0
				}
			case "pgdown":
				m.detailScroll += pageSize
				if max := m.detailMaxScroll(); m.detailScroll > max {
					m.detailScroll = max
				}
			}
			return m, nil
		}

		switch m.mode {
		case modeNormal:
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "d":
				switch m.view {
				case viewEC2:
					if inst := m.selectedInstance(); inst != nil {
						m.selectedInst = inst
						m.selectedSG, m.selectedVPC, m.selectedSubnet = nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
					}
				case viewSG:
					if sg := m.selectedSG_(); sg != nil {
						m.selectedSG = sg
						m.selectedInst, m.selectedVPC, m.selectedSubnet = nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
					}
				case viewVPC:
					if vpc := m.selectedVPC_(); vpc != nil {
						m.selectedVPC = vpc
						m.selectedInst, m.selectedSG, m.selectedSubnet = nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
					}
				case viewSubnet:
					if subnet := m.selectedSubnet_(); subnet != nil {
						m.selectedSubnet = subnet
						m.selectedInst, m.selectedSG, m.selectedVPC = nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
					}
				case viewTGW:
					if att := m.selectedTGWAtt_(); att != nil {
						m.selectedTGWAtt = att
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet = nil, nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
					}
				}
				return m, nil
			case "c":
				if m.view == viewVPC {
					if vpc := m.selectedVPC_(); vpc != nil {
						m.connectivitySrcVPC = vpc
						m.connectivityResult = nil
						m.connectivityCursor = 0
						m.screen = screenConnectivity
						m.input.Placeholder = "filter..."
						m.input.SetValue("")
						m.input.Focus()
						return m, textinput.Blink
					}
				}
				return m, nil
			case "1", "2", "3", "4", "5", "6", "7", "8", "9":
				n := int(msg.String()[0]-'0') - 1
				m.sortByIndex(n)
				m.table = m.buildCurrentTable()
				return m, nil
			case "0":
				m.sortByIndex(9)
				m.table = m.buildCurrentTable()
				return m, nil
			case "r":
				m.loading = true
				m.instances = nil
				m.groups = nil
				m.fetchErr = nil
				m.vpcs = nil
				m.subnets = nil
				m.tgws = nil
				m.tgwAttachments = nil
				m.tgwRouteTables = nil
				m.tgwRoutes = nil
				m.tgwAssociations = nil
				m.routeTables = nil
				ids := selectedRegionIDs(m.regions)
				return m, tea.Batch(m.spinner.Tick, fetchInstancesWithRegions(ids), fetchSGWithRegions(ids), fetchVPCsWithRegions(ids), fetchTGWsWithRegions(ids), fetchRouteTablesWithRegions(ids))
			case "R":
				m.screen = screenRegion
				m.regionCursor = 0
				return m, nil
			case "/":
				m.mode = modeSearch
				m.input.Placeholder = "type to search..."
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case ":":
				m.mode = modeCommand
				m.input.Placeholder = "ec2 / sg / vpc / subnet / tgw"
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case "esc":
				m.filters = nil
				m.input.SetValue("")
				m.table = m.buildCurrentTable()
				return m, nil
			}

		case modeSearch:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = modeNormal
				m.filters = nil
				m.input.SetValue("")
				m.input.Blur()
				m.table = m.buildCurrentTable()
				return m, nil
			case "enter":
				if v := strings.TrimSpace(m.input.Value()); v != "" {
					m.filters = append(m.filters, v)
				}
				m.mode = modeNormal
				m.input.SetValue("")
				m.input.Blur()
				m.table = m.buildCurrentTable()
				return m, nil
			}

		case modeCommand:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.mode = modeNormal
				m.input.SetValue("")
				m.input.Blur()
				return m, nil
			case "enter":
				m.mode = modeNormal
				m.input.Blur()
				m.applyCommand(m.input.Value())
				m.input.SetValue("")
				return m, nil
			}
		}
	}

	if !m.loading {
		var cmds []tea.Cmd
		var cmd tea.Cmd

		if m.mode == modeSearch || m.mode == modeCommand {
			prevVal := m.input.Value()
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
			if m.mode == modeSearch && m.input.Value() != prevVal {
				m.table = m.buildCurrentTable()
			}
		}

		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// selectedInstance returns the Instance matching the currently highlighted table row.
func (m *Model) selectedInstance() *awsclient.Instance {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	id := row[2] // Instance ID column
	for i := range m.instances {
		if m.instances[i].InstanceID == id {
			return &m.instances[i]
		}
	}
	return nil
}

// selectedSG_ returns the SecurityGroup matching the currently highlighted table row.
func (m *Model) selectedSG_() *awsclient.SecurityGroup {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	id := row[2] // Group ID column
	for i := range m.groups {
		if m.groups[i].GroupID == id {
			return &m.groups[i]
		}
	}
	return nil
}

func (m *Model) selectedVPC_() *awsclient.VPC {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	id := row[2] // VPC ID column
	for i := range m.vpcs {
		if m.vpcs[i].VpcID == id {
			return &m.vpcs[i]
		}
	}
	return nil
}

func (m *Model) selectedSubnet_() *awsclient.Subnet {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	id := row[2] // Subnet ID column
	for i := range m.subnets {
		if m.subnets[i].SubnetID == id {
			return &m.subnets[i]
		}
	}
	return nil
}

func (m *Model) connectivityPickerVPCs() []awsclient.VPC {
	filter := strings.ToLower(m.input.Value())
	var result []awsclient.VPC
	for _, v := range m.vpcs {
		if m.connectivitySrcVPC != nil && v.VpcID == m.connectivitySrcVPC.VpcID {
			continue // 소스 VPC 제외
		}
		if filter == "" || matchAll([]string{filter}, v.Profile, v.Name, v.VpcID, v.CidrBlock, v.Region) {
			result = append(result, v)
		}
	}
	return result
}

func (m *Model) selectedTGWAtt_() *awsclient.TGWAttachment {
	row := m.table.SelectedRow()
	if row == nil {
		return nil
	}
	id := row[2] // Attachment ID column
	for i := range m.tgwAttachments {
		if m.tgwAttachments[i].AttachmentID == id {
			return &m.tgwAttachments[i]
		}
	}
	return nil
}

func (m *Model) sortByIndex(n int) {
	cols := ec2SortCols
	switch m.view {
	case viewSG:
		cols = sgSortCols
	case viewVPC:
		cols = vpcSortCols
	case viewSubnet:
		cols = subnetSortCols
	case viewTGW:
		cols = tgwSortCols
	}
	if n < 0 || n >= len(cols) {
		return
	}
	col := cols[n]
	if col == sortNone {
		return
	}
	if m.sortBy == col {
		// 같은 컬럼: asc → desc → 해제
		if m.sortAsc {
			m.sortAsc = false
		} else {
			m.sortBy = sortNone
		}
	} else {
		m.sortBy = col
		m.sortAsc = true
	}
}

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
	}
	m.sortBy = sortNone
	m.filters = nil
	m.input.SetValue("")
	m.table = m.buildCurrentTable()
}

func (m *Model) buildCurrentTable() table.Model {
	switch m.view {
	case viewSG:
		return buildSGTable(filterSGRows(m.sortedGroups(), m.filters), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc)
	case viewVPC:
		return buildVPCTable(filterVPCRows(m.sortedVPCs(), m.filters), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc)
	case viewSubnet:
		return buildSubnetTable(filterSubnetRows(m.sortedSubnets(), m.filters), m.height, m.maxProfileWidth(), m.width, m.sortBy, m.sortAsc)
	case viewTGW:
		return buildTGWTable(filterTGWRows(m.sortedAttachments(), m.filters, m.accountToProfile), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc)
	default:
		return buildEC2Table(filterEC2Rows(m.sortedInstances(), m.filters), m.height, m.maxProfileWidth(), m.sortBy, m.sortAsc)
	}
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

func buildEC2Table(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	cols := []table.Column{
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
	return newTable(cols, rows, height)
}

func filterEC2Rows(instances []awsclient.Instance, filters []string) []table.Row {
	if len(filters) == 0 {
		return ec2Rows(instances)
	}
	var filtered []awsclient.Instance
	for _, inst := range instances {
		if matchAll(filters, inst.Profile, inst.Region, inst.Name, inst.InstanceID, inst.State, inst.Type, inst.PrivateIP, inst.VpcID, inst.SubnetID) {
			filtered = append(filtered, inst)
		}
	}
	return ec2Rows(filtered)
}

func ec2Rows(instances []awsclient.Instance) []table.Row {
	rows := make([]table.Row, len(instances))
	for i, inst := range instances {
		rows[i] = table.Row{inst.Profile, inst.Name, inst.InstanceID, inst.State, inst.Type, inst.PrivateIP, inst.PublicIP, inst.VpcID, inst.SubnetID, inst.Region}
	}
	return rows
}

// --- SG table ---

func buildSGTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	cols := []table.Column{
		h(1, sortProfile,     "Profile",     profileWidth),
		h(2, sortName,        "Name",        22),
		h(3, sortGroupID,     "Group ID",    22),
		h(4, sortVpcID,       "VPC ID",      22),
		h(5, sortNone,        "Description", 36),
		h(6, sortRegion,      "Region",      18),
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

func filterSGRows(groups []awsclient.SecurityGroup, filters []string) []table.Row {
	if len(filters) == 0 {
		return sgRows(groups)
	}
	var filtered []awsclient.SecurityGroup
	for _, sg := range groups {
		if matchAll(filters, sg.Profile, sg.Region, sg.Name, sg.GroupID, sg.VpcID, sg.Description) {
			filtered = append(filtered, sg)
		}
	}
	return sgRows(filtered)
}

func sgRows(groups []awsclient.SecurityGroup) []table.Row {
	rows := make([]table.Row, len(groups))
	for i, sg := range groups {
		rows[i] = table.Row{sg.Profile, sg.Name, sg.GroupID, sg.VpcID, sg.Description, sg.Region}
	}
	return rows
}

// --- VPC table ---

func buildVPCTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	cols := []table.Column{
		h(1, sortProfile, "Profile",  profileWidth),
		h(2, sortName,    "Name",     20),
		h(3, sortVpcID,   "VPC ID",   22),
		h(4, sortCidr,    "CIDR",     18),
		h(5, sortState,   "State",    12),
		h(6, sortRegion,  "Region",   18),
	}
	return newTable(cols, rows, height)
}

func filterVPCRows(vpcs []awsclient.VPC, filters []string) []table.Row {
	if len(filters) == 0 {
		return vpcRows(vpcs)
	}
	var filtered []awsclient.VPC
	for _, v := range vpcs {
		if matchAll(filters, v.Profile, v.Name, v.VpcID, v.CidrBlock, v.State, v.Region) {
			filtered = append(filtered, v)
		}
	}
	return vpcRows(filtered)
}

func vpcRows(vpcs []awsclient.VPC) []table.Row {
	rows := make([]table.Row, len(vpcs))
	for i, v := range vpcs {
		def := ""
		if v.IsDefault {
			def = "default"
		}
		_ = def
		rows[i] = table.Row{v.Profile, v.Name, v.VpcID, v.CidrBlock, v.State, v.Region}
	}
	return rows
}

// --- Subnet table ---

func buildSubnetTable(rows []table.Row, height, profileWidth, termWidth int, sortBy sortCol, sortAsc bool) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	const fixedCols = 24 + 22 + 18 + 20 + 18 // SubnetID+VpcID+CIDR+AZ+Region
	const padding = 16                          // 테이블 내부 패딩 여유분
	const minNameWidth = 18
	nameWidth := termWidth - profileWidth - fixedCols - padding
	if nameWidth < minNameWidth {
		nameWidth = minNameWidth
	}
	cols := []table.Column{
		h(1, sortProfile,    "Profile",   profileWidth),
		h(2, sortName,       "Name",      nameWidth),
		h(3, sortInstanceID, "Subnet ID", 24),
		h(4, sortVpcID,      "VPC ID",    22),
		h(5, sortCidr,       "CIDR",      18),
		h(6, sortAZ,         "AZ",        20),
		h(7, sortRegion,     "Region",    18),
	}
	return newTable(cols, rows, height)
}

func filterSubnetRows(subnets []awsclient.Subnet, filters []string) []table.Row {
	if len(filters) == 0 {
		return subnetRows(subnets)
	}
	var filtered []awsclient.Subnet
	for _, s := range subnets {
		if matchAll(filters, s.Profile, s.Name, s.SubnetID, s.VpcID, s.CidrBlock, s.AvailabilityZone, s.Region) {
			filtered = append(filtered, s)
		}
	}
	return subnetRows(filtered)
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

func buildTGWTable(rows []table.Row, height, profileWidth int, sortBy sortCol, sortAsc bool) table.Model {
	h := func(n int, col sortCol, title string, w int) table.Column {
		return table.Column{Title: colTitle(n, col, title, sortBy, sortAsc), Width: w}
	}
	cols := []table.Column{
		h(1, sortProfile,      "Profile",       profileWidth),
		h(2, sortTgwID,        "TGW ID",        22),
		h(3, sortAttachmentID, "Attachment ID", 26),
		h(4, sortResourceType, "Type",          10),
		h(5, sortResourceID,   "Resource ID",  22),
		h(6, sortOwnerID,      "Owner",        20),
		h(7, sortOwnerID,      "TGW Owner",    20),
		h(8, sortState,        "State",        14),
		h(9, sortRegion,       "Region",       18),
	}
	return newTable(cols, rows, height)
}

func filterTGWRows(atts []awsclient.TGWAttachment, filters []string, accountToProfile map[string]string) []table.Row {
	if len(filters) == 0 {
		return tgwRows(atts, accountToProfile)
	}
	var filtered []awsclient.TGWAttachment
	for _, a := range atts {
		ownerProfile := accountToProfile[a.ResourceOwnerID]
		tgwOwnerProfile := accountToProfile[a.TgwOwnerID]
		if matchAll(filters, a.Profile, a.TgwID, a.AttachmentID, a.ResourceType, a.ResourceID,
			a.ResourceOwnerID, ownerProfile, a.TgwOwnerID, tgwOwnerProfile, a.State, a.Region) {
			filtered = append(filtered, a)
		}
	}
	return tgwRows(filtered, accountToProfile)
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
