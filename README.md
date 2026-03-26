# Learning Professor Agent

An educational AI agent implemented using the [Kagent ADK](https://github.com/kagent-dev/kagent/tree/main/go/adk) in Go. It generates technically precise academic sentences and collaborates with a student agent.

## Features

- **ADK Integration**: Built with the latest Kagent Go SDK.
- **Gemini Powered**: Uses Google Gemini API for high-quality content generation.
- **A2A Collaboration**: Automatically interacts with a student agent to critique the generated content.
- **Dockerized**: Ready for containerized deployment.
- **CI/CD**: Automated builds and pushes to GHCR.

## Getting Started

### Prerequisites

- [Go 1.25+](https://golang.org/dl/)
- [mise](https://mise.jdx.st/) (optional, but recommended)
- [Docker](https://www.docker.com/) (for containerization)

### Environment Configuration

The following environment variables are required:

- `KAGENT_CONTROLLER_URL`: URL of the Kagent controller.
- `STUDENT_AGENT_NAME`: Name of the student agent to collaborate with.
- `STUDENT_AGENT_NAMESPACE`: Namespace of the student agent.
- `CRITIQUE_SKILL_ID`: ID of the skill in the student agent to call.
- `LLM_NAME`: Name of the Gemini model (e.g., `gemini-2.0-flash`).
- `GOOGLE_API_KEY`: Your Google Cloud API key for Gemini.

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
cd src && go build -o ../learning-professor main.go

# Run
./learning-professor
```

The agent will be available at `http://localhost:8080`.

### Documentation

- [DESIGN_SPEC.md](DESIGN_SPEC.md): Full technical specification and requirements.
- [CODEBASE.md](CODEBASE.md): Detailed project structure overview.

### Endpoints

- `GET /.well-known/agent-card.json`: Agent Card (metadata).
- `POST /api/skill/generate-lecture-skill`: Main skill for generating academic content.

## License

MIT
