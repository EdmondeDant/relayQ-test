package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

var ErrLeonardoVideoPricingEvidenceUnavailable = errors.New("Leonardo video pricing evidence is unavailable")

const LeonardoVideoPricingPolicyVersion = "leonardo-video-pricing-policy/2026-08-15-v7"

type LeonardoVideoPriceRequest struct {
	Model          string
	Duration       int
	Width          int
	Height         int
	Quantity       int
	MotionHasAudio bool
	QualityTier    string
}

type LeonardoVideoPriceEstimate struct {
	EstimatedCostUSD decimal.Decimal
	PricingVersion   string
	PricingSource    string
	MatchType        string
}

type LeonardoVideoPriceResolver struct{}

func NewLeonardoVideoPriceResolver() LeonardoVideoPriceResolver {
	return LeonardoVideoPriceResolver{}
}

func (LeonardoVideoPriceResolver) Estimate(ctx context.Context, request LeonardoVideoPriceRequest) (*LeonardoVideoPriceEstimate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if request.Model == "" || request.Duration < 0 || request.Width <= 0 || request.Height <= 0 || request.Quantity <= 0 {
		return nil, ErrLeonardoVideoPricingEvidenceUnavailable
	}
	if request.MotionHasAudio || request.Quantity != 1 {
		return nil, ErrLeonardoVideoPricingEvidenceUnavailable
	}
	seedancePrices := map[string]map[[2]int]string{
		"seedance-1.0-pro-fast": {
			{4, 480}: "0.0449", {6, 480}: "0.0748", {8, 480}: "0.1047", {10, 480}: "0.1346",
			{4, 720}: "0.1047", {6, 720}: "0.1645", {8, 720}: "0.2243", {10, 720}: "0.2841",
			{4, 1080}: "0.2691", {6, 1080}: "0.3887", {8, 1080}: "0.5233", {10, 1080}: "0.6578",
		},
		"seedance-1.0-pro": {
			{4, 480}: "0.1346", {6, 480}: "0.1944", {8, 480}: "0.2542", {10, 480}: "0.3289",
			{4, 720}: "0.2841", {6, 720}: "0.4186", {8, 720}: "0.5532", {10, 720}: "0.6877",
			{4, 1080}: "0.6578", {6, 1080}: "0.9867", {8, 1080}: "1.3156", {10, 1080}: "1.6445",
		},
	}
	resolution, ok := LeonardoVideoResolution(request.Model, request.Width, request.Height)
	if !ok {
		return nil, ErrLeonardoVideoPricingEvidenceUnavailable
	}
	var price decimal.Decimal
	switch request.Model {
	case "seedance-1.0-pro-fast", "seedance-1.0-pro":
		value, found := seedancePrices[request.Model][[2]int{request.Duration, resolution}]
		if !found {
			return nil, ErrLeonardoVideoPricingEvidenceUnavailable
		}
		price = decimal.RequireFromString(value)
	case "motion_2.0-fast":
		if request.Duration != 0 {
			return nil, ErrLeonardoVideoPricingEvidenceUnavailable
		}
		price = decimal.RequireFromString("0.1047")
	case "wan-2.7":
		if request.Duration < 2 || request.Duration > 10 {
			return nil, ErrLeonardoVideoPricingEvidenceUnavailable
		}
		price = decimal.RequireFromString("0.1645").Add(decimal.NewFromInt(int64(request.Duration - 2)).Mul(decimal.RequireFromString("0.082225")))
	case "minimax-h3":
		if request.Duration < 5 || request.Duration > 15 {
			return nil, ErrLeonardoVideoPricingEvidenceUnavailable
		}
		price = interpolateLeonardoVideoPrice("0.897", "2.691", request.Duration, 5, 15)
	case "kling-video-o-3":
		if request.Duration < 3 || request.Duration > 15 {
			return nil, ErrLeonardoVideoPricingEvidenceUnavailable
		}
		price = decimal.RequireFromString("1.0046").Add(decimal.NewFromInt(int64(request.Duration - 3)).Mul(decimal.RequireFromString("0.3348833333333333")))
	case "seedance-2.0", "seedance-2.0-fast", "seedance-2.0-mini":
		endpoints := map[string]map[int][2]string{
			"seedance-2.0":      {480: {"0.8402", "3.153"}, 720: {"1.8075", "6.7813"}, 1080: {"4.0679", "15.258"}, 2160: {"11.3859", "42.6972"}},
			"seedance-2.0-fast": {480: {"0.6713", "2.5221"}, 720: {"1.4457", "5.4239"}},
			"seedance-2.0-mini": {480: {"0.3588", "1.3455"}, 720: {"0.7774", "2.9153"}},
		}
		values, found := endpoints[request.Model][resolution]
		if !found || request.Duration < 4 || request.Duration > 15 {
			return nil, ErrLeonardoVideoPricingEvidenceUnavailable
		}
		price = interpolateLeonardoVideoPrice(values[0], values[1], request.Duration, 4, 15)
	case "kling-2.1", "kling-2.5", "kling-2.5-turbo-standard", "kling-2.6", "kling-video-o-1":
		prices := map[string][2]string{
			"kling-2.1": {"0.613", "1.2259"}, "kling-2.5": {"0.3513", "0.7027"},
			"kling-2.5-turbo-standard": {"0.2841", "0.5681"}, "kling-2.6": {"0.903", "1.806"},
			"kling-video-o-1": {"0.755", "1.51"},
		}
		if request.Duration != 5 && request.Duration != 10 {
			return nil, ErrLeonardoVideoPricingEvidenceUnavailable
		}
		price = decimal.RequireFromString(prices[request.Model][request.Duration/5-1])
	case "kling-3.0", "kling-3.0-turbo":
		endpoints := map[string][2]string{"kling-3.0": {"0.5651", "2.8256"}, "kling-3.0-turbo": {"0.7176", "3.588"}}
		if request.Duration < 3 || request.Duration > 15 {
			return nil, ErrLeonardoVideoPricingEvidenceUnavailable
		}
		price = interpolateLeonardoVideoPrice(endpoints[request.Model][0], endpoints[request.Model][1], request.Duration, 3, 15)
	default:
		return nil, ErrLeonardoVideoPricingEvidenceUnavailable
	}
	return &LeonardoVideoPriceEstimate{
		EstimatedCostUSD: price,
		PricingVersion:   LeonardoVideoPricingPolicyVersion,
		PricingSource:    "leonardo_authenticated_pricing_calculator",
		MatchType:        "model_duration_resolution_exact",
	}, nil
}

