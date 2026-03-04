## HTTP API Spec

이 문서는 현재 구현된 HTTP API의 실제 동작을 정리한 스펙입니다.
모든 내용은 `main.go`, `api/handlers.go`, `docs/API.md`를 기준으로 합니다.

---

### 공통

- **Base URL**: `http://localhost:{SERVER_PORT}`
  - 기본값: `4000` (`config.server.port`가 0이거나 설정되지 않은 경우)
- **Content-Type**
  - 요청: 엔드포인트별로 상이 (JSON 바디가 있는 경우 `application/json`)
  - 응답: 성공 및 에러 모두 `application/json`
- **인증**
  - 현재 버전에서는 모든 엔드포인트에 인증이 필요하지 않음.

에러 응답 공통 포맷 (`api.ErrorResponse`):

```json
{
  "success": false,
  "error": "string",
  "message": "string"
}
```

---

### 1. `GET /` (인덱스)

- **핸들러**: `Handler.HandleIndex`
- **메서드**: `GET`
- **경로 매칭 규칙**
  - `r.URL.Path`가 정확히 `/`인 경우에만 200 응답.
  - 그 외 경로로 들어오면 `404 Not Found` + JSON 에러 응답으로 처리.

#### 정상 응답

- **HTTP Status**: `200 OK`
- **Body**

```json
{
  "message": "Image Converting Server"
}
```

#### 에러 응답

- **경로가 `/`가 아닌 경우**
  - **HTTP Status**: `404 Not Found`
  - **Body**

  ```json
  {
    "success": false,
    "error": "not_found",
    "message": "The requested resource was not found"
  }
  ```

---

### 2. `GET /health` (헬스 체크)

- **핸들러**: `Handler.HandleHealth`
- **메서드**: `GET`
- **기능**: 서버가 살아 있는지만 단순 확인 (추가적인 의존성 체크는 없음).

#### 정상 응답

- **HTTP Status**: `200 OK`
- **Body**

```json
{
  "status": "ok"
}
```

---

### 3. `/api/convert` (이미지 변환)

- **핸들러**: `Handler.HandleConvert`
- **지원 메서드**: `GET`, `POST`
- **기능 개요**
  - 입력된 이미지를 WebP로 변환하고, 필요 시 리사이즈.
  - 변환된 결과를 R2 버킷에 업로드.
  - 응답으로 원본/결과 경로 및 크기를 JSON으로 반환.

#### 3.1 요청 파라미터

##### 공통 입력: `source`

- **필수**: 예
- **위치**
  - `GET`: 쿼리 파라미터 `source`
  - `POST`: JSON 바디 필드 `source`
- **형식**
  - R2 객체: `r2://bucket-name/object-key`
    - 실제 구현에서는 `bucket-name`은 무시하고 `object-key`만 사용함.
  - 외부 URL: `http://...` 또는 `https://...`

##### 리사이즈 옵션 (쿼리 파라미터)

- `width` (선택, 정수)
  - 0 이상 정수.
  - 0 미만이거나 숫자가 아니면 `400 invalid_resize_params`.
- `height` (선택, 정수)
  - 0 이상 정수.
  - 0 미만이거나 숫자가 아니면 `400 invalid_resize_params`.
- `preset` (선택, 문자열)
  - `config.resize.presets` 키 중 하나여야 함.
  - 존재하지 않는 이름이면 `400 invalid_preset`.

`width`/`height`/`preset` 동작:

- `width` 또는 `height` 중 하나라도 > 0 이면, 해당 값으로 직접 리사이즈.
- 둘 다 0이고 `preset`이 설정된 경우, preset에 정의된 `(width, height)`로 리사이즈.
- 모두 비어 있으면 리사이즈 없이 포맷 변환만 수행.

#### 3.2 요청 형식

##### `GET /api/convert`

- 쿼리 파라미터:
  - `source` (필수)
  - `width`, `height`, `preset` (선택)

##### `POST /api/convert`

- 헤더:
  - `Content-Type: application/json`
- 바디(JSON):

```json
{
  "source": "string (required)"
}
```

쿼리 파라미터로 리사이즈 옵션(`width`, `height`, `preset`)을 함께 전달할 수 있음.

#### 3.3 내부 처리 플로우 (요약)

1. 메서드에 따라 `source` 추출
   - `GET`: `source` 쿼리값.
   - `POST`: JSON 바디 디코딩 실패 시 `400 invalid_request`.
2. `source`가 비어 있으면 `400 missing_source`.
3. 쿼리에서 리사이즈 옵션 파싱
   - `width` / `height` 정수 변환 실패 또는 음수면 `400 invalid_resize_params`.
   - `preset`이 설정됐지만 config에 없음 → `400 invalid_preset`.
4. 원본 이미지 다운로드
   - `source`가 `r2://`로 시작:
     - `r2://bucket/key` 형식 강제.
     - `/` 기준 2조각 미만이면 `400 invalid_source_format`.
     - bucket 부분은 무시하고 key만 사용.
     - `DownloadImage` 실패 시 `404 image_not_found`.
   - `source`가 `http://` 또는 `https://`:
     - `http.Get` 호출.
     - 네트워크/IO 에러 또는 HTTP status 200이 아니면 `404 url_not_accessible`.
   - 그 외:
     - `400 invalid_source_format`.
5. Processor를 이용해 변환 (`processor.Processor.Process`)
   - 지원되지 않는 이미지 포맷이거나 디코딩/변환 실패 시
     - `500 conversion_failed` + 실제 에러 메시지를 포함한 문자열.
