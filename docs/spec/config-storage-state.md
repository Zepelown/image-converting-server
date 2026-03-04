## Config, Storage(R2), State Spec

이 문서는 이미지 변환 서버의 설정(`config`), 스토리지(R2 연동), 상태(state) 관리에 대한
현재 구현 기준 스펙을 정리합니다.

기준 구현 파일:
- 설정: `config/config.go`, `config/config.yaml`, `.env` 관련
- 스토리지: `r2/client.go`
- 상태: `state/state.go`

---

### 1. 설정 로딩 및 구조

#### 1.1 설정 파일 및 .env 처리

- 진입점: `config.Load("config/config.yaml")`
- 처리 순서:
  1. `.env` 파일 로딩 시도
     - `godotenv.Load()` 호출.
     - 파일이 없어도 에러를 무시하고 계속 진행.
  2. `config/config.yaml` 파일 읽기
     - 실패 시: `failed to read config file` 에러 반환.
  3. YAML 파싱 → `Config` 구조체로 언마샬.
     - 실패 시: `failed to parse config file` 에러 반환.
  4. `applyEnvironmentVariables` 호출
     - 특정 필드는 환경 변수로 YAML 값을 덮어씀.
  5. `setDefaults` 호출
     - 선택 필드에 대해 기본값 설정.
  6. `Validate` 호출
     - 필수 값 및 제약 조건 검증.

#### 1.2 Config 구조체

```go
type Config struct {
    R2         R2Config         `yaml:"r2"`
    Conversion ConversionConfig `yaml:"conversion"`
    Resize     ResizeConfig     `yaml:"resize"`
    Cron       CronConfig       `yaml:"cron"`
    Server     ServerConfig     `yaml:"server"`
    Webhook    WebhookConfig    `yaml:"webhook"`
}
```

각 서브 구조체는 아래와 같습니다.

##### R2Config

```yaml
r2:
  access_key: string
  secret_key: string
  endpoint: string
  bucket: string
```

- 모든 필드는 필수이며, 비어 있으면 검증 단계에서 에러.

##### ConversionConfig

```yaml
conversion:
  formats: [ "jpeg", "jpg", "png", "gif", "bmp", "tiff" ] # etc.
  quality: 85
  max_size_mb: 50
```

- `formats` ([]string)
  - 변환 대상 확장자 목록.
  - 비어 있을 경우 기본값:
    - `["jpeg", "jpg", "png", "gif", "bmp", "tiff"]`
- `quality` (int)
  - WebP 인코딩 품질(0~100).
  - 0이면 기본값 85로 설정.
  - 검증: 0~100 범위 밖이면 에러.
- `max_size_mb` (int)
  - 최대 이미지 크기(MB 단위).
  - 0 이하이면 검증 단계에서 에러.

※ 현재 `max_size_mb`는 직접적으로 처리 코드에서 검사하지 않지만,
   스펙 상 제한값으로 유지됨 (향후 구현 확장 포인트).

##### ResizeConfig

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
```

- `presets` (map[string]PresetConfig)
  - 이름 기반 리사이즈 프리셋 집합.
- `PresetConfig`:
  - `width` (int): > 0이어야 함.
  - `height` (int): > 0이어야 함.
- 검증:
  - 각 preset의 width/height가 0 이하이면 에러.

##### CronConfig

```yaml
cron:
  schedule: "0 2 * * *"
  enabled: true
```

- `schedule` (string)
  - Crontab 형식.
  - 비어 있으면 기본 `"0 2 * * *"`로 설정.
- `enabled` (bool)
  - YAML 값 그대로 사용 (별도 기본 로직 없음).
  - `true`여야 실제 크론 잡이 동작.

##### ServerConfig

```yaml
server:
  port: 4000
  timeout_seconds: 30
```

- `port` (int)
  - 1~65535 범위 내 필수 값.
  - 0이면 기본값 4000으로 설정.
- `timeout_seconds` (int)
  - > 0 이어야 함.
  - 0이면 기본값 30으로 설정.
  - HTTP 서버의 `ReadTimeout`/`WriteTimeout`에 모두 사용됨.

##### WebhookConfig

```yaml
webhook:
  url: ""                  # 빈 문자열이면 비활성
  retry_enabled: false
  retry_interval_minutes: 5
  max_retries: 5
  pending_dir: "data/webhook_pending"
