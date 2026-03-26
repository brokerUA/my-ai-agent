package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/kagent-dev/kagent/go/adk"
	"google.golang.org/genai"
)

// MockLLMClient implements LLMClient interface for testing
type MockLLMClient struct {
	Response *genai.GenerateContentResponse
	Err      error
}

func (m *MockLLMClient) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return m.Response, m.Err
}

// MockA2AClient implements A2AClient interface for testing
type MockA2AClient struct {
	Response *adk.InvokeResponse
	Err      error
}

func (m *MockA2AClient) Invoke(ctx context.Context, url string, req adk.InvokeRequest) (*adk.InvokeResponse, error) {
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
		Response: &adk.InvokeResponse{
			Output: "Student feedback",
		},
	}

	p := &Professor{
		llmClient:     mockLLM,
		a2aClient:     mockA2A,
		llmName:       "gemini-pro",
		studentURL:    "http://student",
		critiqueSkill: "critique",
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
		studentURL: "http://student",
	}
	args := map[string]interface{}{"request": "topic"}

	_, err := p.GenerateLecture(context.Background(), args)
	if err == nil {
		t.Fatal("Expected error from A2A, got nil")
	}
}
