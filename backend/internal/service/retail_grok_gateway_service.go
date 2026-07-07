package service

import (
	"strings"
)

type RetailGrokGatewayService struct {
	retailService *RetailGrokService
	openaiService *OpenAIGatewayService
}

func NewRetailGrokGatewayService(retailService *RetailGrokService, openaiService *OpenAIGatewayService) *RetailGrokGatewayService {
	return &RetailGrokGatewayService{
		retailService: retailService,
		openaiService: openaiService,
	}
}

func (s *RetailGrokGatewayService) OpenAI() *OpenAIGatewayService {
	return s.openaiService
}

func (s *RetailGrokGatewayService) Retail() *RetailGrokService {
	return s.retailService
}

func (s *RetailGrokGatewayService) IsAllowedChatModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return false
	}
	return strings.Contains(model, "grok")
}

func (s *RetailGrokGatewayService) IsAllowedImageModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	switch model {
	case "grok-imagine-image", "grok-imagine-image-quality":
		return true
	default:
		return false
	}
}

func (s *RetailGrokGatewayService) IsAllowedVideoModel(model string) bool {
	return strings.EqualFold(strings.TrimSpace(model), "grok-imagine-video")
}

func (s *RetailGrokGatewayService) BuildUsageLogFromForwardResult(keyID int64, endpoint, model string, result *OpenAIForwardResult) *RetailGrokUsageLog {
	log := &RetailGrokUsageLog{
		RetailGrokKeyID: keyID,
		InboundEndpoint: endpoint,
		Model:           strings.TrimSpace(model),
		Status:          "success",
	}
	if isRetailGrokVideoEndpoint(endpoint) {
		log.VideoCount = 1
	}
	if result == nil {
		return log
	}
	log.RequestID = strings.TrimSpace(result.RequestID)
	log.UpstreamRequestID = strings.TrimSpace(result.RequestID)
	log.UpstreamModel = strings.TrimSpace(result.UpstreamModel)
	log.InputTokens = int64(result.Usage.InputTokens + result.Usage.CacheCreationInputTokens + result.Usage.CacheReadInputTokens)
	log.OutputTokens = int64(result.Usage.OutputTokens)
	log.TotalTokens = log.InputTokens + log.OutputTokens
	log.ImageCount = int64(result.ImageCount)
	if log.RequestID == "" {
		log.RequestID = strings.TrimSpace(result.ResponseID)
	}
	return log
}

func isRetailGrokVideoEndpoint(endpoint string) bool {
	return endpoint == "/retail/v1/videos/generations" || endpoint == "/retail/v1/videos/edits" || endpoint == "/retail/v1/videos/extensions"
}
