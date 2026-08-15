## Batch Cron (정기 이미지 변환) Spec

이 문서는 정기 배치 이미지 변환 기능(크론 잡)의 현재 동작을 정리한 스펙입니다.
구현 기준 파일: `cron/job.go`, `state/state.go`, `config/config.go`, `r2/client.go`, `webhook/*`.

---

### 1. 개요

- 기능 이름: **배치 이미지 변환 크론 잡**
- 역할:
  - R2 버킷에서 아직 WebP로 변환되지 않은 지원 포맷 이미지를 주기적으로 탐색.
  - 이미지를 WebP로 변환 후 다시 R2에 업로드.
  - 실행 결과를 상태 파일(`state.json`)에 저장.
  - 설정된 경우, 배치 완료 후 웹훅 호출.
- 동작 위치:
  - `main.go`에서 애플리케이션 시작 시 `cron.NewJob(...)` + `Start()` 호출.

---

### 2. 설정 및 활성화 조건

#### 2.1 Cron 설정 (`config.CronConfig`)

- 필드:
  - `schedule` (string)
    - Crontab 형식 문자열.
    - 기본값: `"0 2 * * *"` (매일 새벽 2시)
  - `enabled` (bool)
    - `true`일 때만 크론 잡이 실제로 등록 및 실행됨.
    - `CRON_ENABLED=true` 또는 `CRON_ENABLED=1`일 때 활성화.
    - env 미설정 시 false.

#### 2.2 시작/중지 동작

- `NewJob(cfg, r2Client, proc, statePath)`로 `Job` 생성.
  - `statePath` 기본값: `"data/state.json"` (main에서 고정 문자열로 전달).
  - `lockPath`: `filepath.Join(filepath.Dir(statePath), ".lock")` → 기본 `"data/.lock"`.
- `Job.Start()`
  - `cfg.Cron.Enabled`가 `false`이면:
    - `[INFO] Cron job is disabled` 로그 후 바로 `nil` 리턴 (크론 미등록).
  - `cron.AddFunc(cfg.Cron.Schedule, j.ProcessImages)` 등록.
  - `cron.Start()` 호출 후
    - `[INFO] Cron job started with schedule: {schedule}` 로그.
- `Job.Stop()`
  - `cron.Stop()`만 호출 (기타 정리 동작 없음).
  - `main.go`에서 `defer cronJob.Stop()`로 서버 종료 시 함께 호출됨.

---

### 3. 락(lock) 메커니즘

동일 시점에 배치가 중복 실행되는 것을 방지하기 위해 파일 기반 락을 사용합니다.

#### 3.1 락 위치

- `lockPath = {dirname(statePath)}/.lock`
  - 기본값: `data/.lock`

#### 3.2 락 획득(`acquireLock`)

1. 기존 락 파일 존재 여부 확인 (`os.Stat(lockPath)`).
2. 락 파일이 존재하는 경우:
   - `modTime`이 1시간보다 오래된 경우:
     - "stale lock"으로 판단하고 제거:
       - `[WARN] Found stale lock file, removing...`
       - `os.Remove(lockPath)`
   - 1시간 이내에 생성/수정된 락 파일인 경우:
     - 에러 리턴: `"cron job is already running (lock file exists: {lockPath})"`.
3. 디렉터리 생성 (`os.MkdirAll(dir, 0755)`).
4. `os.OpenFile(lockPath, O_CREATE|O_EXCL|O_WRONLY, 0644)`로 락 파일 생성.
   - 이미 존재하면 `"cron job is already running"` 에러.
5. 모든 과정 성공 시 `nil` 리턴.

#### 3.3 락 해제(`releaseLock`)

- `os.Remove(lockPath)` 호출.
- 실패 시:
  - `[ERROR] Failed to release lock: {err}` 로그만 남기고 배치는 계속 종료.

#### 3.4 배치 실행과의 관계

- `ProcessImages()` 시작 시:
  - `acquireLock()` 호출 실패 → 에러 로그 후 바로 return.
