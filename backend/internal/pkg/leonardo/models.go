package leonardo

import (
	"encoding/json"
	"strings"
)

type ModelModality string

const (
	ModelModalityImage ModelModality = "image"
	ModelModalityVideo ModelModality = "video"
	ModelModalityAudio ModelModality = "audio"
	ModelModality3D    ModelModality = "3d"
)

type VerifiedModel struct {
	DisplayName       string
	ProviderModelID   string
	RequestModelSlug  string
	Modality          ModelModality
	ImageCapabilities *VerifiedImageCapabilities
}

type VerifiedImageCapabilities struct {
	MinWidth         int
	MaxWidth         int
	MinHeight        int
	MaxHeight        int
	Multiple         int
	MaxQuantity      int
	AllowedQualities []string
	ImageReference   *VerifiedImageGuidance
	Content          *VerifiedImageGuidance
	Style            *VerifiedImageGuidance
}

type VerifiedImageGuidance struct {
	MaxItems          int
	AllowedStrengths  []string
	AllowedImageTypes []string
	DefaultStrength   string
	StrengthRequired  bool
}

var verifiedModels = []VerifiedModel{
	{
		DisplayName:      "FLUX Schnell",
		ProviderModelID:  "1dd50843-d653-4516-a8e3-f0238ee453ff",
		RequestModelSlug: "flux-schnell",
		Modality:         ModelModalityImage,
		ImageCapabilities: &VerifiedImageCapabilities{
			MinWidth: 32, MaxWidth: 2048, MinHeight: 32, MaxHeight: 2048, Multiple: 8, MaxQuantity: 1,
			Content: &VerifiedImageGuidance{MaxItems: 1, AllowedStrengths: []string{"LOW", "MID", "HIGH"}, AllowedImageTypes: []string{"INIT", "GENERATION", "UPLOADED", "GENERATED", "VARIATION"}, DefaultStrength: "MID"},
			Style:   &VerifiedImageGuidance{MaxItems: 1, AllowedStrengths: []string{"LOW", "MID", "HIGH", "ULTRA", "MAX"}, AllowedImageTypes: []string{"INIT", "GENERATION", "UPLOADED", "GENERATED", "VARIATION"}, DefaultStrength: "MID"},
		},
	},
	{
		DisplayName: "GPT Image 2", ProviderModelID: "135b2740-a20b-48c8-8f86-6f68199e06c5", RequestModelSlug: "gpt-image-2", Modality: ModelModalityImage,
		ImageCapabilities: &VerifiedImageCapabilities{MinWidth: 768, MaxWidth: 3808, MinHeight: 672, MaxHeight: 3584, Multiple: 16, MaxQuantity: 8, AllowedQualities: []string{"low", "medium", "high"}, ImageReference: &VerifiedImageGuidance{MaxItems: 6, AllowedImageTypes: []string{"UPLOADED", "GENERATED"}}},
	},
	{
		DisplayName: "Nano Banana 2", ProviderModelID: "7418e71f-4133-4e1b-9895-bee19f48f2ce", RequestModelSlug: "nano-banana-2", Modality: ModelModalityImage,
		ImageCapabilities: &VerifiedImageCapabilities{MinWidth: 0, MaxWidth: 6336, MinHeight: 0, MaxHeight: 5504, MaxQuantity: 8, AllowedQualities: []string{"low"}, ImageReference: &VerifiedImageGuidance{MaxItems: 6, AllowedStrengths: []string{"LOW", "MID", "HIGH"}, AllowedImageTypes: []string{"UPLOADED", "GENERATED"}, DefaultStrength: "MID", StrengthRequired: true}},
	},
	{
		DisplayName: "Nano Banana 2 Lite", ProviderModelID: "21278dfe-ac26-4292-82e0-8e588373a30c", RequestModelSlug: "nano-banana-2-lite", Modality: ModelModalityImage,
		ImageCapabilities: &VerifiedImageCapabilities{MinWidth: 0, MaxWidth: 1584, MinHeight: 0, MaxHeight: 1376, MaxQuantity: 8, AllowedQualities: []string{"low"}, ImageReference: &VerifiedImageGuidance{MaxItems: 6, AllowedStrengths: []string{"LOW", "MID", "HIGH"}, AllowedImageTypes: []string{"UPLOADED", "GENERATED"}, DefaultStrength: "MID", StrengthRequired: true}},
	},
	{
		DisplayName: "Cinematic Kino", ProviderModelID: "aa77f04e-3eec-4034-9c07-d0f619684628", RequestModelSlug: "kino-xl", Modality: ModelModalityImage,
		ImageCapabilities: &VerifiedImageCapabilities{MinWidth: 32, MaxWidth: 1536, MinHeight: 32, MaxHeight: 1536, MaxQuantity: 8, AllowedQualities: []string{"low", "high"}},
	},
	{
		DisplayName: "Concept Art", ProviderModelID: "dd29ac47-ea88-4720-8678-b8633245c09c", RequestModelSlug: "concept-art", Modality: ModelModalityImage,
		ImageCapabilities: &VerifiedImageCapabilities{MinWidth: 32, MaxWidth: 1584, MinHeight: 32, MaxHeight: 1536, MaxQuantity: 8, AllowedQualities: []string{"low", "high"}},
	},
	{
		DisplayName: "Graphic Design", ProviderModelID: "9d4ace10-25dd-42fd-a6be-a301a7ac614f", RequestModelSlug: "graphic-design", Modality: ModelModalityImage,
		ImageCapabilities: &VerifiedImageCapabilities{MinWidth: 32, MaxWidth: 1584, MinHeight: 32, MaxHeight: 1536, MaxQuantity: 8, AllowedQualities: []string{"low", "high"}},
	},
	{
		DisplayName: "Illustrative Albedo", ProviderModelID: "2067ae52-33fd-4a82-bb92-c2c55e7d2786", RequestModelSlug: "illustrative-albedo", Modality: ModelModalityImage,
		ImageCapabilities: &VerifiedImageCapabilities{MinWidth: 32, MaxWidth: 1584, MinHeight: 32, MaxHeight: 1536, MaxQuantity: 8, AllowedQualities: []string{"low", "high"}},
	},
	{DisplayName: "Seedance 1.0 Pro Fast", ProviderModelID: "b959ecc2-a7f0-4618-9877-1bc45fc27570", RequestModelSlug: "seedance-1.0-pro-fast", Modality: ModelModalityVideo},
	{DisplayName: "Motion 2.0 Fast", ProviderModelID: "0a7a3eb2-3905-480b-a89a-2f3ffff545e7", RequestModelSlug: "motion_2.0-fast", Modality: ModelModalityVideo},
	{DisplayName: "Seedance 1.0 Pro", ProviderModelID: "728c9eac-b17d-47fe-b382-b9a28687fa85", RequestModelSlug: "seedance-1.0-pro", Modality: ModelModalityVideo},
	{DisplayName: "Wan 2.7", ProviderModelID: "52884d8c-e2b9-4bb1-8ed5-927d390fe53a", RequestModelSlug: "wan-2.7", Modality: ModelModalityVideo},
	{DisplayName: "Kling Video O3 Omni", ProviderModelID: "0d5109cf-d256-4720-86d3-d8e5ff5a3ce2", RequestModelSlug: "kling-video-o-3", Modality: ModelModalityVideo},
}

