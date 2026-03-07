# Config → .env 전환 남은 작업 구현 계획

설정을 `.env` 단일 소스로 쓰기 위해 남은 작업을 단계별로 정리한 계획이다.

---

## 목표

- 앱 진입점에서 **YAML 없이** `config.LoadFromEnv()`만 사용.
- `.env.example`에 필요한 모든 env 키가 문서화되어 있음.
- 스펙 문서가 현재 동작(env-only)과 일치.

---

## Phase 1: 진입점 전환 (필수)

**목표**: 실행 시 `.env`만 읽고 `config.yaml`을 사용하지 않도록 변경.

### 1.1 main.go 수정

| 항목 | 내용 |
|------|------|
| **파일** | `main.go` |
| **변경** | 23번째 줄 `config.Load("config/config.yaml")` → `config.LoadFromEnv()` |
| **검증** | `go run .` 실행 후, `.env`만 존재할 때 정상 기동되는지 확인. (config.yaml 삭제/이동 후 테스트 권장) |

**예시 diff:**

```go
// Before
cfg, err := config.Load("config/config.yaml")

// After
cfg, err := config.LoadFromEnv()
```

### 1.2 완료 조건

- [ ] `main.go`에서 `config.LoadFromEnv()` 호출
- [ ] `go build ./...` 성공
- [ ] `go test ./...` 통과
- [ ] (선택) `config/config.yaml` 없이 `.env`만 두고 `go run .` 성공

---

## Phase 2: .env.example 보완 (필수)

**목표**: 새 개발자가 `cp .env.example .env` 후 필요한 키를 한눈에 볼 수 있도록 한다.

### 2.1 추가할 env 키 및 주석

| 키 | 설명 | 예시 값 |
|----|------|---------|
| `CONVERSION_FORMATS` | 변환 대상 확장자 (쉼표 구분) | `jpeg,jpg,png,gif,bmp,tiff` |
| `CONVERSION_QUALITY` | WebP 품질 (0–100) | `85` |
| `CONVERSION_MAX_SIZE_MB` | 최대 이미지 크기 (MB) | `50` |
| `RESIZE_PRESETS` | 리사이즈 프리셋 (이름:폭x높이, 쉼표 구분) | `thumbnail:150x150,medium:800x800,large:1920x1920` |
| `CRON_SCHEDULE` | Crontab 표현식 | `0 2 * * *` |
| `CRON_ENABLED` | 크론 활성화 여부 | `true` / `false` |
| `SERVER_TIMEOUT_SECONDS` | HTTP 타임아웃(초) | `30` |

기존 `.env.example`에 이미 있는 항목(R2_*, SERVER_PORT, WEBHOOK_*)은 유지하고, 위 항목을 **주석 + 예시 값** 형태로 추가.

### 2.2 작업 내용

- [ ] `.env.example` 상단 또는 적절한 위치에 "이 파일을 .env로 복사한 뒤 값 입력" 안내 유지
- [ ] Conversion 섹션 주석 추가: CONVERSION_FORMATS, CONVERSION_QUALITY, CONVERSION_MAX_SIZE_MB
- [ ] Resize 섹션 주석 추가: RESIZE_PRESETS 형식 설명 및 예시
- [ ] Cron 섹션 주석 추가: CRON_SCHEDULE, CRON_ENABLED
- [ ] Server 섹션에 SERVER_TIMEOUT_SECONDS 주석 추가 (SERVER_PORT는 기존 유지)

### 2.3 완료 조건

- [ ] `.env.example`에 위 모든 키가 (주석이든 값이든) 존재
- [ ] 형식(쉼표 구분, `이름:폭x높이` 등)이 `config/load_env.go` 파싱 규칙과 일치

---

## Phase 3: 스펙 문서 갱신 (선택)

**목표**: `docs/spec/config-storage-state.md`가 현재 구현(env-only 진입점)을 반영하도록 수정.

### 3.1 변경 포인트

| 섹션 | 변경 내용 |
|------|-----------|
| **1.1 설정 파일 및 .env 처리** | 진입점을 `config.LoadFromEnv()`로 기술. 기존 "YAML 읽기 → env 오버라이드" 흐름을 "env만 로드" 흐름으로 교체 또는 보조 설명으로 이동. |
| **기준 구현 파일** | `config/config.go` → `config/load_env.go`, `config/defaults.go`, `config/validate.go` 등으로 보조 명시. |
| **§2 환경 변수 오버라이드 규칙** | "env가 유일한 소스"임을 명시하고, 지원 env 목록에 CONVERSION_*, RESIZE_PRESETS, CRON_*, SERVER_TIMEOUT_SECONDS 추가. |

### 3.2 완료 조건

- [ ] 진입점이 `LoadFromEnv()`로 문서화됨
- [ ] 지원 env 목록이 `load_env.go`와 일치

---

## Phase 4: YAML 경로 제거 (나중에, 선택)

**목표**: env-only로 완전 전환 후 레거시 코드 정리. **Phase 1–2가 안정화된 뒤** 진행.

### 4.1 제거/정리 대상

| 대상 | 조치 |
|------|------|
| `config.Load(configPath string)` | 제거 또는 deprecated 주석 후 내부에서 `LoadFromEnv()` 호출하도록 위임 |
| `applyEnvironmentVariables` | `config.go`에서 제거 (호출처 없어짐) |
| `config/config.yaml` | 삭제 또는 `config/config.yaml.example`로 이름 변경 후 샘플로만 유지 |
| `config_test.go` 내 YAML 기반 테스트 | `Load(configPath)` 제거 시 해당 테스트 제거 또는 `LoadFromEnv` 기반으로 대체 |

### 4.2 주의

- **테스트 불변 규칙**: 테스트 수정/삭제 시 스펙 변경 사유, 영향 범위, 승인 내역을 남긴다.
- `config_test.go`의 `TestValidateConfig`는 `Validate` 단위 테스트이므로 유지. YAML `Load` 관련 테스트만 단계적으로 정리.

### 4.3 완료 조건

- [ ] 코드베이스에 `config.Load(` 호출이 없음 (또는 deprecated 래퍼만 존재)
- [ ] `config.yaml` 미사용 상태가 문서/README에 반영됨
- [ ] `go test ./config` 통과

---

## 체크리스트 요약

| Phase | 작업 | 우선순위 |
|-------|------|----------|
| 1 | main.go → LoadFromEnv() | 필수 |
| 2 | .env.example에 새 키/주석 추가 | 필수 |
| 3 | docs/spec/config-storage-state.md 갱신 | 선택 |
| 4 | YAML 경로 제거 (Load, applyEnvironmentVariables, config.yaml, YAML 테스트) | 나중에 |

---

## 권장 실행 순서

1. **Phase 1** 수행 → 빌드/테스트/실행 확인
2. **Phase 2** 수행 → .env.example으로 새 환경 구성 테스트
3. 필요 시 **Phase 3**으로 스펙 문서 동기화
4. env-only 운영이 충분히 안정된 뒤 **Phase 4** 검토
