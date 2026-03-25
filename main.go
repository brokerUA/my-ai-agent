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
	"strings"
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
			Jsonrpc string      `json:"jsonrpc"`
			Method  string      `json:"method"`
			Params  interface{} `json:"params"`
			Id      interface{} `json:"id"`
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

		if req.Method != "" && req.Method != "OnSendMessageStream" {
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

		// Simulate partial response generation
		chunks := []string{"Hello", ", I ", "am ", "your ", "Go", "-agent", ". ", "I ", "support ", "streaming!"}
		fullResult := strings.Join(chunks, "")

		for _, chunk := range chunks {
			chunkData := map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "OnChunk",
				"params": map[string]interface{}{
					"chunk": chunk,
				},
			}
			data, _ := json.Marshal(chunkData)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				slog.Error("Failed to write chunk", "error", err)
				return
			}
			flusher.Flush()
			time.Sleep(200 * time.Millisecond)
		}

		// Send the final result
		resultData := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "OnResult",
			"params": map[string]interface{}{
				"result": fullResult,
			},
		}
		data, _ := json.Marshal(resultData)
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			slog.Error("Failed to write result", "error", err)
			return
		}
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

		fmt.Fprintf(w, "ADK Agent %s is running. Agent card is available at /.well-known/agent-card.json", card.Name)
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
