package adk

import (
	"context"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"strings"

	_ "embed"
)

//go:embed agent-card.json
var agentCard []byte

type App struct {
	name   string
	tools  []*Tool
	logger logr.Logger
}

func NewApp(name string) *App {
	zapLogger, _ := zap.NewProduction()
	logger := zapr.NewLogger(zapLogger)
	return &App{
		name:   name,
		logger: logger,
	}
}

type Tool struct {
	Name        string
	Description string
	Handler     func(context.Context, map[string]interface{}) (interface{}, error)
}

func (a *App) AddTool(t *Tool) {
	a.tools = append(a.tools, t)
}

func (a *App) Run() error {
	if len(a.tools) == 0 {
		return fmt.Errorf("no tools added")
	}

	mux := http.NewServeMux()
	
	mux.HandleFunc("GET /.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(agentCard)
	})

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "online",
			"name":   a.name,
			"info":   "AI Agent Professor is running",
		})
	})

	mux.HandleFunc("POST /api/skill/", func(w http.ResponseWriter, r *http.Request) {
		skillID := r.URL.Path[len("/api/skill/"):]
		var tool *Tool
		for _, t := range a.tools {
			if t.Name == skillID {
				tool = t
				break
			}
		}

		if tool == nil {
			http.Error(w, "Skill not found", http.StatusNotFound)
			return
		}

		var args map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		result, err := tool.Handler(r.Context(), args)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"output": result})
	})

	a.logger.Info("Agent started", "name", a.name, "addr", ":8080")
	return http.ListenAndServe(":8080", mux)
}

type InvokeRequest struct {
	SkillID string
	Input   map[string]interface{}
}

type InvokeResponse struct {
	Output interface{}
}

type A2AClient struct{}

func NewA2AClient() *A2AClient {
	return &A2AClient{}
}

func (c *A2AClient) Invoke(ctx context.Context, baseURL string, req InvokeRequest) (*InvokeResponse, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %v", err)
	}

	skillPath := "/api/skill/" + req.SkillID
	if strings.HasSuffix(u.Path, "/") && strings.HasPrefix(skillPath, "/") {
		u.Path = u.Path + skillPath[1:]
	} else if !strings.HasSuffix(u.Path, "/") && !strings.HasPrefix(skillPath, "/") {
		u.Path = u.Path + "/" + skillPath
	} else {
		u.Path = u.Path + skillPath
	}

	body, err := json.Marshal(req.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request input: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	var invokeResp InvokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&invokeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &invokeResp, nil
}
