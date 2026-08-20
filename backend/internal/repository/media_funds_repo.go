package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mediaFundsRepository struct {
	client *dbent.Client
	db     *sql.DB
}

type mediaFundsExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type mediaFundsRow struct {
	reference    string
	userID       int64
	publicID     string
	productID    int64
	amount       decimal.Decimal
	priceVersion string
	status       string
}

func NewMediaFundsRepository(client *dbent.Client, db *sql.DB) service.MediaFundsRepository {
	return &mediaFundsRepository{client: client, db: db}
}

func (r *mediaFundsRepository) Reserve(ctx context.Context, request service.MediaFundsReserveRequest) (*service.MediaFundsReservation, error) {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		exec := sqlExecutorFromEntClient(tx.Client())
		if exec == nil {
			return nil, errors.New("media funds transaction executor is unavailable")
		}
		return reserveMediaFunds(ctx, exec, request)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	reservation, err := reserveMediaFunds(ctx, tx, request)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return reservation, nil
}

func reserveMediaFunds(ctx context.Context, exec mediaFundsExecutor, request service.MediaFundsReserveRequest) (*service.MediaFundsReservation, error) {
	if err := lockMediaFunds(ctx, exec, request.UserID, request.PublicID); err != nil {
		return nil, err
	}
	row, err := getMediaFunds(ctx, exec, request.UserID, request.PublicID)
	if err == nil {
		if row.productID != request.ProductID || !row.amount.Equal(request.Amount) || row.priceVersion != request.PriceVersion || row.status != "reserved" {
			return nil, service.ErrMediaReservationConflict
		}
		return mediaFundsResult(row, true), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	result, err := exec.ExecContext(ctx, `UPDATE users SET balance=balance-$1, updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL AND status=$3 AND balance >= $1`, request.Amount, request.UserID, service.StatusActive)
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
	if _, err = exec.ExecContext(ctx, `INSERT INTO media_funds_reservations(reference,user_id,public_id,product_id,amount,price_version,status) VALUES($1,$2,$3,$4,$5,$6,'reserved')`, row.reference, row.userID, row.publicID, row.productID, row.amount, row.priceVersion); err != nil {
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

func lockMediaFunds(ctx context.Context, exec mediaFundsExecutor, userID int64, publicID string) error {
	_, err := exec.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, fmt.Sprintf("media_funds:%d:%s", userID, publicID))
	return err
}

func getMediaFunds(ctx context.Context, exec mediaFundsExecutor, userID int64, publicID string) (*mediaFundsRow, error) {
	row := &mediaFundsRow{}
	rows, err := exec.QueryContext(ctx, `SELECT reference,user_id,public_id,product_id,amount,price_version,status FROM media_funds_reservations WHERE user_id=$1 AND public_id=$2 FOR UPDATE`, userID, publicID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err = rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}
	if err = rows.Scan(&row.reference, &row.userID, &row.publicID, &row.productID, &row.amount, &row.priceVersion, &row.status); err != nil {
		return nil, err
	}
	return row, rows.Err()
}

func mediaFundsResult(row *mediaFundsRow, alreadyExists bool) *service.MediaFundsReservation {
	return &service.MediaFundsReservation{Reference: row.reference, UserID: row.userID, PublicID: row.publicID, ProductID: row.productID, Amount: row.amount, PriceVersion: row.priceVersion, Status: row.status, AlreadyExists: alreadyExists}
}
