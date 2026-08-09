package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

var ErrLeonardoVideoPricingEvidenceUnavailable = errors.New("Leonardo video pricing evidence is unavailable")

const LeonardoVideoPricingPolicyVersion = "leonardo-video-pricing-policy/2026-08-10-v5"

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

func LeonardoVideoResolution(model string, width, height int) (int, bool) {
	dimensions := map[string]map[[2]int]int{
		"seedance-1.0-pro-fast": {{864, 480}: 480, {736, 544}: 480, {640, 640}: 480, {544, 736}: 480, {480, 864}: 480, {960, 416}: 480, {1248, 704}: 720, {1120, 832}: 720, {960, 960}: 720, {832, 1120}: 720, {704, 1248}: 720, {1504, 640}: 720, {1920, 1088}: 1080, {1664, 1248}: 1080, {1440, 1440}: 1080, {1248, 1664}: 1080, {1088, 1920}: 1080, {2176, 928}: 1080},
		"seedance-1.0-pro":      {{864, 480}: 480, {736, 544}: 480, {640, 640}: 480, {544, 736}: 480, {480, 864}: 480, {960, 416}: 480, {1248, 704}: 720, {1120, 832}: 720, {960, 960}: 720, {832, 1120}: 720, {704, 1248}: 720, {1504, 640}: 720, {1920, 1088}: 1080, {1664, 1248}: 1080, {1440, 1440}: 1080, {1248, 1664}: 1080, {1088, 1920}: 1080, {2176, 928}: 1080},
		"motion_2.0-fast":       {{832, 480}: 480, {480, 832}: 480, {512, 768}: 480, {576, 720}: 480, {1280, 720}: 720, {720, 1152}: 720, {768, 1152}: 720, {864, 1024}: 720},
		"wan-2.7":               {{1280, 720}: 720, {960, 960}: 720, {720, 1280}: 720, {1920, 1080}: 1080, {1440, 1440}: 1080, {1080, 1920}: 1080},
	}
	resolution, ok := dimensions[model][[2]int{width, height}]
	return resolution, ok
}

func LeonardoVideoSize(model, resolution, ratio string) (string, error) {
	sizes := map[string]map[string]map[string]string{
		"seedance-1.0-pro-fast": {"480p": {"16:9": "864x480", "4:3": "736x544", "1:1": "640x640", "3:4": "544x736", "9:16": "480x864", "21:9": "960x416"}, "720p": {"16:9": "1248x704", "4:3": "1120x832", "1:1": "960x960", "3:4": "832x1120", "9:16": "704x1248", "21:9": "1504x640"}, "1080p": {"16:9": "1920x1088", "4:3": "1664x1248", "1:1": "1440x1440", "3:4": "1248x1664", "9:16": "1088x1920", "21:9": "2176x928"}},
		"seedance-1.0-pro":      {"480p": {"16:9": "864x480", "4:3": "736x544", "1:1": "640x640", "3:4": "544x736", "9:16": "480x864", "21:9": "960x416"}, "720p": {"16:9": "1248x704", "4:3": "1120x832", "1:1": "960x960", "3:4": "832x1120", "9:16": "704x1248", "21:9": "1504x640"}, "1080p": {"16:9": "1920x1088", "4:3": "1664x1248", "1:1": "1440x1440", "3:4": "1248x1664", "9:16": "1088x1920", "21:9": "2176x928"}},
		"motion_2.0-fast":       {"480p": {"16:9": "832x480", "9:16": "480x832", "2:3": "512x768", "4:5": "576x720"}, "720p": {"16:9": "1280x720", "9:16": "720x1152", "2:3": "768x1152", "4:5": "864x1024"}},
		"wan-2.7":               {"720p": {"16:9": "1280x720", "1:1": "960x960", "9:16": "720x1280"}, "1080p": {"16:9": "1920x1080", "1:1": "1440x1440", "9:16": "1080x1920"}},
	}
	if size := sizes[model][resolution][ratio]; size != "" {
		return size, nil
	}
	return "", fmt.Errorf("%w: unsupported video size", ErrLeonardoVideoPricingEvidenceUnavailable)
}

func LeonardoVideoGenerationParameters(model, prompt string, duration, width, height, quantity int) (map[string]any, error) {
	if _, err := NewLeonardoVideoPriceResolver().Estimate(context.Background(), LeonardoVideoPriceRequest{Model: model, Duration: duration, Width: width, Height: height, Quantity: quantity}); err != nil {
		return nil, err
	}
	parameters := map[string]any{"prompt": prompt, "width": width, "height": height, "quantity": quantity}
	resolution, _ := LeonardoVideoResolution(model, width, height)
	switch model {
	case "motion_2.0-fast":
		parameters["mode"] = fmt.Sprintf("RESOLUTION_%d", resolution)
	case "wan-2.7":
		parameters["duration"] = duration
		parameters["resolution"] = fmt.Sprintf("%dp", resolution)
	default:
		parameters["duration"] = duration
		parameters["mode"] = fmt.Sprintf("RESOLUTION_%d", resolution)
		parameters["prompt_enhance"] = "OFF"
	}
	return parameters, nil
}
