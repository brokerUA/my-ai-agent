# Design Spec: Learning Professor Agent

## Purpose
Learning Professor is an educational AI agent that demonstrates complex interaction patterns using the Kagent ADK. It focuses on generating high-quality academic content and collaborating with other agents (Student Agent) for peer-review and critique.

## Core Capabilities
- **Academic Lecture Generation**: Generates exactly one technically precise academic sentence about a given topic using Google Gemini LLM.
- **A2A (Agent-to-Agent) Interaction**: Automatically sends the generated content to a student agent for critique.
- **Agent Discovery**: Provides metadata about its version, author, and capabilities via `/.well-known/agent-card.json`.
- **REST API**: Exposes ADK-compliant endpoints for skill execution.

## Technical Requirements
- **Runtime**: Go 1.25+
- **Framework**: github.com/kagent-dev/kagent/go/adk
- **LLM**: Google Gemini (via `google.golang.org/genai`)
- **Deployment**: Dockerized (Alpine-based)
- **Environment**: Requires access to a Kagent Controller for A2A communication.

## Success Criteria
- [x] Correctly implements the `GenerateLecture` skill.
- [x] Successfully integrates with Google Gemini for text generation.
- [x] Demonstrates functional A2A communication with the student agent.
- [x] Exposes compliant `agent-card.json` metadata.
- [x] Includes evaluation set in `evals/config.yaml`.
- [x] Passes unit tests in `src/main_test.go`.
- [x] Successfully builds via Docker and pushes to GHCR.

## Constraints
- Must respond in English only for academic precision.
- Must follow the single-sentence constraint for lecture generation.
- Must use secure practices (non-root user in Docker, environment-based secrets).
