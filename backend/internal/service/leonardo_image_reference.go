package service

import (
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/leonardo"
)

var ErrLeonardoImageReferenceInvalid = errors.New("leonardo image reference is invalid")

type LeonardoImageReferenceCapability struct {
	MaxItems         int
	AllowedStrengths []string
	DefaultStrength  string
	StrengthRequired bool
}

type LeonardoFluxGuidances struct {
	Content []leonardo.ImageReference `json:"content,omitempty"`
	Style   []leonardo.ImageReference `json:"style,omitempty"`
}

func LeonardoImageReferenceCapabilityForModel(model string) *LeonardoImageReferenceCapability {
	verified, ok := leonardo.ResolveByRequestModelSlug(strings.TrimSpace(model))
	if !ok || verified.ImageCapabilities == nil || verified.ImageCapabilities.ImageReference == nil {
		return nil
	}
	guidance := verified.ImageCapabilities.ImageReference
	return &LeonardoImageReferenceCapability{MaxItems: guidance.MaxItems, AllowedStrengths: append([]string(nil), guidance.AllowedStrengths...), DefaultStrength: guidance.DefaultStrength, StrengthRequired: guidance.StrengthRequired}
}

func BuildLeonardoImageReferenceGuidance(references []leonardo.ImageReference, capability *LeonardoImageReferenceCapability) (map[string]any, error) {
	if len(references) == 0 {
		return nil, nil
	}
	if capability == nil || capability.MaxItems <= 0 || len(references) > capability.MaxItems {
		return nil, ErrLeonardoImageReferenceInvalid
	}
	allowedStrengths := make(map[string]struct{}, len(capability.AllowedStrengths))
	for _, strength := range capability.AllowedStrengths {
		allowedStrengths[strength] = struct{}{}
	}
	normalized := make([]leonardo.ImageReference, len(references))
	for i, reference := range references {
		reference.Image.ID = strings.TrimSpace(reference.Image.ID)
		if reference.Image.ID == "" || (reference.Image.Type != "UPLOADED" && reference.Image.Type != "GENERATED") {
			return nil, ErrLeonardoImageReferenceInvalid
		}
		reference.Strength = strings.TrimSpace(reference.Strength)
		strengthRequired := capability.StrengthRequired || len(capability.AllowedStrengths) > 0
		if strengthRequired {
			if reference.Strength == "" {
				reference.Strength = capability.DefaultStrength
			}
			if _, ok := allowedStrengths[reference.Strength]; !ok {
				return nil, ErrLeonardoImageReferenceInvalid
			}
		} else if reference.Strength != "" {
			return nil, ErrLeonardoImageReferenceInvalid
		}
		normalized[i] = reference
	}
	return map[string]any{"image_reference": normalized}, nil
}

func BuildLeonardoFluxGuidances(model string, guidances LeonardoFluxGuidances) (map[string]any, error) {
	if len(guidances.Content) == 0 && len(guidances.Style) == 0 {
		return nil, nil
	}
	verified, ok := leonardo.ResolveByRequestModelSlug(strings.TrimSpace(model))
	if !ok || verified.ImageCapabilities == nil {
		return nil, ErrLeonardoImageReferenceInvalid
	}
	content, err := normalizeLeonardoGuidance(guidances.Content, verified.ImageCapabilities.Content)
	if err != nil {
		return nil, err
	}
	style, err := normalizeLeonardoGuidance(guidances.Style, verified.ImageCapabilities.Style)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any, 2)
	if len(content) > 0 {
		result["content"] = content
	}
	if len(style) > 0 {
		result["style"] = style
	}
	return result, nil
}

func normalizeLeonardoGuidance(references []leonardo.ImageReference, capability *leonardo.VerifiedImageGuidance) ([]leonardo.ImageReference, error) {
	if len(references) == 0 {
		return nil, nil
	}
	if capability == nil || capability.MaxItems <= 0 || len(references) > capability.MaxItems {
		return nil, ErrLeonardoImageReferenceInvalid
	}
	strengths := make(map[string]struct{}, len(capability.AllowedStrengths))
	for _, value := range capability.AllowedStrengths {
		strengths[value] = struct{}{}
	}
	types := make(map[string]struct{}, len(capability.AllowedImageTypes))
	for _, value := range capability.AllowedImageTypes {
		types[value] = struct{}{}
	}
	normalized := make([]leonardo.ImageReference, len(references))
	for i, reference := range references {
		reference.Image.ID = strings.TrimSpace(reference.Image.ID)
		reference.Image.Type = strings.TrimSpace(reference.Image.Type)
		reference.Strength = strings.TrimSpace(reference.Strength)
		if reference.Strength == "" {
			reference.Strength = capability.DefaultStrength
		}
		if reference.Image.ID == "" {
			return nil, ErrLeonardoImageReferenceInvalid
		}
		if _, ok := types[reference.Image.Type]; !ok {
			return nil, ErrLeonardoImageReferenceInvalid
		}
		if _, ok := strengths[reference.Strength]; !ok {
			return nil, ErrLeonardoImageReferenceInvalid
		}
		normalized[i] = reference
	}
	return normalized, nil
}
