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
	leonardoImagePricingVersion = "2026-08-01"
	leonardoImagePricingSource  = "leonardo_authenticated_pricing_calculator"
	leonardoImagePricingMatch   = "exact"
)

type LeonardoImagePriceRequest struct {
	Model    string
	Width    int
	Height   int
	Quantity int
	Public   bool
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
	if model == "" || request.Width <= 0 || request.Height <= 0 || request.Quantity <= 0 {
		return nil, ErrLeonardoImagePricingRequestInvalid
	}
	if model != "flux-schnell" || request.Width != 896 || request.Height != 896 || request.Quantity != 1 || request.Public {
		return nil, ErrLeonardoImagePricingNotFound
	}
	unitCost := decimal.RequireFromString("0.003")
	return &LeonardoImagePriceEstimate{
		UnitCostUSD:      unitCost,
		Quantity:         request.Quantity,
		EstimatedCostUSD: unitCost.Mul(decimal.NewFromInt(int64(request.Quantity))),
		PricingVersion:   leonardoImagePricingVersion,
		PricingSource:    leonardoImagePricingSource,
		MatchType:        leonardoImagePricingMatch,
	}, nil
}
