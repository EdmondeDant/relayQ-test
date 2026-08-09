package service

import (
	"context"
	"errors"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	ErrLeonardoImagePricingNotFound       = errors.New("leonardo image pricing not found")
	ErrLeonardoImagePricingRequestInvalid = errors.New("leonardo image pricing request is invalid")
)

const (
	leonardoImagePricingVersion = "2026-08-08"
	leonardoImagePricingSource  = "leonardo_authenticated_pricing_calculator"
	leonardoImagePricingMatch   = "exact"
	leonardoImagePricingTierMax = "quality_tier_max"
)

type LeonardoImagePriceRequest struct {
	Model       string
	Width       int
	Height      int
	Quantity    int
	Public      bool
	QualityTier string
}

type LeonardoImagePriceEstimate struct {
	UnitCostUSD      decimal.Decimal
	Quantity         int
	EstimatedCostUSD decimal.Decimal
	PricingVersion   string
	PricingSource    string
	MatchType        string
}

type LeonardoImagePriceResolver interface {
	Estimate(context.Context, LeonardoImagePriceRequest) (*LeonardoImagePriceEstimate, error)
}

type leonardoImagePriceResolver struct{}

func NewLeonardoImagePriceResolver() LeonardoImagePriceResolver {
	return leonardoImagePriceResolver{}
}

func (leonardoImagePriceResolver) Estimate(ctx context.Context, request LeonardoImagePriceRequest) (*LeonardoImagePriceEstimate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	model := strings.TrimSpace(request.Model)
	matchReferenceSize := (model == "nano-banana-2" || model == "nano-banana-2-lite") && request.Width == 0 && request.Height == 0
	if model == "" || (!matchReferenceSize && (request.Width <= 0 || request.Height <= 0)) || request.Quantity <= 0 {
		return nil, ErrLeonardoImagePricingRequestInvalid
	}
	if request.Public || request.Quantity > 8 {
		return nil, ErrLeonardoImagePricingNotFound
	}
	tier := strings.TrimSpace(request.QualityTier)
	if tier == "" {
		tier = "low"
	}
	if tier != "low" && tier != "medium" && tier != "high" {
		return nil, ErrLeonardoImagePricingRequestInvalid
	}
	price, match, ok := leonardoImageUnitPrice(model, request.Width, request.Height, tier)
	if !ok {
		return nil, ErrLeonardoImagePricingNotFound
	}
	unitCost := decimal.RequireFromString(price)
	return &LeonardoImagePriceEstimate{
		UnitCostUSD:      unitCost,
		Quantity:         request.Quantity,
		EstimatedCostUSD: unitCost.Mul(decimal.NewFromInt(int64(request.Quantity))),
		PricingVersion:   leonardoImagePricingVersion,
		PricingSource:    leonardoImagePricingSource,
		MatchType:        match,
	}, nil
}

func leonardoImageUnitPrice(model string, width, height int, quality string) (string, string, bool) {
	if width == 0 && height == 0 {
		if model == "nano-banana-2" && quality == "low" {
			return "0.0389", leonardoImagePricingMatch, true
		}
		if model == "nano-banana-2-lite" && quality == "low" {
			return "0.0449", leonardoImagePricingMatch, true
		}
	}
	if width != height {
		return "", "", false
	}
	size := map[int]int{1024: 0, 2048: 1, 2880: 2}[width]
	switch model {
	case "flux-schnell":
		if width > 2048 {
			return "", "", false
		}
		if price, ok := map[int]string{896: "0.003", 1024: "0.003", 1120: "0.0045"}[width]; ok {
			return price, leonardoImagePricingMatch, true
		}
		return "0.0045", leonardoImagePricingTierMax, true
	case "gpt-image-2":
		prices := map[string][3]string{"low": {"0.012", "0.0254", "0.0762"}, "medium": {"0.0987", "0.2153", "0.6683"}, "high": {"0.3902", "0.8566", "2.6596"}}
		values, ok := prices[quality]
		return values[size], leonardoImagePricingMatch, ok && (width == 1024 || width == 2048 || width == 2880)
	case "nano-banana-2":
		if quality != "low" {
			return "", "", false
		}
		values := [3]string{"0.0389", "0.0583", "0.0777"}
		return values[size], leonardoImagePricingMatch, width == 1024 || width == 2048
	case "nano-banana-2-lite":
		if quality == "low" && width == 1024 {
			return "0.0449", leonardoImagePricingMatch, true
		}
	case "kino-xl":
		prices := map[string]map[int]string{"low": {896: "0.0045", 1024: "0.006", 1120: "0.006"}, "high": {896: "0.0239", 1024: "0.0239", 1120: "0.0269"}}
		price, ok := prices[quality][width]
		return price, leonardoImagePricingMatch, ok
	case "concept-art", "graphic-design", "illustrative-albedo":
		prices := map[string]map[int]string{"low": {888: "0.0045", 960: "0.0045", 1024: "0.006"}, "high": {888: "0.0239", 960: "0.0239", 1024: "0.0239"}}
		price, ok := prices[quality][width]
		return price, leonardoImagePricingMatch, ok
	}
	return "", "", false
}
