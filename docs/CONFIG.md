# 설정 가이드

Image Converting Server의 설정 파일 구조 및 각 설정 항목에 대한 상세 가이드입니다.

## 설정 파일 위치

기본 설정 파일 경로: `config/config.yaml`

서버 실행 시 `-config` 플래그로 다른 경로를 지정할 수 있습니다:
```bash
go run main.go -config /path/to/custom-config.yaml
```

## 설정 파일 구조

### 전체 구조

```yaml
r2:
  access_key: "your-access-key"
  secret_key: "your-secret-key"
  endpoint: "https://your-account-id.r2.cloudflarestorage.com"
  bucket: "your-bucket-name"

conversion:
  formats: ["jpeg", "jpg", "png", "gif", "bmp", "tiff"]
  quality: 85
  max_size_mb: 50

resize:
  presets:
    thumbnail:
      width: 150
      height: 150
    medium:
      width: 800
      height: 800
    large:
      width: 1920
      height: 1920

cron:
  schedule: "0 2 * * *"
  enabled: true

server:
  port: 8080
  timeout_seconds: 30
```

---

## 설정 항목 상세 설명

### R2 설정 (`r2`)

Cloudflare R2 접속 정보를 설정합니다.

#### `access_key` (필수)
- **타입**: string
- **설명**: Cloudflare R2 API Access Key ID
- **예시**: `"abc123def456ghi789"`
- **보안**: 민감 정보 - 환경 변수 사용 권장

#### `secret_key` (필수)
- **타입**: string
- **설명**: Cloudflare R2 API Secret Access Key
- **예시**: `"secret123456789abcdef"`
- **보안**: 민감 정보 - 환경 변수 사용 권장

#### `endpoint` (필수)
- **타입**: string
- **설명**: Cloudflare R2 엔드포인트 URL
- **형식**: `https://{account-id}.r2.cloudflarestorage.com`
- **예시**: `"https://abc123def456.r2.cloudflarestorage.com"`
- **참고**: Cloudflare 대시보드에서 확인 가능

#### `bucket` (필수)
- **타입**: string
- **설명**: 이미지가 저장된 R2 버킷 이름
- **예시**: `"my-image-bucket"`

**예시**:
```yaml
r2:
  access_key: "your-access-key"
  secret_key: "your-secret-key"
  endpoint: "https://abc123def456.r2.cloudflarestorage.com"
  bucket: "my-image-bucket"
```

---

### 변환 설정 (`conversion`)

이미지 변환 관련 설정입니다.

#### `formats` (선택)
- **타입**: array of strings
- **설명**: WebP로 변환할 이미지 포맷 목록
- **기본값**: `["jpeg", "jpg", "png", "gif", "bmp", "tiff"]`
- **지원 포맷**: `jpeg`, `jpg`, `png`, `gif`, `bmp`, `tiff`
- **예시**: `["jpeg", "jpg", "png"]`

#### `quality` (선택)
- **타입**: integer
- **설명**: WebP 변환 품질 (0-100)
- **기본값**: `85`
- **범위**: 0 (최저 품질, 최소 크기) ~ 100 (최고 품질, 최대 크기)
- **권장값**: 80-90

#### `max_size_mb` (선택)
- **타입**: integer
- **설명**: 처리할 수 있는 최대 이미지 크기 (MB)
- **기본값**: `50`
- **제한**: 메모리 제약에 따라 조정 필요

**예시**:
```yaml
conversion:
  formats: ["jpeg", "jpg", "png", "gif"]
  quality: 85
  max_size_mb: 50
```

---

### 리사이징 설정 (`resize`)

이미지 리사이징 프리셋을 정의합니다.

#### `presets` (선택)
- **타입**: object
- **설명**: 프리셋 크기 정의
- **구조**: 각 프리셋은 `width`와 `height`를 가짐

**프리셋 예시**:
```yaml
resize:
  presets:
    thumbnail:
      width: 150
      height: 150
    medium:
      width: 800
      height: 800
    large:
      width: 1920
      height: 1920
    custom_small:
      width: 400
      height: 300
```

