## Webhook & Retry Spec

이 문서는 이미지 변환 배치 및 수동 트리거에서 사용되는 웹훅 전송과,
전송 실패 시 재시도 메커니즘의 현재 동작을 정리한 스펙입니다.

기준 구현 파일: `webhook/notify.go`, `webhook/store.go`, `api/handlers.go`, `cron/job.go`.

---

### 1. 개념 및 역할

- **웹훅(Webhook)**
  - 이미지 변환 배치가 완료되었을 때 또는 운영자가 수동으로 요청했을 때,
    외부 시스템(예: CMS, 백오피스)에 처리 결과를 통지하는 HTTP POST 콜백.
- **재시도(Retry)**
  - 웹훅 전송이 실패했을 때, 페이로드를 디스크에 저장해 두고
    별도 백그라운드 워커가 주기적으로 재전송을 시도.

각 기능은 다음과 같이 호출됩니다.

- 배치 완료 시: `cron.Job.ProcessImages()` → `webhook.SendBulk()` (+ 실패 시 `StorePending`)
- 수동 트리거: `POST /api/webhook/send` → `webhook.SendBulk()` (+ 실패 시 `StorePending`)
- 재시도 워커: `main.go`에서 `webhook.RunRetryWorker()`를 고루틴으로 실행.

---

### 2. Webhook 페이로드 구조

#### 2.1 ImageEntry

```json
{
  "source": "string",
  "destination": "string"
}
```

- `source`: 원본 객체 키(또는 경로).
- `destination`: 변환된 객체 키(또는 경로).

#### 2.2 BatchPayload

```json
{
  "event": "string",
  "processed_count": 0,
  "failed_count": 0,
  "images": [
    { "source": "...", "destination": "..." }
  ]
}
```

- `event`
  - `"batch.completed"`: 크론 배치 완료 시.
  - `"manual.triggered"`: `POST /api/webhook/send`에서 수동 호출 시.
- `processed_count`: 성공적으로 처리된 이미지 수.
- `failed_count`: 처리 실패한 이미지 수.
- `images`: 성공적으로 변환된(또는 수동으로 전달된) 이미지 목록.

---

### 3. Webhook 전송 (SendBulk)

#### 3.1 함수 시그니처 및 동작

- 시그니처: `SendBulk(ctx context.Context, url string, payload *BatchPayload, timeout time.Duration) error`
- 동작:
  1. `url`이 빈 문자열이면 아무 작업도 하지 않고 `nil` 반환.
  2. `payload`를 JSON으로 직렬화.
     - 실패 시 `[ERROR] Webhook: failed to marshal payload` 로그 후 에러 반환.
  3. `http.NewRequestWithContext(ctx, POST, url, body)` 생성.
     - 실패 시 `[ERROR] Webhook: failed to create request` 로그 후 에러 반환.
  4. `Content-Type: application/json` 헤더 설정.
  5. `http.Client{Timeout: timeout}`으로 요청 전송.
     - 네트워크 오류 등 발생 시 `[ERROR] Webhook: POST failed: {err}` 로그 후 에러 반환.
  6. 응답 상태 코드 확인:
     - `2xx`가 아닌 경우:
       - 에러 메시지: `webhook returned status {statusCode}`
       - `[ERROR] Webhook: {err} for {url}` 로그 후 에러 반환.
  7. 모든 과정이 성공하면 `nil` 반환.

#### 3.2 배치/수동 호출 관점

- **배치(`batch.completed`)**
  - 실패하더라도 배치 처리 자체에는 영향을 주지 않으며, 로그 및 재시도 저장만 수행.
- **수동 호출(`manual.triggered`)**
  - 실패 시 API 응답을 `502 Bad Gateway` + `webhook_delivery_failed`로 반환.

---

### 4. Pending 레코드 저장 (StorePending)

웹훅 전송 실패 시, 재시도를 위한 정보를 파일에 저장합니다.

#### 4.1 PendingRecord 구조

```json
{
  "url": "https://example.com/webhook",
  "payload": { ... BatchPayload ... },
  "retry_count": 0,
  "created_at": "2024-01-01T00:00:00Z"
}
```

- `url`: 재시도 시 타겟이 될 웹훅 URL.
- `payload`: 전송에 실패한 `BatchPayload` 전체.
- `retry_count`: 재시도 횟수 (초기값 0).
- `created_at`: 레코드 생성 시간.

#### 4.2 저장 위치 및 파일명

