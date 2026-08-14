package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type MediaJobStatus string
type MediaAttemptStatus string
type MediaReservationState string

const (
	MediaJobAccepted         MediaJobStatus = "accepted"
	MediaJobReserved         MediaJobStatus = "reserved"
	MediaJobAttempting       MediaJobStatus = "attempting"
	MediaJobSucceeded        MediaJobStatus = "succeeded"
	MediaJobFailed           MediaJobStatus = "failed"
	MediaJobPartialSucceeded MediaJobStatus = "partial_succeeded"
	MediaJobReleased         MediaJobStatus = "released"
	MediaJobSettled          MediaJobStatus = "settled"

	MediaAttemptPlanned         MediaAttemptStatus = "planned"
	MediaAttemptRunning         MediaAttemptStatus = "running"
	MediaAttemptSucceeded       MediaAttemptStatus = "succeeded"
	MediaAttemptFailedRetryable MediaAttemptStatus = "failed_retryable"
	MediaAttemptFailedTerminal  MediaAttemptStatus = "failed_terminal"
	MediaAttemptSkipped         MediaAttemptStatus = "skipped"

	MediaReservationNone          MediaReservationState = "none"
	MediaReservationHeld          MediaReservationState = "held"
	MediaReservationCaptured      MediaReservationState = "captured"
	MediaReservationReleased      MediaReservationState = "released"
	MediaReservationCaptureFailed MediaReservationState = "capture_failed"
)

var (
	ErrMediaJobInvalidTransition = errors.New("invalid media job transition")
	ErrMediaJobNoCandidate       = errors.New("media job has no offer candidate")
	ErrMediaJobOutputExists      = errors.New("media job already has billable output")
	ErrMediaAttemptNotFound      = errors.New("media attempt not found")
)

type MediaJobInput struct {
	JobID              string
	RequestID          string
	IdempotencyKeyHash string
	RequestFingerprint string
	APIKeyID           int64
	UserID             int64
	CustomerGroupID    *int64
	ProductID          int64
	PublicModel        string
	Modality           MediaModality
	Op                 string
	CreatedAt          time.Time
}

type MediaJob struct {
	JobID              string
	RequestID          string
	IdempotencyKeyHash string
	RequestFingerprint string
	APIKeyID           int64
	UserID             int64
	CustomerGroupID    *int64
	ProductID          int64
	PublicModel        string
	Modality           MediaModality
	Op                 string
	Reservation        MediaReservation
	Attempts           []MediaAttempt
	SelectedOfferID    int64
	Status             MediaJobStatus
	Audit              MediaAuditTrail
	CreatedAt          time.Time
	UpdatedAt          time.Time
	candidates         []MediaOfferCandidate
	nextCandidate      int
}

type MediaAttempt struct {
	AttemptNo          int
	OfferID            int64
	Provider           string
	SourceGroupID      int64
	AccountID          *int64
	UpstreamModel      string
	TrustedCostSnap    float64
	CustomerChargeSnap float64
	Status             MediaAttemptStatus
	UpstreamRef        string
	ErrorClass         string
	ErrorCode          string
	BillableOutput     int
	StartedAt          *time.Time
	EndedAt            *time.Time
}

type AttemptEndInput struct {
	Status         MediaAttemptStatus
	UpstreamRef    string
	ErrorClass     string
	ErrorCode      string
	BillableOutput int
}

type MediaReservation struct {
	State    MediaReservationState
	Amount   float64
	Currency string
	Basis    string
	Quantity int
	HoldKey  string
	DedupKey string
}

type MediaAuditEvent struct {
	At      time.Time
	Type    string
	OfferID int64
	Detail  map[string]any
}

type MediaAuditTrail struct {
	Events        []MediaAuditEvent
	SelectSummary map[string]any
	MoneySummary  map[string]any
}

type UsageLogDraft struct {
	RequestID            string
	APIKeyID             int64
	UserID               int64
	CustomerGroupID      *int64
	AccountID            *int64
	RequestedModel       string
	UpstreamModel        string
	MediaType            string
	ImageCount           int
	ActualCost           float64
	ProductID            int64
	OfferID              int64
	UpstreamPlatform     string
	SourceGroupID        int64
	TrustedCost          float64
	TrustedCostUnit      string
	TrustedCostSource    string
	TrustedCostVersion   string
	CustomerPriceVersion string
}

type MediaUsageAuditRepository interface {
	CreateMediaUsageAudit(context.Context, *UsageLog) (bool, error)
}

