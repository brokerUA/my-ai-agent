package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/kagent-dev/kagent/go/adk"
	"google.golang.org/genai"
)

// MockLLMClient implements LLMClient interface for testing.
type MockLLMClient struct {
	Response *genai.GenerateContentResponse
	Err      error
}

func (m *MockLLMClient) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return m.Response, m.Err
}

// MockA2AClient implements A2AClient interface for testing.
type MockA2AClient struct {
	Response a2a.SendMessageResult
	Err      error
}

func (m *MockA2AClient) SendMessage(ctx context.Context, req *a2a.SendMessageRequest) (a2a.SendMessageResult, error) {
	return m.Response, m.Err
}

func TestGenerateLecture_Success(t *testing.T) {
	mockLLM := &MockLLMClient{
		Response: &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "This is a technical sentence."},
						},
					},
				},
			},
		},
	}
	mockA2A := &MockA2AClient{
		Response: a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewDataPart(map[string]interface{}{"output": "Student feedback"})),
	}

	p := &Professor{
		llmClient:     mockLLM,
		a2aClient:     mockA2A,
		llmName:       "gemini-pro",
		studentUrl:    "http://student",
		critiqueSkill: "critique-content-skill",
	}

	args := map[string]interface{}{
		"request": "Kubernetes",
	}

	resp, err := p.GenerateLecture(context.Background(), args)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp != "Student feedback" {
		t.Errorf("Expected 'Student feedback', got %v", resp)
	}
}

func TestGenerateLecture_MissingRequest(t *testing.T) {
	p := &Professor{}
	args := map[string]interface{}{}

	_, err := p.GenerateLecture(context.Background(), args)
	if err == nil {
		t.Fatal("Expected error for missing request, got nil")
	}
	if err.Error() != "missing 'request' argument" {
		t.Errorf("Expected error 'missing request argument', got %v", err)
	}
}

func TestGenerateLecture_LLMError(t *testing.T) {
	mockLLM := &MockLLMClient{
		Err: fmt.Errorf("LLM failed"),
	}
	p := &Professor{
		llmClient: mockLLM,
	}
	args := map[string]interface{}{"request": "topic"}

	_, err := p.GenerateLecture(context.Background(), args)
	if err == nil {
		t.Fatal("Expected error from LLM, got nil")
	}
}

func TestGenerateLecture_A2AError(t *testing.T) {
	mockLLM := &MockLLMClient{
		Response: &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Sentence"},
						},
					},
				},
			},
		},
	}
	mockA2A := &MockA2AClient{
		Err: fmt.Errorf("A2A failed"),
	}
	p := &Professor{
		llmClient:  mockLLM,
		a2aClient:  mockA2A,
		studentUrl: "http://student",
	}
	args := map[string]interface{}{"request": "topic"}

	_, err := p.GenerateLecture(context.Background(), args)
	if err == nil {
		t.Fatal("Expected error from A2A, got nil")
	}
}

func TestHandleMessage_Success(t *testing.T) {
	mockLLM := &MockLLMClient{
		Response: &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "OTel is an observability framework."},
						},
					},
				},
			},
		},
	}
	mockA2A := &MockA2AClient{
		Response: a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewDataPart(map[string]interface{}{"output": "Feedback"})),
	}

	p := &Professor{
		llmClient:     mockLLM,
		a2aClient:     mockA2A,
		studentUrl:    "http://student",
		critiqueSkill: "skill",
	}

	rawParams := json.RawMessage(`{"message":{"parts":[{"kind":"text","text":"What is OTel?"}]}}`)
	req := &adk.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "test-uuid",
		Method:  "message/send",
		Params:  rawParams,
	}

	resp, err := p.HandleMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	if resp.ID != "test-uuid" {
		t.Errorf("Expected ID 'test-uuid', got %v", resp.ID)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result is not a map: %T", resp.Result)
	}

	if result["kind"] != "message" {
		t.Errorf("Expected kind 'message', got %v", result["kind"])
	}

	parts := result["parts"].([]map[string]interface{})
	if len(parts) == 0 || parts[0]["text"] != "Feedback" {
		t.Errorf("Expected text 'Feedback', got %v", parts[0]["text"])
	}
}

func TestHandleMessage_MethodNotFound(t *testing.T) {
	p := &Professor{}
	req := &adk.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "test-id",
		Method:  "", // Empty method
		Params:  json.RawMessage(`{}`),
	}

	resp, err := p.HandleMessage(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Expected error response, got nil")
	}

	errMap := resp.Error.(map[string]interface{})
	if errMap["code"].(int) != -32601 {
		t.Errorf("Expected error code -32601, got %v", errMap["code"])
	}
	if errMap["message"].(string) != "Method not found" {
		t.Errorf("Expected error message 'Method not found', got %v", errMap["message"])
	}
}
