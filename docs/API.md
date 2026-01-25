# API 명세서

Image Converting Server의 HTTP API 엔드포인트 상세 명세입니다.

## 기본 정보

- **Base URL**: `http://localhost:8080`
- **Content-Type**: `application/json`
- **인증**: 현재 버전에서는 인증이 필요하지 않습니다 (필요시 추후 추가)

## 엔드포인트 목록

### 1. 메인 엔드포인트

#### `GET /`

서버 정보를 반환합니다.

**요청**:
```http
GET / HTTP/1.1
Host: localhost:8080
```

**응답** (200 OK):
```json
{
  "message": "Image Converting Server"
}
```

---

### 2. 헬스 체크

#### `GET /health`

서버 상태를 확인합니다.

**요청**:
```http
GET /health HTTP/1.1
Host: localhost:8080
```

**응답** (200 OK):
```json
{
  "status": "ok"
}
```

---

### 3. 이미지 변환

#### `POST /api/convert`

이미지를 WebP로 변환하고 선택적으로 리사이징합니다.

**요청**:

**헤더**:
```http
POST /api/convert HTTP/1.1
Host: localhost:8080
Content-Type: application/json
```

**본문** (JSON):
```json
{
  "source": "r2://my-bucket/images/photo.jpg"
}
```

또는 외부 URL:
```json
{
  "source": "https://example.com/image.png"
}
```

**쿼리 파라미터** (선택적):
- `width` (integer): 리사이징할 너비 (픽셀)
- `height` (integer): 리사이징할 높이 (픽셀)
- `preset` (string): 프리셋 크기 이름 (`thumbnail`, `medium`, `large`)

**예시**:
```http
POST /api/convert?width=800&height=600 HTTP/1.1
Content-Type: application/json

{
  "source": "r2://my-bucket/images/photo.jpg"
}
```

또는 프리셋 사용:
```http
POST /api/convert?preset=medium HTTP/1.1
Content-Type: application/json

{
  "source": "r2://my-bucket/images/photo.jpg"
}
```

**응답** (200 OK):
```json
{
  "success": true,
  "message": "Image converted successfully",
  "source": "r2://my-bucket/images/photo.jpg",
  "destination": "r2://my-bucket/images/photo.webp",
  "original_size": 1024000,
  "converted_size": 512000,
  "width": 800,
  "height": 600
}
```

**에러 응답** (400 Bad Request):
```json
{
  "success": false,
  "error": "invalid_source_format",
  "message": "Source must be either r2://bucket/key or https:// URL"
}
```

**에러 응답** (404 Not Found):
```json
{
  "success": false,
  "error": "image_not_found",
  "message": "Image not found in R2 bucket"
}
```

**에러 응답** (500 Internal Server Error):
```json
{
  "success": false,
  "error": "conversion_failed",
  "message": "Failed to convert image: invalid image format"
}
```

---

#### `GET /api/convert`

GET 방식으로 이미지 변환을 요청합니다.

**요청**:

**쿼리 파라미터**:
- `source` (string, 필수): 이미지 소스 (R2 키 또는 URL)
- `width` (integer, 선택): 리사이징할 너비
- `height` (integer, 선택): 리사이징할 높이
- `preset` (string, 선택): 프리셋 크기 이름

**예시**:
```http
GET /api/convert?source=r2://my-bucket/images/photo.jpg&width=800&height=600 HTTP/1.1
Host: localhost:8080
```

또는 프리셋 사용:
```http
GET /api/convert?source=r2://my-bucket/images/photo.jpg&preset=medium HTTP/1.1
Host: localhost:8080
```

**응답**: POST와 동일

---

## 요청/응답 스키마

### 변환 요청 (POST 본문)

```json
{
  "source": "string (required)"
}
```

**source 형식**:
- R2 객체: `r2://bucket-name/object-key`
- 외부 URL: `https://example.com/image.jpg`

### 변환 응답 (성공)

```json
{
  "success": true,
  "message": "string",
  "source": "string",
  "destination": "string",
  "original_size": "integer (bytes)",
  "converted_size": "integer (bytes)",
  "width": "integer (optional)",
  "height": "integer (optional)"
}
```

### 에러 응답

```json
{
  "success": false,
  "error": "string",
  "message": "string"
}
```

---

## 에러 코드