type MediaUsageAuditService struct{ repo MediaUsageAuditRepository }

func NewMediaUsageAuditService(repo MediaUsageAuditRepository) *MediaUsageAuditService {
	return &MediaUsageAuditService{repo: repo}
}

func (s *MediaUsageAuditService) Write(ctx context.Context, draft UsageLogDraft) (bool, error) {
	if s == nil || s.repo == nil || draft.RequestID == "" || draft.APIKeyID <= 0 || draft.UserID <= 0 || draft.AccountID == nil || draft.ProductID <= 0 || draft.OfferID <= 0 || draft.SourceGroupID <= 0 || draft.ActualCost <= 0 || draft.TrustedCost <= 0 {
		return false, errors.New("invalid media usage audit")
	}
	mediaProductID, mediaOfferID, sourceGroupID := draft.ProductID, draft.OfferID, draft.SourceGroupID
	upstreamPlatform, trustedUnit, trustedSource, trustedVersion, priceVersion := draft.UpstreamPlatform, draft.TrustedCostUnit, draft.TrustedCostSource, draft.TrustedCostVersion, draft.CustomerPriceVersion
	upstreamModel, mediaType, billingMode := draft.UpstreamModel, draft.MediaType, string(BillingModeImage)
	if mediaType == string(MediaModalityVideo) {
		billingMode = string(BillingModePerRequest)
	}
	log := &UsageLog{UserID: draft.UserID, APIKeyID: draft.APIKeyID, AccountID: *draft.AccountID, RequestID: draft.RequestID, Model: draft.RequestedModel, RequestedModel: draft.RequestedModel, UpstreamModel: &upstreamModel, GroupID: draft.CustomerGroupID, ActualCost: draft.ActualCost, TotalCost: draft.ActualCost, ImageCount: draft.ImageCount, MediaType: &mediaType, BillingMode: &billingMode, MediaProductID: &mediaProductID, MediaOfferID: &mediaOfferID, UpstreamPlatform: &upstreamPlatform, SourceGroupID: &sourceGroupID, TrustedCostAmount: &draft.TrustedCost, TrustedCostUnit: &trustedUnit, TrustedCostSource: &trustedSource, TrustedCostVersion: &trustedVersion, CustomerPriceVersion: &priceVersion}
	return s.repo.CreateMediaUsageAudit(ctx, log)
}

func NewMediaJob(input MediaJobInput) MediaJob {
	now := input.CreatedAt
	if now.IsZero() {
		now = time.Now()
	}
	return MediaJob{
		JobID: input.JobID, RequestID: input.RequestID, IdempotencyKeyHash: input.IdempotencyKeyHash,
		RequestFingerprint: input.RequestFingerprint, APIKeyID: input.APIKeyID, UserID: input.UserID,
		CustomerGroupID: input.CustomerGroupID, ProductID: input.ProductID, PublicModel: input.PublicModel,
		Modality: input.Modality, Op: input.Op, Status: MediaJobAccepted, CreatedAt: now, UpdatedAt: now,
		Reservation: MediaReservation{State: MediaReservationNone},
		Audit:       MediaAuditTrail{SelectSummary: map[string]any{}, MoneySummary: map[string]any{}},
	}
}

func (j *MediaJob) ApplySelect(result MediaSelectResult, now time.Time) error {
	if j.Status != MediaJobAccepted || result.Err != nil || result.Selected == nil {
		if result.Err != nil {
			return result.Err
		}
		return ErrMediaJobInvalidTransition
	}
	j.candidates = append([]MediaOfferCandidate(nil), result.RankedEligible...)
	j.nextCandidate = 0
	for _, skipped := range result.Skipped {
		j.appendAudit(now, "skip_offer", skipped.Offer.ID, map[string]any{"reason": skipped.SkipReason, "detail": skipped.SkipDetail})
	}
	j.Audit.SelectSummary = map[string]any{
		"eligible_count": len(result.RankedEligible), "skipped_count": len(result.Skipped),
		"selected_offer_id": result.Selected.Offer.ID, "trusted_cost": result.Selected.TrustedCost,
	}
	j.appendAudit(now, "select", result.Selected.Offer.ID, j.Audit.SelectSummary)
	j.Reservation = MediaReservation{
		State: MediaReservationNone, Amount: result.Selected.CustomerCharge,
		Currency: result.Product.CustomerPrice.Currency, Basis: result.Product.CustomerPrice.Basis,
		Quantity: result.Selected.Quantity,
		HoldKey:  fmt.Sprintf("media_hold:%s:%d", j.RequestID, j.APIKeyID),
		DedupKey: fmt.Sprintf("%s:%d", j.RequestID, j.APIKeyID),
	}
	j.UpdatedAt = now
	return nil
}