6. 업로드 대상 key 결정
   - R2 소스가 아니었던 경우(URL 등): `url.Path`를 기반으로 key 생성.
   - key가 비어 있으면 기본 `"downloaded_image"`.
   - 확장자를 `.webp`로 변경.
7. R2 업로드
   - `UploadImage` 실패 시 `500 upload_failed`.
8. 원본 삭제
   - 현재 구현에서는 삭제 로직이 **주석 처리**되어 있어, 원본은 항상 유지됨.
9. 응답 JSON 생성 및 반환.

#### 3.4 정상 응답

- **HTTP Status**: `200 OK`
- **Body** (`ConvertResponse`):

```json
{
  "success": true,
  "message": "Image converted successfully",
  "source": "원본 source 값 그대로",
  "destination": "r2://{config.r2.bucket}/{destKey}",
  "original_size": 1024000,
  "converted_size": 512000,
  "width": 800,
  "height": 600
}
```

- `width` / `height` 필드는 다음 중 하나로 설정됨:
  - `options.Width` / `options.Height`가 > 0 인 경우 해당 값.
  - `preset` 사용 시, 해당 preset의 `(width, height)`.
  - 둘 다 없으면 응답에서 생략(`omitempty`).

#### 3.5 에러 응답

- **공통 에러 코드** (실제 구현 기준)
  - `invalid_request` (400): JSON 바디 파싱 실패.
  - `missing_source` (400): `source` 누락.
  - `invalid_resize_params` (400): `width`/`height` 파싱 실패 또는 음수.
  - `invalid_preset` (400): 존재하지 않는 preset.
  - `invalid_source_format` (400): `source` 형식이 `r2://bucket/key` 또는 `http(s)://`가 아님, 혹은 R2 형식이 잘못됨.
  - `image_not_found` (404): R2 다운로드 실패.
  - `url_not_accessible` (404): 외부 URL 접근 실패 또는 비-200 응답.
  - `conversion_failed` (500): 포맷 변환 실패 또는 디코딩 실패.
  - `upload_failed` (500): R2 업로드 실패.

메서드가 `GET`/`POST`가 아닌 경우:

- **HTTP Status**: `405 Method Not Allowed`
- **Body**

```json
{
  "success": false,
  "error": "method_not_allowed",
  "message": "Method not allowed"
}
```

이때 `Allow` 헤더는 `"GET, POST"`로 설정됨.

---

### 4. `POST /api/webhook/send` (웹훅 수동 발송)

- **핸들러**: `Handler.HandleTriggerWebhook`
- **메서드**: `POST` (기타 메서드는 405)
- **기능 개요**
  - 배치 완료와 무관하게, 지정한 이미지 목록을 payload로 하여 웹훅 URL로 즉시 전송.
  - 어드민/운영자가 테스트 또는 특정 이미지에 대한 알림을 보낼 때 사용.

#### 4.1 사전 조건

- `config.webhook.url`이 **빈 문자열이 아니어야 함**.
  - 비어 있는 경우:
    - **HTTP Status**: `503 Service Unavailable`
    - **에러 코드**: `webhook_not_configured`

#### 4.2 요청 형식

- 헤더:
  - `Content-Type: application/json`
- 바디(JSON):

```json
{
  "images": [
    { "source": "string", "destination": "string" }
  ]
}
```

- `images`
  - 필수, 길이 >= 1.
  - 각 요소는 `webhook.ImageEntry` 구조와 동일.

잘못된 요청의 경우:

- JSON 파싱 실패 → `400 invalid_request`.
- `images`가 비어 있음 → `400 empty_images`.

#### 4.3 내부 처리 플로우 (요약)

1. 메서드 확인: `POST`가 아니면 405 + `method_not_allowed`.
2. `config.webhook.url` 체크, 비어 있으면 503 + `webhook_not_configured`.
3. JSON 바디 파싱 (`TriggerWebhookRequest`).
4. `images`가 비어 있으면 400 + `empty_images`.
5. `webhook.BatchPayload` 생성
   - `event`: `"manual.triggered"`
   - `processed_count`: `len(images)`
   - `failed_count`: `0`
   - `images`: 요청 바디 그대로 사용.
6. `webhook.SendBulk` 호출 (타임아웃 10초).
7. 실패 시:
   - `config.webhook.retry_enabled`가 true이면 `webhook.StorePending`으로 재시도용 레코드 저장.
   - 응답은 항상 502 + `webhook_delivery_failed`.
8. 성공 시:
   - `200 OK` + `{ "success": true, "message": "Webhook sent" }`.

#### 4.4 에러 응답 요약

- `method_not_allowed` (405): POST 이외의 메서드.
- `webhook_not_configured` (503): `WEBHOOK_URL`/`config.webhook.url` 미설정.
- `invalid_request` (400): JSON 파싱 실패.
- `empty_images` (400): images 배열이 비어 있음.
- `webhook_delivery_failed` (502): 외부 웹훅 엔드포인트 호출 실패 (네트워크/HTTP 오류).

---

### 5. 타임아웃 및 서버 설정 관련

- HTTP 서버 전체 타임아웃은 `config.server.timeout_seconds`에 의해 제어됨.
  - `ReadTimeout`, `WriteTimeout` 모두 동일 값이 사용됨.
  - 기본값: 30초 (`config.server.timeout_seconds`가 0이면 30으로 설정).