```

- `url` (string)
  - 빈 문자열이면:
    - 배치 완료 시 웹훅 전송을 시도하지 않음.
    - `POST /api/webhook/send` 엔드포인트는 503 `webhook_not_configured`.
- `retry_enabled` (bool)
  - `true`이면:
    - 웹훅 전송 실패 시 pending 파일을 저장 (`StorePending`).
    - 애플리케이션 시작 시 재시도 워커(`RunRetryWorker`) 실행.
  - `false`이면 재시도 관련 동작 없음.
- `retry_interval_minutes` (int)
  - 0이면 기본값 5로 설정.
  - `Webhook.RetryEnabled == true`인 경우 1 미만이면 검증 단계에서 에러.
- `max_retries` (int)
  - 0이면 기본값 5로 설정.
  - `Webhook.RetryEnabled == true`인 경우 0 미만이면 검증 단계에서 에러.
- `pending_dir` (string)
  - 비어 있으면 기본 `"data/webhook_pending"`으로 설정.
  - pending 레코드 및 재시도 워커에서 사용.

---

### 2. 환경 변수 오버라이드 규칙

`applyEnvironmentVariables` 단계에서 다음 환경 변수들이 YAML 값을 덮어씁니다.

- `R2_ACCESS_KEY` → `config.R2.AccessKey`
- `R2_SECRET_KEY` → `config.R2.SecretKey`
- `R2_ENDPOINT` → `config.R2.Endpoint`
- `R2_BUCKET` → `config.R2.Bucket`
- `SERVER_PORT` → `config.Server.Port`
  - 정수 파싱 성공 시에만 반영.
- `WEBHOOK_URL` → `config.Webhook.URL`
- `WEBHOOK_RETRY_ENABLED`
  - `"false"` 또는 `"0"` → `RetryEnabled = false`
  - `"true"` 또는 `"1"` → `RetryEnabled = true`
- `WEBHOOK_RETRY_INTERVAL_MINUTES`
  - 정수 파싱 성공 & `>=1`이면 `RetryIntervalMinutes` 덮어씀.
- `WEBHOOK_MAX_RETRIES`
  - 정수 파싱 성공 & `>=0`이면 `MaxRetries` 덮어씀.
- `WEBHOOK_PENDING_DIR`
  - 비어 있지 않은 경우 `PendingDir` 덮어씀.

주의: 환경 변수는 YAML보다 **우선** 적용됩니다.

---

### 3. 설정 검증 규칙 (Validate)

`config.Validate`는 다음 조건을 만족하지 못하면 에러를 반환합니다.

#### 3.1 R2 필수 값

- `config.R2.AccessKey == ""` → 에러: `"required field missing: r2.access_key"`
- `config.R2.SecretKey == ""` → 에러: `"required field missing: r2.secret_key"`
- `config.R2.Endpoint == ""` → 에러: `"required field missing: r2.endpoint"`
- `config.R2.Bucket == ""` → 에러: `"required field missing: r2.bucket"`

#### 3.2 Conversion 설정

- `config.Conversion.Quality < 0 || > 100`
  - 에러: `"conversion.quality must be between 0 and 100, got: %d"`
- `config.Conversion.MaxSizeMB <= 0`
  - 에러: `"conversion.max_size_mb must be positive, got: %d"`

#### 3.3 서버 설정

- `config.Server.Port < 1 || > 65535`
  - 에러: `"server.port must be between 1 and 65535, got: %d"`
- `config.Server.TimeoutSeconds <= 0`
  - 에러: `"server.timeout_seconds must be positive, got: %d"`

#### 3.4 리사이즈 프리셋

- 각 preset에 대해:
  - `width <= 0` → 에러: `"resize.presets.%s.width must be positive, got: %d"`
  - `height <= 0` → 에러: `"resize.presets.%s.height must be positive, got: %d"`

#### 3.5 웹훅 재시도 관련

- `config.Webhook.RetryEnabled == true`인 경우:
  - `RetryIntervalMinutes < 1`
    - 에러: `"webhook.retry_interval_minutes must be at least 1, got: %d"`
  - `MaxRetries < 0`
    - 에러: `"webhook.max_retries must be non-negative, got: %d"`

---

### 4. R2 Storage Client 스펙

R2는 S3 호환 API를 통해 접근하며, `r2.StorageClient` 인터페이스로 추상화됩니다.

#### 4.1 StorageClient 인터페이스

```go
type StorageClient interface {
    DownloadImage(ctx context.Context, key string) ([]byte, error)
    UploadImage(ctx context.Context, key string, data []byte, contentType string) error
    ListObjects(ctx context.Context, since time.Time) ([]string, error)
    TestConnection(ctx context.Context) error
    DeleteObject(ctx context.Context, key string) error
}
```

#### 4.2 NewClient 동작

- 시그니처: `NewClient(ctx, *R2Config) (StorageClient, error)`
- 내부:
  1. `awsConfig.LoadDefaultConfig` 호출
     - `credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")`
     - `region = "auto"` (Cloudflare R2 권장 값).
  2. `s3.NewFromConfig` 생성 시 `BaseEndpoint = cfg.Endpoint`로 설정.
  3. 위 클라이언트를 감싼 `r2Client` 반환.

#### 4.3 메서드별 동작

