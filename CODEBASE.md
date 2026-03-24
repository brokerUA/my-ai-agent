# Project Structure & Implementation Details

## File Inventory

- **`evals/config.yaml`**: Evaluation dataset for automated quality checks using `adk eval`.
- **`main.go`**: The entry point of the ADK agent.
  - Implements the `AgentCard` structure with Auth, Endpoints, and API Spec.
  - Configures the ADK agent using `google.golang.org/adk/agent`.
  - Implements simulated Search and A2A capabilities.
  - Sets up an HTTP server with endpoints for `/.well-known/ai-agent.json` and the standard ADK REST API.
- **`main_test.go`**: Unit tests for the agent and its endpoints.
- **`DESIGN_SPEC.md`**: Technical specification and requirements document as per ADK development guide.
- **`Dockerfile`**: Defines the multi-stage build process for creating a lightweight container image based on Alpine, with a non-root user for security.
- **`mise.toml`**: Project configuration for `mise`. Includes Go version and automation tasks for building and running.
- **`.github/workflows/deploy.yml`**: GitHub Actions workflow for CI/CD. Builds and pushes the Docker image to GHCR.
- **`go.mod` / `go.sum`**: Go dependency management.

## Technical Architecture

The agent is built on the [Google ADK](https://github.com/google/adk-go). It follows the ADK pattern for implementing agents and exposing them via REST API.

Key components:
1. **Agent Metadata**: Managed by `AgentCard`.
2. **REST API**: Handled by `adkrest.NewServer`.
3. **Well-known URI**: Custom handler at `/.well-known/ai-agent.json` to provide agent discovery.
4. **Observability**: Structured logging using `log/slog`.

## CI/CD Pipeline

The pipeline uses `docker/build-push-action@v5` to:
- Authenticate with GHCR.
- Build the image from `Dockerfile`.
- Tag it with `latest` and the full git commit SHA.
- Push the image to the repository's container registry.
