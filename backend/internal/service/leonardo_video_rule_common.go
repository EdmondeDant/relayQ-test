package service

import (
	"fmt"
	"strings"
)

type leonardoVideoParameterSpec struct {
	model              string
	durations          []int
	defaultWidth       int
	defaultHeight      int
	durationOptional   bool
	seed               bool
	motionHasAudio     *bool
	resolution         func(width, height int) string
	normalizeSize      func(width, height int) (int, int)
	startFrame         bool
	endFrame           bool
	startFrameRequired bool
	maxReferences      int
	referenceStrength  bool
}

func newLeonardoVideoRoute(spec leonardoVideoParameterSpec) LeonardoVideoRoute {
	return LeonardoVideoRoute{
		Model: spec.model, Durations: append([]int(nil), spec.durations...),
		StartFrame: spec.startFrame, EndFrame: spec.endFrame,
		StartFrameRequired: spec.startFrameRequired, MaxReferenceImages: spec.maxReferences,
		ReferenceImageStrength: spec.referenceStrength,
		BuildParameters: func(prompt string, duration, width, height, quantity int) (map[string]any, error) {
			return buildLeonardoVideoParameters(spec, prompt, duration, width, height, quantity)
		},
	}
}

func buildLeonardoVideoParameters(spec leonardoVideoParameterSpec, prompt string, duration, width, height, quantity int) (map[string]any, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" || quantity != 1 || !spec.durationOptional && !containsInt(spec.durations, duration) {
		return nil, fmt.Errorf("%w: unsupported parameters for %s", ErrLeonardoVideoPricingEvidenceUnavailable, spec.model)
	}
	if width <= 0 || height <= 0 {
		width, height = spec.defaultWidth, spec.defaultHeight
	}
	if spec.normalizeSize != nil {
		width, height = spec.normalizeSize(width, height)
	}
	parameters := map[string]any{"prompt": prompt, "width": width, "height": height, "quantity": quantity}
	if !spec.durationOptional {
		parameters["duration"] = duration
	}
	if spec.seed {
		parameters["seed"] = -1
	}
	if spec.motionHasAudio != nil {
		parameters["motion_has_audio"] = *spec.motionHasAudio
	}
	if spec.resolution != nil {
		if value := spec.resolution(width, height); value != "" {
			parameters["resolution"] = value
		}
	}
	return parameters, nil
}

func boolPointer(value bool) *bool { return &value }

func normalizeVideoSize(horizontalWidth, horizontalHeight, square int) func(int, int) (int, int) {
	return func(width, height int) (int, int) {
		switch {
		case width == height:
			return square, square
		case width < height:
			return horizontalHeight, horizontalWidth
		default:
			return horizontalWidth, horizontalHeight
		}
	}
}

func preserveAllowedVideoSizes(allowed [][2]int, fallback func(int, int) (int, int)) func(int, int) (int, int) {
	return func(width, height int) (int, int) {
		for _, size := range allowed {
			if width == size[0] && height == size[1] {
				return width, height
			}
		}
		return fallback(width, height)
	}
}
