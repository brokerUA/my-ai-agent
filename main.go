package main

import (
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"os"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/server/adkrest"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// AgentCard represents the structure of the agent card according to requirements.
type AgentCard struct {
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Version            string            `json:"version"`
	Capabilities       Capabilities      `json:"capabilities"`
	Author             string            `json:"author"`
	Auth               map[string]string `json:"auth"`
	Endpoints          []string          `json:"endpoints"`
	APISpec            string            `json:"api_spec"`
	PreferredTransport string            `json:"preferredTransport"`
	ProtocolVersion    string            `json:"protocolVersion"`
}

// Capabilities represents the agent's capabilities.
type Capabilities struct {
	Streaming              bool `json:"streaming"`
	StateTransitionHistory bool `json:"stateTransitionHistory"`
}

// NewMyAgent creates an instance of the agent with a card.
func NewMyAgent() (agent.Agent, AgentCard, error) {
	card := AgentCard{
		Name:        "MyGoAgent",
		Description: "This is an agent implemented in Go using ADK with search and A2A capabilities.",
		Version:     "1.1.0",
		Capabilities: Capabilities{
			Streaming:              true,
			StateTransitionHistory: true,
		},
		Author: "Dmytro Andrieiev",
		Auth: map[string]string{
			"type": "none",
		},
		Endpoints: []string{
			"/api/v1/sessions",
			"/.well-known/agent-card.json",
			"/",
		},
		APISpec:            "https://example.com/api-spec.yaml",
		PreferredTransport: "JSONRPC",
		ProtocolVersion:    "0.3.0",
	}

	// Mock search tool simulation
	searchTool := func(query string) string {
		slog.Info("Simulating search", "query", query)
		return fmt.Sprintf("Search results for '%s': [Result 1: ADK Go is awesome], [Result 2: How to build agents in Go]", query)
	}

	// Use a wrapper function for Run logic.
	runFunc := func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
		return func(yield func(*session.Event, error) bool) {
			userInput := ""
			if ctx.UserContent() != nil && len(ctx.UserContent().Parts) > 0 {
				userInput = ctx.UserContent().Parts[0].Text
			}

			var responseText string
			if userInput != "" && (userInput == "search" || userInput == "find") {
				// Simple search simulation trigger
				responseText = searchTool("ADK Go")
			} else if userInput != "" && (userInput == "a2a" || userInput == "delegate") {
				// A2A simulation trigger
				responseText = "I can delegate tasks to other agents. For example, I can call a WeatherAgent to get current conditions."
			} else {
				responseText = "Hello! I am your ADK agent. I can perform searches and support A2A communication."
			}

			event := &session.Event{
				LLMResponse: model.LLMResponse{
					Content: genai.NewContentFromText(responseText, genai.RoleModel),
				},
			}
			yield(event, nil)
		}
	}

	a, err := agent.New(agent.Config{
		Name:        card.Name,
		Description: card.Description,
		Run:         runFunc,
	})

	return a, card, err
}

