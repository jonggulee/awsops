# awsops

k9s-style TUI for viewing AWS resources across multiple accounts in a single terminal screen.

## Requirements

- Go 1.21+
- `~/.aws/config` with one or more profiles configured

## Installation

```bash
git clone https://github.com/jgulee/awsops
cd awsops
go build -o awsops .
```

## Usage

```bash
./awsops
```

Reads all profiles from `~/.aws/config` and fetches resources from the selected regions on startup (default: `ap-northeast-2`).

## Views

Switch views with the `:` command (type to filter):

| Command | View |
|---------|------|
| `:ec2` | EC2 Instances |
| `:sg` | Security Groups |
| `:eni` | Network Interfaces |
| `:elb` | Load Balancers (ALB/NLB) |
| `:vpc` | VPCs |
| `:subnet` | Subnets |
| `:tgw` | Transit Gateway Attachments |
| `:eks` | EKS Clusters |
| `:acm` | ACM Certificates |
| `:route53` | Route 53 Records |

## Key Bindings

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move cursor |
| `◀` / `▶` | Scroll columns left / right |
| `q` / `ctrl+c` | Quit |

### Search & Filter

| Key | Action |
|-----|--------|
| `/` | Enter search mode |
| `enter` | Confirm search term (stacks with AND logic) |
| `esc` | Clear all filters and exit search mode |
| `t` | Open tag picker (2-step key → value selection) |

Multiple search terms stack with AND logic.  
In search mode, `key=value` syntax filters by tag (e.g. `env=production`).  
Plain text also matches against tag keys and values.

### Detail View

| Key | Action |
|-----|--------|
| `d` | Open detail screen for selected row |
| `↑` / `↓` | Navigate interactive fields |
| `enter` | Jump to linked resource |
| `esc` / `q` | Back to list (or previous detail if navigated) |
| `j` / `k` | Scroll detail content up / down |

Cross-resource navigation is supported in detail views:

- **EC2**: jump to VPC, Subnet, or any attached SG
- **EKS**: jump to Nodes (→ EC2), Subnets, or Security Groups
- **ALB**: jump to any attached Security Group
- **Route 53**: jump to ALB when alias target matches a loaded load balancer
- **SG**: shows associated ENIs

### Sort

Press a number key to sort by that column. Same key again reverses order. One more press clears sort.

| View | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 0 |
|------|---|---|---|---|---|---|---|---|---|---|
| EC2 | Profile | Name | Instance ID | State | Type | Private IP | Public IP | VPC ID | Subnet ID | Region |
| SG | Profile | Name | Group ID | VPC ID | Description | Region | | | | |
| ENI | Profile | ID | Name | Status | Type | Private IP | Instance ID | VPC ID | Subnet ID | Region |
| ALB | Profile | Name | Type | Scheme | State | VPC ID | DNS Name | Region | | |
| VPC | Profile | Name | VPC ID | CIDR | State | Region | | | | |
| Subnet | Profile | Name | Subnet ID | VPC ID | CIDR | AZ | Region | | | |
| TGW | Profile | TGW ID | Attachment ID | Type | Resource ID | Owner | TGW Owner | State | Region | |
| EKS | Profile | Name | Status | Version | VPC ID | Endpoint | Region | | | |
| ACM | Profile | Domain | Status | Type | Expiry | Region | | | | |
| Route 53 | Zone Name | Zone Type | Record Name | Type | TTL | Values | | | | |

### Connectivity Check (Subnet view)

| Key | Action |
|-----|--------|
| `c` | Start connectivity check from selected subnet |
| `↑` / `↓` / `pgup` / `pgdn` | Navigate route / subnet list |
| `enter` | Select route (phase 1) / run check (phase 2) |
| `esc` | Back to previous step |
| type to filter | Filter the picker list |

Performs a 5-step TGW-based connectivity analysis between two subnets.

### Regions

| Key | Action |
|-----|--------|
| `R` | Open region selector |
| `↑` / `↓` | Move cursor |
| `space` | Toggle region on/off |
| `a` | Select all |
| `n` | Deselect all |
| `enter` | Apply and re-fetch |
| `esc` / `q` | Cancel (discard changes) |

Regions are grouped by geography (Asia Pacific / United States).  
Pressing `esc`/`q` after making changes shows a discard confirmation prompt.

### Other

| Key | Action |
|-----|--------|
| `r` | Refresh all resources |
| `R` | Open region selector |

---

# awsops (한국어)

여러 AWS 어카운트의 리소스를 하나의 터미널 화면에서 조회하는 k9s 스타일 TUI 도구.

## 요구사항

- Go 1.21+
- 하나 이상의 프로필이 설정된 `~/.aws/config`

## 설치

```bash
git clone https://github.com/jgulee/awsops
cd awsops
go build -o awsops .
```

## 실행

```bash
./awsops
```

`~/.aws/config`의 모든 프로필을 읽어 선택된 리전에서 리소스를 조회한다 (기본값: `ap-northeast-2`).

