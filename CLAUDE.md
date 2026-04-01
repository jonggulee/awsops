# awsops - 프로젝트 컨텍스트

## 프로젝트 목표

k9s처럼 터미널에서 동작하는 AWS 멀티 어카운트 리소스 뷰어.
여러 AWS 어카운트의 리소스를 하나의 TUI 화면에서 한눈에 조회한다.

## 핵심 원칙

- **Read-only**: 조회만 한다. 생성/삭제/수정 기능은 만들지 않는다
- **멀티 어카운트**: `~/.aws/config`의 여러 프로필을 동시에 조회
- **TUI**: bubbletea 기반 터미널 UI (k9s 스타일)

## 기술 스택

- **언어**: Go
- **TUI**: bubbletea (Elm 아키텍처: Model / Update / View)
- **스타일**: lipgloss, bubbles
- **AWS**: aws-sdk-go-v2

## 현재 AWS 프로필 환경

`~/.aws/config`에 다음 프로필이 설정되어 있다:
- `default`
- `infra-team`
- `lam-dev`
- `sgr-ai`
- `s3-test`
- `center`
- `sgr-ai-learn`

기본 리전: `ap-northeast-2` (서울)

## 프로젝트 구조 (목표)

```
awsops/
├── main.go
├── internal/
│   ├── aws/
│   │   ├── client.go      # 멀티 프로필 AWS 클라이언트 관리
│   │   └── ec2.go         # EC2 리소스 조회
│   ├── ui/
│   │   ├── app.go         # bubbletea 앱 진입점
│   │   ├── model.go       # 상태 관리
│   │   └── view.go        # 화면 렌더링
│   └── config/
│       └── config.go      # 설정 로드
└── CLAUDE.md
```

## 협업 방식

- 사용자가 Go와 bubbletea를 공부하는 목적도 있으므로, 코드를 대신 짜주기보다 개념 설명 → 직접 작성 → 리뷰 순서로 진행한다
- 막히는 부분은 힌트를 주고, 완성된 코드는 리뷰해준다
- 고도화 아이디어는 적극적으로 제안한다

## 로드맵

### v1 - 기본 뷰어
- [ ] EC2 인스턴스 멀티 어카운트 조회
- [ ] 리소스 타입 검색
- [ ] 어카운트 / 리전 필터링

### v2 - 고도화
- [ ] 리소스간 관계 시각화 (EC2 ↔ SG ↔ VPC)
- [ ] 태그 기반 검색
- [ ] 실시간 상태 갱신 (polling)
- [ ] 어카운트간 리소스 diff

### v3 - 심화
- [ ] Cost Explorer 비용 오버레이
- [ ] 즐겨찾기 리소스
- [ ] 어카운트/리전 프리셋 설정 파일