func (j *MediaJob) ReserveCustomerCharge(now time.Time) error {
	if j.Status != MediaJobAccepted || len(j.candidates) == 0 || j.Reservation.State != MediaReservationNone || j.Reservation.Amount <= 0 {
		return ErrMediaJobInvalidTransition
	}
	j.Reservation.State = MediaReservationHeld
	j.Status = MediaJobReserved
	j.Audit.MoneySummary = map[string]any{"reserved": j.Reservation.Amount, "captured": float64(0), "released": float64(0), "customer_charge": j.Reservation.Amount}
	j.appendAudit(now, "reserve", 0, map[string]any{"amount": j.Reservation.Amount, "currency": j.Reservation.Currency})
	j.UpdatedAt = now
	return nil
}

func (j *MediaJob) BeginAttempt(candidate MediaOfferCandidate, now time.Time) (*MediaAttempt, error) {
	if j.Reservation.State != MediaReservationHeld || j.Status != MediaJobReserved && j.Status != MediaJobAttempting || j.TotalBillableOutput() > 0 {
		return nil, ErrMediaJobInvalidTransition
	}
	if len(j.Attempts) > 0 && j.Attempts[len(j.Attempts)-1].Status != MediaAttemptFailedRetryable {
		return nil, ErrMediaJobInvalidTransition
	}
	attempt := MediaAttempt{
		AttemptNo: len(j.Attempts) + 1, OfferID: candidate.Offer.ID, Provider: candidate.Offer.Provider,
		SourceGroupID: candidate.Offer.SourceGroupID, AccountID: candidate.Offer.SourceAccountID,
		UpstreamModel: candidate.Offer.UpstreamModel, TrustedCostSnap: candidate.TrustedCost,
		CustomerChargeSnap: j.Reservation.Amount, Status: MediaAttemptRunning, StartedAt: &now,
	}
	j.Attempts = append(j.Attempts, attempt)
	j.Status = MediaJobAttempting
	j.appendAudit(now, "attempt_start", candidate.Offer.ID, map[string]any{"attempt_no": attempt.AttemptNo, "trusted_cost": attempt.TrustedCostSnap})
	j.UpdatedAt = now
	return &j.Attempts[len(j.Attempts)-1], nil
}

func (j *MediaJob) BeginNextAttempt(now time.Time) (*MediaAttempt, error) {
	if j.nextCandidate >= len(j.candidates) {
		return nil, ErrMediaJobNoCandidate
	}
	candidate := j.candidates[j.nextCandidate]
	j.nextCandidate++
	return j.BeginAttempt(candidate, now)
}

func (j *MediaJob) EndAttempt(attemptNo int, end AttemptEndInput, now time.Time) error {
	if attemptNo <= 0 || attemptNo > len(j.Attempts) {
		return ErrMediaAttemptNotFound
	}
	attempt := &j.Attempts[attemptNo-1]
	if attempt.Status != MediaAttemptRunning || end.BillableOutput < 0 {
		return ErrMediaJobInvalidTransition
	}
	if end.Status != MediaAttemptSucceeded && end.Status != MediaAttemptFailedRetryable && end.Status != MediaAttemptFailedTerminal {
		return ErrMediaJobInvalidTransition
	}
	attempt.Status, attempt.UpstreamRef = end.Status, end.UpstreamRef
	attempt.ErrorClass, attempt.ErrorCode, attempt.BillableOutput = end.ErrorClass, end.ErrorCode, end.BillableOutput
	attempt.EndedAt = &now
	if end.Status == MediaAttemptSucceeded {
		j.SelectedOfferID = attempt.OfferID
		if end.BillableOutput >= j.Reservation.Quantity {
			j.Status = MediaJobSucceeded
		} else {
			j.Status = MediaJobPartialSucceeded
		}
	} else if end.Status == MediaAttemptFailedTerminal || end.BillableOutput > 0 {
		if end.BillableOutput > 0 {
			j.SelectedOfferID = attempt.OfferID
			j.Status = MediaJobPartialSucceeded
		} else {
			j.Status = MediaJobFailed
		}
	} else if j.nextCandidate >= len(j.candidates) {
		j.Status = MediaJobFailed
	}
	j.appendAudit(now, "attempt_end", attempt.OfferID, map[string]any{"attempt_no": attemptNo, "status": end.Status, "billable_output": end.BillableOutput, "error_class": end.ErrorClass})
	j.UpdatedAt = now
	return nil
}

