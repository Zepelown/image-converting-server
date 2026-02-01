# 구현 순서 가이드

이 문서는 Image Converting Server의 단계별 구현 계획과 세부 작업 목록을 제공합니다.

## 구현 단계 개요

전체 구현은 다음 7단계로 진행됩니다:

1. **설정 관리** - 설정 파일 구조 및 로더 구현
2. **R2 클라이언트** - R2 접속 및 이미지 다운로드/업로드
3. **이미지 프로세서** - WebP 변환 및 리사이징
4. **상태 관리** - 마지막 처리 시간 추적
5. **크론 잡** - 일일 실행 스케줄러
6. **API 핸들러** - HTTP 엔드포인트 구현
7. **메인 통합** - 서버 시작 및 모든 컴포넌트 통합

## 단계별 상세 작업

### Phase 1: 설정 관리

**목표**: YAML 설정 파일을 로드하고 검증하는 기능 구현

**파일 생성**:
- `config/config.go` - 설정 구조체 및 로더
- `config/config.yaml` - 설정 파일 예시

**작업 목록**:
- [x] 설정 구조체 정의 (`Config`, `R2Config`, `ConversionConfig`, `ResizeConfig`, `CronConfig`)
- [x] YAML 파일 로드 함수 구현
- [x] 설정 검증 함수 구현 (필수 필드 확인)
- [x] 환경 변수 지원 (선택적)
- [x] 기본값 설정
- [x] 설정 파일 예시 작성

**의존성**: 없음

**참고 문서**: [CONFIG.md](./CONFIG.md)

---

### Phase 2: R2 클라이언트

**목표**: Cloudflare R2에 접속하여 이미지를 다운로드/업로드하고 객체 목록을 조회하는 기능 구현

**파일 생성**:
- `r2/client.go` - R2 클라이언트 구현 및 인터페이스 정의

**작업 목록**:
- [x] AWS SDK v2 설정 및 R2 엔드포인트 구성 (`s3.Options.BaseEndpoint`)
- [x] R2 client 인터페이스 (`StorageClient`) 및 구조체 정의
- [x] R2 client 초기화 함수 구현 (`NewClient`)
- [x] 이미지 다운로드 함수 구현 (`DownloadImage`)
- [x] 이미지 업로드 함수 구현 (`UploadImage`)
- [x] 객체 목록 조회 함수 구현 (`ListObjects`)
- [x] 마지막 수정 시간 이후 객체 필터링 로직 구현
- [x] 에러 처리 및 로깅 (AWS SDK 에러 감지)
- [x] `r2/client_test.go` 작성 및 단위 테스트
- [x] 연결 테스트 코드 작성

**의존성**: 
- Phase 1 (설정 관리)

**참고 문서**: [ARCHITECTURE.md](./ARCHITECTURE.md), [CONFIG.md](./CONFIG.md)

---

### Phase 3: 이미지 프로세서

**목표**: 이미지를 WebP로 변환하고 리사이징하는 기능 구현

**파일 생성**:
- `processor/converter.go` - 이미지 변환 및 리사이징

**작업 목록**:
- [x] 이미지 포맷 감지 함수 구현
- [x] WebP 변환 함수 구현 (`ConvertToWebP`)
  - [x] 이미지 디코딩
  - [x] WebP 인코딩 (품질 설정)
  - [x] 메타데이터 보존 (선택적)
- [x] 리사이징 함수 구현 (`ResizeImage`)
  - [x] 프리셋 크기 지원
  - [x] 커스텀 크기 지원
  - [x] 비율 유지 옵션
- [x] 변환 + 리사이징 통합 함수 (`ConvertAndResize`)
- [x] 에러 처리 (지원하지 않는 포맷, 손상된 이미지 등)
- [x] 메모리 최적화 (큰 이미지 처리)

**의존성**: 없음 (독립적으로 테스트 가능)

**참고 문서**: [ARCHITECTURE.md](./ARCHITECTURE.md), [CONFIG.md](./CONFIG.md)