func interpolateLeonardoVideoPrice(minValue, maxValue string, value, minValueAt, maxValueAt int) decimal.Decimal {
	minPrice := decimal.RequireFromString(minValue)
	return minPrice.Add(decimal.NewFromInt(int64(value - minValueAt)).Mul(decimal.RequireFromString(maxValue).Sub(minPrice)).Div(decimal.NewFromInt(int64(maxValueAt - minValueAt))))
}

func LeonardoVideoResolution(model string, width, height int) (int, bool) {
	dimensions := map[string]map[[2]int]int{
		"seedance-1.0-pro-fast":    {{864, 480}: 480, {736, 544}: 480, {640, 640}: 480, {544, 736}: 480, {480, 864}: 480, {960, 416}: 480, {1248, 704}: 720, {1120, 832}: 720, {960, 960}: 720, {832, 1120}: 720, {704, 1248}: 720, {1504, 640}: 720, {1920, 1088}: 1080, {1664, 1248}: 1080, {1440, 1440}: 1080, {1248, 1664}: 1080, {1088, 1920}: 1080, {2176, 928}: 1080},
		"seedance-1.0-pro":         {{864, 480}: 480, {736, 544}: 480, {640, 640}: 480, {544, 736}: 480, {480, 864}: 480, {960, 416}: 480, {1248, 704}: 720, {1120, 832}: 720, {960, 960}: 720, {832, 1120}: 720, {704, 1248}: 720, {1504, 640}: 720, {1920, 1088}: 1080, {1664, 1248}: 1080, {1440, 1440}: 1080, {1248, 1664}: 1080, {1088, 1920}: 1080, {2176, 928}: 1080},
		"motion_2.0-fast":          {{832, 480}: 480, {480, 832}: 480, {512, 768}: 480, {576, 720}: 480, {1280, 720}: 720, {720, 1152}: 720, {768, 1152}: 720, {864, 1024}: 720},
		"wan-2.7":                  {{1280, 720}: 720, {960, 960}: 720, {720, 1280}: 720, {1920, 1080}: 1080, {1440, 1440}: 1080, {1080, 1920}: 1080},
		"minimax-h3":               {{1376, 768}: 768},
		"kling-video-o-3":          {{1280, 720}: 720, {960, 960}: 720, {720, 1280}: 720, {1920, 1080}: 1080, {1440, 1440}: 1080, {1080, 1920}: 1080, {3840, 2160}: 2160, {2880, 2880}: 2160, {2160, 3840}: 2160},
		"seedance-2.0":             {{864, 496}: 480, {496, 864}: 480, {640, 640}: 480, {1280, 720}: 720, {720, 1280}: 720, {960, 960}: 720, {1920, 1080}: 1080, {1080, 1920}: 1080, {1440, 1440}: 1080, {3840, 2160}: 2160, {2160, 3840}: 2160, {2880, 2880}: 2160},
		"seedance-2.0-fast":        {{864, 496}: 480, {496, 864}: 480, {640, 640}: 480, {1280, 720}: 720, {720, 1280}: 720, {960, 960}: 720},
		"seedance-2.0-mini":        {{864, 496}: 480, {496, 864}: 480, {640, 640}: 480, {1280, 720}: 720, {720, 1280}: 720, {960, 960}: 720},
		"kling-2.1":                {{1920, 1080}: 1080, {1080, 1920}: 1080},
		"kling-2.5":                {{1280, 720}: 720, {720, 1280}: 720, {960, 960}: 720, {1920, 1080}: 1080, {1080, 1920}: 1080, {1440, 1440}: 1080},
		"kling-2.5-turbo-standard": {{1280, 720}: 720, {720, 1280}: 720, {960, 960}: 720},
		"kling-2.6":                {{1920, 1080}: 1080, {1080, 1920}: 1080, {1440, 1440}: 1080},
		"kling-3.0":                {{1280, 720}: 720, {720, 1280}: 720, {960, 960}: 720, {1920, 1080}: 1080, {1080, 1920}: 1080, {1440, 1440}: 1080, {3840, 2160}: 2160, {2160, 3840}: 2160, {2880, 2880}: 2160},
		"kling-3.0-turbo":          {{1280, 720}: 720, {720, 1280}: 720, {960, 960}: 720, {1920, 1080}: 1080, {1080, 1920}: 1080, {1440, 1440}: 1080},
		"kling-video-o-1":          {{1920, 1080}: 1080, {1080, 1920}: 1080, {1440, 1440}: 1080},
	}
	resolution, ok := dimensions[model][[2]int{width, height}]
	return resolution, ok
}

