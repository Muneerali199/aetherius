package service

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/aetherius/platform/services/ai/internal/model"
)

type AIService struct {
	models map[string]*model.AIModel
}

func New() *AIService {
	s := &AIService{
		models: make(map[string]*model.AIModel),
	}
	s.initDefaultModels()
	return s
}

func (s *AIService) initDefaultModels() {
	s.models["llama-3-70b"] = &model.AIModel{
		ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Name:        "Llama 3 70B",
		Type:        model.ModelLLM,
		Version:     "3.0",
		SizeGB:      140,
		MinVRAM:     80,
		Description: "Meta's largest Llama 3 model with 70B parameters, ideal for complex reasoning and code generation",
		IsAvailable: true,
	}

	s.models["llama-3-8b"] = &model.AIModel{
		ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Name:        "Llama 3 8B",
		Type:        model.ModelLLM,
		Version:     "3.0",
		SizeGB:      16,
		MinVRAM:     16,
		Description: "Meta's efficient Llama 3 model with 8B parameters, suitable for general-purpose inference",
		IsAvailable: true,
	}

	s.models["stable-diffusion-3"] = &model.AIModel{
		ID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Name:        "Stable Diffusion 3",
		Type:        model.ModelImageGen,
		Version:     "3.0",
		SizeGB:      8,
		MinVRAM:     16,
		Description: "Stability AI's state-of-the-art text-to-image generation model",
		IsAvailable: true,
	}

	s.models["nomic-embed-text"] = &model.AIModel{
		ID:          uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		Name:        "Nomic Embed Text",
		Type:        model.ModelEmbedding,
		Version:     "1.5",
		SizeGB:      0.5,
		MinVRAM:     2,
		Description: "Lightweight text embedding model for semantic search and RAG pipelines",
		IsAvailable: true,
	}
}

func (s *AIService) ListModels() []*model.AIModel {
	models := make([]*model.AIModel, 0, len(s.models))
	for _, m := range s.models {
		models = append(models, m)
	}
	return models
}

func (s *AIService) GetModel(id string) *model.AIModel {
	return s.models[id]
}

func (s *AIService) Infer(req *model.InferenceRequest, userID string) *model.InferenceResponse {
	start := time.Now()

	modelObj, ok := s.models[req.ModelID]
	if !ok {
		modelObj = &model.AIModel{Type: model.ModelLLM}
	}

	promptTokens := estimateTokens(req.Prompt)
	if promptTokens == 0 {
		promptTokens = 32
	}

	latencyMs := 50 + rand.Int63n(151)
	time.Sleep(time.Duration(latencyMs) * time.Millisecond)

	resp := &model.InferenceResponse{
		ID:        uuid.New().String(),
		ModelID:   req.ModelID,
		CreatedAt: time.Now(),
	}

	switch modelObj.Type {
	case model.ModelLLM:
		completionTokens := 80 + rand.Intn(121)
		resp.Output = map[string]interface{}{
			"text": generateLLMResponse(req.Prompt, req.MaxTokens),
		}
		resp.Usage = model.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		}

	case model.ModelImageGen:
		completionTokens := 1000 + rand.Intn(501)
		resp.Output = map[string]interface{}{
			"status":    "Image generation requested",
			"prompt":    req.Prompt,
			"model":     req.ModelID,
			"seed":      rand.Intn(999999),
			"image_url": fmt.Sprintf("https://api.aetherius.cloud/v1/ai/generations/%s.png", uuid.New().String()),
		}
		resp.Usage = model.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		}

	case model.ModelEmbedding:
		completionTokens := 0
		vec := make([]float64, 384)
		for i := range vec {
			vec[i] = math.Round((rand.Float64()*2-1)*10000) / 10000
		}
		resp.Output = map[string]interface{}{
			"embedding": vec,
			"dimension": 384,
		}
		resp.Usage = model.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		}

	case model.ModelASR:
		resp.Output = map[string]interface{}{
			"text":    "Transcribed audio from input.",
			"model":   req.ModelID,
			"duration_seconds": 12.5,
		}
		resp.Usage = model.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: 64,
			TotalTokens:      promptTokens + 64,
		}

	case model.ModelTTS:
		resp.Output = map[string]interface{}{
			"status":     "Text-to-speech generation requested",
			"text":       req.Prompt,
			"audio_url":  fmt.Sprintf("https://api.aetherius.cloud/v1/ai/audio/%s.mp3", uuid.New().String()),
			"duration_seconds": 8.0,
		}
		resp.Usage = model.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: 48,
			TotalTokens:      promptTokens + 48,
		}

	default:
		resp.Output = map[string]interface{}{
			"text": "Inference completed for unknown model type.",
		}
		resp.Usage = model.TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: 16,
			TotalTokens:      promptTokens + 16,
		}
	}

	resp.LatencyMs = time.Since(start).Milliseconds()
	return resp
}

func generateLLMResponse(prompt string, maxTokens int) string {
	templates := []string{
		"Based on current market conditions and your infrastructure requirements, I recommend implementing a distributed architecture with horizontal scaling. Key considerations include: 1) Resource allocation across availability zones, 2) Cost optimization through reserved instances, 3) Performance monitoring with real-time dashboards. Let me elaborate on each point.",
		"After analyzing the provided data, here are the key insights: The system shows optimal performance under moderate load conditions. However, I've identified several optimization opportunities: memory management tuning, connection pooling adjustments, and caching strategy improvements. Would you like me to dive deeper into any of these areas?",
		"Thank you for your question. Based on best practices and current industry standards, the recommended approach involves a multi-layered strategy: first, establish baseline metrics; second, implement progressive enhancement; third, validate through A/B testing. This methodology has shown 40%% improvement in similar deployments.",
	}
	text := templates[rand.Intn(len(templates))]
	if maxTokens > 0 && len(text) > maxTokens {
		text = text[:maxTokens]
	}
	return text
}

func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	words := len(strings.Fields(text))
	return int(float64(words) * 1.3)
}
