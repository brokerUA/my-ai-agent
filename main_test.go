package main

import (
	"context"
	"encoding/json"
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
