package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

type LeonardoVideoRoute struct {
	Model                  string
	Durations              []int
	StartFrame             bool
	EndFrame               bool
	StartFrameRequired     bool
	MaxReferenceImages     int
	ReferenceImageStrength bool
	BuildParameters        func(prompt string, duration, width, height, quantity int) (map[string]any, error)
}

var leonardoVideoRoutes = map[string]LeonardoVideoRoute{
	kling21VideoRoute.Model:              kling21VideoRoute,
	kling25VideoRoute.Model:              kling25VideoRoute,
	kling25TurboStandardVideoRoute.Model: kling25TurboStandardVideoRoute,
	kling26VideoRoute.Model:              kling26VideoRoute,
	kling30VideoRoute.Model:              kling30VideoRoute,
	kling30TurboVideoRoute.Model:         kling30TurboVideoRoute,
	klingVideoO1Route.Model:              klingVideoO1Route,
	klingVideoO3Route.Model:              klingVideoO3Route,
	motion20FastVideoRoute.Model:         motion20FastVideoRoute,
	seedance10ProVideoRoute.Model:        seedance10ProVideoRoute,
	seedance10ProFastVideoRoute.Model:    seedance10ProFastVideoRoute,
	seedance20VideoRoute.Model:           seedance20VideoRoute,
	seedance20FastVideoRoute.Model:       seedance20FastVideoRoute,
	seedance20MiniVideoRoute.Model:       seedance20MiniVideoRoute,
	wan27VideoRoute.Model:                wan27VideoRoute,
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

func NormalizeLeonardoVideoRequestSize(model string, width, height int) (int, int, error) {
	route, ok := LeonardoVideoRouteFor(model)
	if !ok || route.BuildParameters == nil {
		return 0, 0, ErrLeonardoMediaCreateInputInvalid
	}
	if width <= 0 || height <= 0 {
		parameters, err := route.BuildParameters("size normalization", firstDuration(route), 0, 0, 1)
		if err != nil {
			return 0, 0, err
		}
		resolvedWidth, widthOK := parameters["width"].(int)
		resolvedHeight, heightOK := parameters["height"].(int)
		if !widthOK || !heightOK {
			return 0, 0, ErrLeonardoMediaCreateInputInvalid
		}
		return resolvedWidth, resolvedHeight, nil
	}
	if spec := route.BuildParameters; spec != nil {
		parameters, err := spec("size normalization", firstDuration(route), width, height, 1)
		if err != nil {
			return 0, 0, err
		}
		resolvedWidth, widthOK := parameters["width"].(int)
		resolvedHeight, heightOK := parameters["height"].(int)
		if !widthOK || !heightOK {
			return 0, 0, fmt.Errorf("%w: route returned invalid dimensions", ErrLeonardoMediaCreateInputInvalid)
		}
		return resolvedWidth, resolvedHeight, nil
	}
	return width, height, nil
}

func firstDuration(route LeonardoVideoRoute) int {
	if len(route.Durations) > 0 {
		return route.Durations[0]
	}
	return 0
}

func LeonardoVideoUpstreamModel(model string) string {
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
	if route.StartFrameRequired && references.StartFrame == "" || references.StartFrame != "" && !route.StartFrame || references.EndFrame != "" && !route.EndFrame || len(references.ReferenceImages) > route.MaxReferenceImages {
		return nil, fmt.Errorf("%w: requested guidance is not supported by %s", ErrLeonardoVideoParameterUnsupported, model)
	}
	if route.BuildParameters == nil {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	parameters, err := route.BuildParameters(prompt, duration, width, height, quantity)
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
			strength := ""
			if route.ReferenceImageStrength {
				strength = "MID"
			}
			item := leonardoVideoSourceReference(source, strength)
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
