# Image Converting Server

A high-performance image processing service built with Go, designed for seamless integration with **Cloudflare R2** (S3-compatible storage). This server automates image conversion to **WebP** and provides on-the-fly **resizing** via HTTP API or scheduled cron jobs.

## Features

- **Automated WebP Conversion**: Automatically converts various image formats (JPEG, PNG, GIF, BMP, TIFF) to WebP to reduce storage and bandwidth costs.
- **Cloudflare R2 Integration**: Specifically optimized for Cloudflare R2, providing a cost-effective alternative to AWS S3.
- **On-the-fly Resizing**: Resize images dynamically using HTTP query parameters or predefined presets (e.g., thumbnail, medium, large).
- **Scheduled Cron Jobs**: Periodically scans your R2 bucket for new images and converts them automatically to keep your storage optimized.
- **State Management**: Tracks processing status to ensure efficient incremental updates without reprocessing existing images.
- **Webhooks**: Notifies your endpoint when a cron batch completes, with optional retry for failed deliveries.

## Getting Started

### Prerequisites

- Go 1.21 or higher

### Installation

```bash
go mod download
```

### Running the Server

```bash
go run main.go
```

The server will start on `http://localhost:4000`

### Running with Docker (Recommended)

1.  **Environment Setup**:
    Copy `.env.example` to `.env` and fill in your R2 credentials.
    ```bash
    cp .env.example .env
    ```

2.  **Run with Docker Compose**:
    ```bash
    docker-compose up -d
    ```
    The server will start on `http://localhost:4000` (default).

3.  **Port Customization**:
    To use a different port, set `SERVER_PORT` in your `.env` file or environment:
    ```bash
    SERVER_PORT=9000 docker-compose up -d
    ```

### Configuration

Configuration is done via **environment variables** (and optionally a `.env` file). Copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
```

All settings (R2, conversion, resize presets, cron, server) can be set in `.env`. If `config/config.yaml` exists, it is loaded first and env vars override it; if the file is missing, the server runs with env (and defaults) only.

- **R2**: set `R2_BUCKET` for one bucket, or `R2_BUCKETS=bucket-a,bucket-b` to process multiple buckets in the same R2 account.
- **Conversion**: `CONVERSION_FORMATS`, `CONVERSION_QUALITY`, `CONVERSION_MAX_SIZE_MB`
- **Resize presets**: optional `RESIZE_PRESETS=thumbnail:150x150,medium:800x800` (used as `?preset=thumbnail` in API)
- **Cron**: `CRON_SCHEDULE`, `CRON_ENABLED` (set `CRON_ENABLED=true` to enable the scheduled job)

Cron uses standard cron expression **(minute hour day month weekday)** in **server local time**. Default `"0 12 * * *"` runs at **UTC 12:00** (noon UTC). On a UTC server that equals **21:00 KST**. Adjust the hour if your server uses a different timezone. See [docs/CRON.md](docs/CRON.md) for details.

### Webhook (batch completion notification)

When a **cron** batch run finishes, the server can send a **POST** request to a URL you configure. This is useful for logging, monitoring, or triggering downstream workflows (e.g. cache invalidation, CDN purge).

#### When it is sent

- **Only after the scheduled cron job** completes (not on one-off API conversions).
- Sent only if `WEBHOOK_URL` is set. If the URL is empty, no request is made.

#### Request details

| Item | Value |
|------|--------|
| **Method** | `POST` |
| **Content-Type** | `application/json` |
| **Timeout** | 10 seconds |

#### Request body (JSON)

| Field | Type | Description |
|-------|------|-------------|
| `event` | string | Always `"batch.completed"` for cron batch completion. |
| `processed_count` | number | Number of images successfully converted in this run. |
| `failed_count` | number | Number of images that failed (download, convert, or upload). |
| `images` | array | List of converted images in this run. |
| `images[].bucket` | string | R2 bucket name. Present for cron conversions. |
| `images[].source` | string | Original object key (e.g. `path/to/image.jpg`). |
| `images[].destination` | string | New object key after conversion (e.g. `path/to/image.webp`). |

**Example body:**

```json
{
  "event": "batch.completed",
  "processed_count": 3,
  "failed_count": 0,
  "images": [
    { "bucket": "images-a", "source": "uploads/photo.jpg", "destination": "uploads/photo.webp" },
    { "bucket": "images-b", "source": "assets/logo.png", "destination": "assets/logo.webp" },
    { "bucket": "images-c", "source": "gallery/1.gif", "destination": "gallery/1.webp" }
  ]
}
```

If no images were converted in the run, `images` is an empty array and counts may be zero.

#### Configuration

Set the webhook URL (and optional retry settings) via environment variables in `.env`:

| Variable | Description | Default |
|----------|-------------|---------|
| `WEBHOOK_URL` | Full URL to receive the POST (e.g. `https://your-server.com/webhook/conversion-batch`). | (none) |
| `WEBHOOK_RETRY_ENABLED` | If `true`, failed deliveries are stored and retried. | `false` |
| `WEBHOOK_RETRY_INTERVAL_MINUTES` | Minutes between retry passes. | `5` |
| `WEBHOOK_MAX_RETRIES` | Max retries per failed payload; then the pending file is removed (or moved to dead letter if configured). | `5` |
| `WEBHOOK_PENDING_DIR` | Directory for storing failed webhook payloads for retry. | `data/webhook_pending` |

**Receiver expectations:** Your endpoint should respond with an HTTP status in the **2xx** range. Any other status (or timeout/connection error) is treated as failure. If retry is enabled, the payload is written under `WEBHOOK_PENDING_DIR` and retried periodically until success or `WEBHOOK_MAX_RETRIES` is reached.

#### Manual webhook trigger (POST /api/webhook/send)

Admins can trigger the same webhook on demand (e.g. for testing or to notify about specific images) without waiting for a cron batch.

- **Method:** `POST`
- **URL:** `/api/webhook/send`
- **Request body (JSON):**

  | Field   | Type  | Description                                        |
  |---------|-------|----------------------------------------------------|
  | `images` | array | List of image entries (source and destination keys). Must not be empty. |
  | `images[].source`      | string | Original object key (e.g. `uploads/a.jpg`).       |
  | `images[].destination`| string | Converted object key (e.g. `uploads/a.webp`).     |

  Example:
  ```json
  { "images": [ { "source": "uploads/a.jpg", "destination": "uploads/a.webp" } ] }
  ```

- **Response:** Same webhook payload shape is POSTed to `WEBHOOK_URL` with `event: "manual.triggered"` (so receivers can distinguish from `batch.completed`).
- **Status codes:** `200` success; `400` invalid/empty body; `502` webhook delivery failed; `503` webhook URL not configured.

### Endpoints

- `GET /` - Main endpoint
- `GET /health` - Health check endpoint
- `POST /api/webhook/send` - Trigger webhook manually (see Webhook section above)

### Documentation

- [API](docs/API.md) · [Config](docs/CONFIG.md) · [Cron](docs/CRON.md) · [Usage](docs/USAGE.md)

## License

MIT License - see [LICENSE](LICENSE) file for details