func (j *MediaJob) FailoverToNext(now time.Time) (*MediaAttempt, error) {
	if j.TotalBillableOutput() > 0 {
		return nil, ErrMediaJobOutputExists
	}
	if len(j.Attempts) == 0 || j.Attempts[len(j.Attempts)-1].Status != MediaAttemptFailedRetryable || j.Status != MediaJobAttempting {
		return nil, ErrMediaJobInvalidTransition
	}
	if j.nextCandidate >= len(j.candidates) {
		j.Status = MediaJobFailed
		return nil, ErrMediaJobNoCandidate
	}
	next := j.candidates[j.nextCandidate]
	j.appendAudit(now, "failover", next.Offer.ID, map[string]any{"from_offer_id": j.Attempts[len(j.Attempts)-1].OfferID})
	return j.BeginNextAttempt(now)
}

func (j *MediaJob) SettleOrRelease(now time.Time) error {
	switch j.Status {
	case MediaJobSucceeded, MediaJobPartialSucceeded:
		if j.Reservation.State != MediaReservationHeld || j.TotalBillableOutput() == 0 {
			return ErrMediaJobInvalidTransition
		}
		j.Reservation.State = MediaReservationCaptured
		j.Status = MediaJobSettled
		j.Audit.MoneySummary["captured"] = j.Reservation.Amount
		j.appendAudit(now, "settle", j.SelectedOfferID, map[string]any{"amount": j.Reservation.Amount})
	case MediaJobFailed:
		if j.Reservation.State != MediaReservationHeld || j.TotalBillableOutput() != 0 {
			return ErrMediaJobInvalidTransition
		}
		j.Reservation.State = MediaReservationReleased
		j.Status = MediaJobReleased
		j.Audit.MoneySummary["released"] = j.Reservation.Amount
		j.appendAudit(now, "release", 0, map[string]any{"amount": j.Reservation.Amount})
	default:
		return ErrMediaJobInvalidTransition
	}
	j.UpdatedAt = now
	return nil
}

func (j MediaJob) TotalBillableOutput() int {
	total := 0
	for _, attempt := range j.Attempts {
		total += attempt.BillableOutput
	}
	return total
}

func (j MediaJob) AuditJSON() map[string]any {
	return map[string]any{
		"job_id": j.JobID, "request_id": j.RequestID, "customer_group_id": j.CustomerGroupID,
		"product_id": j.ProductID, "public_model": j.PublicModel, "modality": j.Modality,
		"status": j.Status, "reservation": j.Reservation, "attempts": j.Attempts,
		"selected_offer_id": j.SelectedOfferID, "events": j.Audit.Events,
		"select_summary": j.Audit.SelectSummary, "money_summary": j.Audit.MoneySummary,
	}
}

func BuildUsageLogDraft(job MediaJob, successAttempt *MediaAttempt) UsageLogDraft {
	draft := UsageLogDraft{
		RequestID: job.RequestID, APIKeyID: job.APIKeyID, UserID: job.UserID,
		CustomerGroupID: job.CustomerGroupID, RequestedModel: job.PublicModel,
		MediaType: string(job.Modality), ActualCost: job.Reservation.Amount, ProductID: job.ProductID,
		ImageCount: job.TotalBillableOutput(),
	}
	if successAttempt != nil {
		draft.AccountID = successAttempt.AccountID
		draft.UpstreamModel = successAttempt.UpstreamModel
		draft.OfferID = successAttempt.OfferID
		draft.UpstreamPlatform = successAttempt.Provider
		draft.SourceGroupID = successAttempt.SourceGroupID
		draft.TrustedCost = successAttempt.TrustedCostSnap
	}
	return draft
}

func BuildIdempotencyScope(job MediaJob) (string, string, string) {
	return "media:" + string(job.Modality) + ":" + job.Op, job.IdempotencyKeyHash, job.RequestFingerprint
}

func (j *MediaJob) appendAudit(at time.Time, eventType string, offerID int64, detail map[string]any) {
	j.Audit.Events = append(j.Audit.Events, MediaAuditEvent{At: at, Type: eventType, OfferID: offerID, Detail: detail})
}