| HTTP 상태 코드 | 에러 코드 | 설명 |
|---------------|----------|------|
| 400 | `invalid_source_format` | 소스 형식이 올바르지 않음 |
| 400 | `missing_source` | source 파라미터가 누락됨 |
| 400 | `invalid_resize_params` | 리사이징 파라미터가 올바르지 않음 |
| 400 | `invalid_preset` | 존재하지 않는 프리셋 이름 |
| 404 | `image_not_found` | R2에서 이미지를 찾을 수 없음 |
| 404 | `url_not_accessible` | 외부 URL에 접근할 수 없음 |
| 500 | `conversion_failed` | 이미지 변환 실패 |
| 500 | `upload_failed` | R2 업로드 실패 |
| 500 | `internal_error` | 내부 서버 오류 |

---

## 사용 예시

### cURL 예시

#### 1. R2 이미지 변환 (기본)
```bash
curl -X POST http://localhost:8080/api/convert \
  -H "Content-Type: application/json" \
  -d '{"source": "r2://my-bucket/images/photo.jpg"}'
```

#### 2. R2 이미지 변환 + 리사이징
```bash
curl -X POST "http://localhost:8080/api/convert?width=800&height=600" \
  -H "Content-Type: application/json" \
  -d '{"source": "r2://my-bucket/images/photo.jpg"}'
```

#### 3. 프리셋 크기 사용
```bash
curl -X POST "http://localhost:8080/api/convert?preset=medium" \
  -H "Content-Type: application/json" \
  -d '{"source": "r2://my-bucket/images/photo.jpg"}'
```

#### 4. 외부 URL 변환
```bash
curl -X POST http://localhost:8080/api/convert \
  -H "Content-Type: application/json" \
  -d '{"source": "https://example.com/image.png"}'
```

#### 5. GET 방식 사용
```bash
curl "http://localhost:8080/api/convert?source=r2://my-bucket/images/photo.jpg&width=800&height=600"
```

### JavaScript (Fetch API) 예시

```javascript
// 기본 변환
async function convertImage(source) {
  const response = await fetch('http://localhost:8080/api/convert', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ source }),
  });
  
  const result = await response.json();
  return result;
}

// 리사이징 포함
async function convertAndResize(source, width, height) {
  const url = new URL('http://localhost:8080/api/convert');
  url.searchParams.set('width', width);
  url.searchParams.set('height', height);
  
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ source }),
  });
  
  const result = await response.json();
  return result;
}

// 사용 예시
convertImage('r2://my-bucket/images/photo.jpg')
  .then(result => {
    if (result.success) {
      console.log('변환 완료:', result.destination);
    } else {
      console.error('에러:', result.message);
    }
  });
```

### Python 예시

```python
import requests

# 기본 변환
def convert_image(source):
    url = 'http://localhost:8080/api/convert'
    response = requests.post(
        url,
        json={'source': source}
    )
    return response.json()

# 리사이징 포함
def convert_and_resize(source, width, height):
    url = 'http://localhost:8080/api/convert'
    params = {'width': width, 'height': height}
    response = requests.post(
        url,
        params=params,
        json={'source': source}
    )
    return response.json()

# 사용 예시
result = convert_image('r2://my-bucket/images/photo.jpg')
if result['success']:
    print(f"변환 완료: {result['destination']}")
else:
    print(f"에러: {result['message']}")
```

---

## 리사이징 옵션

### 프리셋 크기

설정 파일에서 정의된 프리셋 크기를 사용할 수 있습니다:

- `thumbnail`: 150x150 픽셀
- `medium`: 800x800 픽셀
- `large`: 1920x1920 픽셀

프리셋은 설정 파일에서 변경 가능합니다. 자세한 내용은 [CONFIG.md](./CONFIG.md) 참조.

### 커스텀 크기

`width`와 `height` 쿼리 파라미터를 사용하여 원하는 크기를 지정할 수 있습니다.

**주의사항**:
- `width`와 `height`를 모두 지정하면 정확히 그 크기로 리사이징됩니다 (비율이 깨질 수 있음)
- 하나만 지정하면 비율을 유지하면서 리사이징됩니다
- 둘 다 지정하지 않으면 리사이징하지 않고 WebP 변환만 수행합니다

---

## 제한사항

- **최대 이미지 크기**: 설정 파일에서 지정 (기본값: 50MB)
- **지원 이미지 포맷**: JPEG, PNG, GIF, BMP, TIFF
- **출력 포맷**: WebP만 지원
- **동시 요청**: 현재 버전에서는 제한 없음 (필요시 추후 추가)

---

## 참고 문서

- [ARCHITECTURE.md](./ARCHITECTURE.md) - 시스템 아키텍처
- [CONFIG.md](./CONFIG.md) - 설정 파일 가이드
- [USAGE.md](./USAGE.md) - 사용 가이드
