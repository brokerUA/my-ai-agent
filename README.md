# My ADK Go Agent

A custom AI agent implemented using the [Google ADK (Agent Development Kit)](https://github.com/google/adk-go) in Go.

## Features

- **ADK Integration**: Built with the official Go SDK.
- **Agent Card**: Serves metadata via `/.well-known/ai-agent.json`.
- **Observability**: Structured logging using `log/slog` for better production monitoring.
- **Dockerized**: Ready for containerized deployment with a non-root user for improved security.
- **CI/CD**: Automated image builds and pushes to GHCR.
- **Mise support**: Task management and environment setup using `mise`.

## Getting Started

### Prerequisites

- [Go 1.25+](https://golang.org/dl/)
- [mise](https://mise.jdx.st/) (optional, but recommended)
- [Docker](https://www.docker.com/) (for containerization)

### Development

Using `mise`:

```bash
# Build the agent
mise run build

# Run the agent
mise run run
```

Without `mise`:

```bash
# Build
go build -o agent-app main.go

# Run
./agent-app
```

The agent will be available at `http://localhost:8080`.

### Documentation

- [DESIGN_SPEC.md](DESIGN_SPEC.md): Full technical specification and requirements.
- [CODEBASE.md](CODEBASE.md): Detailed project structure overview.

### Endpoints

- `GET /`: Health check and basic info.
- `GET /.well-known/ai-agent.json`: Agent Card (metadata).
- `POST /api/...`: ADK REST API endpoints.

## CI/CD

The project includes a GitHub Actions workflow that:
1. Builds a Docker image on every push to the `main` branch.
2. Pushes the image to GitHub Container Registry (GHCR).

## License

MIT