---

### Phase 4: 상태 관리

**목표**: 크론 잡의 마지막 실행 시간을 추적하여 증분 처리 지원

**파일 생성**:
- `state/state.go` - 상태 관리

**작업 목록**:
- [x] 상태 구조체 정의 (`State` - LastProcessedTime 등)
- [x] 상태 파일 경로 설정 (로컬 JSON 파일)
- [x] 상태 로드 함수 구현 (`LoadState`)
- [x] 상태 저장 함수 구현 (`SaveState`)
- [x] 마지막 처리 시간 업데이트 함수
- [x] 파일이 없을 때 초기화 처리
- [x] 에러 처리 (파일 읽기/쓰기 실패)

**의존성**: 없음

**참고 문서**: [CRON.md](./CRON.md)

---

### Phase 5: 크론 잡

**목표**: 일일 스케줄로 R2의 이미지를 자동 변환하는 크론 잡 구현

**파일 생성**:
- `cron/job.go` - 크론 잡 실행

**작업 목록**:
- [x] 크론 스케줄러 초기화
- [x] 크론 잡 함수 구현 (`ProcessImages`)
  - [x] 상태에서 마지막 처리 시간 로드
  - [x] R2에서 새 이미지 목록 조회 (마지막 처리 시간 이후)
  - [x] 각 이미지에 대해:
    - [x] 이미지 다운로드
    - [x] WebP 변환
    - [x] R2에 업로드 (원본 대체)
  - [x] 상태 업데이트 (마지막 처리 시간)
- [x] 에러 처리 및 로깅
- [x] 중복 실행 방지 (락 메커니즘)
- [x] 실패한 이미지 추적 (선택적)

**의존성**: 
- Phase 1 (설정 관리)
- Phase 2 (R2 클라이언트)
- Phase 3 (이미지 프로세서)
- Phase 4 (상태 관리)

**참고 문서**: [CRON.md](./CRON.md), [ARCHITECTURE.md](./ARCHITECTURE.md)

---

### Phase 6: API 핸들러

**목표**: 프론트엔드에서 호출할 수 있는 HTTP API 엔드포인트 구현

**파일 생성**:
- `api/handlers.go` - HTTP 핸들러

**작업 목록**:
- [x] 변환 핸들러 구현 (`ConvertHandler`)
  - [x] POST `/api/convert` 엔드포인트
  - [x] GET `/api/convert` 엔드포인트 (쿼리 파라미터)
  - [x] 요청 파싱 (JSON 또는 쿼리 파라미터)
  - [x] 소스 타입 감지 (R2 키 vs URL)
  - [x] 리사이징 파라미터 파싱 (width, height, preset)
  - [x] 이미지 다운로드 (R2 또는 URL)
  - [x] 이미지 변환 및 리사이징
  - [x] R2에 업로드
  - [x] 응답 반환 (성공/실패)
- [x] 에러 처리 및 HTTP 상태 코드
- [x] 요청 검증 (필수 필드 확인)
- [x] 로깅
- [x] 기존 `/health` 엔드포인트 유지

**의존성**: 
- Phase 1 (설정 관리)
- Phase 2 (R2 클라이언트)
- Phase 3 (이미지 프로세서)

**참고 문서**: [API.md](./API.md), [ARCHITECTURE.md](./ARCHITECTURE.md)

---

### Phase 7: 메인 통합

**목표**: 모든 컴포넌트를 통합하여 서버를 시작하고 크론 잡을 등록

**파일 수정**:
- `main.go` - 서버 진입점

**작업 목록**:
- [ ] 설정 로드
- [ ] R2 클라이언트 초기화
- [ ] 이미지 프로세서 초기화
- [ ] 상태 관리자 초기화
- [ ] 크론 스케줄러 초기화 및 등록
- [ ] HTTP 라우터 설정
  - [ ] `/` - 메인 엔드포인트
  - [ ] `/health` - 헬스 체크
  - [ ] `/api/convert` - 변환 API
