# Image Converting Server

A high-performance image processing service built with Go, designed for seamless integration with **Cloudflare R2** (S3-compatible storage). This server automates image conversion to **WebP** and provides on-the-fly **resizing** via HTTP API or scheduled cron jobs.

## Features

- **Automated WebP Conversion**: Automatically converts various image formats (JPEG, PNG, GIF, BMP, TIFF) to WebP to reduce storage and bandwidth costs.
- **Cloudflare R2 Integration**: Specifically optimized for Cloudflare R2, providing a cost-effective alternative to AWS S3.
- **On-the-fly Resizing**: Resize images dynamically using HTTP query parameters or predefined presets (e.g., thumbnail, medium, large).
- **Scheduled Cron Jobs**: Periodically scans your R2 bucket for new images and converts them automatically to keep your storage optimized.
- **State Management**: Tracks processing status to ensure efficient incremental updates without reprocessing existing images.

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

### Endpoints

- `GET /` - Main endpoint
- `GET /health` - Health check endpoint

## License

MIT License - see [LICENSE](LICENSE) file for details