- 인수: `StorePending(pendingDir, url, payload)`
- `pendingDir`
  - 비어 있으면 `defaultPendingDir = "data/webhook_pending"` 사용.
- 저장 경로: `{pendingDir}/{timestamp}-{shortID}.json`
  - `timestamp`: `"20060102-150405"` 형식 (로컬 시간 기준).
  - `shortID`: 4바이트 랜덤을 hex로 인코딩한 8자리 문자열.

#### 4.3 저장 절차

1. `pendingDir` 디렉터리 생성 (`os.MkdirAll`).
2. `PendingRecord` 생성 후 JSON 직렬화 (`MarshalIndent`).
3. 임시 파일에 먼저 기록: `{path}.tmp`.
4. `os.Rename(tmpPath, path)`로 원자적으로 교체.
5. `Rename` 실패 시 임시 파일 삭제 후 에러 반환.

실패 시 호출자 쪽 처리:

- 배치/수동 API 모두, `StorePending` 실패는 **로그만 남기고** 원래 에러 플로우를 유지.

---

### 5. 재시도 워커 (RunRetryWorker)

웹훅 재시도 워커는 애플리케이션 시작 시 별도의 고루틴으로 실행됩니다.

#### 5.1 실행 조건 및 설정

`main.go` 기준:

- `cfg.Webhook.URL != ""` **그리고**
- `cfg.Webhook.RetryEnabled == true`

이면:

```go
go webhook.RunRetryWorker(retryCtx, webhook.RetryWorkerOptions{
    PendingDir:  cfg.Webhook.PendingDir,
    Interval:    time.Duration(cfg.Webhook.RetryIntervalMinutes) * time.Minute,
    MaxRetries:  cfg.Webhook.MaxRetries,
    SendTimeout: 10 * time.Second,
})
```

#### 5.2 RetryWorkerOptions

- `PendingDir` (string)
  - 재시도 대상 JSON 파일들이 저장된 디렉터리.
  - 비어 있으면 `defaultPendingDir`(`data/webhook_pending`) 사용.
- `Interval` (time.Duration)
  - 재시도 사이 간격.
  - 0이면 기본값 `5분`.
- `MaxRetries` (int)
  - 각 레코드별 최대 재시도 횟수.
  - 0 이상 정수여야 하며, `config` 레벨에서 `Webhook.MaxRetries` 검증.
- `SendTimeout` (time.Duration)
  - 각 재시도 요청의 HTTP 타임아웃.
  - 0이면 기본값 `10초`.
- `DeadLetterDir` (string, 선택)
  - 설정된 경우, 최대 재시도 초과 시 해당 디렉터리로 파일을 이동.
  - 비어 있으면 초과 시 해당 pending 파일을 삭제.

#### 5.3 워커 루프 동작

1. `RunRetryWorker(ctx, opts)`
   - `PendingDir`/`Interval`/`SendTimeout`에 기본값 적용.
   - `time.NewTicker(opts.Interval)` 생성.
   - 첫 실행 시 즉시 한 번 `runRetryPass()` 호출 (ticker를 기다리지 않음).
2. 메인 루프:
   - `ctx.Done()` 수신 시 종료.
   - 매 tick마다 `runRetryPass(ctx, opts)` 실행.

#### 5.4 단일 패스 처리 (runRetryPass)

1. `os.ReadDir(opts.PendingDir)`로 디렉터리 읽기.
   - 디렉터리가 없으면 조용히 리턴.
   - 기타 에러 시 `[WARN] Webhook retry: read dir {PendingDir}: {err}` 로그 후 리턴.
2. 각 엔트리에 대해:
   - 디렉터리거나 `.json`이 아니거나 `.tmp`로 끝나면 무시.
   - `path := PendingDir + "/" + name`.
   - `readPendingFile(path)`로 `PendingRecord` 로딩.
     - 실패 시 `[WARN] Webhook retry: read {path}: {err}` 후 다음 파일.
3. `SendBulk(ctx, rec.URL, &rec.Payload, opts.SendTimeout)` 호출.
   - 성공 시:
     - `os.Remove(path)` 시도.
     - 실패 시 `[WARN] Webhook retry: remove {path}: {err}` 로그.
     - 다음 파일로 continue.
