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

type enisLoadedMsg struct {
	enis []awsclient.ENI
	errs []error
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

// --- detail history ---

// detailSnapshot captures the state of a detail view for back-navigation.
type detailSnapshot struct {
	selectedInst   *awsclient.Instance
	selectedSG     *awsclient.SecurityGroup
	selectedVPC    *awsclient.VPC
	selectedSubnet *awsclient.Subnet
	detailScroll   int
	detailCursor   int
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
	selectedTGWAtt            *awsclient.TGWAttachment
	connectivitySrcSubnet     *awsclient.Subnet
	connectivitySelectedRoute *awsclient.SubnetTGWRoute // nil = phase1(route 선택), non-nil = phase2(subnet 선택)
	connectivityResult        *awsclient.ConnectivityResult
	connectivityCursor  int
	routeTables         []awsclient.VPCRouteTable
	enis                []awsclient.ENI
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
	detailCursor   int              // detail 화면에서 선택된 interactive 필드 인덱스 (-1 = 없음)
	detailHistory  []detailSnapshot // 뒤로가기 스택
	colOffset      int // 가로 스크롤: 첫 번째로 보이는 컬럼 인덱스
	// 현재 테이블에 표시 중인 데이터 (커서 기반 선택에 사용)
	displayedInstances   []awsclient.Instance
	displayedGroups      []awsclient.SecurityGroup
	displayedVPCs        []awsclient.VPC
	displayedSubnets     []awsclient.Subnet
	displayedAttachments []awsclient.TGWAttachment
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
		fetchENIsWithRegions(regions),
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

	case enisLoadedMsg:
		m.enis = msg.enis
		m.fetchErr = append(m.fetchErr, msg.errs...)

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
				m.enis = nil
				m.fetchErr = nil
				return m, tea.Batch(m.spinner.Tick, fetchInstancesWithRegions(ids), fetchSGWithRegions(ids), fetchVPCsWithRegions(ids), fetchTGWsWithRegions(ids), fetchRouteTablesWithRegions(ids), fetchENIsWithRegions(ids))
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

			// 피커 화면 (2단계)
			pageSize := m.height / 2
			if pageSize < 1 {
				pageSize = 1
			}

			if m.connectivitySelectedRoute == nil {
				// ── Phase 1: 소스 서브넷 route table의 TGW route 선택 ──────────
				routes := m.connectivityPickerRoutes()
				switch msg.String() {
				case "ctrl+c":
					return m, tea.Quit
				case "esc":
					m.screen = screenTable
					m.connectivitySrcSubnet = nil
					m.connectivityCursor = 0
					m.input.SetValue("")
				case "up", "k":
					if m.connectivityCursor > 0 {
						m.connectivityCursor--
					}
				case "down", "j":
					if m.connectivityCursor < len(routes)-1 {
						m.connectivityCursor++
					}
				case "pgup":
					m.connectivityCursor -= pageSize
					if m.connectivityCursor < 0 {
						m.connectivityCursor = 0
					}
				case "pgdown":
					m.connectivityCursor += pageSize
					if m.connectivityCursor >= len(routes) {
						m.connectivityCursor = len(routes) - 1
					}
					if m.connectivityCursor < 0 {
						m.connectivityCursor = 0
					}
				case "enter":
					if len(routes) > 0 && m.connectivityCursor < len(routes) {
						selected := routes[m.connectivityCursor]
						m.connectivitySelectedRoute = &selected
						m.connectivityCursor = 0
						m.input.SetValue("")
					}
				default:
					prev := m.input.Value()
					var cmd tea.Cmd
					m.input, cmd = m.input.Update(msg)
					if m.input.Value() != prev {
						m.connectivityCursor = 0
					}
					return m, cmd
				}
			} else {
				// ── Phase 2: 해당 CIDR 대역의 목적지 서브넷 선택 ─────────────
				subnets := m.connectivityPickerSubnets()
				switch msg.String() {
				case "ctrl+c":
					return m, tea.Quit
				case "esc":
					// phase 1으로 돌아가기
					m.connectivitySelectedRoute = nil
					m.connectivityCursor = 0
					m.input.SetValue("")
				case "up", "k":
					if m.connectivityCursor > 0 {
						m.connectivityCursor--
					}
				case "down", "j":
					if m.connectivityCursor < len(subnets)-1 {
						m.connectivityCursor++
					}
				case "pgup":
					m.connectivityCursor -= pageSize
					if m.connectivityCursor < 0 {
						m.connectivityCursor = 0
					}
				case "pgdown":
					m.connectivityCursor += pageSize
					if m.connectivityCursor >= len(subnets) {
						m.connectivityCursor = len(subnets) - 1
					}
					if m.connectivityCursor < 0 {
						m.connectivityCursor = 0
					}
				case "enter":
					if len(subnets) > 0 && m.connectivityCursor < len(subnets) && m.connectivitySrcSubnet != nil {
						dst := subnets[m.connectivityCursor]
						res := awsclient.CheckConnectivity(
							m.connectivitySrcSubnet.SubnetID,
							dst.SubnetID,
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
						m.connectivityCursor = 0
					}
					return m, cmd
				}
			}
			return m, nil
		}

		// 디테일 화면: ↑/↓ = 필드 이동, j/k = 스크롤, enter = 이동
		if m.screen == screenDetail {
			pageSize := m.height / 2
			if pageSize < 1 {
				pageSize = 1
			}
			n := m.detailInteractiveFieldCount()
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "q":
				if m.detailCursor >= 0 {
					m.detailCursor = -1
				} else if len(m.detailHistory) > 0 {
					// 히스토리에서 이전 상태 복원
					prev := m.detailHistory[len(m.detailHistory)-1]
					m.detailHistory = m.detailHistory[:len(m.detailHistory)-1]
					m.selectedInst = prev.selectedInst
					m.selectedSG = prev.selectedSG
					m.selectedVPC = prev.selectedVPC
					m.selectedSubnet = prev.selectedSubnet
					m.detailScroll = prev.detailScroll
					m.detailCursor = prev.detailCursor
				} else {
					m.screen = screenTable
					m.selectedInst = nil
					m.selectedTGWAtt = nil
					m.detailScroll = 0
				}
			case "up":
				if n > 0 {
					if m.detailCursor < 0 {
						m.detailCursor = n - 1
					} else {
						m.detailCursor = (m.detailCursor - 1 + n) % n
					}
				}
			case "down":
				if n > 0 {
					if m.detailCursor < 0 {
						m.detailCursor = 0
					} else {
						m.detailCursor = (m.detailCursor + 1) % n
					}
				}
			case "enter":
				m.navigateFromDetail()
			case "k":
				if m.detailScroll > 0 {
					m.detailScroll--
				}
			case "j":
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
						m.detailCursor = -1
						m.detailHistory = nil
					}
				case viewSG:
					if sg := m.selectedSG_(); sg != nil {
						m.selectedSG = sg
						m.selectedInst, m.selectedVPC, m.selectedSubnet = nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
					}
				case viewVPC:
					if vpc := m.selectedVPC_(); vpc != nil {
						m.selectedVPC = vpc
						m.selectedInst, m.selectedSG, m.selectedSubnet = nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
					}
				case viewSubnet:
					if subnet := m.selectedSubnet_(); subnet != nil {
						m.selectedSubnet = subnet
						m.selectedInst, m.selectedSG, m.selectedVPC = nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
					}
				case viewTGW:
					if att := m.selectedTGWAtt_(); att != nil {
						m.selectedTGWAtt = att
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet = nil, nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
					}
				}
				return m, nil
			case "c":
				if m.view == viewSubnet {
					if subnet := m.selectedSubnet_(); subnet != nil {
						m.connectivitySrcSubnet = subnet
						m.connectivitySelectedRoute = nil
						m.connectivityResult = nil
						m.connectivityCursor = 0
						m.detailScroll = 0
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
				m.enis = nil
				ids := selectedRegionIDs(m.regions)
				return m, tea.Batch(m.spinner.Tick, fetchInstancesWithRegions(ids), fetchSGWithRegions(ids), fetchVPCsWithRegions(ids), fetchTGWsWithRegions(ids), fetchRouteTablesWithRegions(ids), fetchENIsWithRegions(ids))
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
			case "right":
				if max := m.maxColOffset(); m.colOffset < max {
					m.colOffset++
					m.table = m.buildCurrentTable()
				}
				return m, nil
			case "left":
				if m.colOffset > 0 {
					m.colOffset--
					m.table = m.buildCurrentTable()
				}
				return m, nil
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

// detailInteractiveFieldCount returns how many arrow-navigable fields the current detail view has.
// EC2: VPC ID(0), Subnet ID(1), SG(2..)
func (m *Model) detailInteractiveFieldCount() int {
	if m.selectedInst != nil {
		return 2 + len(m.selectedInst.SecurityGroups)
	}
	return 0
}

// navigateFromDetail navigates to the resource pointed to by the current detailCursor.
func (m *Model) navigateFromDetail() {
	if m.selectedInst == nil || m.detailCursor < 0 {
		return
	}
	// 현재 상태를 히스토리에 push
	snapshot := detailSnapshot{
		selectedInst:   m.selectedInst,
		selectedSG:     m.selectedSG,
		selectedVPC:    m.selectedVPC,
		selectedSubnet: m.selectedSubnet,
		detailScroll:   m.detailScroll,
		detailCursor:   m.detailCursor,
	}

	switch m.detailCursor {
	case 0: // VPC ID
		for i := range m.vpcs {
			if m.vpcs[i].VpcID == m.selectedInst.VpcID {
				m.detailHistory = append(m.detailHistory, snapshot)
				m.selectedVPC = &m.vpcs[i]
				m.selectedInst, m.selectedSG, m.selectedSubnet = nil, nil, nil
				m.detailScroll = 0
				m.detailCursor = -1
				return
			}
		}
	case 1: // Subnet ID
		for i := range m.subnets {
			if m.subnets[i].SubnetID == m.selectedInst.SubnetID {
				m.detailHistory = append(m.detailHistory, snapshot)
				m.selectedSubnet = &m.subnets[i]
				m.selectedInst, m.selectedSG, m.selectedVPC = nil, nil, nil
				m.detailScroll = 0
				m.detailCursor = -1
				return
			}
		}
	default: // SG (cursor 2+)
		sgIdx := m.detailCursor - 2
		if sgIdx < len(m.selectedInst.SecurityGroups) {
			targetID := m.selectedInst.SecurityGroups[sgIdx].ID
			for i := range m.groups {
				if m.groups[i].GroupID == targetID {
					m.detailHistory = append(m.detailHistory, snapshot)
					m.selectedSG = &m.groups[i]
					m.selectedInst, m.selectedVPC, m.selectedSubnet = nil, nil, nil
					m.detailScroll = 0
					m.detailCursor = -1
					return
				}
			}
		}
	}
}

// selectedInstance returns the Instance at the current table cursor position.
func (m *Model) selectedInstance() *awsclient.Instance {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedInstances) {
		return &m.displayedInstances[c]
	}
	return nil
}

// selectedSG_ returns the SecurityGroup at the current table cursor position.
func (m *Model) selectedSG_() *awsclient.SecurityGroup {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedGroups) {
		return &m.displayedGroups[c]
	}
	return nil
}

func (m *Model) selectedVPC_() *awsclient.VPC {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedVPCs) {
		return &m.displayedVPCs[c]
	}
	return nil
}