func main() {
	// Configure structured logging.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Create the agent and its card
	myAgent, card, err := NewMyAgent()
	if err != nil {
		slog.Error("Failed to create agent", "error", err)
		os.Exit(1)
	}

	// 2. Configure ADK REST server
	restServer, err := adkrest.NewServer(adkrest.ServerConfig{
		AgentLoader:     agent.NewSingleLoader(myAgent),
		SessionService:  session.InMemoryService(),
		SSEWriteTimeout: 30 * time.Second,
	})
	if err != nil {
		slog.Error("Failed to create ADK REST server", "error", err)
		os.Exit(1)
	}

	// 3. Create the main HTTP router
	mux := http.NewServeMux()

	handleRPC := func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Received RPC request", "method", r.Method, "url", r.URL.String())

		var req struct {
			Jsonrpc string `json:"jsonrpc"`
			Method  string `json:"method"`
			Params  struct {
				Message struct {
					Role      string `json:"role"`
					Parts     []struct {
						Kind string `json:"kind"`
						Text string `json:"text,omitempty"`
						File *struct {
							MimeType string `json:"mimeType"`
							Data     string `json:"data"`
						} `json:"file,omitempty"`
					} `json:"parts"`
					MessageId string `json:"messageId"`
				} `json:"message"`
				Metadata map[string]interface{} `json:"metadata"`
			} `json:"params"`
			Id interface{} `json:"id"`
		}

		// Try to decode JSON-RPC request from body if it's a POST
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				slog.Error("Failed to decode JSON-RPC request", "error", err)
				http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
				return
			}
		}
		slog.Info("RPC method", "method", req.Method, "id", req.Id)

		if req.Method != "" && req.Method != "message/stream" && req.Method != "OnSendMessageStream" {
			slog.Warn("Method not found", "method", req.Method)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"error": map[string]interface{}{
					"code":    -32601,
					"message": "Method not found",
				},
				"id": req.Id,
			})
			return
		}

		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			slog.Error("Streaming unsupported by response writer")
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Explicitly send the first byte to establish the stream and Content-Type
		fmt.Fprintf(w, ": ok\n\n")
		flusher.Flush()

		// Generate IDs for the task/context
		taskId := "225d6247-06ba-4cda-a08b-33ae35c8dcfa"
		contextId := "05217e44-7e9f-473e-ab4f-2c2dde50a2b1"
		artifactId := "9b6934dd-37e3-4eb1-8766-962efaab63a1"
		timestamp := time.Now().Format("2006-01-02T15:04:05.000000")

		// 1. Initial task status update: submitted
		initialEvent := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.Id,
			"result": map[string]interface{}{
				"id":        taskId,
				"contextId": contextId,
				"status": map[string]interface{}{
					"state":     "submitted",
					"timestamp": timestamp,
				},
				"history": []map[string]interface{}{
					{
						"role":      req.Params.Message.Role,
						"parts":     req.Params.Message.Parts,
						"messageId": req.Params.Message.MessageId,
						"taskId":    taskId,
						"contextId": contextId,
					},
				},
				"kind":     "task",
				"metadata": map[string]interface{}{},
			},
		}
		data, _ := json.Marshal(initialEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		time.Sleep(500 * time.Millisecond)

		// 2. Artifact updates
		sections := []string{"<section 1...>", "<section 2...>", "<section 3...>"}
		for i, section := range sections {
			appendMode := i > 0
			lastChunk := i == len(sections)-1

			updateEvent := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.Id,
				"result": map[string]interface{}{
					"taskId":    taskId,
					"contextId": contextId,
					"artifact": map[string]interface{}{
						"artifactId": artifactId,
						"parts": []map[string]interface{}{
							{"type": "text", "text": section},
						},
					},
					"append":    appendMode,
					"lastChunk": lastChunk,
					"kind":      "artifact-update",
				},
			}
			data, _ := json.Marshal(updateEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			time.Sleep(500 * time.Millisecond)
		}

		// 3. Final status update: completed
		finalTimestamp := time.Now().Format("2006-01-02T15:04:05.000000")
		finalEvent := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req.Id,
			"result": map[string]interface{}{
				"taskId":    taskId,
				"contextId": contextId,
				"status": map[string]interface{}{
					"state":     "completed",
					"timestamp": finalTimestamp,
				},
				"final": true,
				"kind":  "status-update",
			},
		}
		data, _ = json.Marshal(finalEvent)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		slog.Info("Finished SSE stream")
	}

	// 4. Add Well-Known URI for Agent Card
	mux.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		data, err := json.MarshalIndent(card, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
	})

	// 5. Connect ADK API
	mux.Handle("/api/", http.StripPrefix("/api", restServer))

	// 6. Root endpoint handling SSE and regular requests
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Check if it's an SSE request (A2A proxy often uses Accept: text/event-stream)
		if r.Header.Get("Accept") == "text/event-stream" || r.Method == http.MethodPost {
			handleRPC(w, r)
			return
		}

		fmt.Fprintf(w, "ADK Agent %s is running. Agent card is available at /.well-known/agent-card.json. Enjoy!", card.Name)
	})

	port := ":8080"
	slog.Info("Starting server", "port", port)
	slog.Info("Agent Card URL", "url", fmt.Sprintf("http://localhost%s/.well-known/agent-card.json", port))

	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
