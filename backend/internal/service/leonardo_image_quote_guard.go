package service

import (
	"context"
	"errors"
	"strings"

	"github.com/shopspring/decimal"
)

var (
	ErrLeonardoImageQuoteInvalid            = errors.New("leonardo image quote is invalid")
	ErrLeonardoImageQuoteBelowEstimatedCost = errors.New("leonardo image quote is below estimated upstream cost")
)

type LeonardoImageQuoteBalanceReader interface {
	GetAvailableBalanceUSD(ctx context.Context, userID int64) (decimal.Decimal, error)
}

type LeonardoImageQuoteRequest struct {
	UserID           int64
	PricingRequest   LeonardoImagePriceRequest
	CustomerQuoteUSD decimal.Decimal
}

type LeonardoImageQuote struct {
	EstimatedUpstreamCostUSD decimal.Decimal
	CustomerQuoteUSD         decimal.Decimal
	PricingVersion           string
	PricingSource            string
	MatchType                string
}

type LeonardoImageQuoteGuard struct {
	priceResolver LeonardoImagePriceResolver
	balanceReader LeonardoImageQuoteBalanceReader
}

func NewLeonardoImageQuoteGuard(priceResolver LeonardoImagePriceResolver, balanceReader LeonardoImageQuoteBalanceReader) *LeonardoImageQuoteGuard {
	return &LeonardoImageQuoteGuard{priceResolver: priceResolver, balanceReader: balanceReader}
}

func (g *LeonardoImageQuoteGuard) Prepare(ctx context.Context, request LeonardoImageQuoteRequest) (*LeonardoImageQuote, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if g == nil || g.priceResolver == nil || g.balanceReader == nil || request.UserID <= 0 || request.CustomerQuoteUSD.Sign() <= 0 {
		return nil, ErrLeonardoImageQuoteInvalid
	}
	estimate, err := g.priceResolver.Estimate(ctx, request.PricingRequest)
	if err != nil {
		return nil, err
	}
	if estimate == nil || estimate.EstimatedCostUSD.Sign() <= 0 || strings.TrimSpace(estimate.PricingVersion) == "" || strings.TrimSpace(estimate.PricingSource) == "" || strings.TrimSpace(estimate.MatchType) == "" {
		return nil, ErrLeonardoImageQuoteInvalid
	}
	if request.CustomerQuoteUSD.Cmp(estimate.EstimatedCostUSD) < 0 {
		return nil, ErrLeonardoImageQuoteBelowEstimatedCost
	}
	balance, err := g.balanceReader.GetAvailableBalanceUSD(ctx, request.UserID)
	if err != nil {
		return nil, err
	}
	if balance.Cmp(request.CustomerQuoteUSD) < 0 {
		return nil, ErrInsufficientBalance
	}
	return &LeonardoImageQuote{
		EstimatedUpstreamCostUSD: estimate.EstimatedCostUSD,
		CustomerQuoteUSD:         request.CustomerQuoteUSD,
		PricingVersion:           estimate.PricingVersion,
		PricingSource:            estimate.PricingSource,
		MatchType:                estimate.MatchType,
	}, nil
}