4. 실패 시:
   - `rec.RetryCount++`.
   - `rec.RetryCount >= opts.MaxRetries` ?
     - **예**:
       - `DeadLetterDir`가 설정된 경우:
         - `os.MkdirAll(DeadLetterDir, 0755)` 후
         - `dest := DeadLetterDir + "/" + basename(path)`.
         - `os.Rename(path, dest)` 시도.
           - 성공 시: `[INFO] Webhook retry: moved to dead letter after {MaxRetries} retries: {dest}`.
           - 실패 시: `[WARN] Webhook retry: move to dead letter {dest}: {err}` 후 `os.Remove(path)` 시도.
       - `DeadLetterDir`가 비어 있는 경우:
         - `os.Remove(path)` 시도.
         - 성공 시: `[INFO] Webhook retry: removed after {MaxRetries} retries: {path}`.
         - 실패 시: `[WARN] Webhook retry: remove {path}: {err}`.
       - 이후 이 레코드는 더 이상 재시도 대상이 아님.
     - **아니오** (`RetryCount`가 아직 한도 미만):
       - `RetryCount`가 증가된 `PendingRecord`를 다시 파일에 저장.
       - 저장 방식:
         - `json.MarshalIndent(rec)` → `{path}.tmp`에 쓰기 → `os.Rename(tmp, path)`.
         - 중간 실패 시 각각 `[WARN] Webhook retry: marshal/write/rename ...` 로그 후 다음 파일.

---

### 6. API 레벨에서의 사용

#### 6.1 크론 배치 (`batch.completed`)

- 호출 위치: `cron.Job.ProcessImages()` 끝부분.
- 조건: `cfg.Webhook.URL != ""`일 때만.
- 실패 처리:
  - `[WARN] Webhook send failed (batch completed successfully): {err}` 로그.
  - `cfg.Webhook.RetryEnabled`가 `true`이면 `StorePending` 호출.
  - API 레벨 응답은 없고, 배치 잡은 정상 완료로 간주.

#### 6.2 수동 API (`POST /api/webhook/send`, `manual.triggered`)

- 조건:
  - `config.Webhook.URL`이 비어 있으면 아예 호출하지 않고 503 리턴.
- 실패 처리:
  - `SendBulk` 에러 발생 시:
    - `config.Webhook.RetryEnabled`가 `true`이면 `StorePending` 호출.
    - HTTP 응답:

    ```json
    {
      "success": false,
      "error": "webhook_delivery_failed",
      "message": "Webhook delivery failed"
    }
    ```

    - Status: `502 Bad Gateway`.

---

### 7. 설정 관련 연결점 요약

웹훅 및 재시도 동작은 `config.Config`의 다음 필드들에 의해 제어됩니다.

- `webhook.url`
  - 비어 있으면:
    - 배치 완료 시 SendBulk 호출 안 함.
    - `POST /api/webhook/send`는 `503 webhook_not_configured` 반환.
- `webhook.retry_enabled`
  - `true`일 때:
    - SendBulk 실패 시 `StorePending`으로 재시도 큐에 저장.
    - 애플리케이션 시작 시 `RunRetryWorker` 고루틴 실행.
  - `false`일 때:
    - 실패 시 재시도 저장 및 워커 실행 모두 하지 않음.
- `webhook.retry_interval_minutes`
  - 재시도 워커의 `Interval` 값 (분 단위).
  - 0이면 기본값 5분 (코드 레벨 기본값) 사용.
- `webhook.max_retries`
  - 각 레코드별 최대 재시도 횟수.
  - 검증 로직:
    - `RetryEnabled == true`일 때 0 이상이 아닌 경우 에러.
- `webhook.pending_dir`
  - `StorePending` 및 `RunRetryWorker`가 사용하는 디렉터리.
  - 비어 있으면 `data/webhook_pending` 사용.

---

### 8. 현재 버전의 제약/특이사항

- 요청 헤더/인증:
  - 웹훅 요청에는 별도의 인증 헤더나 시그니처가 포함되지 않음.
  - 필요 시 향후 버전에서 확장 가능.
- Dead letter 디렉터리:
  - `RetryWorkerOptions.DeadLetterDir`는 구조상 존재하지만,
    현재 `main.go`에서는 값을 전달하지 않아 기본적으로 사용되지 않음.
  - 즉, `MaxRetries` 초과 시 pending 파일은 삭제되는 것이 기본 동작.
- 실패 원인 구분:
  - 재시도 로직은 HTTP status 코드만 보고 성공/실패를 판단하며,
    `2xx` 이외의 모든 코드는 동일하게 실패로 취급.
- Concurrency:
  - 동일 디렉터리를 여러 워커가 동시에 훑는 상황은 고려하지 않은 구현.
  - 현재는 단일 애플리케이션 인스턴스에서 하나의 워커만 실행된다는 가정을 둠.

