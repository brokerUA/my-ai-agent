# Design Spec: MyGoAgent

## Purpose
MyGoAgent is a reference implementation of an AI agent using the Google ADK for Go. It demonstrates core agent capabilities, discovery via Agent Card, and REST API integration.

## Core Capabilities
- **Greeting**: Responds with a friendly greeting and introduces itself.
- **Search**: Simulated tool to demonstrate information retrieval.
- **A2A (Agent-to-Agent)**: Supports delegating tasks to other specialized agents.
- **Discovery**: Provides metadata about its version, author, and capabilities via a standard well-known URI.
- **REST API**: Exposes standard ADK endpoints for session-based interactions.

## Technical Requirements
- **Runtime**: Go 1.25+
- **Framework**: google.golang.org/adk (ADK-Go)
- **Deployment**: Dockerized (Alpine-based)
- **Observability**: Structured logging and OpenTelemetry tracing (planned).

## Success Criteria
- [x] Correctly implements `agent.Agent` interface.
- [x] Exposes `/.well-known/ai-agent.json` with expanded metadata (Auth, Endpoints, API Spec).
- [x] Simulated search tool returns relevant mock data.
- [x] Supports A2A communication patterns.
- [x] Includes evaluation set in `evals/config.yaml`.
- [x] Passes unit tests for core endpoints.
- [x] Successfully builds and pushes Docker image via CI/CD.

## Constraints
- Must use standard Go library where possible.
- Must follow ADK-Go patterns for session management.
- Deployment must be secure (non-root user in Docker).
