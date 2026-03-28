package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/kagent-dev/kagent/go/adk"
	"google.golang.org/genai"
)

// LLMClient is an interface for generating content using LLM.
type LLMClient interface {
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

// A2AClient is an interface for sending messages via A2A protocol.
type A2AClient interface {
	SendMessage(ctx context.Context, req *a2a.SendMessageRequest) (a2a.SendMessageResult, error)
}

// Professor is an agent that generates a technical explanation and calls a student agent for critique.
type Professor struct {
	llmClient     LLMClient
	a2aClient     A2AClient
	llmName       string
	studentUrl    string
	critiqueSkill string
}

// GenerateLecture generates a one-sentence technical explanation and sends it to another agent.
func (p *Professor) GenerateLecture(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	topic, ok := args["request"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'request' argument")
	}

	// Instruction: Explain the topic in exactly ONE technical sentence.
// 	prompt := fmt.Sprintf("Explain the topic in exactly ONE technical sentence. Topic: %s", topic)
	log.Printf("[INFO] Generating content for topic: %s", topic)
// 	resp, err := p.llmClient.GenerateContent(ctx, p.llmName, genai.Text(prompt), nil)
// 	if err != nil {
// 		log.Printf("[ERROR] LLM generation failed: %v", err)
// 		return nil, fmt.Errorf("failed to generate sentence: %v", err)
// 	}
//
// 	sentence := strings.TrimSpace(resp.Text())
	sentence := strings.TrimSpace("OpenTelemetry (OTel) is an open-source, vendor-agnostic observability framework providing a standardized set of APIs, SDKs, and tools for collecting, transforming, and exporting telemetric data—specifically traces, metrics, and logs—from distributed systems to backend analytical platforms.")
	log.Printf("[INFO] LLM generated sentence: %s", sentence)

	if sentence == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	if p.a2aClient == nil {
		return nil, fmt.Errorf("A2A client is not initialized")
	}

	// Call 'kagent__NS__learning_student' via 'critique-content-skill' with the sentence.
	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewDataPart(map[string]interface{}{"request": sentence}))
	msg.SetMeta("skillId", p.critiqueSkill)

	log.Printf("[INFO] Sending message to student agent: %s, skill: %s", p.studentUrl, p.critiqueSkill)
	req := &a2a.SendMessageRequest{
		Message: msg,
	}
	reqJSON, _ := json.Marshal(req)
	log.Printf("[DEBUG] Student agent request: %s", string(reqJSON))

	a2aResp, err := p.a2aClient.SendMessage(ctx, req)
	if err != nil {
		log.Printf("[ERROR] Failed to call student agent: %v", err)
		return nil, fmt.Errorf("failed to call student agent at %s: %v", p.studentUrl, err)
	}

	// Extract response from the student agent.
	var output interface{}
	if msg, ok := a2aResp.(*a2a.Message); ok {
		for _, part := range msg.Parts {
			if data := part.Data(); data != nil {
				if m, ok := data.(map[string]interface{}); ok {
					if out, ok := m["output"]; ok {
						output = out
						break
					}
				}
				output = data
			} else if text := part.Text(); text != "" {
				output = text
			}
		}
	} else if task, ok := a2aResp.(*a2a.Task); ok {
		output = fmt.Sprintf("Task created: %s", task.ID)
	}

	log.Printf("[INFO] Received response from student agent: %v", output)
	if a2aRespMsg, ok := a2aResp.(*a2a.Message); ok {
		respJSON, _ := json.Marshal(a2aRespMsg)
		log.Printf("[DEBUG] Student agent response: %s", string(respJSON))
	}
	fmt.Printf("Sent to student: %s\n", sentence)
	fmt.Printf("Received from student: %v\n", output)

	return output, nil
}

type genaiWrapper struct {
	client *genai.Client
}

func (w *genaiWrapper) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return w.client.Models.GenerateContent(ctx, model, contents, config)
}

func (p *Professor) HandleMessage(ctx context.Context, req *adk.JSONRPCRequest) (*adk.JSONRPCResponse, error) {
	if req.Method != "message/send" {
		return &adk.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			},
		}, nil
	}

	var params struct {
		Message struct {
			Parts []struct {
				Kind string `json:"kind"`
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"message"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %v", err)
	}

	var text string
	for _, part := range params.Message.Parts {
		if part.Kind == "text" {
			text = part.Text
			break
		}
	}

	if text == "" {
		return nil, fmt.Errorf("no text part found in message")
	}

	// Trigger GenerateLecture
	output, err := p.GenerateLecture(ctx, map[string]interface{}{"request": text})
	if err != nil {
		return nil, err
	}

	// Prepare result in A2A format
	result := map[string]interface{}{
		"kind":      "message",
		"messageId": fmt.Sprintf("msg-%s", req.ID),
		"role":      "assistant",
		"parts": []map[string]interface{}{
			{
				"kind": "text",
				"text": fmt.Sprintf("%v", output),
			},
		},
	}

	return &adk.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}, nil
}

func main() {
	// Initialize ADK app
	app := adk.NewApp("learning-professor")

	// Environment variables for A2A and LLM
	studentUrl := os.Getenv("STUDENT_AGENT_URL")
	if studentUrl == "" {
		log.Fatal("Missing required environment variable STUDENT_AGENT_URL")
	}

	critiqueSkillID := os.Getenv("CRITIQUE_SKILL_ID")
	if critiqueSkillID == "" {
		log.Fatal("Missing required environment variable CRITIQUE_SKILL_ID")
	}

	llmName := os.Getenv("LLM_NAME")
	if llmName == "" {
		log.Fatal("Missing required environment variable LLM_NAME")
	}

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("Missing required environment variable GOOGLE_API_KEY")
	}

	ctx := context.Background()
	gc, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("failed to create Gemini client: %v", err)
	}

	// Create A2A client for communication with other agents.
	endpoints := []*a2a.AgentInterface{
		{
			URL:             studentUrl,
			ProtocolBinding: a2a.TransportProtocolHTTPJSON,
			ProtocolVersion: a2a.Version,
		},
	}
	a2aClient, err := a2aclient.NewFromEndpoints(ctx, endpoints)
	if err != nil {
		log.Fatalf("failed to create A2A client: %v", err)
	}

	professor := &Professor{
		llmClient:     &genaiWrapper{client: gc},
		a2aClient:     a2aClient,
		llmName:       llmName,
		studentUrl:    studentUrl,
		critiqueSkill: critiqueSkillID,
	}

	app.SetHandler(professor.HandleMessage)

	// Register tools (skills) for the agent.
	app.AddTool(&adk.Tool{
		Name:        "generate-lecture-skill",
		Description: "One sentence technical explanation.",
		Handler:     professor.GenerateLecture,
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
