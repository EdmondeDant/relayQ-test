package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrLeonardoManualReviewInvalid = errors.New("invalid Leonardo manual review operation")

type LeonardoManualReviewService struct {
	repository GenerationJobRepository
	funds      LeonardoImageTerminalFunds
}

func NewLeonardoManualReviewService(repository GenerationJobRepository, funds LeonardoImageTerminalFunds) *LeonardoManualReviewService {
	return &LeonardoManualReviewService{repository: repository, funds: funds}
}

func (s *LeonardoManualReviewService) Get(ctx context.Context, publicID string) (*GenerationJob, error) {
	if s == nil || s.repository == nil || strings.TrimSpace(publicID) == "" {
		return nil, ErrLeonardoManualReviewInvalid
	}
	job, err := s.repository.GetByPublicID(ctx, strings.TrimSpace(publicID))
	if err != nil {
		return nil, err
	}
	if job.Provider != PlatformLeonardo || job.BillingStatus != GenerationJobBillingStatusManualReview {
		return nil, ErrLeonardoManualReviewInvalid
	}
	return job, nil
}

func (s *LeonardoManualReviewService) AttachUpstreamID(ctx context.Context, publicID, upstreamID string) (*GenerationJob, error) {
	if !validLeonardoGenerationUUID(strings.TrimSpace(upstreamID)) {
		return nil, ErrLeonardoManualReviewInvalid
	}
	job, err := s.Get(ctx, publicID)
	if err != nil || job.Status != GenerationJobStatusUnknown || job.UpstreamGenerationID != nil {
		return nil, errors.Join(ErrLeonardoManualReviewInvalid, err)
	}
	if existing, lookupErr := s.repository.GetByUpstreamGenerationID(ctx, strings.TrimSpace(upstreamID)); lookupErr == nil && existing != nil {
		return nil, ErrGenerationJobConflict
	} else if lookupErr != nil && !errors.Is(lookupErr, ErrGenerationJobNotFound) {
		return nil, lookupErr
	}
	updated := *job
	updated.UpstreamGenerationID = stringPointer(strings.TrimSpace(upstreamID))
	updated.NextPollAt = timePointerValue(time.Now().UTC())
	if err = s.repository.CompareAndSwapStatus(ctx, job.PublicID, job.Status, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *LeonardoManualReviewService) Refund(ctx context.Context, publicID, reason string) (*GenerationJob, error) {
	job, err := s.Get(ctx, publicID)
	if err != nil || s.funds == nil || job.Status != GenerationJobStatusUnknown || job.CustomerCost == nil || job.BillingReference == nil || strings.TrimSpace(reason) == "" {
		return nil, errors.Join(ErrLeonardoManualReviewInvalid, err)
	}
	if err = s.funds.Release(ctx, LeonardoImageFundsReleaseRequest{UserID: job.UserID, PublicID: job.PublicID, Reference: strings.TrimSpace(*job.BillingReference), AmountUSD: *job.CustomerCost, Reason: strings.TrimSpace(reason)}); err != nil {
		return nil, err
	}
	updated := *job
	updated.Status = GenerationJobStatusFailed
	updated.BillingStatus = GenerationJobBillingStatusRefunded
	updated.ErrorCode = stringPointer("manual_refund")
	updated.ErrorMessage = stringPointer("Leonardo submission was manually refunded")
	updated.FailedAt = timePointerValue(time.Now().UTC())
	if err = s.repository.CompareAndSwapStatus(ctx, job.PublicID, job.Status, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}
