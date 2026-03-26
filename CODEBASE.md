# Project Structure & Implementation Details

## File Inventory

- **`agent-card.json`**: Agent metadata defining name, version, and capabilities. Used for agent discovery.
- **`src/main.go`**: The entry point of the ADK agent.
  - Implements the `Professor` structure with LLM and A2A clients.
  - Configures the ADK agent using `github.com/kagent-dev/kagent/go/adk`.
  - Implements `GenerateLecture` skill using Google Gemini LLM.
  - Communicates with a student agent via A2A.
- **`src/main_test.go`**: Unit tests for the agent logic and skill handlers.
- **`DESIGN_SPEC.md`**: Technical specification and requirements document for the Learning Professor agent.
- **`Dockerfile`**: Defines the multi-stage build process for creating a lightweight container image based on Alpine.
- **`mise.toml`**: Project configuration for `mise`. Includes Go version and automation tasks for building and running.
- **`evals/config.yaml`**: Evaluation dataset for automated quality checks using `adk eval`.
- **`src/go.mod` / `src/go.sum`**: Go dependency management.

## Technical Architecture

The agent is built on the [Kagent ADK](https://github.com/kagent-dev/kagent/tree/main/go/adk). It follows the ADK pattern for implementing tools (skills) and exposing them.

Key components:
1. **ADK App**: Managed by `adk.NewApp`.
2. **Gemini Integration**: Uses `google.golang.org/genai` to generate technically precise sentences.
3. **A2A Client**: Uses `adk.NewA2AClient()` to delegate tasks (critique) to a student agent.
4. **Environment Configuration**: Requires several environment variables for connection and authentication.

## CI/CD Pipeline

The pipeline is configured to:
- Build the image from `Dockerfile`.
- Push the image to the repository's container registry (GHCR).
- Tag it with `latest` and the git commit SHA.
