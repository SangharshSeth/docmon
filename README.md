# DocMon - Docker Container Monitor

DocMon is a standalone application that provides a web-based interface for monitoring and managing Docker containers and images. It includes both a backend API server and an embedded React frontend, all packaged into a single binary.

## Features

- Real-time Docker container monitoring
- Docker image management
- Built-in caching with Redis
- Modern React-based UI
- Single binary distribution
- Gzip compression for optimal performance

## Prerequisites

- Docker installed and running
- Redis server (optional, for caching)
- Go 1.21 or higher (for building from source)

## Quick Start

1. Download the latest binary from the releases page
2. Create a `.env` file with your Redis configuration (optional):
   ```
   REDIS=redis://localhost:6379
   ```
3. Run the binary:
   ```bash
   ./docmon
   ```
4. Open your browser and navigate to `http://localhost:8080`

## Building from Source

```bash
git clone https://github.com/sangharshseth/docmon.git
cd docmon
go build -o docmon ./cmd/main.go
```

## Architecture

- Backend: Go with Gin framework
- Frontend: React (embedded in the binary)
- Caching: Redis
- Static File Serving: Built-in middleware

## Configuration

The application can be configured using environment variables:

- `REDIS`: Redis connection URL (optional)
- Default port: 8080 (Web UI and API)

## Development

For development, the application uses:
- Gin for the REST API
- CORS enabled for development
- Gzip compression
- Structured logging (slog)

## Security Notes

- The application runs without proxy trust settings as it's designed to run locally
- CORS is configured for local development
- Release mode is enabled by default for production use

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## License

MIT © [Sangharsh Seth](https://github.com/sangharshseth)

