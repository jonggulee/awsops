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

Switch views with the `:` command:

| Command | View |
|---------|------|
| `:ec2` | EC2 Instances |
| `:sg` | Security Groups |
| `:vpc` | VPCs |
| `:subnet` | Subnets |
| `:tgw` | Transit Gateway Attachments |
| `:acm` | ACM Certificates |

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

Multiple search terms stack with AND. Example: `/` → `prod` → `enter` → `/` → `m7i` → `enter` shows only rows matching both.

### Detail

| Key | Action |
|-----|--------|
| `d` | Open detail screen for selected row |
| `↑` / `↓` | Navigate interactive fields (EC2 detail) |
| `enter` | Jump to linked resource (VPC / Subnet / SG) |
| `esc` / `q` | Back to list (or previous detail) |
| `j` / `k` | Scroll detail content up / down |

EC2 detail shows `[vpc name]`, `[subnet name]`, `[sg name]` hints next to IDs.  
SG detail shows `[sg name]` in inbound/outbound rules and lists associated ENIs.

### Sort

Press a number key to sort by that column. Same key again reverses. One more press clears sort.

| View | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 0 |
|------|---|---|---|---|---|---|---|---|---|---|
| EC2 | Profile | Name | Instance ID | State | Type | Private IP | Public IP | VPC ID | Subnet ID | Region |
| SG | Profile | Name | Group ID | VPC ID | Description | Region | | | | |
| VPC | Profile | Name | VPC ID | CIDR | State | Region | | | | |
| Subnet | Profile | Name | Subnet ID | VPC ID | CIDR | AZ | Region | | | |
| TGW | Profile | TGW ID | Attachment ID | Type | Resource ID | Owner | TGW Owner | State | Region | |
| ACM | Profile | Domain | Status | Type | Expiry | Region | | | | |

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
| `enter` | Apply and re-fetch (save) |
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

`:` 명령어로 뷰를 전환한다:

| 명령어 | 뷰 |
|--------|----|
| `:ec2` | EC2 인스턴스 |
| `:sg` | 보안 그룹 |
| `:vpc` | VPC |
| `:subnet` | 서브넷 |
| `:tgw` | Transit Gateway 어태치먼트 |
| `:acm` | ACM 인증서 |

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

검색어는 AND 조건으로 누적된다. 예: `/` → `prod` → `enter` → `/` → `m7i` → `enter` 입력 시 두 조건을 모두 만족하는 행만 표시.

### 상세 보기

| 키 | 동작 |
|----|------|
| `d` | 선택한 행의 상세 화면 표시 |
| `↑` / `↓` | 인터랙티브 필드 이동 (EC2 상세) |
| `enter` | 연결된 리소스로 이동 (VPC / Subnet / SG) |
| `esc` / `q` | 목록으로 돌아가기 (또는 이전 상세로) |
| `j` / `k` | 상세 내용 위/아래 스크롤 |

EC2 상세에서는 ID 옆에 `[vpc name]`, `[subnet name]`, `[sg name]` 힌트가 표시된다.  
SG 상세에서는 인바운드/아웃바운드 규칙에 `[sg name]`이 표시되고 연결된 ENI 목록도 확인할 수 있다.

### 정렬

숫자 키로 해당 컬럼 기준 정렬. 같은 키를 다시 누르면 역순, 한 번 더 누르면 정렬 해제.

| 뷰 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 0 |
|----|---|---|---|---|---|---|---|---|---|---|
| EC2 | Profile | Name | Instance ID | State | Type | Private IP | Public IP | VPC ID | Subnet ID | Region |
| SG | Profile | Name | Group ID | VPC ID | Description | Region | | | | |
| VPC | Profile | Name | VPC ID | CIDR | State | Region | | | | |
| Subnet | Profile | Name | Subnet ID | VPC ID | CIDR | AZ | Region | | | |
| TGW | Profile | TGW ID | Attachment ID | Type | Resource ID | Owner | TGW Owner | State | Region | |
| ACM | Profile | Domain | Status | Type | Expiry | Region | | | | |

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
| `enter` | 선택 적용 및 재조회 (저장) |
| `esc` / `q` | 취소 (변경 사항 버리기) |

리전은 지역별로 그룹화되어 표시된다 (Asia Pacific / United States).  
변경 사항이 있는 상태에서 `esc`/`q`를 누르면 폐기 확인창이 나타난다.

### 기타

| 키 | 동작 |
|----|------|
| `r` | 전체 리소스 새로고침 |
| `R` | 리전 선택 화면 열기 |
