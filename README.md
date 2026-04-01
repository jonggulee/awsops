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

Reads all profiles from `~/.aws/config` and fetches resources from the default region (`ap-northeast-2`) on startup.

## Key Bindings

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move cursor |
| `q` / `ctrl+c` | Quit |

### Views

| Key | Action |
|-----|--------|
| `:ec2` + `enter` | Switch to EC2 instances view |
| `:sg` + `enter` | Switch to Security Groups view |

### Search

| Key | Action |
|-----|--------|
| `/` | Enter search mode |
| `enter` | Confirm search term (stacks with AND logic) |
| `esc` | Clear all search terms and exit search mode |

Multiple search terms are combined with AND. Example: `/` → `center` → `enter` → `/` → `m7i` → `enter` shows only rows matching both.

### Detail

| Key | Action |
|-----|--------|
| `d` | Show detail screen for selected row |
| `esc` / `q` | Back to list |

### Sort

Press a number key to sort by that column. Press the same key again to reverse. Press once more to clear.

**EC2**

| Key | Column |
|-----|--------|
| `1` | Profile |
| `2` | Name |
| `3` | Instance ID |
| `4` | State |
| `5` | Type |
| `6` | Private IP |
| `7` | Public IP |
| `8` | Region |

**Security Groups**

| Key | Column |
|-----|--------|
| `1` | Profile |
| `2` | Name |
| `3` | Group ID |
| `4` | VPC ID |
| `6` | Region |

### Regions

| Key | Action |
|-----|--------|
| `R` | Open region selector |
| `space` | Toggle region on/off |
| `enter` | Apply and re-fetch |
| `esc` | Cancel |
| `r` | Refresh with current regions |

Default region: `ap-northeast-2` (Seoul)

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

`~/.aws/config`의 모든 프로필을 읽어 기본 리전(`ap-northeast-2`)에서 리소스를 조회한다.

## 키 바인딩

### 이동

| 키 | 동작 |
|-----|--------|
| `↑` / `↓` | 커서 이동 |
| `q` / `ctrl+c` | 종료 |

### 뷰 전환

| 키 | 동작 |
|-----|--------|
| `:ec2` + `enter` | EC2 인스턴스 뷰로 전환 |
| `:sg` + `enter` | 보안 그룹 뷰로 전환 |

### 검색

| 키 | 동작 |
|-----|--------|
| `/` | 검색 모드 진입 |
| `enter` | 검색어 확정 (AND로 누적) |
| `esc` | 검색어 전체 초기화 및 검색 모드 종료 |

검색어는 AND 조건으로 누적된다. 예: `/` → `center` → `enter` → `/` → `m7i` → `enter` 입력 시 두 조건을 모두 만족하는 행만 표시.

### 상세 보기

| 키 | 동작 |
|-----|--------|
| `d` | 선택한 행의 상세 화면 표시 |
| `esc` / `q` | 목록으로 돌아가기 |

### 정렬

숫자 키로 해당 컬럼 기준 정렬. 같은 키를 다시 누르면 역순, 한 번 더 누르면 정렬 해제.

**EC2**

| 키 | 컬럼 |
|-----|--------|
| `1` | Profile |
| `2` | Name |
| `3` | Instance ID |
| `4` | State |
| `5` | Type |
| `6` | Private IP |
| `7` | Public IP |
| `8` | Region |

**보안 그룹**

| 키 | 컬럼 |
|-----|--------|
| `1` | Profile |
| `2` | Name |
| `3` | Group ID |
| `4` | VPC ID |
| `6` | Region |

### 리전 선택

| 키 | 동작 |
|-----|--------|
| `R` | 리전 선택 화면 열기 |
| `space` | 리전 on/off 토글 |
| `enter` | 선택 적용 및 재조회 |
| `esc` | 취소 |
| `r` | 현재 리전으로 새로고침 |

기본 리전: `ap-northeast-2` (서울)