- `defer releaseLock()`으로 항상 종료 시 락 파일 제거 시도.

---

### 4. 처리 대상 이미지 선정 로직

#### 4.1 상태 로딩

- `state.LoadState(statePath)` 호출.
- 상태 파일이 없으면:
  - 새 초기 상태(`NewState()`) 생성 후 사용.
- 상태 구조 (`state.State`):
  - `last_processed_time` (`time.Time`)
  - `last_run_time` (`time.Time`)
  - `processed_count` (`int`)
  - `failed_count` (`int`)

#### 4.2 R2 객체 목록 조회

- `r2Client.ListObjects(ctx, currentState.LastProcessedTime)`
  - `since` 파라미터로 `LastProcessedTime` 전달.
- `ListObjects` 구현 세부:
  - R2 버킷 전체 페이지네이션 조회.
  - `since.IsZero()`이면 모든 객체 포함.
  - 그렇지 않으면 `obj.LastModified.After(since)` 인 객체만 포함.

#### 4.3 확장자 필터링

- 각 key에 대해:
  1. `strings.HasSuffix(strings.ToLower(key), ".webp")`이면 **건너뜀**.
  2. `isSupportedExtension(key)` 검사:
     - `filepath.Ext(key)` → 소문자로 변환.
     - 앞의 `.` 제거 후 확장자 문자열만 비교.
     - `cfg.Conversion.Formats` 배열 중 하나와 일치하면 통과.
       - `"jpeg"`/`"jpg"` 상호 매핑 지원:
         - `ext == "jpeg"` && format == `"jpg"` → 허용.
         - `ext == "jpg"` && format == `"jpeg"` → 허용.
     - 둘 다 아니면 **건너뜀**.

결론: **webp가 아닌** && **지원 포맷 목록에 포함된** 객체만 배치 대상.

---

### 5. 개별 이미지 처리 플로우

각 대상 key마다 다음 단계를 수행합니다.

1. **다운로드**
   - `r2Client.DownloadImage(ctx, key)`
   - 실패 시:
     - `[ERROR] Failed to download image {key}: {err}`
     - `failedCount++`
     - 다음 key로 continue.
2. **변환**
   - `processor.Processor.Process(data, processor.ProcessOptions{})`
   - 현재 배치에서는 리사이즈 옵션 없이 순수 포맷 변환만 수행.
   - 실패 시:
     - `[ERROR] Failed to convert image {key}: {err}`
     - `failedCount++`
     - 다음 key로 continue.
3. **업로드**
   - `destKey := changeExtensionToWebp(key)`
     - 확장자가 없으면 `key + ".webp"`.
     - 있으면 기존 확장자를 `.webp`로 치환.
   - `r2Client.UploadImage(ctx, destKey, webpData, "image/webp")`
   - 실패 시:
     - `[ERROR] Failed to upload converted image {destKey}: {err}`
     - `failedCount++`
     - 다음 key로 continue.
4. **성공 처리**
   - `[INFO] Successfully converted {key} to {destKey}`
   - `processedCount++`
   - `converted` 리스트에 `webhook.ImageEntry{Source: key, Destination: destKey}` 추가.
5. **원본 삭제**
   - 현재 구현에서는 삭제 로직이 **주석 처리**되어 있어, 원본은 항상 유지됨.

---

### 6. 상태(state) 업데이트 및 저장

배치 처리 루프가 끝난 후, 다음 값을 업데이트합니다.

- `currentState.ProcessedCount = processedCount`
- `currentState.FailedCount = failedCount`
- `currentState.LastRunTime = startTime`
- `currentState.UpdateLastProcessedTime(startTime)`
  - 즉, **이번 배치 시작 시각**을 기준으로 이후 리스트업에 사용할 `since`가 갱신됩니다.
  - 이로 인해 다음 배치에서는 이 시각 이후에 수정된 객체만 대상으로 삼게 됨.

저장 방식:

