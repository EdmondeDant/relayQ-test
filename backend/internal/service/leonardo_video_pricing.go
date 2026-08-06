package service

import (
	"context"
	"errors"
	"strings"

	"github.com/shopspring/decimal"
)

var ErrLeonardoVideoPricingEvidenceUnavailable = errors.New("Leonardo video pricing evidence is unavailable")

const LeonardoVideoPricingPolicyVersion = "leonardo-video-pricing-policy/2026-08-06-v2"

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
	if strings.TrimSpace(request.Model) == "" || request.Duration <= 0 || request.Width <= 0 || request.Height <= 0 || request.Quantity <= 0 {
		return nil, ErrLeonardoVideoPricingEvidenceUnavailable
	}
	tier := strings.TrimSpace(request.QualityTier)
	if tier == "" {
		tier = "low"
	}
	maxima := map[string]string{"low": "7.176", "medium": "6.7813", "high": "42.6972"}
	price, ok := maxima[tier]
	if !ok {
		return nil, ErrLeonardoVideoPricingEvidenceUnavailable
	}
	return &LeonardoVideoPriceEstimate{
		EstimatedCostUSD: decimal.RequireFromString(price).Mul(decimal.NewFromInt(int64(request.Quantity))),
		PricingVersion:   LeonardoVideoPricingPolicyVersion,
		PricingSource:    "leonardo_authenticated_pricing_calculator",
		MatchType:        "quality_tier_max",
	}, nil
}
