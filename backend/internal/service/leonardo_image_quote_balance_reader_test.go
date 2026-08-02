package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type leonardoImageQuoteUserReaderFake struct {
	user    *User
	err     error
	calls   int
	userIDs []int64
}

func (f *leonardoImageQuoteUserReaderFake) GetByID(_ context.Context, userID int64) (*User, error) {
	f.calls++
	f.userIDs = append(f.userIDs, userID)
	return f.user, f.err
}

func TestLeonardoImageQuoteUserBalanceReaderSuccess(t *testing.T) {
	users := &leonardoImageQuoteUserReaderFake{user: &User{ID: 42, Balance: 0.005}}
	balance, err := NewLeonardoImageQuoteUserBalanceReader(users).GetAvailableBalanceUSD(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, "0.005", balance.String())
	require.Equal(t, 1, users.calls)
	require.Equal(t, []int64{42}, users.userIDs)
}

func TestLeonardoImageQuoteUserBalanceReaderZeroAndNegative(t *testing.T) {
	for _, test := range []struct {
		balance  float64
		expected string
	}{{0, "0"}, {-0.001, "-0.001"}} {
		users := &leonardoImageQuoteUserReaderFake{user: &User{ID: 42, Balance: test.balance}}
		balance, err := NewLeonardoImageQuoteUserBalanceReader(users).GetAvailableBalanceUSD(context.Background(), 42)
		require.NoError(t, err)
		require.Equal(t, test.expected, balance.String())
	}
}

func TestLeonardoImageQuoteUserBalanceReaderPrecision(t *testing.T) {
	for _, test := range []struct {
		balance  float64
		expected string
	}{{0.003, "0.003"}, {0.005, "0.005"}, {0.1, "0.1"}, {123456.789012345, "123456.789012345"}} {
		users := &leonardoImageQuoteUserReaderFake{user: &User{ID: 42, Balance: test.balance}}
		balance, err := NewLeonardoImageQuoteUserBalanceReader(users).GetAvailableBalanceUSD(context.Background(), 42)
		require.NoError(t, err)
		require.Equal(t, decimal.RequireFromString(test.expected), balance)
	}
}

func TestLeonardoImageQuoteUserBalanceReaderInvalidConfigurationAndID(t *testing.T) {
	var reader *LeonardoImageQuoteUserBalanceReader
	_, err := reader.GetAvailableBalanceUSD(context.Background(), 42)
	require.ErrorIs(t, err, ErrLeonardoImageQuoteBalanceInvalid)
	_, err = NewLeonardoImageQuoteUserBalanceReader(nil).GetAvailableBalanceUSD(context.Background(), 42)
	require.ErrorIs(t, err, ErrLeonardoImageQuoteBalanceInvalid)
	for _, userID := range []int64{0, -1} {
		users := &leonardoImageQuoteUserReaderFake{}
		_, err = NewLeonardoImageQuoteUserBalanceReader(users).GetAvailableBalanceUSD(context.Background(), userID)
		require.ErrorIs(t, err, ErrLeonardoImageQuoteBalanceInvalid)
		require.Zero(t, users.calls)
	}
}

func TestLeonardoImageQuoteUserBalanceReaderCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	users := &leonardoImageQuoteUserReaderFake{}
	_, err := NewLeonardoImageQuoteUserBalanceReader(users).GetAvailableBalanceUSD(ctx, 42)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, users.calls)
}

func TestLeonardoImageQuoteUserBalanceReaderRepositoryErrors(t *testing.T) {
	for _, repositoryErr := range []error{ErrUserNotFound, errors.New("user repository unavailable"), context.DeadlineExceeded} {
		users := &leonardoImageQuoteUserReaderFake{err: repositoryErr}
		_, err := NewLeonardoImageQuoteUserBalanceReader(users).GetAvailableBalanceUSD(context.Background(), 42)
		require.ErrorIs(t, err, repositoryErr)
		require.NotErrorIs(t, err, ErrInsufficientBalance)
		require.NotErrorIs(t, err, ErrLeonardoImageQuoteBalanceInvalid)
		require.Equal(t, 1, users.calls)
	}
}

func TestLeonardoImageQuoteUserBalanceReaderInvalidUser(t *testing.T) {
	for _, user := range []*User{nil, {ID: 43, Balance: 1}, {ID: 42, Balance: math.NaN()}, {ID: 42, Balance: math.Inf(1)}, {ID: 42, Balance: math.Inf(-1)}} {
		users := &leonardoImageQuoteUserReaderFake{user: user}
		_, err := NewLeonardoImageQuoteUserBalanceReader(users).GetAvailableBalanceUSD(context.Background(), 42)
		require.ErrorIs(t, err, ErrLeonardoImageQuoteBalanceInvalid)
		require.Equal(t, 1, users.calls)
	}
}

func TestLeonardoImageQuoteUserBalanceReaderGuardIntegration(t *testing.T) {
	request := quoteRequest("0.005")
	users := &leonardoImageQuoteUserReaderFake{user: &User{ID: 1, Balance: 0.005}}
	guard := NewLeonardoImageQuoteGuard(NewLeonardoImagePriceResolver(), NewLeonardoImageQuoteUserBalanceReader(users))
	quote, err := guard.Prepare(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "0.005", quote.CustomerQuoteUSD.String())
	require.Equal(t, 1, users.calls)

	users = &leonardoImageQuoteUserReaderFake{user: &User{ID: 1, Balance: 0.004999}}
	_, err = NewLeonardoImageQuoteGuard(NewLeonardoImagePriceResolver(), NewLeonardoImageQuoteUserBalanceReader(users)).Prepare(context.Background(), request)
	require.ErrorIs(t, err, ErrInsufficientBalance)

	users = &leonardoImageQuoteUserReaderFake{user: &User{ID: 1, Balance: 1}}
	request = quoteRequest("0.002999")
	_, err = NewLeonardoImageQuoteGuard(NewLeonardoImagePriceResolver(), NewLeonardoImageQuoteUserBalanceReader(users)).Prepare(context.Background(), request)
	require.ErrorIs(t, err, ErrLeonardoImageQuoteBelowEstimatedCost)
	require.Zero(t, users.calls)

	users = &leonardoImageQuoteUserReaderFake{user: &User{ID: 1, Balance: 1}}
	request = quoteRequest("0.005")
	request.PricingRequest.Model = "unknown"
	_, err = NewLeonardoImageQuoteGuard(NewLeonardoImagePriceResolver(), NewLeonardoImageQuoteUserBalanceReader(users)).Prepare(context.Background(), request)
	require.ErrorIs(t, err, ErrLeonardoImagePricingNotFound)
	require.Zero(t, users.calls)
}
