package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const leonardoFundsReferencePrefix = "leo_hold_"

var leonardoFundsMaxAmount = decimal.RequireFromString("999999999999.99999999")

type leonardoImageFundsRepository struct {
	db *sql.DB
}

type leonardoFundReservationRow struct {
	reference        string
	userID           int64
	publicID         string
	amount           decimal.Decimal
	pricingVersion   string
	pricingSource    string
	pricingMatchType string
	status           string
}

var _ service.LeonardoImageFunds = (*leonardoImageFundsRepository)(nil)

func NewLeonardoImageFundsRepository(_ *dbent.Client, sqlDB *sql.DB) service.LeonardoImageFunds {
	return &leonardoImageFundsRepository{db: sqlDB}
}

func (r *leonardoImageFundsRepository) Reserve(ctx context.Context, request service.LeonardoImageFundsReserveRequest) (*service.LeonardoImageFundsReservation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request.PublicID = strings.TrimSpace(request.PublicID)
	request.PricingVersion = strings.TrimSpace(request.PricingVersion)
	request.PricingSource = strings.TrimSpace(request.PricingSource)
	request.PricingMatchType = strings.TrimSpace(request.PricingMatchType)
	if r == nil || r.db == nil || request.UserID <= 0 || request.PublicID == "" || len(request.PublicID) > 64 || !validLeonardoFundsAmount(request.AmountUSD) || request.PricingVersion == "" || len(request.PricingVersion) > 64 || request.PricingSource == "" || len(request.PricingSource) > 128 || request.PricingMatchType == "" || len(request.PricingMatchType) > 32 {
		return nil, service.ErrLeonardoImageCreateReservationInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	lockKey := fmt.Sprintf("leonardo_image_funds:%d:%s", request.UserID, request.PublicID)
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return nil, err
	}
	existing, err := getLeonardoFundReservation(ctx, tx, request.UserID, request.PublicID)
	if err == nil {
		if !sameLeonardoFundReservation(existing, request) {
			return nil, service.ErrLeonardoImageCreateReservationConflict
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return leonardoFundReservationResult(existing, true), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var remaining decimal.Decimal
	err = tx.QueryRowContext(ctx, `
UPDATE users
SET balance = balance - $1, updated_at = NOW()
WHERE id = $2 AND deleted_at IS NULL AND status = $3 AND balance >= $1
RETURNING balance`, request.AmountUSD, request.UserID, service.StatusActive).Scan(&remaining)
	if errors.Is(err, sql.ErrNoRows) {
		var status string
		var deletedAt sql.NullTime
		if userErr := tx.QueryRowContext(ctx, `SELECT status, deleted_at FROM users WHERE id = $1`, request.UserID).Scan(&status, &deletedAt); errors.Is(userErr, sql.ErrNoRows) {
			return nil, service.ErrUserNotFound
		} else if userErr != nil {
			return nil, userErr
		} else if status != service.StatusActive || deletedAt.Valid {
			return nil, service.ErrLeonardoImageCreateReservationInvalid
		}
		return nil, service.ErrInsufficientBalance
	}
	if err != nil {
		return nil, err
	}
	row := &leonardoFundReservationRow{
		reference:        leonardoFundsReferencePrefix + uuid.NewString(),
		userID:           request.UserID,
		publicID:         request.PublicID,
		amount:           request.AmountUSD,
		pricingVersion:   request.PricingVersion,
		pricingSource:    request.PricingSource,
		pricingMatchType: request.PricingMatchType,
		status:           "reserved",
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO leonardo_image_funds_reservations
    (reference, user_id, public_id, amount_usd, pricing_version, pricing_source, pricing_match_type)
VALUES ($1, $2, $3, $4, $5, $6, $7)`, row.reference, row.userID, row.publicID, row.amount, row.pricingVersion, row.pricingSource, row.pricingMatchType)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return leonardoFundReservationResult(row, false), nil
}

func (r *leonardoImageFundsRepository) Release(ctx context.Context, request service.LeonardoImageFundsReleaseRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	request.PublicID = strings.TrimSpace(request.PublicID)
	request.Reference = strings.TrimSpace(request.Reference)
	request.Reason = strings.TrimSpace(request.Reason)
	if r == nil || r.db == nil || request.UserID <= 0 || request.PublicID == "" || len(request.PublicID) > 64 || request.Reference == "" || len(request.Reference) > 128 || !validLeonardoFundsAmount(request.AmountUSD) || request.Reason == "" || len(request.Reason) > 128 {
		return service.ErrLeonardoImageCreateReservationInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	lockKey := fmt.Sprintf("leonardo_image_funds:%d:%s", request.UserID, request.PublicID)
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return err
	}
	row, err := getLeonardoFundReservation(ctx, tx, request.UserID, request.PublicID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrLeonardoImageCreateReservationInvalid
	}
	if err != nil {
		return err
	}
	if row.userID != request.UserID || row.publicID != request.PublicID || row.reference != request.Reference || !row.amount.Equal(request.AmountUSD) {
		return service.ErrLeonardoImageCreateReservationConflict
	}
	if row.status == "released" {
		return tx.Commit()
	}
	if row.status != "reserved" {
		return service.ErrLeonardoImageCreateReservationConflict
	}
	result, err := tx.ExecContext(ctx, `
UPDATE users
SET balance = balance + $1, updated_at = NOW()
WHERE id = $2`, request.AmountUSD, request.UserID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return service.ErrLeonardoImageCreateReservationConflict
	}
	result, err = tx.ExecContext(ctx, `
UPDATE leonardo_image_funds_reservations
SET status = 'released', released_at = NOW(), updated_at = NOW()
WHERE user_id = $1 AND public_id = $2 AND reference = $3 AND status = 'reserved'`, request.UserID, request.PublicID, request.Reference)
	if err != nil {
		return err
	}
	affected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return service.ErrLeonardoImageCreateReservationConflict
	}
	// Keep the persisted job ledger consistent with the balance refund in the
	// same transaction. A job may not exist when submission failed before
	// creation, so zero affected rows is valid.
	if _, err = tx.ExecContext(ctx, `
UPDATE generation_jobs
SET billing_status = 'refunded', updated_at = NOW()
WHERE public_id = $1 AND billing_reference = $2 AND billing_status = 'reserved'`, request.PublicID, request.Reference); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *leonardoImageFundsRepository) Settle(ctx context.Context, request service.LeonardoImageFundsSettleRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	request.PublicID = strings.TrimSpace(request.PublicID)
	request.Reference = strings.TrimSpace(request.Reference)
	if r == nil || r.db == nil || request.UserID <= 0 || request.PublicID == "" || len(request.PublicID) > 64 || request.Reference == "" || len(request.Reference) > 128 || !validLeonardoFundsAmount(request.AmountUSD) {
		return service.ErrLeonardoImageCreateReservationInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	lockKey := fmt.Sprintf("leonardo_image_funds:%d:%s", request.UserID, request.PublicID)
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return err
	}
	row, err := getLeonardoFundReservation(ctx, tx, request.UserID, request.PublicID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrLeonardoImageCreateReservationInvalid
	}
	if err != nil {
		return err
	}
	if row.userID != request.UserID || row.publicID != request.PublicID || row.reference != request.Reference || !row.amount.Equal(request.AmountUSD) {
		return service.ErrLeonardoImageCreateReservationConflict
	}
	if row.status == "settled" {
		return tx.Commit()
	}
	if row.status != "reserved" {
		return service.ErrLeonardoImageCreateReservationConflict
	}
	result, err := tx.ExecContext(ctx, `
UPDATE leonardo_image_funds_reservations
SET status = 'settled', updated_at = NOW()
WHERE user_id = $1 AND public_id = $2 AND reference = $3 AND status = 'reserved'`, request.UserID, request.PublicID, request.Reference)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return service.ErrLeonardoImageCreateReservationConflict
	}
	return tx.Commit()
}

func getLeonardoFundReservation(ctx context.Context, tx *sql.Tx, userID int64, publicID string) (*leonardoFundReservationRow, error) {
	row := &leonardoFundReservationRow{}
	err := tx.QueryRowContext(ctx, `
SELECT reference, user_id, public_id, amount_usd, pricing_version, pricing_source, pricing_match_type, status
FROM leonardo_image_funds_reservations
WHERE user_id = $1 AND public_id = $2
FOR UPDATE`, userID, publicID).Scan(&row.reference, &row.userID, &row.publicID, &row.amount, &row.pricingVersion, &row.pricingSource, &row.pricingMatchType, &row.status)
	return row, err
}

func validLeonardoFundsAmount(amount decimal.Decimal) bool {
	return amount.Sign() > 0 && amount.Cmp(leonardoFundsMaxAmount) <= 0 && amount.Equal(amount.Truncate(8))
}

func sameLeonardoFundReservation(row *leonardoFundReservationRow, request service.LeonardoImageFundsReserveRequest) bool {
	return row != nil && row.status == "reserved" && row.userID == request.UserID && row.publicID == request.PublicID && row.amount.Equal(request.AmountUSD) && row.pricingVersion == request.PricingVersion && row.pricingSource == request.PricingSource && row.pricingMatchType == request.PricingMatchType
}

func leonardoFundReservationResult(row *leonardoFundReservationRow, alreadyReserved bool) *service.LeonardoImageFundsReservation {
	return &service.LeonardoImageFundsReservation{Reference: row.reference, UserID: row.userID, PublicID: row.publicID, AmountUSD: row.amount, PricingVersion: row.pricingVersion, PricingSource: row.pricingSource, PricingMatchType: row.pricingMatchType, AlreadyReserved: alreadyReserved}
}