- `DownloadImage`
  - `GetObject` 호출 후 `Body` 전체를 읽어 `[]byte`로 반환.
  - 실패 시: `"failed to download image from R2 (key: %s): %w"` 형식의 에러.
- `UploadImage`
  - `PutObject` 호출 (`ContentType` 포함).
  - 실패 시: `"failed to upload image to R2 (key: %s): %w"`.
- `ListObjects`
  - `s3.NewListObjectsV2Paginator` 사용.
  - 각 페이지에서:
    - `since.IsZero()`이면 모든 key 추가.
    - 아니면 `obj.LastModified.After(since)`인 경우에만 추가.
- `TestConnection`
  - `HeadBucket` 호출로 버킷 접근 가능 여부 확인.
  - 실패 시: `"failed to connect to R2 bucket %s: %w"`.
- `DeleteObject`
  - `DeleteObject` 호출.
  - 실패 시: `"failed to delete object from R2 (key: %s): %w"`.

#### 4.4 사용 위치 요약

- HTTP API (`/api/convert`)
  - `DownloadImage`, `UploadImage` 사용.
- 배치 크론
  - `ListObjects`, `DownloadImage`, `UploadImage` 사용.
  - (원본 삭제는 현재 주석 처리되어 미사용이지만, 필요 시 `DeleteObject`를 다시 사용할 수 있음.)
- 애플리케이션 초기화
  - `TestConnection`으로 R2 연결 확인.

---

### 5. State 관리 스펙

배치 크론의 실행 이력 및 마지막 처리 시점을 디스크에 저장합니다.

#### 5.1 State 구조

```json
{
  "last_processed_time": "time.RFC3339 string",
  "last_run_time": "time.RFC3339 string",
  "processed_count": 0,
  "failed_count": 0
}
```

- `last_processed_time`
  - 최근에 처리한 기준 시각.
  - 다음 `ListObjects` 호출 시 `since` 인자로 사용.
- `last_run_time`
  - 마지막 배치 실행 시각.
- `processed_count`
  - 마지막 배치에서 성공적으로 처리한 이미지 수.
- `failed_count`
  - 마지막 배치에서 실패한 이미지 수.

#### 5.2 생성 및 로딩

- `NewState()`
  - 모든 필드를 zero 값으로 초기화.
- `LoadState(filePath)`
  - `os.ReadFile(filePath)` 실패 시:
    - `os.IsNotExist`이면 `NewState()`를 반환 (에러 없음).
    - 그 외 에러는 호출자에게 그대로 전달.
  - JSON 언마샬 실패 시 에러 반환.

#### 5.3 저장

- `SaveState(filePath, state)`
  - 디렉터리 존재 여부 확인 후, 없으면 `os.MkdirAll(dir, 0755)`로 생성.
  - `json.MarshalIndent`로 직렬화.
  - `{filePath}.tmp`에 먼저 쓰고, `os.Rename(tmp, filePath)`로 원자적 교체.
  - 임시 파일 쓰기/rename 실패 시 적절한 에러 반환.

#### 5.4 헬퍼 메서드

- `UpdateLastProcessedTime(t time.Time)`
  - `LastProcessedTime = t`.
- `UpdateLastRunTime()`
  - `LastRunTime = time.Now()`.

#### 5.5 배치와의 관계

- `cron.Job.ProcessImages()`에서:
  - 시작 시 `LoadState(statePath)` 호출.
  - `ListObjects(ctx, state.LastProcessedTime)`로 신규/변경된 객체만 조회.
  - 처리 후:
    - `ProcessedCount`/`FailedCount` 갱신.
    - `LastRunTime = startTime`.
    - `UpdateLastProcessedTime(startTime)` 호출.
    - `SaveState(statePath, currentState)` 저장.

---

### 6. 현재 버전의 제약 및 향후 확장 포인트

- **MaxSizeMB 미사용**
  - `conversion.max_size_mb`는 설정/검증만 하고 실제 처리 코드에서 크기 제한 검사를 하지 않음.
  - 향후 업로드/다운로드 전후로 크기 체크를 추가할 수 있음.
- **타임존**
  - 시간 값(`LastProcessedTime`, `LastRunTime`, `CreatedAt` 등)은 Go의 `time.Time` 기본 직렬화 형식을 사용.
  - 내부적으로 UTC를 사용하지만, 설정/로그 등에서 별도의 타임존 변환 로직은 없음.
- **멀티리전/멀티버킷**
  - 현재는 단일 `endpoint`와 단일 `bucket`만 지원.
  - `r2://bucket/key` 형식 입력 중 `bucket`은 무시하고 항상 `config.R2.Bucket`을 사용.
- **다중 인스턴스 환경**
  - `state.json` 및 `.lock`, `webhook_pending` 디렉터리를 공유하는 다중 인스턴스에서의 동시성은 기본적인 파일 시스템 보장에 의존.
  - 필요 시 분산 락/스토어로 마이그레이션 가능.

