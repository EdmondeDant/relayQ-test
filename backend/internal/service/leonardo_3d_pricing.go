package service

import (
	"context"
	"errors"
	"strings"
)

var ErrLeonardo3DPricingEvidenceUnavailable = errors.New("Leonardo 3d pricing evidence is unavailable")

const Leonardo3DPricingPolicyVersion = "leonardo-3d-pricing-policy/2026-08-05-v1"

type Leonardo3DPriceRequest struct {
	Model    string
	Quantity int
}

type Leonardo3DPriceResolver struct{}

func NewLeonardo3DPriceResolver() Leonardo3DPriceResolver {
	return Leonardo3DPriceResolver{}
}

func (Leonardo3DPriceResolver) Estimate(ctx context.Context, request Leonardo3DPriceRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(request.Model) == "" || request.Quantity <= 0 {
		return ErrLeonardo3DPricingEvidenceUnavailable
	}
	return ErrLeonardo3DPricingEvidenceUnavailable
}
