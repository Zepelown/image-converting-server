# Image Converting Server - 개요

## 프로젝트 소개

Image Converting Server는 Cloudflare R2에 저장된 이미지를 WebP 형식으로 변환하고 리사이징하는 소형 Go 서버입니다. 프론트엔드에서 presigned URL을 통해 R2에 업로드한 이미지를 자동으로 최적화하여 저장 공간을 절약하고 웹 성능을 향상시킵니다.

## 주요 기능

### 1. 자동 이미지 변환 (크론 잡)
- **일일 실행**: 매일 지정된 시간에 R2 버킷을 스캔하여 WebP가 아닌 모든 이미지를 자동으로 변환
- **증분 처리**: 마지막 실행 이후 새로 추가된 이미지만 처리하여 효율성 극대화
- **원본 대체**: 변환된 WebP 이미지로 원본을 대체하여 저장 공간 절약

### 2. 온디맨드 변환 (API)
- **즉시 변환**: 프론트엔드에서 API를 호출하여 특정 이미지를 즉시 변환
- **다양한 소스 지원**: R2 객체 키 또는 외부 URL 모두 지원
- **리사이징 옵션**: 프리셋 크기 또는 커스텀 크기로 리사이징 가능

### 3. 이미지 리사이징
- **프리셋 크기**: thumbnail, medium, large 등 미리 정의된 크기 사용
- **커스텀 크기**: API 요청 시 width, height 파라미터로 원하는 크기 지정
- **비율 유지**: 원본 이미지 비율을 유지하면서 리사이징

## 기술 스택

- **언어**: Go 1.21+
- **스토리지**: Cloudflare R2 (S3 호환 API)
- **이미지 처리**: 
  - WebP 변환: `github.com/chai2010/webp` 또는 `github.com/tidbyt/go-libwebp`
  - 리사이징: `github.com/disintegration/imaging`
- **스케줄링**: `github.com/robfig/cron/v3`
- **설정 관리**: YAML 파일 (`gopkg.in/yaml.v3`)

## 문서 구조

이 프로젝트의 문서는 다음과 같이 구성되어 있습니다:

### 📘 [OVERVIEW.md](./OVERVIEW.md) (현재 문서)
프로젝트 전체 개요, 주요 기능, 기술 스택, 문서 구조 안내

### 📋 [IMPLEMENTATION.md](./IMPLEMENTATION.md)
단계별 구현 계획, 컴포넌트별 구현 순서, 의존성 관계, 파일 구조, 구현 체크리스트

### 🔌 [API.md](./API.md)
API 엔드포인트 상세 명세, 요청/응답 형식, 에러 처리, 사용 예시

### 🏗️ [ARCHITECTURE.md](./ARCHITECTURE.md)
시스템 아키텍처, 컴포넌트 설계, 데이터 흐름, 의존성 관계

### ⚙️ [CONFIG.md](./CONFIG.md)
설정 파일 구조, 각 설정 항목 설명, 환경 변수, 보안 고려사항

### 📖 [USAGE.md](./USAGE.md)
설치 방법, 서버 실행, 설정 파일 준비, API 사용 예시, 트러블슈팅

### ⏰ [CRON.md](./CRON.md)
크론 잡 스케줄 설정, 증분 처리 로직, 상태 관리, 실패 처리 및 재시도

## 빠른 시작

### 1. 설정 파일 준비
```bash
# config/config.yaml 파일 생성 및 R2 접속 정보 입력
```

자세한 내용은 [CONFIG.md](./CONFIG.md) 참조

### 2. 서버 실행
```bash
go run main.go
```

자세한 내용은 [USAGE.md](./USAGE.md) 참조

### 3. API 호출 예시
```bash
# 이미지 변환 요청
curl -X POST http://localhost:8080/api/convert \
  -H "Content-Type: application/json" \
  -d '{"source": "r2://my-bucket/images/photo.jpg"}'
```

자세한 내용은 [API.md](./API.md) 참조

## 아키텍처 개요

```
┌─────────────┐
│  Frontend   │
└──────┬──────┘
       │ API Request
       ▼
┌─────────────────┐
│   HTTP Server   │
└──────┬──────────┘
       │
       ├─────────────────┐
       │                 │
       ▼                 ▼
┌──────────────┐  ┌──────────────┐
│ API Handler  │  │  Cron Job    │
└──────┬───────┘  └──────┬───────┘
       │                 │
       └────────┬────────┘
                │
                ▼
        ┌───────────────┐
        │ Image Processor│
        └───────┬───────┘
                │
                ▼
        ┌───────────────┐
        │ Cloudflare R2 │
        └───────────────┘
```

자세한 아키텍처 설명은 [ARCHITECTURE.md](./ARCHITECTURE.md) 참조

## 주요 컴포넌트

1. **설정 관리** (`config/`): YAML 기반 설정 로드 및 검증
2. **R2 클라이언트** (`r2/`): R2 접속, 이미지 다운로드/업로드
3. **이미지 프로세서** (`processor/`): WebP 변환 및 리사이징
4. **크론 잡** (`cron/`): 일일 스케줄 실행 및 이미지 변환
5. **API 핸들러** (`api/`): HTTP 엔드포인트 처리
6. **상태 관리** (`state/`): 마지막 처리 시간 추적

각 컴포넌트의 상세 설명은 [ARCHITECTURE.md](./ARCHITECTURE.md) 참조

## 다음 단계

1. **구현 계획 확인**: [IMPLEMENTATION.md](./IMPLEMENTATION.md)에서 단계별 구현 계획 확인
2. **API 명세 확인**: [API.md](./API.md)에서 엔드포인트 상세 확인
3. **설정 준비**: [CONFIG.md](./CONFIG.md)에서 설정 파일 작성
4. **사용 가이드**: [USAGE.md](./USAGE.md)에서 설치 및 실행 방법 확인

## 라이선스

MIT License - 자세한 내용은 [LICENSE](../LICENSE) 파일 참조
