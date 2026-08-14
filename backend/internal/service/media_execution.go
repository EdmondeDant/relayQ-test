package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type MediaSubmissionState string

const (
	MediaSubmissionNotWritten        MediaSubmissionState = "not_written"
	MediaSubmissionSubmitted         MediaSubmissionState = "submitted"
	MediaSubmissionSideEffectUnknown MediaSubmissionState = "side_effect_unknown"
)

var (
	ErrMediaAttemptInvalid      = errors.New("invalid media job attempt")
	ErrMediaReservationInvalid  = errors.New("invalid media funds reservation")
	ErrMediaReservationConflict = errors.New("media funds reservation conflict")
)

type MediaJobAttempt struct {
	ID                  int64
	JobID               int64
	OfferID             int64
	Provider            string
	SourceGroupID       int64
	AccountID           *int64
	UpstreamModel       string
	TrustedCostSnapshot map[string]any
	SubmissionState     MediaSubmissionState
	ErrorCode           *string
	ErrorMessage        *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type MediaJobAttemptRepository interface {
	Create(ctx context.Context, attempt *MediaJobAttempt) error
	ListByJobID(ctx context.Context, jobID int64) ([]MediaJobAttempt, error)
}

type MediaFundsReservation struct {
	Reference     string
	UserID        int64
	PublicID      string
	ProductID     int64
	Amount        decimal.Decimal
	PriceVersion  string
	Status        string
	AlreadyExists bool
}

type MediaFundsReserveRequest struct {
	UserID       int64
	PublicID     string
	ProductID    int64
	Amount       decimal.Decimal
	PriceVersion string
}

type MediaFundsTransitionRequest struct {
	UserID    int64
	PublicID  string
	Reference string
	Amount    decimal.Decimal
}

type MediaFundsRepository interface {
	Reserve(ctx context.Context, request MediaFundsReserveRequest) (*MediaFundsReservation, error)
	Settle(ctx context.Context, request MediaFundsTransitionRequest) error
	Release(ctx context.Context, request MediaFundsTransitionRequest) error
}

type MediaFundsService struct{ repo MediaFundsRepository }

func NewMediaFundsService(repo MediaFundsRepository) *MediaFundsService {
	return &MediaFundsService{repo: repo}
}

func (s *MediaFundsService) Reserve(ctx context.Context, request MediaFundsReserveRequest) (*MediaFundsReservation, error) {
	request.PublicID = strings.TrimSpace(request.PublicID)
	request.PriceVersion = strings.TrimSpace(request.PriceVersion)
	if s == nil || s.repo == nil || request.UserID <= 0 || request.ProductID <= 0 || request.PublicID == "" || request.PriceVersion == "" || !validMediaFundsAmount(request.Amount) {
		return nil, ErrMediaReservationInvalid
	}
	return s.repo.Reserve(ctx, request)
}

func (s *MediaFundsService) Settle(ctx context.Context, request MediaFundsTransitionRequest) error {
	if !validMediaFundsTransition(s, &request) {
		return ErrMediaReservationInvalid
	}
	return s.repo.Settle(ctx, request)
}

func (s *MediaFundsService) Release(ctx context.Context, request MediaFundsTransitionRequest) error {
	if !validMediaFundsTransition(s, &request) {
		return ErrMediaReservationInvalid
	}
	return s.repo.Release(ctx, request)
}

func validMediaFundsTransition(service *MediaFundsService, request *MediaFundsTransitionRequest) bool {
	request.PublicID = strings.TrimSpace(request.PublicID)
	request.Reference = strings.TrimSpace(request.Reference)
	return service != nil && service.repo != nil && request.UserID > 0 && request.PublicID != "" && request.Reference != "" && validMediaFundsAmount(request.Amount)
}

func validMediaFundsAmount(amount decimal.Decimal) bool {
	return amount.IsPositive() && amount.Equal(amount.Truncate(10))
}
