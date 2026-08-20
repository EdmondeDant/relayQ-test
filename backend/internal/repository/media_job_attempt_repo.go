package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type mediaJobAttemptRepository struct{ db *sql.DB }

func NewMediaJobAttemptRepository(db *sql.DB) service.MediaJobAttemptRepository {
	return &mediaJobAttemptRepository{db: db}
}

func (r *mediaJobAttemptRepository) Create(ctx context.Context, attempt *service.MediaJobAttempt) error {
	if attempt == nil || attempt.JobID <= 0 || attempt.OfferID <= 0 || attempt.SourceGroupID <= 0 || strings.TrimSpace(attempt.Provider) == "" || strings.TrimSpace(attempt.UpstreamModel) == "" || !validMediaSubmissionState(attempt.SubmissionState) {
		return service.ErrMediaAttemptInvalid
	}
	snapshot, err := json.Marshal(attempt.TrustedCostSnapshot)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `INSERT INTO media_job_attempts(job_id, offer_id, provider, source_group_id, account_id, upstream_model, trusted_cost_snapshot, submission_state, error_code, error_message) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id, created_at, updated_at`, attempt.JobID, attempt.OfferID, attempt.Provider, attempt.SourceGroupID, attempt.AccountID, attempt.UpstreamModel, snapshot, attempt.SubmissionState, attempt.ErrorCode, attempt.ErrorMessage).Scan(&attempt.ID, &attempt.CreatedAt, &attempt.UpdatedAt)
}

func (r *mediaJobAttemptRepository) ListByJobID(ctx context.Context, jobID int64) ([]service.MediaJobAttempt, error) {
	if jobID <= 0 {
		return nil, service.ErrMediaAttemptInvalid
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, job_id, offer_id, provider, source_group_id, account_id, upstream_model, trusted_cost_snapshot, submission_state, error_code, error_message, created_at, updated_at FROM media_job_attempts WHERE job_id=$1 ORDER BY id`, jobID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	attempts := []service.MediaJobAttempt{}
	for rows.Next() {
		var attempt service.MediaJobAttempt
		var snapshot []byte
		var accountID sql.NullInt64
		var errorCode, errorMessage sql.NullString
		if err = rows.Scan(&attempt.ID, &attempt.JobID, &attempt.OfferID, &attempt.Provider, &attempt.SourceGroupID, &accountID, &attempt.UpstreamModel, &snapshot, &attempt.SubmissionState, &errorCode, &errorMessage, &attempt.CreatedAt, &attempt.UpdatedAt); err != nil {
			return nil, err
		}
		if accountID.Valid {
			attempt.AccountID = &accountID.Int64
		}
		if errorCode.Valid {
			attempt.ErrorCode = &errorCode.String
		}
		if errorMessage.Valid {
			attempt.ErrorMessage = &errorMessage.String
		}
		if err = json.Unmarshal(snapshot, &attempt.TrustedCostSnapshot); err != nil {
			return nil, err
		}
		attempts = append(attempts, attempt)
	}
	return attempts, rows.Err()
}

func validMediaSubmissionState(state service.MediaSubmissionState) bool {
	return state == service.MediaSubmissionNotWritten || state == service.MediaSubmissionSubmitted || state == service.MediaSubmissionSideEffectUnknown
}