func LeonardoVideoSize(model, resolution, ratio string) (string, error) {
	sizes := map[string]map[string]map[string]string{
		"seedance-1.0-pro-fast":    {"480p": {"16:9": "864x480", "4:3": "736x544", "1:1": "640x640", "3:4": "544x736", "9:16": "480x864", "21:9": "960x416"}, "720p": {"16:9": "1248x704", "4:3": "1120x832", "1:1": "960x960", "3:4": "832x1120", "9:16": "704x1248", "21:9": "1504x640"}, "1080p": {"16:9": "1920x1088", "4:3": "1664x1248", "1:1": "1440x1440", "3:4": "1248x1664", "9:16": "1088x1920", "21:9": "2176x928"}},
		"seedance-1.0-pro":         {"480p": {"16:9": "864x480", "4:3": "736x544", "1:1": "640x640", "3:4": "544x736", "9:16": "480x864", "21:9": "960x416"}, "720p": {"16:9": "1248x704", "4:3": "1120x832", "1:1": "960x960", "3:4": "832x1120", "9:16": "704x1248", "21:9": "1504x640"}, "1080p": {"16:9": "1920x1088", "4:3": "1664x1248", "1:1": "1440x1440", "3:4": "1248x1664", "9:16": "1088x1920", "21:9": "2176x928"}},
		"motion_2.0-fast":          {"480p": {"16:9": "832x480", "9:16": "480x832", "2:3": "512x768", "4:5": "576x720"}, "720p": {"16:9": "1280x720", "9:16": "720x1152", "2:3": "768x1152", "4:5": "864x1024"}},
		"wan-2.7":                  {"720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}, "1080p": {"16:9": "1920x1080", "1:1": "1440x1440", "9:16": "1080x1920"}},
		"kling-video-o-3":          {"720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}, "1080p": {"16:9": "1920x1080", "1:1": "1440x1440", "9:16": "1080x1920"}, "2160p": {"16:9": "3840x2160", "1:1": "2880x2880", "9:16": "2160x3840"}},
		"seedance-2.0":             {"480p": {"16:9": "864x496", "1:1": "640x640", "9:16": "496x864"}, "720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}, "1080p": {"16:9": "1920x1080", "1:1": "1440x1440", "9:16": "1080x1920"}, "2160p": {"16:9": "3840x2160", "1:1": "2880x2880", "9:16": "2160x3840"}},
		"seedance-2.0-fast":        {"480p": {"16:9": "864x496", "1:1": "640x640", "9:16": "496x864"}, "720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}},
		"seedance-2.0-mini":        {"480p": {"16:9": "864x496", "1:1": "640x640", "9:16": "496x864"}, "720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}},
		"kling-2.1":                {"1080p": {"16:9": "1920x1080", "9:16": "1080x1920"}},
		"kling-2.5":                {"720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}, "1080p": {"16:9": "1920x1080", "1:1": "1440x1440", "9:16": "1080x1920"}},
		"kling-2.5-turbo-standard": {"720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}},
		"kling-2.6":                {"1080p": {"16:9": "1920x1080", "1:1": "1440x1440", "9:16": "1080x1920"}},
		"kling-3.0":                {"720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}, "1080p": {"16:9": "1920x1080", "1:1": "1440x1440", "9:16": "1080x1920"}, "2160p": {"16:9": "3840x2160", "1:1": "2880x2880", "9:16": "2160x3840"}},
		"kling-3.0-turbo":          {"720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}, "1080p": {"16:9": "1920x1080", "1:1": "1440x1440", "9:16": "1080x1920"}},
		"kling-video-o-1":          {"1080p": {"16:9": "1920x1080", "1:1": "1440x1440", "9:16": "1080x1920"}},
	}
	if size := sizes[model][resolution][ratio]; size != "" {
		return size, nil
	}
	return "", fmt.Errorf("%w: unsupported video size", ErrLeonardoVideoPricingEvidenceUnavailable)
}

func SupportsLeonardoVideoStartFrame(model string) bool {
	switch model {
	case "seedance-1.0-pro-fast", "seedance-1.0-pro", "wan-2.7", "motion_2.0-fast", "kling-video-o-3",
		"seedance-2.0", "seedance-2.0-fast", "seedance-2.0-mini",
		"kling-2.1", "kling-2.5", "kling-2.5-turbo-standard", "kling-2.6", "kling-3.0", "kling-3.0-turbo", "kling-video-o-1":
		return true
	default:
		return false
	}
}

// LeonardoVideoGenerationParameters builds the REST v2 `parameters` object for a
// video model, replicating the official Leonardo schema 1:1 per model.
// It returns the model's official model identifier via LeonardoVideoUpstreamModel
// only when the caller needs it; here it always builds with the public model slug
// (the v2 envelope `model` field is mapped separately by the orchestrator).
// sliceContains reports whether value is present in the int slice.
func sliceContains(value int, allowed []int) bool {
	for _, v := range allowed {
		if v == value {
			return true
		}
	}
	return false
}

// wanVideoResolution maps the official Wan 2.7 width/height enum to the matching
// `resolution` tier (validation is silent otherwise). Returns "" when no known
// combination matches, in which case the caller omits the resolution field.
func wanVideoResolution(width, height int) string {
	switch {
	case width == 1280 && height == 720,
		width == 960 && height == 960,
		width == 720 && height == 1280:
		return "720p"
	case width == 1920 && height == 1080,
		width == 1440 && height == 1440,
		width == 1080 && height == 1920:
		return "1080p"
	default:
		return ""
	}
}

// LeonardoVideoGenerationParameters delegates to the exact per-model REST v2
// rule used by both the customer OpenAI v1 API and Infinite Canvas multipart.
func LeonardoVideoGenerationParameters(model, prompt string, duration, width, height, quantity int) (map[string]any, error) {
	route, ok := LeonardoVideoRouteFor(model)
	if !ok || route.BuildParameters == nil {
		return nil, fmt.Errorf("%w: unsupported video model %s", ErrLeonardoVideoPricingEvidenceUnavailable, strings.TrimSpace(model))
	}
	return route.BuildParameters(prompt, duration, width, height, quantity)
}