**프리셋 사용**:
- API 요청 시 `?preset=thumbnail` 형식으로 사용
- 프리셋 이름은 자유롭게 정의 가능
- 각 프리셋은 `width`와 `height` 필수

---

### 크론 설정 (`cron`)

크론 잡 스케줄링 설정입니다.

#### `schedule` (선택)
- **타입**: string
- **설명**: Cron 표현식 형식의 실행 스케줄
- **기본값**: `"0 2 * * *"` (매일 새벽 2시)
- **형식**: `"분 시 일 월 요일"`
- **예시**:
  - `"0 2 * * *"` - 매일 새벽 2시
  - `"0 */6 * * *"` - 6시간마다
  - `"0 0 * * 0"` - 매주 일요일 자정

#### `enabled` (선택)
- **타입**: boolean
- **설명**: 크론 잡 활성화 여부
- **기본값**: `true`
- **사용**: 개발/테스트 환경에서 비활성화 가능

**예시**:
```yaml
cron:
  schedule: "0 2 * * *"
  enabled: true
```

**Cron 표현식 참고**:
```
분(0-59) 시(0-23) 일(1-31) 월(1-12) 요일(0-7, 0과 7은 일요일)

특수 문자:
* : 모든 값
, : 값 목록 (예: 1,3,5)
- : 범위 (예: 1-5)
/ : 간격 (예: */2는 2마다)
```

---

### 서버 설정 (`server`)

HTTP 서버 설정입니다.

#### `port` (선택)
- **타입**: integer
- **설명**: HTTP 서버 포트 번호
- **기본값**: `8080`
- **예시**: `8080`, `3000`, `9000`

#### `timeout_seconds` (선택)
- **타입**: integer
- **설명**: 요청 타임아웃 시간 (초)
- **기본값**: `30`
- **권장값**: 큰 이미지 처리 시 더 긴 시간 설정

**예시**:
```yaml
server:
  port: 8080
  timeout_seconds: 30
```

---

## 전체 설정 파일 예시

```yaml
# Cloudflare R2 설정
r2:
  access_key: "your-access-key-here"
  secret_key: "your-secret-key-here"
  endpoint: "https://abc123def456.r2.cloudflarestorage.com"
  bucket: "my-image-bucket"

# 이미지 변환 설정
conversion:
  formats: ["jpeg", "jpg", "png", "gif", "bmp", "tiff"]
  quality: 85
  max_size_mb: 50

# 리사이징 프리셋
resize:
  presets:
    thumbnail:
      width: 150
      height: 150
    medium:
      width: 800
      height: 800
    large:
      width: 1920
      height: 1920

# 크론 잡 설정
cron:
  schedule: "0 2 * * *"  # 매일 새벽 2시
  enabled: true

# 서버 설정
server:
  port: 8080
  timeout_seconds: 30
```

---

## 환경 변수 지원

민감한 정보는 환경 변수로 제공할 수 있습니다. 환경 변수는 설정 파일의 값보다 우선순위가 높습니다.

### 지원하는 환경 변수

| 환경 변수 | 설정 파일 경로 | 설명 |
|----------|---------------|------|
| `R2_ACCESS_KEY` | `r2.access_key` | R2 Access Key |
| `R2_SECRET_KEY` | `r2.secret_key` | R2 Secret Key |
| `R2_ENDPOINT` | `r2.endpoint` | R2 Endpoint URL |
| `R2_BUCKET` | `r2.bucket` | R2 Bucket 이름 |
| `SERVER_PORT` | `server.port` | 서버 포트 |

### 환경 변수 사용 예시

```bash
# .env 파일 또는 환경 변수 설정
export R2_ACCESS_KEY="your-access-key"
export R2_SECRET_KEY="your-secret-key"
export R2_ENDPOINT="https://abc123def456.r2.cloudflarestorage.com"
export R2_BUCKET="my-image-bucket"

# 서버 실행
go run main.go
```

