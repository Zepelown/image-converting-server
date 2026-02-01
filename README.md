# image-converting-server

Image converting server built with Go.

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

The server will start on `http://localhost:8080`

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
    The server will start on `http://localhost:8080` (default).

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