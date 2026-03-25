package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

func (m *mockInvocationContext) Context() context.Context {
	return context.Background()
}

func TestNewMyAgent(t *testing.T) {
	myAgent, card, err := NewMyAgent()
	if err != nil {
		t.Fatalf("NewMyAgent() error = %v", err)
	}

	if myAgent == nil {
		t.Error("NewMyAgent() returned nil agent")
	}

	if card.Name != "MyGoAgent" {
		t.Errorf("Expected agent name 'MyGoAgent', got '%s'", card.Name)
	}

	if card.Version != "1.1.0" {
		t.Errorf("Expected agent version '1.1.0', got '%s'", card.Version)
	}

	if !card.Capabilities.Streaming {
		t.Error("Expected streaming capability to be true")
	}

	if !card.Capabilities.StateTransitionHistory {
		t.Error("Expected stateTransitionHistory capability to be true")
	}

	if card.ProtocolVersion != "0.3.0" {
		t.Errorf("Expected protocol version '0.3.0', got '%s'", card.ProtocolVersion)
	}

	if card.PreferredTransport != "JSONRPC" {
		t.Errorf("Expected preferred transport 'JSONRPC', got '%s'", card.PreferredTransport)
	}
}

func TestAgentSearch(t *testing.T) {
	myAgent, _, _ := NewMyAgent()
	
	// Test simulated search
	ctx := &mockInvocationContext{
		userContent: genai.NewContentFromText("search", genai.RoleUser),
	}
	
	found := false
	for event, err := range myAgent.Run(ctx) {
		if err != nil {
			t.Fatalf("Run error: %v", err)
		}
		if event.LLMResponse.Content != nil && len(event.LLMResponse.Content.Parts) > 0 {
			text := event.LLMResponse.Content.Parts[0].Text
			if text != "" && (len(text) >= 6 && text[:6] == "Search") {
				found = true
			}
		}
	}
	if !found {
		t.Error("Simulated search response not found for input 'search'")
	}
}

type mockInvocationContext struct {
	agent.InvocationContext
	userContent *genai.Content
}

func (m *mockInvocationContext) UserContent() *genai.Content {
	return m.userContent
}

func (m *mockInvocationContext) Value(key any) any {
	return m.Context().Value(key)
}

func (m *mockInvocationContext) Deadline() (deadline time.Time, ok bool) {
	return m.Context().Deadline()
}

func (m *mockInvocationContext) Done() <-chan struct{} {
	return m.Context().Done()
}

func (m *mockInvocationContext) Err() error {
	return m.Context().Err()
}

func (m *mockInvocationContext) Session() session.Session {
	return &mockSession{}
}

type mockSession struct {
	session.Session
}

func (s *mockSession) ID() string {
	return "test-session-id"
}

func (m *mockInvocationContext) InvocationID() string {
	return "test-id"
}

func (m *mockInvocationContext) Branch() string {
	return "main"
}

func (m *mockInvocationContext) Artifacts() agent.Artifacts { return nil }
func (m *mockInvocationContext) Memory() agent.Memory       { return nil }
func (m *mockInvocationContext) RunConfig() *agent.RunConfig { return nil }
func (m *mockInvocationContext) EndInvocation()             {}
func (m *mockInvocationContext) Ended() bool                { return false }
func (m *mockInvocationContext) WithContext(ctx context.Context) agent.InvocationContext { return m }
func (m *mockInvocationContext) Agent() agent.Agent {
	a, _, _ := NewMyAgent()
	return a
}

func TestWellKnownEndpoint(t *testing.T) {
	_, card, _ := NewMyAgent()

	req, err := http.NewRequest("GET", "/.well-known/agent-card.json", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	
	// Create a handler manually since main() is not easily testable without refactoring
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.MarshalIndent(card, "", "  ")
		w.Write(data)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expectedType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedType {
		t.Errorf("handler returned unexpected content type: got %v want %v", contentType, expectedType)
	}

	var responseCard AgentCard
	if err := json.NewDecoder(rr.Body).Decode(&responseCard); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if responseCard.Name != card.Name {
		t.Errorf("Response card name mismatch: got %v want %v", responseCard.Name, card.Name)
	}
}

func TestRootEndpoint(t *testing.T) {
	_, card, _ := NewMyAgent()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("ADK Agent " + card.Name + " is running."))
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !testing.Short() {
		// Basic content check
		expected := "ADK Agent MyGoAgent is running."
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
		}
	}
}

func TestRPCEndpoint(t *testing.T) {
	// In the real app, we registered the handler at the root with SSE support if Accept: text/event-stream is present
	mux := http.NewServeMux()
	
	// Mock implementation similar to the one in main.go
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") == "text/event-stream" || r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
				return
			}
			
			fmt.Fprintf(w, ": ok\n\n")
			flusher.Flush()
			
			fmt.Fprintf(w, "data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"kind\":\"task\",\"status\":{\"state\":\"submitted\"}}}\n\n")
			flusher.Flush()
			return
		}
		w.Write([]byte("Regular response"))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Test regular GET request
	respRegular, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if respRegular.Header.Get("Content-Type") == "text/event-stream" {
		t.Error("Regular GET request should not return text/event-stream")
	}
	respRegular.Body.Close()

	// 2. Test GET request with Accept: text/event-stream
	reqSSE, _ := http.NewRequest("GET", ts.URL+"/", nil)
	reqSSE.Header.Set("Accept", "text/event-stream")
	respSSE, err := http.DefaultClient.Do(reqSSE)
	if err != nil {
		t.Fatal(err)
	}
	defer respSSE.Body.Close()

	if respSSE.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream for SSE request, got %s", respSSE.Header.Get("Content-Type"))
	}

	// 3. Test POST request (should also return SSE)
	respPost, err := http.Post(ts.URL+"/", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer respPost.Body.Close()

	if respPost.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream for POST request, got %s", respPost.Header.Get("Content-Type"))
	}
}
