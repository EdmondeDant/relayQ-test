//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const leonardoFundsConcurrency = 8

func TestLeonardoImageFundsRepositoryPostgresConcurrentIdempotentReserveAndRelease(t *testing.T) {
	userID := createLeonardoFundsTestUser(t, "idempotent", "1")
	repo := NewLeonardoImageFundsRepository(nil, integrationDB)
	request := service.LeonardoImageFundsReserveRequest{UserID: userID, PublicID: fmt.Sprintf("job-idempotent-%d", time.Now().UnixNano()), AmountUSD: decimal.RequireFromString("0.6"), PricingVersion: "2026-08-01", PricingSource: "calculator", PricingMatchType: "exact"}

	results, errs := concurrentlyReserveLeonardoFunds(repo, request, leonardoFundsConcurrency)
	for _, err := range errs {
		require.NoError(t, err)
	}
	firstReservations := 0
	var reference string
	for _, reservation := range results {
		require.NotNil(t, reservation)
		if !reservation.AlreadyReserved {
			firstReservations++
		}
		if reference == "" {
			reference = reservation.Reference
		}
		require.Equal(t, reference, reservation.Reference)
	}
	require.Equal(t, 1, firstReservations)
	require.Equal(t, "0.40000000", leonardoTestUserBalance(t, userID))
	require.Equal(t, 1, leonardoTestReservationCount(t, userID))

	release := service.LeonardoImageFundsReleaseRequest{UserID: userID, PublicID: request.PublicID, Reference: reference, AmountUSD: request.AmountUSD, Reason: "request_not_written"}
	releaseErrs := concurrentlyReleaseLeonardoFunds(repo, release, leonardoFundsConcurrency)
	for _, err := range releaseErrs {
		require.NoError(t, err)
	}
	require.Equal(t, "1.00000000", leonardoTestUserBalance(t, userID))
	require.Equal(t, "released", leonardoTestReservationStatus(t, userID, request.PublicID))
}

func TestLeonardoImageFundsRepositoryPostgresConcurrentBalanceGuard(t *testing.T) {
	ctx := context.Background()
	userID := createLeonardoFundsTestUser(t, "balance", "1")
	repo := NewLeonardoImageFundsRepository(nil, integrationDB)
	start := make(chan struct{})
	errCh := make(chan error, leonardoFundsConcurrency)
	var wg sync.WaitGroup
	for i := range leonardoFundsConcurrency {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, err := repo.Reserve(ctx, service.LeonardoImageFundsReserveRequest{UserID: userID, PublicID: fmt.Sprintf("job-balance-%d-%d", time.Now().UnixNano(), index), AmountUSD: decimal.RequireFromString("0.2"), PricingVersion: "2026-08-01", PricingSource: "calculator", PricingMatchType: "exact"})
			errCh <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(errCh)

	succeeded, insufficient := 0, 0
	for err := range errCh {
		if err == nil {
			succeeded++
		} else if errors.Is(err, service.ErrInsufficientBalance) {
			insufficient++
		} else {
			require.NoError(t, err)
		}
	}
	require.Equal(t, 5, succeeded)
	require.Equal(t, 3, insufficient)
	require.Equal(t, "0.00000000", leonardoTestUserBalance(t, userID))
	require.Equal(t, 5, leonardoTestReservationCount(t, userID))
}

func TestLeonardoImageFundsRepositoryPostgresScopeAndConflictContracts(t *testing.T) {
	ctx := context.Background()
	firstUserID := createLeonardoFundsTestUser(t, "scope-first", "1")
	secondUserID := createLeonardoFundsTestUser(t, "scope-second", "1")
	repo := NewLeonardoImageFundsRepository(nil, integrationDB)
	publicID := fmt.Sprintf("shared-job-%d", time.Now().UnixNano())
	first := service.LeonardoImageFundsReserveRequest{UserID: firstUserID, PublicID: publicID, AmountUSD: decimal.RequireFromString("0.25"), PricingVersion: "2026-08-01", PricingSource: "calculator", PricingMatchType: "exact"}
	second := first
	second.UserID = secondUserID
	firstReservation, err := repo.Reserve(ctx, first)
	require.NoError(t, err)
	secondReservation, err := repo.Reserve(ctx, second)
	require.NoError(t, err)
	require.NotEqual(t, firstReservation.Reference, secondReservation.Reference)
	require.Equal(t, 2, leonardoTestReservationCountByPublicID(t, publicID))

	conflict := first
	conflict.AmountUSD = decimal.RequireFromString("0.26")
	_, err = repo.Reserve(ctx, conflict)
	require.ErrorIs(t, err, service.ErrLeonardoImageCreateReservationConflict)
	require.Equal(t, "0.75000000", leonardoTestUserBalance(t, firstUserID))

	err = repo.Release(ctx, service.LeonardoImageFundsReleaseRequest{UserID: firstUserID, PublicID: publicID, Reference: secondReservation.Reference, AmountUSD: first.AmountUSD, Reason: "wrong_reference"})
	require.ErrorIs(t, err, service.ErrLeonardoImageCreateReservationConflict)
	require.Equal(t, "0.75000000", leonardoTestUserBalance(t, firstUserID))

	err = repo.Release(ctx, service.LeonardoImageFundsReleaseRequest{UserID: firstUserID, PublicID: "missing", Reference: firstReservation.Reference, AmountUSD: first.AmountUSD, Reason: "missing"})
	require.ErrorIs(t, err, service.ErrLeonardoImageCreateReservationInvalid)
}

func TestLeonardoImageFundsRepositoryPostgresPrecisionAndReleaseAfterUserDisable(t *testing.T) {
	ctx := context.Background()
	userID := createLeonardoFundsTestUser(t, "precision", "0.00000001")
	repo := NewLeonardoImageFundsRepository(nil, integrationDB)
	request := service.LeonardoImageFundsReserveRequest{UserID: userID, PublicID: fmt.Sprintf("job-precision-%d", time.Now().UnixNano()), AmountUSD: decimal.RequireFromString("0.00000001"), PricingVersion: "2026-08-01", PricingSource: "calculator", PricingMatchType: "exact"}
	reservation, err := repo.Reserve(ctx, request)
	require.NoError(t, err)
	require.Equal(t, "0.00000000", leonardoTestUserBalance(t, userID))
	_, err = integrationDB.ExecContext(ctx, `UPDATE users SET status = 'disabled', deleted_at = NOW() WHERE id = $1`, userID)
	require.NoError(t, err)
	err = repo.Release(ctx, service.LeonardoImageFundsReleaseRequest{UserID: userID, PublicID: request.PublicID, Reference: reservation.Reference, AmountUSD: request.AmountUSD, Reason: "request_not_written"})
	require.NoError(t, err)
	require.Equal(t, "0.00000001", leonardoTestUserBalance(t, userID))
}

func concurrentlyReserveLeonardoFunds(repo service.LeonardoImageCreateFunds, request service.LeonardoImageFundsReserveRequest, count int) ([]*service.LeonardoImageFundsReservation, []error) {
	start := make(chan struct{})
	resultCh := make(chan *service.LeonardoImageFundsReservation, count)
	errCh := make(chan error, count)
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := repo.Reserve(context.Background(), request)
			resultCh <- result
			errCh <- err
		}()
	}
	close(start)
	wg.Wait()
	close(resultCh)
	close(errCh)
	return collectLeonardoReservations(resultCh), collectLeonardoErrors(errCh)
}

