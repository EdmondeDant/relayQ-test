package service

import (
	"context"
	"errors"
	"strings"
)

var ErrLeonardoAudioPricingEvidenceUnavailable = errors.New("leonardo audio pricing evidence is unavailable")

const LeonardoAudioPricingPolicyVersion = "leonardo-audio-pricing-policy/2026-08-05-v1"

type LeonardoAudioPriceRequest struct {
	Model           string
	DurationSeconds int
	Quantity        int
}

type LeonardoAudioPriceResolver struct{}

func NewLeonardoAudioPriceResolver() LeonardoAudioPriceResolver {
	return LeonardoAudioPriceResolver{}
}

func (LeonardoAudioPriceResolver) Estimate(ctx context.Context, request LeonardoAudioPriceRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(request.Model) == "" || request.DurationSeconds <= 0 || request.Quantity <= 0 {
		return ErrLeonardoAudioPricingEvidenceUnavailable
	}
	return ErrLeonardoAudioPricingEvidenceUnavailable
}
