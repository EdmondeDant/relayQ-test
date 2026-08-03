package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestLeonardoImageFundsRepositoryReserve(t *testing.T) {
	t.Run("deducts active user and creates reservation", func(t *testing.T) {
		db, mock := newLeonardoFundsSQLMock(t)
		request := leonardoFundsReserveRequest("job-1", "0.005")
		expectLeonardoFundsBeginAndMiss(mock, request.UserID, request.PublicID)
		mock.ExpectQuery("UPDATE users").WithArgs(request.AmountUSD, request.UserID, service.StatusActive).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("0.99500000"))
		mock.ExpectExec("INSERT INTO leonardo_image_funds_reservations").WithArgs(sqlmock.AnyArg(), request.UserID, request.PublicID, request.AmountUSD, request.PricingVersion, request.PricingSource, request.PricingMatchType).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reservation, err := NewLeonardoImageFundsRepository(nil, db).Reserve(context.Background(), request)
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(reservation.Reference, leonardoFundsReferencePrefix))
		require.Equal(t, request.UserID, reservation.UserID)
		require.Equal(t, request.PublicID, reservation.PublicID)
		require.True(t, request.AmountUSD.Equal(reservation.AmountUSD))
		require.Equal(t, request.PricingVersion, reservation.PricingVersion)
		require.Equal(t, request.PricingSource, reservation.PricingSource)
		require.Equal(t, request.PricingMatchType, reservation.PricingMatchType)
		require.False(t, reservation.AlreadyReserved)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("same request is idempotent", func(t *testing.T) {
		db, mock := newLeonardoFundsSQLMock(t)
		request := leonardoFundsReserveRequest("job-1", "0.005")
		expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
		mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows("reserved"))
		mock.ExpectCommit()

		reservation, err := NewLeonardoImageFundsRepository(nil, db).Reserve(context.Background(), request)
		require.NoError(t, err)
		require.Equal(t, "leo_hold_existing", reservation.Reference)
		require.True(t, reservation.AlreadyReserved)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	for _, mutate := range []struct {
		name string
		fn   func(*service.LeonardoImageFundsReserveRequest)
	}{
		{name: "amount", fn: func(r *service.LeonardoImageFundsReserveRequest) { r.AmountUSD = decimal.RequireFromString("0.006") }},
		{name: "pricing version", fn: func(r *service.LeonardoImageFundsReserveRequest) { r.PricingVersion = "other" }},
		{name: "pricing source", fn: func(r *service.LeonardoImageFundsReserveRequest) { r.PricingSource = "other" }},
		{name: "pricing match type", fn: func(r *service.LeonardoImageFundsReserveRequest) { r.PricingMatchType = "default" }},
	} {
		t.Run("existing reservation conflicts on "+mutate.name, func(t *testing.T) {
			db, mock := newLeonardoFundsSQLMock(t)
			request := leonardoFundsReserveRequest("job-1", "0.005")
			mutate.fn(&request)
			expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
			mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows("reserved"))
			mock.ExpectRollback()

			_, err := NewLeonardoImageFundsRepository(nil, db).Reserve(context.Background(), request)
			require.ErrorIs(t, err, service.ErrLeonardoImageCreateReservationConflict)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	for _, tc := range []struct {
		name     string
		userRows *sqlmock.Rows
		userErr  error
		expected error
	}{
		{name: "insufficient balance", userRows: sqlmock.NewRows([]string{"status", "deleted_at"}).AddRow(service.StatusActive, nil), expected: service.ErrInsufficientBalance},
		{name: "missing user", userErr: sql.ErrNoRows, expected: service.ErrUserNotFound},
		{name: "inactive user", userRows: sqlmock.NewRows([]string{"status", "deleted_at"}).AddRow("disabled", nil), expected: service.ErrLeonardoImageCreateReservationInvalid},
		{name: "deleted user", userRows: sqlmock.NewRows([]string{"status", "deleted_at"}).AddRow(service.StatusActive, time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)), expected: service.ErrLeonardoImageCreateReservationInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newLeonardoFundsSQLMock(t)
			request := leonardoFundsReserveRequest("job-1", "0.005")
			expectLeonardoFundsBeginAndMiss(mock, request.UserID, request.PublicID)
			mock.ExpectQuery("UPDATE users").WithArgs(request.AmountUSD, request.UserID, service.StatusActive).WillReturnError(sql.ErrNoRows)
			userQuery := mock.ExpectQuery("SELECT status, deleted_at FROM users").WithArgs(request.UserID)
			if tc.userErr != nil {
				userQuery.WillReturnError(tc.userErr)
			} else {
				userQuery.WillReturnRows(tc.userRows)
			}
			mock.ExpectRollback()

			_, err := NewLeonardoImageFundsRepository(nil, db).Reserve(context.Background(), request)
			require.ErrorIs(t, err, tc.expected)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLeonardoImageFundsRepositoryRelease(t *testing.T) {
	t.Run("credits once and marks released", func(t *testing.T) {
		db, mock := newLeonardoFundsSQLMock(t)
		request := leonardoFundsReleaseRequest()
		expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
		mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows("reserved"))
		mock.ExpectExec("UPDATE users").WithArgs(request.AmountUSD, request.UserID).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE leonardo_image_funds_reservations").WithArgs(request.UserID, request.PublicID, request.Reference).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		require.NoError(t, NewLeonardoImageFundsRepository(nil, db).Release(context.Background(), request))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	for _, tc := range []struct {
		name         string
		userResult   sql.Result
		userErr      error
		statusResult sql.Result
		statusErr    error
	}{
		{name: "user update failure", userErr: sql.ErrConnDone},
		{name: "missing user", userResult: sqlmock.NewResult(0, 0)},
		{name: "status update failure", userResult: sqlmock.NewResult(0, 1), statusErr: sql.ErrConnDone},
		{name: "status update conflict", userResult: sqlmock.NewResult(0, 1), statusResult: sqlmock.NewResult(0, 0)},
	} {
		t.Run(tc.name+" rolls back", func(t *testing.T) {
			db, mock := newLeonardoFundsSQLMock(t)
			request := leonardoFundsReleaseRequest()
			expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
			mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows("reserved"))
			userUpdate := mock.ExpectExec("UPDATE users").WithArgs(request.AmountUSD, request.UserID)
			if tc.userErr != nil {
				userUpdate.WillReturnError(tc.userErr)
			} else {
				userUpdate.WillReturnResult(tc.userResult)
			}
			if tc.userErr == nil && tc.userResult != nil {
				rows, _ := tc.userResult.RowsAffected()
				if rows == 1 {
					statusUpdate := mock.ExpectExec("UPDATE leonardo_image_funds_reservations").WithArgs(request.UserID, request.PublicID, request.Reference)
					if tc.statusErr != nil {
						statusUpdate.WillReturnError(tc.statusErr)
					} else {
						statusUpdate.WillReturnResult(tc.statusResult)
					}
				}
			}
			mock.ExpectRollback()
			require.Error(t, NewLeonardoImageFundsRepository(nil, db).Release(context.Background(), request))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("already released is idempotent", func(t *testing.T) {
		db, mock := newLeonardoFundsSQLMock(t)
		request := leonardoFundsReleaseRequest()
		expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
		mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows("released"))
		mock.ExpectCommit()

		require.NoError(t, NewLeonardoImageFundsRepository(nil, db).Release(context.Background(), request))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	for _, tc := range []struct {
		name   string
		mutate func(*service.LeonardoImageFundsReleaseRequest)
		status string
	}{
		{name: "reference mismatch", mutate: func(r *service.LeonardoImageFundsReleaseRequest) { r.Reference = "wrong" }, status: "reserved"},
		{name: "amount mismatch", mutate: func(r *service.LeonardoImageFundsReleaseRequest) { r.AmountUSD = decimal.RequireFromString("0.006") }, status: "reserved"},
		{name: "settled reservation", mutate: func(*service.LeonardoImageFundsReleaseRequest) {}, status: "settled"},
	} {
		t.Run(tc.name+" conflicts", func(t *testing.T) {
			db, mock := newLeonardoFundsSQLMock(t)
			request := leonardoFundsReleaseRequest()
			tc.mutate(&request)
			expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
			mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows(tc.status))
			mock.ExpectRollback()

			err := NewLeonardoImageFundsRepository(nil, db).Release(context.Background(), request)
			require.ErrorIs(t, err, service.ErrLeonardoImageCreateReservationConflict)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("missing reservation is invalid", func(t *testing.T) {
		db, mock := newLeonardoFundsSQLMock(t)
		request := leonardoFundsReleaseRequest()
		expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
		mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		err := NewLeonardoImageFundsRepository(nil, db).Release(context.Background(), request)
		require.ErrorIs(t, err, service.ErrLeonardoImageCreateReservationInvalid)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestLeonardoImageFundsRepositorySettle(t *testing.T) {
	t.Run("marks reserved as settled without balance update", func(t *testing.T) {
		db, mock := newLeonardoFundsSQLMock(t)
		request := leonardoFundsSettleRequest()
		expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
		mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows("reserved"))
		mock.ExpectExec("UPDATE leonardo_image_funds_reservations").WithArgs(request.UserID, request.PublicID, request.Reference).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		require.NoError(t, NewLeonardoImageFundsRepository(nil, db).Settle(context.Background(), request))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("already settled is idempotent", func(t *testing.T) {
		db, mock := newLeonardoFundsSQLMock(t)
		request := leonardoFundsSettleRequest()
		expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
		mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows("settled"))
		mock.ExpectCommit()
		require.NoError(t, NewLeonardoImageFundsRepository(nil, db).Settle(context.Background(), request))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	for _, test := range []struct {
		name   string
		status string
		mutate func(*service.LeonardoImageFundsSettleRequest)
	}{
		{name: "released", status: "released", mutate: func(*service.LeonardoImageFundsSettleRequest) {}},
		{name: "reference mismatch", status: "reserved", mutate: func(r *service.LeonardoImageFundsSettleRequest) { r.Reference = "wrong" }},
		{name: "amount mismatch", status: "reserved", mutate: func(r *service.LeonardoImageFundsSettleRequest) { r.AmountUSD = decimal.RequireFromString("0.006") }},
	} {
		t.Run(test.name+" conflicts", func(t *testing.T) {
			db, mock := newLeonardoFundsSQLMock(t)
			request := leonardoFundsSettleRequest()
			test.mutate(&request)
			expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
			mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows(test.status))
			mock.ExpectRollback()
			require.ErrorIs(t, NewLeonardoImageFundsRepository(nil, db).Settle(context.Background(), request), service.ErrLeonardoImageCreateReservationConflict)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("missing reservation is invalid", func(t *testing.T) {
		db, mock := newLeonardoFundsSQLMock(t)
		request := leonardoFundsSettleRequest()
		expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
		mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()
		require.ErrorIs(t, NewLeonardoImageFundsRepository(nil, db).Settle(context.Background(), request), service.ErrLeonardoImageCreateReservationInvalid)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestLeonardoImageFundsRepositoryTransactionFailures(t *testing.T) {
	t.Run("insert failure rolls back balance", func(t *testing.T) {
		db, mock := newLeonardoFundsSQLMock(t)
		request := leonardoFundsReserveRequest("job-1", "0.005")
		expectLeonardoFundsBeginAndMiss(mock, request.UserID, request.PublicID)
		mock.ExpectQuery("UPDATE users").WithArgs(request.AmountUSD, request.UserID, service.StatusActive).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow("0.995"))
		mock.ExpectExec("INSERT INTO leonardo_image_funds_reservations").WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()
		_, err := NewLeonardoImageFundsRepository(nil, db).Reserve(context.Background(), request)
		require.ErrorIs(t, err, sql.ErrConnDone)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	for _, operation := range []string{"reserve", "release"} {
		t.Run(operation+" commit error is returned without retry", func(t *testing.T) {
			db, mock := newLeonardoFundsSQLMock(t)
			if operation == "reserve" {
				request := leonardoFundsReserveRequest("job-1", "0.005")
				expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
				mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows("reserved"))
				mock.ExpectCommit().WillReturnError(sql.ErrTxDone)
				_, err := NewLeonardoImageFundsRepository(nil, db).Reserve(context.Background(), request)
				require.ErrorIs(t, err, sql.ErrTxDone)
			} else {
				request := leonardoFundsReleaseRequest()
				expectLeonardoFundsBegin(mock, request.UserID, request.PublicID)
				mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(request.UserID, request.PublicID).WillReturnRows(leonardoReservationRows("released"))
				mock.ExpectCommit().WillReturnError(sql.ErrTxDone)
				require.ErrorIs(t, NewLeonardoImageFundsRepository(nil, db).Release(context.Background(), request), sql.ErrTxDone)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLeonardoImageFundsRepositoryValidation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := NewLeonardoImageFundsRepository(nil, nil)
	_, err := repo.Reserve(ctx, leonardoFundsReserveRequest("job-1", "0.005"))
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, repo.Release(ctx, leonardoFundsReleaseRequest()), context.Canceled)
	require.ErrorIs(t, repo.Settle(ctx, leonardoFundsSettleRequest()), context.Canceled)

	for _, tc := range []struct {
		name   string
		mutate func(*service.LeonardoImageFundsReserveRequest)
	}{
		{name: "nil database", mutate: func(*service.LeonardoImageFundsReserveRequest) {}},
		{name: "invalid user", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.UserID = 0 }},
		{name: "empty public id", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.PublicID = " " }},
		{name: "long public id", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.PublicID = strings.Repeat("x", 65) }},
		{name: "zero amount", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.AmountUSD = decimal.Zero }},
		{name: "negative amount", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.AmountUSD = decimal.NewFromInt(-1) }},
		{name: "over precision amount", mutate: func(r *service.LeonardoImageFundsReserveRequest) {
			r.AmountUSD = decimal.RequireFromString("0.000000001")
		}},
		{name: "over max amount", mutate: func(r *service.LeonardoImageFundsReserveRequest) {
			r.AmountUSD = decimal.RequireFromString("1000000000000")
		}},
		{name: "empty pricing version", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.PricingVersion = "" }},
		{name: "long pricing version", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.PricingVersion = strings.Repeat("x", 65) }},
		{name: "empty pricing source", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.PricingSource = "" }},
		{name: "long pricing source", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.PricingSource = strings.Repeat("x", 129) }},
		{name: "empty match type", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.PricingMatchType = "" }},
		{name: "long match type", mutate: func(r *service.LeonardoImageFundsReserveRequest) { r.PricingMatchType = strings.Repeat("x", 33) }},
	} {
		t.Run("reserve "+tc.name, func(t *testing.T) {
			request := leonardoFundsReserveRequest("job-1", "0.005")
			tc.mutate(&request)
			_, err := repo.Reserve(context.Background(), request)
			require.ErrorIs(t, err, service.ErrLeonardoImageCreateReservationInvalid)
		})
	}

	for _, tc := range []struct {
		name   string
		mutate func(*service.LeonardoImageFundsReleaseRequest)
	}{
		{name: "invalid user", mutate: func(r *service.LeonardoImageFundsReleaseRequest) { r.UserID = 0 }},
		{name: "empty public id", mutate: func(r *service.LeonardoImageFundsReleaseRequest) { r.PublicID = "" }},
		{name: "long public id", mutate: func(r *service.LeonardoImageFundsReleaseRequest) { r.PublicID = strings.Repeat("x", 65) }},
		{name: "empty reference", mutate: func(r *service.LeonardoImageFundsReleaseRequest) { r.Reference = "" }},
		{name: "long reference", mutate: func(r *service.LeonardoImageFundsReleaseRequest) { r.Reference = strings.Repeat("x", 129) }},
		{name: "invalid amount", mutate: func(r *service.LeonardoImageFundsReleaseRequest) {
			r.AmountUSD = decimal.RequireFromString("0.000000001")
		}},
		{name: "empty reason", mutate: func(r *service.LeonardoImageFundsReleaseRequest) { r.Reason = "" }},
		{name: "long reason", mutate: func(r *service.LeonardoImageFundsReleaseRequest) { r.Reason = strings.Repeat("x", 129) }},
	} {
		t.Run("release "+tc.name, func(t *testing.T) {
			request := leonardoFundsReleaseRequest()
			tc.mutate(&request)
			require.ErrorIs(t, repo.Release(context.Background(), request), service.ErrLeonardoImageCreateReservationInvalid)
		})
	}

	for _, tc := range []struct {
		name   string
		mutate func(*service.LeonardoImageFundsSettleRequest)
	}{
		{name: "invalid user", mutate: func(r *service.LeonardoImageFundsSettleRequest) { r.UserID = 0 }},
		{name: "empty public id", mutate: func(r *service.LeonardoImageFundsSettleRequest) { r.PublicID = " " }},
		{name: "long public id", mutate: func(r *service.LeonardoImageFundsSettleRequest) { r.PublicID = strings.Repeat("x", 65) }},
		{name: "empty reference", mutate: func(r *service.LeonardoImageFundsSettleRequest) { r.Reference = " " }},
		{name: "long reference", mutate: func(r *service.LeonardoImageFundsSettleRequest) { r.Reference = strings.Repeat("x", 129) }},
		{name: "zero amount", mutate: func(r *service.LeonardoImageFundsSettleRequest) { r.AmountUSD = decimal.Zero }},
		{name: "over precision", mutate: func(r *service.LeonardoImageFundsSettleRequest) {
			r.AmountUSD = decimal.RequireFromString("0.000000001")
		}},
		{name: "over max", mutate: func(r *service.LeonardoImageFundsSettleRequest) {
			r.AmountUSD = decimal.RequireFromString("1000000000000")
		}},
	} {
		t.Run("settle "+tc.name, func(t *testing.T) {
			request := leonardoFundsSettleRequest()
			tc.mutate(&request)
			require.ErrorIs(t, repo.Settle(context.Background(), request), service.ErrLeonardoImageCreateReservationInvalid)
		})
	}
}

func newLeonardoFundsSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func expectLeonardoFundsBegin(mock sqlmock.Sqlmock, userID int64, publicID string) {
	mock.ExpectBegin()
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(fmt.Sprintf("leonardo_image_funds:%d:%s", userID, publicID)).WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectLeonardoFundsBeginAndMiss(mock sqlmock.Sqlmock, userID int64, publicID string) {
	expectLeonardoFundsBegin(mock, userID, publicID)
	mock.ExpectQuery("SELECT reference, user_id, public_id").WithArgs(userID, publicID).WillReturnError(sql.ErrNoRows)
}

func leonardoFundsReserveRequest(publicID, amount string) service.LeonardoImageFundsReserveRequest {
	return service.LeonardoImageFundsReserveRequest{UserID: 7, PublicID: publicID, AmountUSD: decimal.RequireFromString(amount), PricingVersion: "2026-08-01", PricingSource: "calculator", PricingMatchType: "exact"}
}

func leonardoFundsReleaseRequest() service.LeonardoImageFundsReleaseRequest {
	return service.LeonardoImageFundsReleaseRequest{UserID: 7, PublicID: "job-1", Reference: "leo_hold_existing", AmountUSD: decimal.RequireFromString("0.005"), Reason: "request_not_written"}
}

func leonardoFundsSettleRequest() service.LeonardoImageFundsSettleRequest {
	return service.LeonardoImageFundsSettleRequest{UserID: 7, PublicID: "job-1", Reference: "leo_hold_existing", AmountUSD: decimal.RequireFromString("0.005")}
}

func leonardoReservationRows(status string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"reference", "user_id", "public_id", "amount_usd", "pricing_version", "pricing_source", "pricing_match_type", "status"}).AddRow("leo_hold_existing", int64(7), "job-1", "0.00500000", "2026-08-01", "calculator", "exact", status)
}
