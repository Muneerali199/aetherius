package model

import (
	"time"

	"github.com/google/uuid"
)

type ModelType string

const (
	ModelLLM       ModelType = "llm"
	ModelImageGen  ModelType = "image-generation"
	ModelEmbedding ModelType = "embedding"
	ModelASR       ModelType = "asr"
	ModelTTS       ModelType = "tts"
)

type AIModel struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Type        ModelType `json:"type"`
	Version     string    `json:"version"`
	SizeGB      float64   `json:"size_gb"`
	MinVRAM     int       `json:"min_vram_gb"`
	Description string    `json:"description"`
	IsAvailable bool      `json:"is_available"`
}

type InferenceRequest struct {
	ModelID     string                 `json:"model_id"`
	Prompt      string                 `json:"prompt,omitempty"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
}

type InferenceResponse struct {
	ID        string                 `json:"id"`
	ModelID   string                 `json:"model_id"`
	Output    map[string]interface{} `json:"output"`
	Usage     TokenUsage             `json:"usage"`
	LatencyMs int64                  `json:"latency_ms"`
	CreatedAt time.Time              `json:"created_at"`
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ModelDeploymentRequest struct {
	ModelID      string `json:"model_id"`
	DeploymentID string `json:"deployment_id"`
	Replicas     int    `json:"replicas"`
}