- `state.SaveState(statePath, currentState)`
  - 디렉터리가 없으면 `os.MkdirAll(dir, 0755)`로 생성.
  - `{statePath}.tmp`에 먼저 쓰고, 이후 `os.Rename(tmp, real)`로 원자적 교체.
  - 저장 실패 시:
    - `[ERROR] Failed to save state: {err}` 로그 후에도 배치 자체는 성공으로 간주.

---

### 7. 웹훅 연동 (배치 완료 알림)

#### 7.1 활성화 조건

- `cfg.Webhook.URL`이 비어 있지 않을 때만 실행.
- `cfg.Webhook.RetryEnabled` 여부에 따라 실패 시 재시도 저장 여부가 달라짐.

#### 7.2 페이로드 내용 (`webhook.BatchPayload`)

```json
{
  "event": "batch.completed",
  "processed_count": <processedCount>,
  "failed_count": <failedCount>,
  "images": [
    { "source": "string", "destination": "string" }
  ]
}
```

- `images` 배열에는 **성공적으로 변환된 항목만** 포함.

#### 7.3 동작

1. 배치 루프 종료 후, `cfg.Webhook.URL != ""`일 때:
   - 위 구조의 `BatchPayload` 생성.
2. `webhook.SendBulk(ctx, cfg.Webhook.URL, payload, 10 * time.Second)` 호출.
3. 실패 시:
   - `[WARN] Webhook send failed (batch completed successfully): {err}`
   - `cfg.Webhook.RetryEnabled == true`이면:
     - `webhook.StorePending(cfg.Webhook.PendingDir, cfg.Webhook.URL, payload)` 호출.
     - 이때 실패하면 `[ERROR] Webhook: failed to store pending for retry: {storeErr}` 로그.
   - 배치 자체는 **성공으로 이미 끝난 상태**이며, 리턴 에러 없이 종료.

---

### 8. 로그 메시지 패턴

현재 구현에 기반한 주요 로그 패턴은 다음과 같습니다.

- 크론 시작/비활성화
  - `[INFO] Cron job is disabled`
  - `[INFO] Cron job started with schedule: {cronExpr}`
- 실행 시작/종료
  - `[INFO] Cron job execution started`
  - `[INFO] Listing bucket: {bucket} (since: {timestamp or 'beginning (full scan)'})`
  - `[INFO] Found {N} objects to check`
  - `[INFO] Cron job execution completed. Processed: {processed}, Failed: {failed}, Duration: {duration}`
- 에러/경고
  - `[ERROR] Failed to acquire lock: {err}`
  - `[ERROR] Failed to load state: {err}`
  - `[ERROR] Failed to list objects from R2: {err}`
  - `[ERROR] Failed to download image {key}: {err}`
  - `[ERROR] Failed to convert image {key}: {err}`
  - `[ERROR] Failed to upload converted image {destKey}: {err}`
  - `[ERROR] Failed to save state: {err}`
  - `[WARN] Found stale lock file, removing...`
  - `[ERROR] Failed to release lock: {err}`
  - `[WARN] Webhook send failed (batch completed successfully): {err}`
  - `[ERROR] Webhook: failed to store pending for retry: {err}`

---

### 9. 현재 버전의 제약/특이사항

- 상태 기반 필터링:
  - `LastProcessedTime`는 배치 **시작 시간**으로만 갱신되며, 객체의 실제 LastModified 기준이 아니라는 점에 유의.
- 리사이즈 옵션:
  - 크론 배치에서는 `ProcessOptions{}`를 사용하므로, 리사이즈 없이 순수 포맷 변환만 수행.
- 원본 삭제:
  - 원본 이미지 삭제 로직은 모두 주석 처리되어 있어, 실제로는 삭제하지 않음.
- 동시 실행 방지:
  - 단일 애플리케이션 프로세스 내에서만 파일 락으로 중복 실행을 제어.
  - 여러 인스턴스가 같은 `data` 디렉터리를 공유하는 경우, 파일 락을 통해 다중 인스턴스 간 경쟁도 어느 정도 방지할 수 있음. 단, 네트워크 파일시스템 환경 등에서는 보장되지 않을 수 있음.

