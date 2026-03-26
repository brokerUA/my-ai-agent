package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kagent-dev/kagent/go/adk"
	"google.golang.org/genai"
)

type LLMClient interface {
	GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error)
}

type A2AClient interface {
	Invoke(ctx context.Context, url string, req adk.InvokeRequest) (*adk.InvokeResponse, error)
}

type Professor struct {
	llmClient     LLMClient
	a2aClient     A2AClient
	llmName       string
	studentURL    string
	critiqueSkill string
}

func (p *Professor) GenerateLecture(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	topic, ok := args["request"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'request' argument")
	}

	prompt := fmt.Sprintf("Provide exactly ONE technically precise academic sentence explaining the topic: %s. Respond in English only. No additional text.", topic)
	resp, err := p.llmClient.GenerateContent(ctx, p.llmName, genai.Text(prompt), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sentence: %v", err)
	}

	sentence := strings.TrimSpace(resp.Text())

	if sentence == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	a2aResp, err := p.a2aClient.Invoke(ctx, p.studentURL, adk.InvokeRequest{
		SkillID: p.critiqueSkill,
		Input:   map[string]interface{}{"request": sentence},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call student agent at %s: %v", p.studentURL, err)
	}

	return a2aResp.Output, nil
}

type genaiWrapper struct {
	client *genai.Client
}

func (w *genaiWrapper) GenerateContent(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return w.client.Models.GenerateContent(ctx, model, contents, config)
}

func main() {
	// Initialize ADK app
	app := adk.NewApp("learning-professor")

	// Environment variables for A2A and LLM
	controllerURL := os.Getenv("KAGENT_CONTROLLER_URL")
	studentAgent := os.Getenv("STUDENT_AGENT_NAME")
	studentNamespace := os.Getenv("STUDENT_AGENT_NAMESPACE")
	critiqueSkillID := os.Getenv("CRITIQUE_SKILL_ID")
	llmName := os.Getenv("LLM_NAME")
	apiKey := os.Getenv("GOOGLE_API_KEY")

	if controllerURL == "" || studentAgent == "" || studentNamespace == "" || critiqueSkillID == "" || llmName == "" || apiKey == "" {
		log.Fatal("Missing required environment variables (KAGENT_CONTROLLER_URL, STUDENT_AGENT_NAME, STUDENT_AGENT_NAMESPACE, CRITIQUE_SKILL_ID, LLM_NAME, GOOGLE_API_KEY)")
	}

	studentURL := fmt.Sprintf("%s/api/a2a/%s/%s", strings.TrimRight(controllerURL, "/"), studentNamespace, studentAgent)

	ctx := context.Background()
	gc, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("failed to create Gemini client: %v", err)
	}

	professor := &Professor{
		llmClient:     &genaiWrapper{client: gc},
		a2aClient:     adk.NewA2AClient(),
		llmName:       llmName,
		studentURL:    studentURL,
		critiqueSkill: critiqueSkillID,
	}

	app.AddTool(&adk.Tool{
		Name:        "generate-lecture-skill",
		Description: "Generates exactly one technically precise academic sentence about a topic and sends it to the student agent.",
		Handler:     professor.GenerateLecture,
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
