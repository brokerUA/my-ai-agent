package adk

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"net/http"

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

func (c *A2AClient) Invoke(ctx context.Context, url string, req InvokeRequest) (*InvokeResponse, error) {
	return &InvokeResponse{Output: "Response from student"}, nil
}