- [ ] 서버 시작
- [ ] Graceful shutdown 구현
- [ ] 에러 처리 및 로깅
- [ ] 환경 변수 지원 (포트 등)

**의존성**: 
- 모든 이전 Phase

**참고 문서**: [USAGE.md](./USAGE.md), [ARCHITECTURE.md](./ARCHITECTURE.md)

---

## 파일 생성 순서

```
1. config/
   ├── config.go
   └── config.yaml (예시)

2. r2/
   └── client.go

3. processor/
   └── converter.go

4. state/
   └── state.go

5. cron/
   └── job.go

6. api/
   └── handlers.go

7. main.go (수정)
```

## 의존성 관계도

```
config/ ──┐
          │
r2/ ──────┼──> cron/
          │
processor/┼──> cron/
          │    │
          │    └──> api/
          │
state/ ───┘
          │
          └──> main.go
```

## 구현 체크리스트

### 공통 작업
- [ ] 각 패키지에 대한 단위 테스트 작성
- [ ] 에러 처리 및 로깅 구현
- [ ] 코드 주석 작성 (Go doc 형식)
- [ ] 의존성 추가 (`go mod tidy`)

### Phase별 체크리스트

#### Phase 1: 설정 관리
- [x] `config/config.go` 구현 완료
- [x] `config/config.yaml` 예시 작성
- [x] 설정 로드 테스트 통과
- [x] 설정 검증 테스트 통과

#### Phase 2: R2 클라이언트
- [x] `r2/client.go` 구현 완료
- [x] R2 연결 테스트 통과
- [x] 다운로드/업로드 테스트 통과
- [x] 객체 목록 조회 테스트 통과

#### Phase 3: 이미지 프로세서
- [x] `processor/converter.go` 구현 완료
- [x] WebP 변환 테스트 통과
- [x] 리사이징 테스트 통과
- [x] 다양한 이미지 포맷 테스트 통과

#### Phase 4: 상태 관리
- [x] `state/state.go` 구현 완료
- [x] 상태 로드/저장 테스트 통과
- [x] 파일 없을 때 초기화 테스트 통과

#### Phase 5: 크론 잡
- [x] `cron/job.go` 구현 완료
- [x] 스케줄러 등록 테스트 통과
- [x] 증분 처리 로직 테스트 통과
- [x] 중복 실행 방지 테스트 통과

#### Phase 6: API 핸들러
- [x] `api/handlers.go` 구현 완료
- [x] POST 엔드포인트 테스트 통과
- [x] GET 엔드포인트 테스트 통과
- [x] 에러 처리 테스트 통과

#### Phase 7: 메인 통합
- [ ] `main.go` 수정 완료
- [ ] 서버 시작 테스트 통과
- [ ] 크론 잡 등록 테스트 통과
- [ ] Graceful shutdown 테스트 통과
- [ ] 전체 통합 테스트 통과

## 테스트 전략

### 단위 테스트
- 각 패키지별로 독립적인 단위 테스트 작성
- Mock 객체 사용 (R2 클라이언트 등)
- 테스트 커버리지 목표: 80% 이상

### 통합 테스트
- 실제 R2 버킷을 사용한 통합 테스트 (테스트 환경)
- API 엔드포인트 통합 테스트
- 크론 잡 통합 테스트

### 테스트 파일 구조
```
config/
  └── config_test.go
r2/
  └── client_test.go
processor/
  └── converter_test.go
state/
  └── state_test.go
cron/
  └── job_test.go
api/
  └── handlers_test.go
```

## 다음 단계

구현을 시작하기 전에 다음 문서들을 참고하세요:

1. [API.md](./API.md) - API 명세 확인
2. [ARCHITECTURE.md](./ARCHITECTURE.md) - 아키텍처 이해
3. [CONFIG.md](./CONFIG.md) - 설정 파일 구조 확인
4. [CRON.md](./CRON.md) - 크론 잡 동작 방식 이해