func (m *Model) selectedSubnet_() *awsclient.Subnet {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedSubnets) {
		return &m.displayedSubnets[c]
	}
	return nil
}

// connectivityPickerRoutes returns the TGW routes from the source subnet's route table (phase 1).
func (m *Model) connectivityPickerRoutes() []awsclient.SubnetTGWRoute {
	if m.connectivitySrcSubnet == nil {
		return nil
	}
	routes := awsclient.TGWRoutesForSubnet(m.routeTables, m.connectivitySrcSubnet.SubnetID, m.connectivitySrcSubnet.VpcID)

	filter := strings.ToLower(m.input.Value())
	if filter == "" {
		return routes
	}
	var result []awsclient.SubnetTGWRoute
	for _, r := range routes {
		if matchAll([]string{filter}, r.DestinationCIDR, r.GatewayID, r.RouteTableID) {
			result = append(result, r)
		}
	}
	return result
}

// connectivityPickerSubnets returns subnets covered by the selected route's CIDR (phase 2).
func (m *Model) connectivityPickerSubnets() []awsclient.Subnet {
	if m.connectivitySelectedRoute == nil {
		return nil
	}
	routeCIDR := m.connectivitySelectedRoute.DestinationCIDR

	// 목적지 VPC CIDR 조회용 맵
	vpcCIDR := map[string]string{}
	for _, v := range m.vpcs {
		vpcCIDR[v.VpcID] = v.CidrBlock
	}

	filter := strings.ToLower(m.input.Value())
	var result []awsclient.Subnet
	for _, s := range m.subnets {
		if m.connectivitySrcSubnet != nil && s.SubnetID == m.connectivitySrcSubnet.SubnetID {
			continue
		}
		// 선택된 route CIDR이 이 서브넷의 VPC CIDR을 포함하는지 확인
		if !awsclient.CIDRCovers(routeCIDR, vpcCIDR[s.VpcID]) {
			continue
		}
		if filter == "" || matchAll([]string{filter}, s.Profile, s.Name, s.SubnetID, s.VpcID, s.CidrBlock, s.AvailabilityZone, s.Region) {
			result = append(result, s)
		}
	}
	return result
}

func (m *Model) selectedTGWAtt_() *awsclient.TGWAttachment {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedAttachments) {
		return &m.displayedAttachments[c]
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
