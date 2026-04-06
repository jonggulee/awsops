package ui

import (
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

type instanceTypeSpecsLoadedMsg struct {
	specs map[string]awsclient.InstanceTypeSpec
	err   error
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

type certsLoadedMsg struct {
	certs []awsclient.Certificate
	errs  []error
}

type eksLoadedMsg struct {
	clusters []awsclient.EKSCluster
	errs     []error
}

type route53LoadedMsg struct {
	records []awsclient.Route53Record
	errs    []error
}

type albLoadedMsg struct {
	lbs  []awsclient.LoadBalancer
	errs []error
}

type s3LoadedMsg struct {
	buckets []awsclient.S3Bucket
	errs    []error
}

type s3TagsLoadedMsg struct {
	tags map[string]string
	err  error
}

type rdsLoadedMsg struct {
	instances []awsclient.DBInstance
	errs      []error
}

type elastiCacheLoadedMsg struct {
	clusters []awsclient.ElastiCacheCluster
	errs     []error
}

type elastiCacheSubnetGroupLoadedMsg struct {
	subnetIDs []string
	err       error
}

type rdsENIsLoadedMsg struct {
	enis         []awsclient.ENI
	primaryENIID string
	err          error
}

type listenersLoadedMsg struct {
	listeners []awsclient.Listener
	err       error
}

type tgListLoadedMsg struct {
	tgs []awsclient.TargetGroup
	err error
}

type rulesLoadedMsg struct {
	rules []awsclient.ListenerRule
	err   error
}

type targetHealthLoadedMsg struct {
	targets []awsclient.TargetEntry
	err     error
}

type mapRulesLoadedMsg struct {
	listenerARN string
	rules       []awsclient.ListenerRule
}

type mapTargetHealthLoadedMsg struct {
	tgARN   string
	targets []awsclient.TargetEntry
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
	viewACM
	viewENI
	viewEKS
	viewRoute53
	viewALB
	viewRDS
	viewS3
	viewElastiCache
	viewAccount
)

type screen int

const (
	screenTable        screen = iota
	screenDetail              // d 눌렀을 때 상세 화면
	screenRegion              // R 눌렀을 때 리전 선택 화면
	screenConnectivity        // c 눌렀을 때 연결 체크 화면
	screenTagPicker           // :tag 명령어로 진입하는 태그 피커
	screenMap                 // m 눌렀을 때 ELB resource map 화면
)

var viewNames = map[viewType]string{
	viewEC2:     "ec2",
	viewSG:      "sg",
	viewVPC:     "vpc",
	viewSubnet:  "subnet",
	viewTGW:     "tgw",
	viewACM:     "acm",
	viewENI:     "eni",
	viewEKS:     "eks",
	viewRoute53: "route53",
	viewALB:     "elb",
	viewRDS:          "rds",
	viewS3:           "s3",
	viewElastiCache:  "elasticache",
	viewAccount:      "profile",
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
	sortDomainName
	sortExpiry
	sortCertStatus
	sortENIID
	sortDescription
	sortInterfaceType
	sortVersion
	sortZoneName
	sortRecordType
	sortTTL
	sortZoneType
	sortLBType
	sortScheme
	sortEngine
	sortDBClass
	sortCreateTime
	sortVersioning
	sortPublicAccess
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

// ACM:    1=Profile 2=DomainName 3=Status 4=Type 5=Expiry 6=Region
var acmSortCols = []sortCol{sortProfile, sortDomainName, sortCertStatus, sortType, sortExpiry, sortRegion}

// ENI:    1=Profile 2=ENIID 3=Name 4=Status 5=Type 6=PrivateIP 7=InstanceID 8=VpcID 9=SubnetID 10=Region
var eniSortCols = []sortCol{sortProfile, sortENIID, sortName, sortState, sortInterfaceType, sortPrivateIP, sortInstanceID, sortVpcID, sortSubnetID, sortRegion}

// EKS:    1=Profile 2=Name 3=Status 4=Version 5=VpcID 6=- 7=Region
var eksSortCols = []sortCol{sortProfile, sortName, sortState, sortVersion, sortVpcID, sortNone, sortRegion}

// RDS:    1=Profile 2=Identifier 3=Engine 4=Class 5=Status 6=VpcID 7=Endpoint 8=Region
var rdsSortCols = []sortCol{sortProfile, sortName, sortEngine, sortDBClass, sortState, sortVpcID, sortNone, sortRegion}

// S3:            1=Profile 2=Name 3=Region 4=Versioning 5=PublicAccess 6=Created
var s3SortCols = []sortCol{sortProfile, sortName, sortRegion, sortVersioning, sortPublicAccess, sortCreateTime}

// ElastiCache:   1=Profile 2=ID 3=Engine 4=NodeType 5=Status 6=Nodes 7=Endpoint 8=Region
var elastiCacheSortCols = []sortCol{sortProfile, sortName, sortEngine, sortType, sortState, sortNone, sortNone, sortRegion}

var sortColNames = map[sortCol]string{
	sortVersion: "Version",
	sortDomainName: "Domain",
	sortExpiry:     "Expiry",
	sortCertStatus: "Status",
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
	selectedInst        *awsclient.Instance
	selectedSG          *awsclient.SecurityGroup
	selectedVPC         *awsclient.VPC
	selectedSubnet      *awsclient.Subnet
	selectedENI         *awsclient.ENI
	selectedEKS         *awsclient.EKSCluster
	selectedRoute53     *awsclient.Route53Record
	selectedALB         *awsclient.LoadBalancer
	selectedRDS          *awsclient.DBInstance
	selectedElastiCache  *awsclient.ElastiCacheCluster
	selectedListener     *awsclient.Listener
	selectedRule         *awsclient.ListenerRule
	selectedTargetGroup  *awsclient.TargetGroup
	detailScroll         int
	detailCursor         int
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
	selectedCert              *awsclient.Certificate
	connectivitySrcSubnet     *awsclient.Subnet
	connectivitySelectedRoute *awsclient.SubnetTGWRoute // nil = phase1(route 선택), non-nil = phase2(subnet 선택)
	connectivityResult        *awsclient.ConnectivityResult
	connectivityCursor  int
	routeTables         []awsclient.VPCRouteTable
	enis                []awsclient.ENI
	certs               []awsclient.Certificate
	profileToAccount    map[string]string // profile → accountID
	accountToProfile    map[string]string // accountID → profile
	fetchErr         []error
	loading        bool
	width          int
	height         int
	regions        []regionEntry
	commandCursor        int           // 리소스 피커 커서 위치
	tagPickerStep        int           // 0 = key 선택, 1 = value 선택
	tagPickerKey         string        // step 1에서 선택된 태그 키
	tagPickerCursor      int           // 태그 피커 커서
	regionsPrev          []regionEntry // 리전 화면 진입 시 스냅샷 (취소 복원용)
	regionCursor         int
	regionErr            bool // 선택 없이 enter 시 경고 표시
	regionConfirmDiscard bool // 변경 후 esc/q 시 확인창
	sortBy         sortCol
	sortAsc        bool
	detailScroll   int
	detailCursor   int              // detail 화면에서 선택된 interactive 필드 인덱스 (-1 = 없음)
	detailHistory  []detailSnapshot // 뒤로가기 스택
	colOffset      int // 가로 스크롤: 첫 번째로 보이는 컬럼 인덱스
	// 현재 테이블에 표시 중인 데이터 (커서 기반 선택에 사용)
	displayedInstances    []awsclient.Instance
	displayedGroups       []awsclient.SecurityGroup
	displayedVPCs         []awsclient.VPC
	displayedSubnets      []awsclient.Subnet
	displayedAttachments  []awsclient.TGWAttachment
	displayedCerts        []awsclient.Certificate
	displayedENIs         []awsclient.ENI
	selectedENI           *awsclient.ENI
	eksClusters           []awsclient.EKSCluster
	displayedEKS          []awsclient.EKSCluster
	selectedEKS           *awsclient.EKSCluster
	instanceTypeSpecs     map[string]awsclient.InstanceTypeSpec
	route53Records        []awsclient.Route53Record
	displayedRoute53      []awsclient.Route53Record
	selectedRoute53       *awsclient.Route53Record
	loadBalancers         []awsclient.LoadBalancer
	displayedALBs         []awsclient.LoadBalancer
	selectedALB           *awsclient.LoadBalancer
	rdsInstances          []awsclient.DBInstance
	displayedRDS          []awsclient.DBInstance
	selectedRDS           *awsclient.DBInstance
	rdsENIs               []awsclient.ENI // nil = loading, non-nil = loaded
	rdsENIPrimaryID       string
	s3Buckets             []awsclient.S3Bucket
	displayedS3           []awsclient.S3Bucket
	selectedS3            *awsclient.S3Bucket
	s3Tags                map[string]string // nil = loading, non-nil = loaded
	elastiCacheClusters   []awsclient.ElastiCacheCluster
	displayedElastiCache  []awsclient.ElastiCacheCluster
	selectedElastiCache   *awsclient.ElastiCacheCluster
	elastiCacheSubnetIDs  []string // nil = loading, non-nil = loaded
	// ELB lazy-loaded detail data (nil = loading, non-nil = loaded)
	albListeners          []awsclient.Listener    // listeners for current selectedALB
	albTargetGroups       []awsclient.TargetGroup // TGs for current selectedALB
	listenerRules         []awsclient.ListenerRule // rules for current selectedListener (nil for NLB)
	tgTargets             []awsclient.TargetEntry  // target health for current selectedTargetGroup
	selectedListener      *awsclient.Listener
	selectedRule          *awsclient.ListenerRule
	selectedTargetGroup   *awsclient.TargetGroup
	// Resource Map 데이터
	mapRules        map[string][]awsclient.ListenerRule // listenerARN → rules
	mapTargetHealth map[string][]awsclient.TargetEntry  // tgARN → targets
	mapPending      int                                 // 로딩 중인 fetch 수 (0 = 완료)
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
		fetchCertificatesWithRegions(regions),
		fetchEKSWithRegions(regions),
		fetchRoute53(),
		fetchALBWithRegions(regions),
		fetchRDSWithRegions(regions),
		fetchS3Buckets(),
		fetchElastiCacheWithRegions(regions),
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
		return m, fetchInstanceTypeSpecs(m.instances)

	case instanceTypeSpecsLoadedMsg:
		if msg.err == nil && msg.specs != nil {
			m.instanceTypeSpecs = msg.specs
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

	case certsLoadedMsg:
		m.certs = msg.certs
		m.fetchErr = append(m.fetchErr, msg.errs...)

	case eksLoadedMsg:
		m.eksClusters = msg.clusters
		m.fetchErr = append(m.fetchErr, msg.errs...)
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case route53LoadedMsg:
		m.route53Records = msg.records
		m.fetchErr = append(m.fetchErr, msg.errs...)
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case albLoadedMsg:
		m.loadBalancers = msg.lbs
		m.fetchErr = append(m.fetchErr, msg.errs...)
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case rdsLoadedMsg:
		m.rdsInstances = msg.instances
		m.fetchErr = append(m.fetchErr, msg.errs...)
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case s3LoadedMsg:
		m.s3Buckets = msg.buckets
		m.fetchErr = append(m.fetchErr, msg.errs...)
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case s3TagsLoadedMsg:
		if msg.err == nil {
			m.s3Tags = msg.tags
		} else {
			m.s3Tags = map[string]string{}
		}

	case elastiCacheLoadedMsg:
		m.elastiCacheClusters = msg.clusters
		m.fetchErr = append(m.fetchErr, msg.errs...)
		if !m.loading {
			m.table = m.buildCurrentTable()
		}

	case elastiCacheSubnetGroupLoadedMsg:
		if msg.err == nil {
			m.elastiCacheSubnetIDs = msg.subnetIDs
		} else {
			m.elastiCacheSubnetIDs = []string{}
		}

	case rdsENIsLoadedMsg:
		if msg.err == nil {
			m.rdsENIs = msg.enis
			m.rdsENIPrimaryID = msg.primaryENIID
		} else {
			m.rdsENIs = []awsclient.ENI{}
			m.rdsENIPrimaryID = ""
		}

	case listenersLoadedMsg:
		m.albListeners = msg.listeners // nil on error → keep showing spinner
		if msg.err == nil && m.albListeners == nil {
			m.albListeners = []awsclient.Listener{} // empty, not nil
		}
		m.detailCursor = -1 // 커서 리셋 (리스너 로드 완료)

	case tgListLoadedMsg:
		m.albTargetGroups = msg.tgs
		if msg.err == nil && m.albTargetGroups == nil {
			m.albTargetGroups = []awsclient.TargetGroup{}
		}

	case rulesLoadedMsg:
		m.listenerRules = msg.rules
		if msg.err == nil && m.listenerRules == nil {
			m.listenerRules = []awsclient.ListenerRule{}
		}
		m.detailCursor = -1

	case targetHealthLoadedMsg:
		m.tgTargets = msg.targets
		if msg.err == nil && m.tgTargets == nil {
			m.tgTargets = []awsclient.TargetEntry{}
		}

	case mapRulesLoadedMsg:
		if m.mapRules != nil {
			m.mapRules[msg.listenerARN] = msg.rules
		}
		m.mapPending--

	case mapTargetHealthLoadedMsg:
		if m.mapTargetHealth != nil {
			m.mapTargetHealth[msg.tgARN] = msg.targets
		}
		m.mapPending--

	case routeTablesLoadedMsg:
		m.routeTables = msg.tables
		m.fetchErr = append(m.fetchErr, msg.errs...)

	case accountIDsLoadedMsg:
		m.profileToAccount = msg.profileToAccount
		m.accountToProfile = msg.accountToProfile

	case tea.KeyMsg:
		// 리전 선택 화면
		if m.screen == screenRegion {
			// 확인창이 떠 있을 때는 y/n/esc만 처리
			if m.regionConfirmDiscard {
				switch msg.String() {
				case "y", "Y":
					m.regions = m.regionsPrev
					m.regionsPrev = nil
					m.regionConfirmDiscard = false
					m.regionErr = false
					m.screen = screenTable
				case "n", "N", "esc":
					m.regionConfirmDiscard = false
				}
				return m, nil
			}

			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "q":
				if regionsChanged(m.regions, m.regionsPrev) {
					// 변경 사항 있음 → 확인창 표시
					m.regionConfirmDiscard = true
					m.regionErr = false
				} else {
					// 변경 없음 → 바로 나가기
					m.regionsPrev = nil
					m.screen = screenTable
				}
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
				m.regionErr = false
			case "a":
				for i := range m.regions {
					m.regions[i].selected = true
				}
			case "n":
				for i := range m.regions {
					m.regions[i].selected = false
				}
				m.regionErr = false
			case "enter":
				ids := selectedRegionIDs(m.regions)
				if len(ids) == 0 {
					m.regionErr = true
					return m, nil
				}
				m.regionErr = false
				m.regionsPrev = nil
				m.regionConfirmDiscard = false
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
				m.certs = nil
				m.eksClusters = nil
				m.route53Records = nil
				m.loadBalancers = nil
				m.rdsInstances = nil
				m.s3Buckets = nil
				m.elastiCacheClusters = nil
				m.fetchErr = nil
				return m, tea.Batch(m.spinner.Tick, fetchInstancesWithRegions(ids), fetchSGWithRegions(ids), fetchVPCsWithRegions(ids), fetchTGWsWithRegions(ids), fetchRouteTablesWithRegions(ids), fetchENIsWithRegions(ids), fetchCertificatesWithRegions(ids), fetchEKSWithRegions(ids), fetchRoute53(), fetchALBWithRegions(ids), fetchRDSWithRegions(ids), fetchS3Buckets(), fetchElastiCacheWithRegions(ids))
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
				// 필터로 리스트가 줄었을 때 커서가 범위를 벗어나지 않도록 클램핑
				if len(routes) == 0 {
					m.connectivityCursor = 0
				} else if m.connectivityCursor >= len(routes) {
					m.connectivityCursor = len(routes) - 1
				}
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
				// 필터로 리스트가 줄었을 때 커서 클램핑
				if len(subnets) == 0 {
					m.connectivityCursor = 0
				} else if m.connectivityCursor >= len(subnets) {
					m.connectivityCursor = len(subnets) - 1
				}
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

		// 태그 피커 화면
		if m.screen == screenTagPicker {
			query := strings.ToLower(m.input.Value())

			// 현재 단계에 맞는 항목 목록 계산
			var items []string
			if m.tagPickerStep == 0 {
				for _, k := range collectTagKeys(m) {
					if query == "" || strings.Contains(strings.ToLower(k), query) {
						items = append(items, k)
					}
				}
			} else {
				items = append(items, "(any)") // key= 형식으로 해당 태그 존재 여부만 필터
				for _, v := range collectTagValues(m, m.tagPickerKey) {
					if query == "" || strings.Contains(strings.ToLower(v), query) {
						items = append(items, v)
					}
				}
			}

			// 커서 클램핑
			if len(items) == 0 {
				m.tagPickerCursor = 0
			} else if m.tagPickerCursor >= len(items) {
				m.tagPickerCursor = len(items) - 1
			}

			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				if m.tagPickerStep == 1 {
					m.tagPickerStep = 0
					m.tagPickerCursor = 0
					m.input.SetValue("")
				} else {
					m.screen = screenTable
					m.input.Blur()
					m.input.SetValue("")
				}
			case "up", "k":
				if m.tagPickerCursor > 0 {
					m.tagPickerCursor--
				}
			case "down", "j":
				if m.tagPickerCursor < len(items)-1 {
					m.tagPickerCursor++
				}
			case "enter":
				if len(items) > 0 && m.tagPickerCursor < len(items) {
					if m.tagPickerStep == 0 {
						// key 선택 → value 선택 단계로
						m.tagPickerKey = items[m.tagPickerCursor]
						m.tagPickerStep = 1
						m.tagPickerCursor = 0
						m.input.SetValue("")
					} else {
						// value 선택 → 필터 적용 후 테이블로 복귀
						var filter string
						if items[m.tagPickerCursor] == "(any)" {
							filter = m.tagPickerKey + "="
						} else {
							filter = m.tagPickerKey + "=" + items[m.tagPickerCursor]
						}
						m.filters = append(m.filters, filter)
						m.screen = screenTable
						m.input.Blur()
						m.input.SetValue("")
						m.tagPickerStep = 0
						m.tagPickerKey = ""
						m.tagPickerCursor = 0
						m.table = m.buildCurrentTable()
					}
				}
			default:
				prev := m.input.Value()
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				if m.input.Value() != prev {
					m.tagPickerCursor = 0
				}
				return m, cmd
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
					m.selectedENI = prev.selectedENI
					m.selectedEKS = prev.selectedEKS
					m.selectedRoute53 = prev.selectedRoute53
					m.selectedALB = prev.selectedALB
					m.selectedRDS = prev.selectedRDS
					m.selectedElastiCache = prev.selectedElastiCache
					m.selectedListener = prev.selectedListener
					m.selectedRule = prev.selectedRule
					m.selectedTargetGroup = prev.selectedTargetGroup
					m.detailScroll = prev.detailScroll
					m.detailCursor = prev.detailCursor
					// RDS로 돌아올 때 ENI re-fetch
					if m.selectedRDS != nil {
						m.rdsENIs = nil
						return m, fetchENIsForRDS(m.selectedRDS.Endpoint, m.selectedRDS.SubnetIDs, m.enis)
					}
					// ElastiCache로 돌아올 때 subnet group re-fetch
					if ec := m.selectedElastiCache; ec != nil {
						m.elastiCacheSubnetIDs = nil
						if ec.SubnetGroupName != "" {
							return m, fetchElastiCacheSubnetGroup(ec.Profile, ec.Region, ec.SubnetGroupName)
						}
						m.elastiCacheSubnetIDs = []string{}
					}
				} else {
					m.screen = screenTable
					m.selectedInst = nil
					m.selectedTGWAtt = nil
					m.selectedCert = nil
					m.selectedENI = nil
					m.selectedEKS = nil
					m.selectedRoute53 = nil
					m.selectedALB = nil
					m.selectedRDS = nil
					m.selectedElastiCache = nil
					m.elastiCacheSubnetIDs = nil
					m.selectedListener = nil
					m.selectedRule = nil
					m.selectedTargetGroup = nil
					m.rdsENIs = nil
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
				if cmd := m.navigateFromDetail(); cmd != nil {
					return m, cmd
				}
			case "k":
				if m.selectedEKS == nil && m.detailScroll > 0 {
					m.detailScroll--
				}
			case "j":
				if m.selectedEKS == nil && m.detailScroll < m.detailMaxScroll() {
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
			case "m":
				// ALB resource map 진입 (listeners/TGs가 로드된 경우에만)
				if m.selectedALB != nil && len(m.albListeners) > 0 && m.albTargetGroups != nil {
					m.mapRules = make(map[string][]awsclient.ListenerRule)
					m.mapTargetHealth = make(map[string][]awsclient.TargetEntry)
					var cmds []tea.Cmd
					pending := 0
					for _, li := range m.albListeners {
						if li.IsALB() {
							pending++
							li := li
							cmds = append(cmds, fetchMapRulesForListener(li.Profile, li.Region, li.ARN))
						} else {
							m.mapRules[li.ARN] = []awsclient.ListenerRule{}
						}
					}
					for _, tg := range m.albTargetGroups {
						pending++
						tg := tg
						cmds = append(cmds, fetchMapTargetHealthForTG(tg.Profile, tg.Region, tg.ARN))
					}
					m.mapPending = pending
					m.screen = screenMap
					m.detailScroll = 0
					return m, tea.Batch(cmds...)
				}
			}
			return m, nil
		}

		// Resource Map 화면
		if m.screen == screenMap {
			pageSize := m.height / 2
			if pageSize < 1 {
				pageSize = 1
			}
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "q", "m":
				m.screen = screenDetail
				m.detailScroll = 0
			case "k", "up":
				if m.detailScroll > 0 {
					m.detailScroll--
				}
			case "j", "down":
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
			case "d", "enter":
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
				case viewACM:
					if cert := m.selectedCert_(); cert != nil {
						m.selectedCert = cert
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet, m.selectedTGWAtt, m.selectedENI = nil, nil, nil, nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
					}
				case viewENI:
					if eni := m.selectedENI_(); eni != nil {
						m.selectedENI = eni
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet, m.selectedTGWAtt, m.selectedCert = nil, nil, nil, nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
					}
				case viewEKS:
					if cluster := m.selectedEKS_(); cluster != nil {
						m.selectedEKS = cluster
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet, m.selectedTGWAtt, m.selectedCert, m.selectedENI = nil, nil, nil, nil, nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
					}
				case viewRoute53:
					if rec := m.selectedRoute53_(); rec != nil {
						m.selectedRoute53 = rec
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet, m.selectedTGWAtt, m.selectedCert, m.selectedENI, m.selectedEKS, m.selectedALB = nil, nil, nil, nil, nil, nil, nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
					}
				case viewALB:
					if lb := m.selectedALB_(); lb != nil {
						m.selectedALB = lb
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet, m.selectedTGWAtt, m.selectedCert, m.selectedENI, m.selectedEKS, m.selectedRoute53, m.selectedRDS = nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
						m.selectedListener, m.selectedRule, m.selectedTargetGroup = nil, nil, nil
						m.albListeners = nil
						m.albTargetGroups = nil
						m.listenerRules = nil
						m.tgTargets = nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
						return m, tea.Batch(
							fetchListenersForLB(lb.Profile, lb.Region, lb.ARN),
							fetchTargetGroupsForLB(lb.Profile, lb.Region, lb.ARN),
						)
					}
				case viewRDS:
					if db := m.selectedRDS_(); db != nil {
						m.selectedRDS = db
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet, m.selectedTGWAtt, m.selectedCert, m.selectedENI, m.selectedEKS, m.selectedRoute53, m.selectedALB = nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
						m.rdsENIs = nil // trigger lazy fetch
						return m, fetchENIsForRDS(db.Endpoint, db.SubnetIDs, m.enis)
					}
				case viewS3:
					if b := m.selectedS3_(); b != nil {
						m.selectedS3 = b
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet, m.selectedTGWAtt, m.selectedCert, m.selectedENI, m.selectedEKS, m.selectedRoute53, m.selectedALB, m.selectedRDS = nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
						m.s3Tags = nil // trigger lazy fetch
						return m, fetchS3Tags(b.Profile, b.Name)
					}
				case viewElastiCache:
					if ec := m.selectedElastiCache_(); ec != nil {
						m.selectedElastiCache = ec
						m.selectedInst, m.selectedSG, m.selectedVPC, m.selectedSubnet, m.selectedTGWAtt, m.selectedCert, m.selectedENI, m.selectedEKS, m.selectedRoute53, m.selectedALB, m.selectedRDS, m.selectedS3 = nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil
						m.screen = screenDetail
						m.detailScroll = 0
						m.detailCursor = -1
						m.detailHistory = nil
						m.elastiCacheSubnetIDs = nil // trigger lazy fetch
						if ec.SubnetGroupName != "" {
							return m, fetchElastiCacheSubnetGroup(ec.Profile, ec.Region, ec.SubnetGroupName)
						}
						m.elastiCacheSubnetIDs = []string{}
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
				m.certs = nil
				m.eksClusters = nil
				m.route53Records = nil
				m.loadBalancers = nil
				m.rdsInstances = nil
				m.s3Buckets = nil
				m.elastiCacheClusters = nil
				ids := selectedRegionIDs(m.regions)
				return m, tea.Batch(m.spinner.Tick, fetchInstancesWithRegions(ids), fetchSGWithRegions(ids), fetchVPCsWithRegions(ids), fetchTGWsWithRegions(ids), fetchRouteTablesWithRegions(ids), fetchENIsWithRegions(ids), fetchCertificatesWithRegions(ids), fetchEKSWithRegions(ids), fetchRoute53(), fetchALBWithRegions(ids), fetchRDSWithRegions(ids), fetchS3Buckets(), fetchElastiCacheWithRegions(ids))
			case "R":
				// 취소 시 복원할 수 있도록 현재 상태 저장
				prev := make([]regionEntry, len(m.regions))
				copy(prev, m.regions)
				m.regionsPrev = prev
				m.screen = screenRegion
				m.regionCursor = 0
				return m, nil
			case "t":
				m.screen = screenTagPicker
				m.tagPickerStep = 0
				m.tagPickerKey = ""
				m.tagPickerCursor = 0
				m.input.Placeholder = "filter..."
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case "/":
				m.mode = modeSearch
				m.input.Placeholder = "type to search..."
				m.input.SetValue("")
				m.input.Focus()
				return m, textinput.Blink
			case ":":
				m.mode = modeCommand
				m.commandCursor = 0
				m.input.Placeholder = ""
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
			items := filteredPickerItems(m.input.Value())
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				if m.input.Value() != "" {
					m.input.SetValue("")
					m.commandCursor = 0
				} else {
					m.mode = modeNormal
					m.input.Blur()
				}
				return m, nil
			case "up":
				if m.commandCursor > 0 {
					m.commandCursor--
				}
				return m, nil
			case "down":
				if m.commandCursor < len(items)-1 {
					m.commandCursor++
				}
				return m, nil
			case "enter":
				if len(items) > 0 && m.commandCursor < len(items) {
					selected := items[m.commandCursor].cmd
					m.mode = modeNormal
					m.commandCursor = 0
					m.input.Blur()
					m.input.SetValue("")
					m.applyCommand(selected)
				}
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
			if m.mode == modeCommand && m.input.Value() != prevVal {
				// 필터 바뀌면 커서 리셋
				m.commandCursor = 0
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
	if m.selectedEKS != nil {
		return len(m.selectedEKS.Nodes) + len(m.selectedEKS.SubnetIDs) + len(m.selectedEKS.SecurityGroupIDs)
	}
	if m.selectedRoute53 != nil && m.selectedRoute53.AliasTarget != "" {
		// alias target이 로드된 ALB와 매칭될 때만 커서 활성화
		if m.lookupALBByDNS(m.selectedRoute53.AliasTarget) != nil {
			return 1
		}
	}
	if m.selectedALB != nil {
		listenerCount := len(m.albListeners) // 0 if nil (loading)
		return listenerCount + len(m.selectedALB.SecurityGroupIDs)
	}
	if m.selectedListener != nil {
		if !m.selectedListener.IsALB() {
			// NLB: default action TG navigable (1 item)
			for _, a := range m.selectedListener.DefaultActions {
				if a.Type == "forward" && a.TargetGroupARN != "" {
					return 1
				}
			}
			return 0
		}
		return len(m.listenerRules) // 0 if nil (loading)
	}
	if m.selectedRule != nil {
		return len(m.selectedRule.ForwardTGARNs())
	}
	if m.selectedTargetGroup != nil {
		return 0
	}
	if m.selectedRDS != nil {
		return len(m.selectedRDS.SubnetIDs) + len(m.selectedRDS.SecurityGroupIDs) + len(m.rdsENIs)
	}
	if m.selectedS3 != nil {
		return 0
	}
	if m.selectedElastiCache != nil {
		subnetCount := len(m.elastiCacheSubnetIDs) // 0 if nil (loading)
		return subnetCount + len(m.selectedElastiCache.SecurityGroupIDs)
	}
	return 0
}

// lookupTGByARN finds a TargetGroup in albTargetGroups by ARN.
func (m *Model) lookupTGByARN(arn string) *awsclient.TargetGroup {
	for i := range m.albTargetGroups {
		if m.albTargetGroups[i].ARN == arn {
			return &m.albTargetGroups[i]
		}
	}
	return nil
}

// lookupALBByDNS returns a pointer to the LoadBalancer whose DNSName matches the given target.
func (m *Model) lookupALBByDNS(dnsName string) *awsclient.LoadBalancer {
	norm := strings.ToLower(strings.TrimSuffix(dnsName, "."))
	for i := range m.loadBalancers {
		if m.loadBalancers[i].DNSNameNorm() == norm {
			return &m.loadBalancers[i]
		}
	}
	return nil
}

// navigateFromDetail navigates to the resource pointed to by the current detailCursor.
// Returns a tea.Cmd if a lazy fetch is required (e.g., rules, target health).
func (m *Model) navigateFromDetail() tea.Cmd {
	if m.detailCursor < 0 {
		return nil
	}

	// ALB detail: Listeners (0..n-1) → Listener 상세, SGs (n..) → SG 상세
	if m.selectedALB != nil {
		listenerCount := len(m.albListeners)
		snapshot := detailSnapshot{
			selectedALB:  m.selectedALB,
			detailScroll: m.detailScroll,
			detailCursor: m.detailCursor,
		}
		if m.detailCursor < listenerCount {
			// Listener 상세 진입 + lazy fetch
			li := m.albListeners[m.detailCursor]
			m.detailHistory = append(m.detailHistory, snapshot)
			m.selectedListener = &li
			m.selectedALB = nil
			m.listenerRules = nil
			m.detailScroll = 0
			m.detailCursor = -1
			if li.IsALB() {
				return fetchRulesForListener(li.Profile, li.Region, li.ARN)
			}
			// NLB: 룰 없음, 빈 슬라이스로 세팅
			m.listenerRules = []awsclient.ListenerRule{}
			return nil
		}
		// SG 진입 (기존 동작, 인덱스를 listenerCount만큼 shift)
		sgIdx := m.detailCursor - listenerCount
		if sgIdx < len(m.selectedALB.SecurityGroupIDs) {
			sgID := m.selectedALB.SecurityGroupIDs[sgIdx]
			for i := range m.groups {
				if m.groups[i].GroupID == sgID {
					m.detailHistory = append(m.detailHistory, snapshot)
					m.selectedSG = &m.groups[i]
					m.selectedALB = nil
					m.detailScroll = 0
					m.detailCursor = -1
					return nil
				}
			}
		}
		return nil
	}

	// Listener detail → Rule 또는 NLB Default TG
	if m.selectedListener != nil {
		snapshot := detailSnapshot{
			selectedListener: m.selectedListener,
			detailScroll:     m.detailScroll,
			detailCursor:     m.detailCursor,
		}
		if m.selectedListener.IsALB() {
			// ALB: Rule 상세 진입 (no fetch needed, rule data already loaded)
			if m.detailCursor < len(m.listenerRules) {
				rule := m.listenerRules[m.detailCursor]
				m.detailHistory = append(m.detailHistory, snapshot)
				m.selectedRule = &rule
				m.selectedListener = nil
				m.detailScroll = 0
				m.detailCursor = -1
			}
			return nil
		}
		// NLB: Default action TG 진입
		if m.detailCursor == 0 {
			for _, a := range m.selectedListener.DefaultActions {
				if a.Type == "forward" && a.TargetGroupARN != "" {
					if tg := m.lookupTGByARN(a.TargetGroupARN); tg != nil {
						m.detailHistory = append(m.detailHistory, snapshot)
						m.selectedTargetGroup = tg
						m.selectedListener = nil
						m.tgTargets = nil
						m.detailScroll = 0
						m.detailCursor = -1
						return fetchTargetHealthForTG(tg.Profile, tg.Region, tg.ARN)
					}
				}
			}
		}
		return nil
	}

	// Rule detail → Target Group
	if m.selectedRule != nil {
		tgARNs := m.selectedRule.ForwardTGARNs()
		if m.detailCursor < len(tgARNs) {
			tgARN := tgARNs[m.detailCursor]
			if tg := m.lookupTGByARN(tgARN); tg != nil {
				m.detailHistory = append(m.detailHistory, detailSnapshot{
					selectedRule: m.selectedRule,
					detailScroll: m.detailScroll,
					detailCursor: m.detailCursor,
				})
				m.selectedTargetGroup = tg
				m.selectedRule = nil
				m.tgTargets = nil
				m.detailScroll = 0
				m.detailCursor = -1
				return fetchTargetHealthForTG(tg.Profile, tg.Region, tg.ARN)
			}
		}
		return nil
	}

	// Route53 alias → ALB 상세 진입
	if m.selectedRoute53 != nil && m.detailCursor == 0 {
		if lb := m.lookupALBByDNS(m.selectedRoute53.AliasTarget); lb != nil {
			m.detailHistory = append(m.detailHistory, detailSnapshot{
				selectedRoute53: m.selectedRoute53,
				detailScroll:    m.detailScroll,
				detailCursor:    m.detailCursor,
			})
			m.selectedALB = lb
			m.selectedRoute53 = nil
			m.albListeners = nil
			m.albTargetGroups = nil
			m.detailScroll = 0
			m.detailCursor = -1
			return tea.Batch(
				fetchListenersForLB(lb.Profile, lb.Region, lb.ARN),
				fetchTargetGroupsForLB(lb.Profile, lb.Region, lb.ARN),
			)
		}
		return nil
	}

	// EKS 노드 → EC2 인스턴스 상세 진입
	if m.selectedEKS != nil {
		eks := m.selectedEKS
		nodeCount   := len(eks.Nodes)
		subnetCount := len(eks.SubnetIDs)
		snapshot := detailSnapshot{
			selectedEKS:  eks,
			detailScroll: m.detailScroll,
			detailCursor: m.detailCursor,
		}
		switch {
		case m.detailCursor < nodeCount:
			targetID := eks.Nodes[m.detailCursor].InstanceID
			for i := range m.instances {
				if m.instances[i].InstanceID == targetID {
					m.detailHistory = append(m.detailHistory, snapshot)
					m.selectedInst = &m.instances[i]
					m.selectedEKS = nil
					m.detailScroll = 0
					m.detailCursor = -1
					return nil
				}
			}
		case m.detailCursor < nodeCount+subnetCount:
			subnetID := eks.SubnetIDs[m.detailCursor-nodeCount]
			for i := range m.subnets {
				if m.subnets[i].SubnetID == subnetID {
					m.detailHistory = append(m.detailHistory, snapshot)
					m.selectedSubnet = &m.subnets[i]
					m.selectedEKS = nil
					m.detailScroll = 0
					m.detailCursor = -1
					return nil
				}
			}
		default:
			sgIdx := m.detailCursor - nodeCount - subnetCount
			if sgIdx < len(eks.SecurityGroupIDs) {
				sgID := eks.SecurityGroupIDs[sgIdx]
				for i := range m.groups {
					if m.groups[i].GroupID == sgID {
						m.detailHistory = append(m.detailHistory, snapshot)
						m.selectedSG = &m.groups[i]
						m.selectedEKS = nil
						m.detailScroll = 0
						m.detailCursor = -1
						return nil
					}
				}
			}
		}
		return nil
	}

	// RDS Subnet / SG / ENI → 상세 진입
	if m.selectedRDS != nil {
		rds := m.selectedRDS
		subnetCount := len(rds.SubnetIDs)
		sgCount := len(rds.SecurityGroupIDs)
		snapshot := detailSnapshot{
			selectedRDS:  rds,
			detailScroll: m.detailScroll,
			detailCursor: m.detailCursor,
		}
		if m.detailCursor < subnetCount {
			subnetID := rds.SubnetIDs[m.detailCursor]
			for i := range m.subnets {
				if m.subnets[i].SubnetID == subnetID {
					m.detailHistory = append(m.detailHistory, snapshot)
					m.selectedSubnet = &m.subnets[i]
					m.selectedRDS = nil
					m.detailScroll = 0
					m.detailCursor = -1
					return nil
				}
			}
		} else if eniIdx := m.detailCursor - subnetCount - sgCount; eniIdx >= 0 && eniIdx < len(m.rdsENIs) {
			eni := m.rdsENIs[eniIdx]
			m.detailHistory = append(m.detailHistory, snapshot)
			m.selectedENI = &eni
			m.selectedRDS = nil
			m.detailScroll = 0
			m.detailCursor = -1
			return nil
		} else {
			sgIdx := m.detailCursor - subnetCount
			if sgIdx < sgCount {
				sgID := rds.SecurityGroupIDs[sgIdx]
				for i := range m.groups {
					if m.groups[i].GroupID == sgID {
						m.detailHistory = append(m.detailHistory, snapshot)
						m.selectedSG = &m.groups[i]
						m.selectedRDS = nil
						m.detailScroll = 0
						m.detailCursor = -1
						return nil
					}
				}
			}
		}
		return nil
	}

	// ElastiCache: Subnet(0..subnetCount-1), SG(subnetCount..)
	if m.selectedElastiCache != nil {
		ec := m.selectedElastiCache
		subnetCount := len(m.elastiCacheSubnetIDs)
		sgCount := len(ec.SecurityGroupIDs)
		snapshot := detailSnapshot{
			selectedElastiCache: ec,
			detailScroll:        m.detailScroll,
			detailCursor:        m.detailCursor,
		}
		if m.detailCursor < subnetCount {
			subnetID := m.elastiCacheSubnetIDs[m.detailCursor]
			for i := range m.subnets {
				if m.subnets[i].SubnetID == subnetID {
					m.detailHistory = append(m.detailHistory, snapshot)
					m.selectedSubnet = &m.subnets[i]
					m.selectedElastiCache = nil
					m.detailScroll = 0
					m.detailCursor = -1
					return nil
				}
			}
		} else {
			sgIdx := m.detailCursor - subnetCount
			if sgIdx < sgCount {
				sgID := ec.SecurityGroupIDs[sgIdx]
				for i := range m.groups {
					if m.groups[i].GroupID == sgID {
						m.detailHistory = append(m.detailHistory, snapshot)
						m.selectedSG = &m.groups[i]
						m.selectedElastiCache = nil
						m.detailScroll = 0
						m.detailCursor = -1
						return nil
					}
				}
			}
		}
		return nil
	}

	// EC2 detail: VPC(0), Subnet(1), SG(2+)
	if m.selectedInst == nil {
		return nil
	}
	snapshot := detailSnapshot{
		selectedInst:   m.selectedInst,
		selectedSG:     m.selectedSG,
		selectedVPC:    m.selectedVPC,
		selectedSubnet: m.selectedSubnet,
		detailScroll:   m.detailScroll,
		detailCursor:   m.detailCursor,
	}
	switch m.detailCursor {
	case 0:
		for i := range m.vpcs {
			if m.vpcs[i].VpcID == m.selectedInst.VpcID {
				m.detailHistory = append(m.detailHistory, snapshot)
				m.selectedVPC = &m.vpcs[i]
				m.selectedInst, m.selectedSG, m.selectedSubnet = nil, nil, nil
				m.detailScroll = 0
				m.detailCursor = -1
				return nil
			}
		}
	case 1:
		for i := range m.subnets {
			if m.subnets[i].SubnetID == m.selectedInst.SubnetID {
				m.detailHistory = append(m.detailHistory, snapshot)
				m.selectedSubnet = &m.subnets[i]
				m.selectedInst, m.selectedSG, m.selectedVPC = nil, nil, nil
				m.detailScroll = 0
				m.detailCursor = -1
				return nil
			}
		}
	default:
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
					return nil
				}
			}
		}
	}
	return nil
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

func (m *Model) selectedCert_() *awsclient.Certificate {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedCerts) {
		return &m.displayedCerts[c]
	}
	return nil
}

func (m *Model) selectedENI_() *awsclient.ENI {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedENIs) {
		return &m.displayedENIs[c]
	}
	return nil
}

func (m *Model) selectedEKS_() *awsclient.EKSCluster {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedEKS) {
		return &m.displayedEKS[c]
	}
	return nil
}

func (m *Model) selectedS3_() *awsclient.S3Bucket {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedS3) {
		return &m.displayedS3[c]
	}
	return nil
}

func (m *Model) selectedElastiCache_() *awsclient.ElastiCacheCluster {
	c := m.table.Cursor()
	if c >= 0 && c < len(m.displayedElastiCache) {
		return &m.displayedElastiCache[c]
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
	case viewACM:
		cols = acmSortCols
	case viewENI:
		cols = eniSortCols
	case viewEKS:
		cols = eksSortCols
	case viewRDS:
		cols = rdsSortCols
	case viewS3:
		cols = s3SortCols
	case viewElastiCache:
		cols = elastiCacheSortCols
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

