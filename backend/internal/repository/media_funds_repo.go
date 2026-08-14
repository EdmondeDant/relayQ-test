package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mediaFundsRepository struct{ db *sql.DB }

type mediaFundsRow struct {
	reference    string
	userID       int64
	publicID     string
	productID    int64
	amount       decimal.Decimal
	priceVersion string
	status       string
}

func NewMediaFundsRepository(db *sql.DB) service.MediaFundsRepository {
	return &mediaFundsRepository{db: db}
}

func (r *mediaFundsRepository) Reserve(ctx context.Context, request service.MediaFundsReserveRequest) (*service.MediaFundsReservation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err = lockMediaFunds(ctx, tx, request.UserID, request.PublicID); err != nil {
		return nil, err
	}
	row, err := getMediaFunds(ctx, tx, request.UserID, request.PublicID)
	if err == nil {
		if row.productID != request.ProductID || !row.amount.Equal(request.Amount) || row.priceVersion != request.PriceVersion || row.status != "reserved" {
			return nil, service.ErrMediaReservationConflict
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return mediaFundsResult(row, true), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE users SET balance=balance-$1, updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL AND status=$3 AND balance >= $1`, request.Amount, request.UserID, service.StatusActive)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected != 1 {
		return nil, service.ErrInsufficientBalance
	}
	row = &mediaFundsRow{reference: "media_hold_" + uuid.NewString(), userID: request.UserID, publicID: request.PublicID, productID: request.ProductID, amount: request.Amount, priceVersion: request.PriceVersion, status: "reserved"}
	if _, err = tx.ExecContext(ctx, `INSERT INTO media_funds_reservations(reference,user_id,public_id,product_id,amount,price_version,status) VALUES($1,$2,$3,$4,$5,$6,'reserved')`, row.reference, row.userID, row.publicID, row.productID, row.amount, row.priceVersion); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return mediaFundsResult(row, false), nil
}

func (r *mediaFundsRepository) Settle(ctx context.Context, request service.MediaFundsTransitionRequest) error {
	return r.transition(ctx, request, "settled", false)
}

func (r *mediaFundsRepository) Release(ctx context.Context, request service.MediaFundsTransitionRequest) error {
	return r.transition(ctx, request, "released", true)
}

func (r *mediaFundsRepository) transition(ctx context.Context, request service.MediaFundsTransitionRequest, target string, refund bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = lockMediaFunds(ctx, tx, request.UserID, request.PublicID); err != nil {
		return err
	}
	row, err := getMediaFunds(ctx, tx, request.UserID, request.PublicID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrMediaReservationInvalid
	}
	if err != nil {
		return err
	}
	if row.reference != request.Reference || !row.amount.Equal(request.Amount) {
		return service.ErrMediaReservationConflict
	}
	if row.status == target {
		return tx.Commit()
	}
	if row.status != "reserved" {
		return service.ErrMediaReservationConflict
	}
	if refund {
		result, updateErr := tx.ExecContext(ctx, `UPDATE users SET balance=balance+$1, updated_at=NOW() WHERE id=$2`, request.Amount, request.UserID)
		if updateErr != nil {
			return updateErr
		}
		if affected, updateErr := result.RowsAffected(); updateErr != nil || affected != 1 {
			if updateErr != nil {
				return updateErr
			}
			return service.ErrMediaReservationConflict
		}
	}
	timeColumn := "settled_at"
	if refund {
		timeColumn = "released_at"
	}
	result, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE media_funds_reservations SET status=$1, %s=NOW(), updated_at=NOW() WHERE user_id=$2 AND public_id=$3 AND reference=$4 AND status='reserved'`, timeColumn), target, request.UserID, request.PublicID, request.Reference)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return err
		}
		return service.ErrMediaReservationConflict
	}
	return tx.Commit()
}

func lockMediaFunds(ctx context.Context, tx *sql.Tx, userID int64, publicID string) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, fmt.Sprintf("media_funds:%d:%s", userID, publicID))
	return err
}

func getMediaFunds(ctx context.Context, tx *sql.Tx, userID int64, publicID string) (*mediaFundsRow, error) {
	row := &mediaFundsRow{}
	err := tx.QueryRowContext(ctx, `SELECT reference,user_id,public_id,product_id,amount,price_version,status FROM media_funds_reservations WHERE user_id=$1 AND public_id=$2 FOR UPDATE`, userID, publicID).Scan(&row.reference, &row.userID, &row.publicID, &row.productID, &row.amount, &row.priceVersion, &row.status)
	return row, err
}

func mediaFundsResult(row *mediaFundsRow, alreadyExists bool) *service.MediaFundsReservation {
	return &service.MediaFundsReservation{Reference: row.reference, UserID: row.userID, PublicID: row.publicID, ProductID: row.productID, Amount: row.amount, PriceVersion: row.priceVersion, Status: row.status, AlreadyExists: alreadyExists}
}
