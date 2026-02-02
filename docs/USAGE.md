# 사용 가이드

Image Converting Server의 설치, 실행, 사용 방법을 설명합니다.

## 목차

1. [필수 요구사항](#필수-요구사항)
2. [설치](#설치)
3. [설정 파일 준비](#설정-파일-준비)
4. [서버 실행](#서버-실행)
5. [API 사용 예시](#api-사용-예시)
6. [크론 잡 동작 확인](#크론-잡-동작-확인)
7. [트러블슈팅](#트러블슈팅)

---

## 필수 요구사항

### 소프트웨어 요구사항

- **Go**: 1.21 이상
- **Cloudflare R2**: 계정 및 버킷 필요
- **운영체제**: Linux, macOS, Windows

### Cloudflare R2 설정

1. Cloudflare 계정 생성 및 로그인
2. R2 버킷 생성
3. API 토큰 생성:
   - Cloudflare 대시보드 → R2 → Manage R2 API Tokens
   - Access Key ID와 Secret Access Key 생성
   - 버킷에 대한 읽기/쓰기 권한 부여

---

## 설치

### 1. 저장소 클론 (또는 다운로드)

```bash
git clone <repository-url>
cd image-converting-server
```

또는 직접 다운로드하여 압축 해제

### 2. 의존성 설치

```bash
go mod download
```

### 3. 빌드 (선택적)

```bash
# 실행 파일 빌드
go build -o image-converting-server

# 또는 크로스 컴파일
GOOS=linux GOARCH=amd64 go build -o image-converting-server
```

---

## 설정 파일 준비

### 1. 설정 파일 생성

```bash
mkdir -p config
cp config/config.yaml.example config/config.yaml
```

또는 직접 생성:
```bash
mkdir -p config
touch config/config.yaml
```

### 2. 설정 파일 작성

`config/config.yaml` 파일을 열고 R2 접속 정보를 입력합니다:

```yaml
# Note: R2 접속 정보는 보안을 위해 환경 변수(.env)로 제공해야 합니다.

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
  port: 4000
  timeout_seconds: 30
```

**보안 권장사항**: 민감한 정보는 환경 변수로 관리하세요. 자세한 내용은 [CONFIG.md](./CONFIG.md) 참조.

### 3. 환경 변수 설정 (필수)

R2 접속 정보는 반드시 환경 변수 또는 `.env` 파일로 제공해야 합니다:

```bash
# .env 파일 예시
R2_ACCESS_KEY="your-access-key"
R2_SECRET_KEY="your-secret-key"
R2_ENDPOINT="https://your-account-id.r2.cloudflarestorage.com"
R2_BUCKET="your-bucket-name"
```

---

## 서버 실행

### 개발 모드 (소스에서 직접 실행)

```bash
go run main.go
```

### 프로덕션 모드 (빌드된 실행 파일 사용)

```bash
# 빌드
go build -o image-converting-server

# 실행
./image-converting-server
```

### 커스텀 설정 파일 경로 지정

```bash
go run main.go -config /path/to/custom-config.yaml
```

또는:

```bash
./image-converting-server -config /path/to/custom-config.yaml
```

### 백그라운드 실행 (Linux/macOS)

```bash
nohup ./image-converting-server > server.log 2>&1 &
```

### 시스템 서비스로 등록 (Linux)

#### systemd 서비스 파일 생성

`/etc/systemd/system/image-converting-server.service`:

```ini
[Unit]
Description=Image Converting Server
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/image-converting-server
ExecStart=/path/to/image-converting-server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

#### 서비스 등록 및 시작

```bash
sudo systemctl daemon-reload
sudo systemctl enable image-converting-server
sudo systemctl start image-converting-server
sudo systemctl status image-converting-server
```

---

## 서버 상태 확인

### 헬스 체크

```bash
curl http://localhost:4000/health
```

**응답**:
```json
{
  "status": "ok"
}
```

### 서버 정보 확인

```bash
curl http://localhost:4000/
```

**응답**:
```json
{
  "message": "Image Converting Server"
}
```

---

## API 사용 예시

### 1. 기본 이미지 변환

R2에 저장된 이미지를 WebP로 변환:

```bash
curl -X POST http://localhost:4000/api/convert \
  -H "Content-Type: application/json" \
  -d '{"source": "r2://my-bucket/images/photo.jpg"}'
```

**응답**:
```json
{
  "success": true,
  "message": "Image converted successfully",
  "source": "r2://my-bucket/images/photo.jpg",
  "destination": "r2://my-bucket/images/photo.webp",
  "original_size": 1024000,
  "converted_size": 512000
}
```

### 2. 리사이징 포함 변환

```bash
curl -X POST "http://localhost:4000/api/convert?width=800&height=600" \
  -H "Content-Type: application/json" \
  -d '{"source": "r2://my-bucket/images/photo.jpg"}'
```

### 3. 프리셋 크기 사용

```bash
curl -X POST "http://localhost:4000/api/convert?preset=medium" \
  -H "Content-Type: application/json" \
  -d '{"source": "r2://my-bucket/images/photo.jpg"}'
```

### 4. 외부 URL 변환

```bash
curl -X POST http://localhost:4000/api/convert \
  -H "Content-Type: application/json" \
  -d '{"source": "https://example.com/image.png"}'
```

### 5. GET 방식 사용

```bash
curl "http://localhost:4000/api/convert?source=r2://my-bucket/images/photo.jpg&width=800&height=600"
```

더 많은 예시는 [API.md](./API.md) 참조.

---

## 크론 잡 동작 확인

### 크론 잡 상태 확인

크론 잡은 설정된 스케줄에 따라 자동으로 실행됩니다. 기본값은 매일 새벽 2시입니다.

### 수동 실행 (개발/테스트)

개발 중에는 크론 잡을 수동으로 트리거할 수 있습니다 (구현 시 추가 예정).

### 로그 확인

크론 잡 실행 로그는 서버 로그에 기록됩니다:

```bash
# 서버 로그 확인
tail -f server.log

# 또는 systemd 사용 시
sudo journalctl -u image-converting-server -f
```

### 크론 잡 비활성화

설정 파일에서 크론 잡을 비활성화할 수 있습니다:

```yaml
cron:
  schedule: "0 2 * * *"
  enabled: false  # 크론 잡 비활성화
```

자세한 내용은 [CRON.md](./CRON.md) 참조.

---

## 트러블슈팅

### 문제: 서버가 시작되지 않음

**증상**:
```
Error: failed to load config
```

**해결 방법**:
1. 설정 파일 경로 확인
2. 설정 파일 형식 확인 (YAML 문법)
3. 필수 필드 누락 확인

---

### 문제: R2 연결 실패

**증상**:
```
Error: failed to connect to R2
```

**해결 방법**:
1. R2 접속 정보 확인:
   - Access Key ID
   - Secret Access Key
   - Endpoint URL
   - Bucket 이름
2. 네트워크 연결 확인
3. R2 버킷이 존재하는지 확인
4. API 토큰 권한 확인

---

### 문제: 이미지 변환 실패

**증상**:
```json
{
  "success": false,
  "error": "conversion_failed",
  "message": "Failed to convert image"
}
```

**해결 방법**:
1. 이미지 포맷 확인 (지원 포맷: JPEG, PNG, GIF, BMP, TIFF)
2. 이미지 크기 확인 (max_size_mb 제한)
3. 이미지 파일이 손상되지 않았는지 확인
4. 서버 로그에서 상세 에러 메시지 확인

---

### 문제: 크론 잡이 실행되지 않음

**증상**: 크론 잡이 예정된 시간에 실행되지 않음

**해결 방법**:
1. 크론 잡이 활성화되어 있는지 확인 (`cron.enabled: true`)
2. Cron 표현식이 올바른지 확인
3. 서버가 실행 중인지 확인
4. 서버 로그에서 에러 메시지 확인
5. 시간대 설정 확인

---

### 문제: 포트가 이미 사용 중

**증상**:
```
Error: listen tcp :4000: bind: address already in use
```

**해결 방법**:
1. 다른 포트 사용:
   ```yaml
   server:
     port: 4001
   ```
2. 또는 기존 프로세스 종료:
   ```bash
   # 포트 사용 중인 프로세스 찾기
   lsof -i :4000
   # 프로세스 종료
   kill <PID>
   ```

---

### 문제: 메모리 부족

**증상**: 큰 이미지 처리 시 서버가 종료되거나 느려짐

**해결 방법**:
1. 최대 이미지 크기 제한 설정:
   ```yaml
   conversion:
     max_size_mb: 20  # 더 작은 값으로 설정
   ```
2. 서버 리소스 증가
3. 이미지 전처리 (업로드 전 리사이징)

---

### 로그 레벨 조정

현재 버전에서는 기본 로그 레벨을 사용합니다. 향후 버전에서 로그 레벨 설정이 추가될 예정입니다.

---

## 성능 최적화

### 1. 동시 처리 수 제한

큰 이미지를 여러 개 동시에 처리할 때 메모리 부족을 방지하기 위해 동시 처리 수를 제한할 수 있습니다 (향후 버전에서 추가 예정).

### 2. 캐싱

이미 변환된 이미지는 재변환하지 않도록 체크합니다 (ETag 또는 해시 기반).

### 3. 리소스 모니터링

서버 실행 중 리소스 사용량을 모니터링:
```bash
# CPU 및 메모리 사용량 확인
top -p $(pgrep image-converting-server)

# 또는 htop 사용
htop -p $(pgrep image-converting-server)
```

---

## 보안 권장사항

1. **민감 정보 보호**: R2 접속 정보는 환경 변수로 관리
2. **방화벽 설정**: 필요한 포트만 열기
3. **HTTPS 사용**: 프로덕션 환경에서는 리버스 프록시(Nginx, Caddy)를 통해 HTTPS 제공
4. **API 인증**: 필요시 API 키 기반 인증 추가 (향후 버전)

---

## 다음 단계

- [API.md](./API.md) - API 상세 명세 확인
- [CONFIG.md](./CONFIG.md) - 설정 파일 상세 가이드
- [CRON.md](./CRON.md) - 크론 잡 동작 방식 이해
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 시스템 아키텍처 이해

---

## 지원

문제가 지속되면 다음을 확인하세요:

1. 서버 로그 확인
2. 설정 파일 검증
3. R2 연결 상태 확인
4. 문서 재검토