func concurrentlyReleaseLeonardoFunds(repo service.LeonardoImageCreateFunds, request service.LeonardoImageFundsReleaseRequest, count int) []error {
	start := make(chan struct{})
	errCh := make(chan error, count)
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errCh <- repo.Release(context.Background(), request)
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	return collectLeonardoErrors(errCh)
}

func collectLeonardoReservations(ch <-chan *service.LeonardoImageFundsReservation) []*service.LeonardoImageFundsReservation {
	values := make([]*service.LeonardoImageFundsReservation, 0, cap(ch))
	for value := range ch {
		values = append(values, value)
	}
	return values
}

func collectLeonardoErrors(ch <-chan error) []error {
	values := make([]error, 0, cap(ch))
	for value := range ch {
		values = append(values, value)
	}
	return values
}

func createLeonardoFundsTestUser(t *testing.T, label, balance string) int64 {
	t.Helper()
	var userID int64
	email := fmt.Sprintf("leonardo-funds-%s-%d@example.com", label, time.Now().UnixNano())
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `INSERT INTO users (email, balance, status) VALUES ($1, $2, $3) RETURNING id`, email, balance, service.StatusActive).Scan(&userID))
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM leonardo_image_funds_reservations WHERE user_id = $1`, userID)
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	return userID
}

func leonardoTestUserBalance(t *testing.T, userID int64) string {
	t.Helper()
	var balance string
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `SELECT balance::text FROM users WHERE id = $1`, userID).Scan(&balance))
	return balance
}

func leonardoTestReservationCount(t *testing.T, userID int64) int {
	t.Helper()
	var count int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM leonardo_image_funds_reservations WHERE user_id = $1`, userID).Scan(&count))
	return count
}

func leonardoTestReservationCountByPublicID(t *testing.T, publicID string) int {
	t.Helper()
	var count int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM leonardo_image_funds_reservations WHERE public_id = $1`, publicID).Scan(&count))
	return count
}

func leonardoTestReservationStatus(t *testing.T, userID int64, publicID string) string {
	t.Helper()
	var status string
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `SELECT status FROM leonardo_image_funds_reservations WHERE user_id = $1 AND public_id = $2`, userID, publicID).Scan(&status))
	return status
}