또는 `.env` 파일 사용:
```bash
# .env 파일
R2_ACCESS_KEY=your-access-key
R2_SECRET_KEY=your-secret-key
R2_ENDPOINT=https://abc123def456.r2.cloudflarestorage.com
R2_BUCKET=my-image-bucket
```

---

## 설정 검증

서버 시작 시 다음 항목들이 검증됩니다:

1. **필수 필드 확인**
   - `r2.access_key`
   - `r2.secret_key`
   - `r2.endpoint`
   - `r2.bucket`

2. **값 유효성 검사**
   - `conversion.quality`: 0-100 범위
   - `conversion.max_size_mb`: 양수
   - `server.port`: 1-65535 범위
   - `cron.schedule`: 유효한 Cron 표현식

3. **R2 연결 테스트** (선택적)
   - 설정 로드 후 R2 연결 가능 여부 확인

---

## 보안 고려사항

### 1. 민감 정보 관리

**권장 사항**:
- `access_key`와 `secret_key`는 환경 변수로 관리
- 설정 파일에는 실제 값 대신 플레이스홀더 사용
- 설정 파일을 버전 관리에 포함하지 않음 (`.gitignore`에 추가)

**설정 파일 예시** (민감 정보 제외):
```yaml
r2:
  access_key: "${R2_ACCESS_KEY}"  # 환경 변수 참조
  secret_key: "${R2_SECRET_KEY}"  # 환경 변수 참조
  endpoint: "https://abc123def456.r2.cloudflarestorage.com"
  bucket: "my-image-bucket"
```

### 2. 파일 권한

설정 파일의 권한을 제한합니다:
```bash
chmod 600 config/config.yaml
```

### 3. R2 권한 최소화

R2 API 키는 필요한 최소 권한만 부여:
- 특정 버킷에 대한 읽기/쓰기 권한만
- 다른 버킷이나 리소스에 대한 접근 권한 없음

---

## 설정 파일 템플릿

새 프로젝트를 시작할 때 사용할 수 있는 최소 설정 템플릿:

```yaml
r2:
  access_key: ""  # 환경 변수 R2_ACCESS_KEY 사용 권장
  secret_key: ""  # 환경 변수 R2_SECRET_KEY 사용 권장
  endpoint: ""
  bucket: ""

conversion:
  formats: ["jpeg", "jpg", "png", "gif"]
  quality: 85
  max_size_mb: 50

resize:
  presets:
    thumbnail:
      width: 150
      height: 150
    medium:
      width: 800
      height: 800
    large:
      width: 1920
      height: 1920

cron:
  schedule: "0 2 * * *"
  enabled: true

server:
  port: 8080
  timeout_seconds: 30
```

---

## 설정 변경 시 주의사항

1. **서버 재시작 필요**: 설정 파일 변경 후 서버를 재시작해야 변경사항이 적용됩니다.

2. **크론 스케줄 변경**: 크론 스케줄을 변경하면 다음 실행 시간부터 적용됩니다.

3. **R2 설정 변경**: R2 접속 정보를 변경하면 서버 재시작 후 즉시 적용됩니다.

---

## 트러블슈팅

### 설정 파일을 찾을 수 없음
```
Error: config file not found: config/config.yaml
```
**해결**: 설정 파일이 올바른 경로에 있는지 확인하거나 `-config` 플래그로 경로 지정

### 필수 필드 누락
```
Error: required field missing: r2.access_key
```
**해결**: 설정 파일에 모든 필수 필드가 있는지 확인

### R2 연결 실패
```
Error: failed to connect to R2
```
**해결**: 
- R2 접속 정보가 올바른지 확인
- 네트워크 연결 확인
- R2 버킷이 존재하는지 확인

---

## 참고 문서

- [USAGE.md](./USAGE.md) - 서버 실행 방법
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 시스템 아키텍처
- [CRON.md](./CRON.md) - 크론 잡 설정