## 뷰

`:` 키로 뷰를 전환한다 (텍스트 입력으로 필터링 가능):

| 명령어 | 뷰 |
|--------|----|
| `:ec2` | EC2 인스턴스 |
| `:sg` | 보안 그룹 |
| `:eni` | 네트워크 인터페이스 |
| `:elb` | 로드 밸런서 (ALB/NLB) |
| `:vpc` | VPC |
| `:subnet` | 서브넷 |
| `:tgw` | Transit Gateway 어태치먼트 |
| `:eks` | EKS 클러스터 |
| `:acm` | ACM 인증서 |
| `:route53` | Route 53 레코드 |

## 키 바인딩

### 이동

| 키 | 동작 |
|----|------|
| `↑` / `↓` | 커서 이동 |
| `◀` / `▶` | 컬럼 좌우 스크롤 |
| `q` / `ctrl+c` | 종료 |

### 검색 / 필터

| 키 | 동작 |
|----|------|
| `/` | 검색 모드 진입 |
| `enter` | 검색어 확정 (AND로 누적) |
| `esc` | 필터 전체 초기화 및 검색 모드 종료 |
| `t` | 태그 피커 열기 (key → value 2단계 선택) |

검색어는 AND 조건으로 누적된다.  
검색 모드에서 `key=value` 형식으로 입력하면 태그 필터가 적용된다 (예: `env=production`).  
일반 텍스트도 태그 키/값에서 함께 검색된다.

### 상세 보기

| 키 | 동작 |
|----|------|
| `d` | 선택한 행의 상세 화면 표시 |
| `↑` / `↓` | 인터랙티브 필드 이동 |
| `enter` | 연결된 리소스로 이동 |
| `esc` / `q` | 목록으로 돌아가기 (또는 이전 상세로) |
| `j` / `k` | 상세 내용 위/아래 스크롤 |

상세 보기에서 다음 리소스 간 이동이 지원된다:

- **EC2**: VPC, 서브넷, 연결된 SG로 이동
- **EKS**: 노드(→ EC2), 서브넷, 보안 그룹으로 이동
- **ALB**: 연결된 보안 그룹으로 이동
- **Route 53**: Alias Target이 로드된 ALB와 매칭되면 ALB 상세로 이동
- **SG**: 연결된 ENI 목록 표시

### 정렬

숫자 키로 해당 컬럼 기준 정렬. 같은 키를 다시 누르면 역순, 한 번 더 누르면 정렬 해제.

| 뷰 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 0 |
|----|---|---|---|---|---|---|---|---|---|---|
| EC2 | Profile | Name | Instance ID | State | Type | Private IP | Public IP | VPC ID | Subnet ID | Region |
| SG | Profile | Name | Group ID | VPC ID | Description | Region | | | | |
| ENI | Profile | ID | Name | Status | Type | Private IP | Instance ID | VPC ID | Subnet ID | Region |
| ALB | Profile | Name | Type | Scheme | State | VPC ID | DNS Name | Region | | |
| VPC | Profile | Name | VPC ID | CIDR | State | Region | | | | |
| Subnet | Profile | Name | Subnet ID | VPC ID | CIDR | AZ | Region | | | |
| TGW | Profile | TGW ID | Attachment ID | Type | Resource ID | Owner | TGW Owner | State | Region | |
| EKS | Profile | Name | Status | Version | VPC ID | Endpoint | Region | | | |
| ACM | Profile | Domain | Status | Type | Expiry | Region | | | | |
| Route 53 | Zone Name | Zone Type | Record Name | Type | TTL | Values | | | | |

### 연결성 검사 (Subnet 뷰)

| 키 | 동작 |
|----|------|
| `c` | 선택한 서브넷에서 연결성 검사 시작 |
| `↑` / `↓` / `pgup` / `pgdn` | 라우트 / 서브넷 목록 이동 |
| `enter` | 라우트 선택 (1단계) / 검사 실행 (2단계) |
| `esc` | 이전 단계로 돌아가기 |
| 문자 입력 | 피커 목록 필터링 |

TGW 기반 5단계 연결성 분석을 수행한다.

### 리전 선택

| 키 | 동작 |
|----|------|
| `R` | 리전 선택 화면 열기 |
| `↑` / `↓` | 커서 이동 |
| `space` | 리전 on/off 토글 |
| `a` | 전체 선택 |
| `n` | 전체 해제 |
| `enter` | 선택 적용 및 재조회 |
| `esc` / `q` | 취소 (변경 사항 버리기) |

리전은 지역별로 그룹화되어 표시된다 (Asia Pacific / United States).  
변경 사항이 있는 상태에서 `esc`/`q`를 누르면 폐기 확인창이 나타난다.

### 기타

| 키 | 동작 |
|----|------|
| `r` | 전체 리소스 새로고침 |
| `R` | 리전 선택 화면 열기 |
