package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

type LeonardoVideoRoute struct {
	Model              string
	UpstreamModel      string
	Durations          []int
	StartFrame         bool
	EndFrame           bool
	MaxReferenceImages int
}

var leonardoVideoRoutes = map[string]LeonardoVideoRoute{
	"kling-video-o-3":       {Model: "kling-video-o-3", Durations: integerRange(3, 15), StartFrame: true, EndFrame: true, MaxReferenceImages: 7},
	"motion_2.0-fast":       {Model: "motion_2.0-fast", StartFrame: true},
	"seedance-1.0-pro":      {Model: "seedance-1.0-pro", Durations: []int{4, 6, 8, 10}, StartFrame: true, EndFrame: true},
	"seedance-1.0-pro-fast": {Model: "seedance-1.0-pro-fast", Durations: []int{4, 6, 8, 10}, StartFrame: true},
	"seedance-2.0":          {Model: "seedance-2.0", Durations: integerRange(4, 15), StartFrame: true, EndFrame: true, MaxReferenceImages: 4},
	"seedance-2.0-fast":     {Model: "seedance-2.0-fast", Durations: integerRange(4, 15), StartFrame: true, EndFrame: true, MaxReferenceImages: 4},
	"seedance-2.0-mini":     {Model: "seedance-2.0-mini", Durations: integerRange(4, 15), StartFrame: true, EndFrame: true, MaxReferenceImages: 4},
	"wan-2.7":               {Model: "wan-2.7", Durations: integerRange(2, 10), StartFrame: true, EndFrame: true, MaxReferenceImages: 6},
}

func integerRange(first, last int) []int {
	values := make([]int, 0, last-first+1)
	for value := first; value <= last; value++ {
		values = append(values, value)
	}
	return values
}

func LeonardoVideoRouteFor(model string) (LeonardoVideoRoute, bool) {
	route, ok := leonardoVideoRoutes[strings.TrimSpace(model)]
	return route, ok
}

func LeonardoVideoUpstreamModel(model string) string {
	route, ok := LeonardoVideoRouteFor(model)
	if ok && route.UpstreamModel != "" {
		return route.UpstreamModel
	}
	return strings.TrimSpace(model)
}

// IsExplicitCanvasVideoModel reports whether the model has an explicit video
// routing rule with a well-known platform.
func IsExplicitCanvasVideoModel(model string) bool {
	return ExplicitCanvasVideoPlatform(model) != ""
}

// ExplicitCanvasVideoPlatform returns the platform a video model must be routed
// to. minimax-h3 is served in-house over the OpenAI-compatible video API
// (account 73, local kai gateway), so it routes to PlatformOpenAI; all other
// explicit video models route to Leonardo REST v2.
func ExplicitCanvasVideoPlatform(model string) string {
	if strings.EqualFold(strings.TrimSpace(model), "minimax-h3") {
		return PlatformOpenAI
	}
	if _, ok := LeonardoVideoRouteFor(model); ok {
		return PlatformLeonardo
	}
	return ""
}

type LeonardoVideoV1References struct {
	StartFrame      string
	EndFrame        string
	ReferenceImages []string
}

func BuildLeonardoVideoV2Request(model, prompt string, duration, width, height, quantity int, public bool, references LeonardoVideoV1References) ([]byte, error) {
	route, ok := LeonardoVideoRouteFor(model)
	if !ok || strings.TrimSpace(prompt) == "" {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	if len(route.Durations) > 0 && !containsInt(route.Durations, duration) {
		return nil, fmt.Errorf("%w: duration is not supported by %s", ErrLeonardoVideoParameterUnsupported, model)
	}
	if references.StartFrame != "" && !route.StartFrame || references.EndFrame != "" && !route.EndFrame || len(references.ReferenceImages) > route.MaxReferenceImages {
		return nil, fmt.Errorf("%w: requested guidance is not supported by %s", ErrLeonardoVideoParameterUnsupported, model)
	}
	parameters, err := LeonardoVideoGenerationParameters(model, prompt, duration, width, height, quantity)
	if err != nil {
		return nil, err
	}
	guidances := map[string]any{}
	if references.StartFrame != "" {
		guidances["start_frame"] = []any{leonardoVideoSourceReference(references.StartFrame, "")}
	}
	if references.EndFrame != "" {
		guidances["end_frame"] = []any{leonardoVideoSourceReference(references.EndFrame, "")}
	}
	if len(references.ReferenceImages) > 0 {
		items := make([]any, 0, len(references.ReferenceImages))
		for index, source := range references.ReferenceImages {
			item := leonardoVideoSourceReference(source, "MID")
			item["order"] = index
			items = append(items, item)
		}
		guidances["image_reference"] = items
	}
	if len(guidances) > 0 {
		parameters["guidances"] = guidances
	}
	return json.Marshal(map[string]any{"model": model, "public": public, "parameters": parameters})
}

func leonardoVideoSourceReference(source, strength string) map[string]any {
	item := map[string]any{"image": map[string]any{"source": strings.TrimSpace(source)}}
	if strength != "" {
		item["strength"] = strength
	}
	return item
}

func containsInt(values []int, wanted int) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func ValidateLeonardoVideoV2Sources(model string, sources []LeonardoRawImageSource) error {
	route, ok := LeonardoVideoRouteFor(model)
	if !ok {
		return ErrLeonardoMediaCreateInputInvalid
	}
	counts := map[string]int{}
	for _, source := range sources {
		counts[source.Section]++
	}
	if counts["start_frame"] > 1 || counts["start_frame"] > 0 && !route.StartFrame || counts["end_frame"] > 1 || counts["end_frame"] > 0 && !route.EndFrame || counts["image_reference"] > route.MaxReferenceImages {
		return ErrLeonardoVideoParameterUnsupported
	}
	for section := range counts {
		if section != "start_frame" && section != "end_frame" && section != "image_reference" {
			return ErrLeonardoVideoParameterUnsupported
		}
	}
	return nil
}