func ValidateSyncedModel(model Model) (VerifiedModel, bool) {
	verified, ok := ResolveByProviderModelID(strings.TrimSpace(model.ID))
	if !ok || (verified.Modality != ModelModalityImage && verified.Modality != ModelModalityVideo && verified.Modality != ModelModalityAudio && verified.Modality != ModelModality3D) {
		return VerifiedModel{}, false
	}
	var schema ParameterSchema
	if len(model.Parameters) == 0 || json.Unmarshal(model.Parameters, &schema) != nil || schema.Type != "object" || schema.Properties == nil || schema.AdditionalProperties == nil || *schema.AdditionalProperties {
		return VerifiedModel{}, false
	}
	for _, required := range schema.Required {
		if strings.TrimSpace(required) == "" {
			return VerifiedModel{}, false
		}
	}
	return verified, true
}

func ListVerifiedModels() []VerifiedModel {
	models := make([]VerifiedModel, len(verifiedModels))
	for i, model := range verifiedModels {
		models[i] = cloneVerifiedModel(model)
	}
	return models
}

func ListVerifiedVideoModels() []VerifiedModel {
	models := make([]VerifiedModel, 0)
	for _, model := range verifiedModels {
		if model.Modality == ModelModalityVideo {
			models = append(models, model)
		}
	}
	return models
}

func ListVerifiedAudioModels() []VerifiedModel {
	models := make([]VerifiedModel, 0)
	for _, model := range verifiedModels {
		if model.Modality == ModelModalityAudio {
			models = append(models, model)
		}
	}
	return models
}

func ListVerified3DModels() []VerifiedModel {
	models := make([]VerifiedModel, 0)
	for _, model := range verifiedModels {
		if model.Modality == ModelModality3D {
			models = append(models, model)
		}
	}
	return models
}

func ResolveByRequestModelSlug(slug string) (VerifiedModel, bool) {
	for _, model := range verifiedModels {
		if model.RequestModelSlug == slug {
			return cloneVerifiedModel(model), true
		}
	}
	return VerifiedModel{}, false
}

func ResolveByProviderModelID(id string) (VerifiedModel, bool) {
	for _, model := range verifiedModels {
		if model.ProviderModelID == id {
			return cloneVerifiedModel(model), true
		}
	}
	return VerifiedModel{}, false
}

func cloneVerifiedModel(model VerifiedModel) VerifiedModel {
	if model.ImageCapabilities == nil {
		return model
	}
	capabilities := *model.ImageCapabilities
	capabilities.AllowedQualities = append([]string(nil), capabilities.AllowedQualities...)
	capabilities.ImageReference = cloneVerifiedImageGuidance(capabilities.ImageReference)
	capabilities.Content = cloneVerifiedImageGuidance(capabilities.Content)
	capabilities.Style = cloneVerifiedImageGuidance(capabilities.Style)
	model.ImageCapabilities = &capabilities
	return model
}

func cloneVerifiedImageGuidance(guidance *VerifiedImageGuidance) *VerifiedImageGuidance {
	if guidance == nil {
		return nil
	}
	clone := *guidance
	clone.AllowedStrengths = append([]string(nil), guidance.AllowedStrengths...)
	clone.AllowedImageTypes = append([]string(nil), guidance.AllowedImageTypes...)
	return &clone
}
