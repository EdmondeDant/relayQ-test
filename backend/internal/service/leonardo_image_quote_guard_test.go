package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type leonardoImageQuotePriceResolverFake struct {
	estimate *LeonardoImagePriceEstimate
	err      error
	calls    int
}

func (f *leonardoImageQuotePriceResolverFake) Estimate(context.Context, LeonardoImagePriceRequest) (*LeonardoImagePriceEstimate, error) {
	f.calls++
	return f.estimate, f.err
}

type leonardoImageQuoteBalanceReaderFake struct {
	balance decimal.Decimal
	err     error
	calls   int
	userIDs []int64
}

func (f *leonardoImageQuoteBalanceReaderFake) GetAvailableBalanceUSD(_ context.Context, userID int64) (decimal.Decimal, error) {
	f.calls++
	f.userIDs = append(f.userIDs, userID)
	return f.balance, f.err
}

func TestLeonardoImageQuoteGuardSuccess(t *testing.T) {
	resolver := quoteResolverFake()
	balance := &leonardoImageQuoteBalanceReaderFake{balance: quoteDecimal("1")}
	quote, err := NewLeonardoImageQuoteGuard(resolver, balance).Prepare(context.Background(), quoteRequest("0.005"))
	require.NoError(t, err)
	require.Equal(t, "0.003", quote.EstimatedUpstreamCostUSD.String())
	require.Equal(t, "0.005", quote.CustomerQuoteUSD.String())
	require.Equal(t, "2026-08-01", quote.PricingVersion)
	require.Equal(t, "leonardo_authenticated_pricing_calculator", quote.PricingSource)
	require.Equal(t, "exact", quote.MatchType)
	require.Equal(t, 1, resolver.calls)
	require.Equal(t, 1, balance.calls)
	require.Equal(t, []int64{1}, balance.userIDs)
}

func TestLeonardoImageQuoteGuardEqualBoundaries(t *testing.T) {
	quote, err := NewLeonardoImageQuoteGuard(quoteResolverFake(), &leonardoImageQuoteBalanceReaderFake{balance: quoteDecimal("0.003")}).Prepare(context.Background(), quoteRequest("0.003"))
	require.NoError(t, err)
	require.Equal(t, "0.003", quote.CustomerQuoteUSD.String())
}

func TestLeonardoImageQuoteGuardBelowEstimatedCost(t *testing.T) {
	balance := &leonardoImageQuoteBalanceReaderFake{balance: quoteDecimal("1")}
	quote, err := NewLeonardoImageQuoteGuard(quoteResolverFake(), balance).Prepare(context.Background(), quoteRequest("0.002999"))
	require.Nil(t, quote)
	require.ErrorIs(t, err, ErrLeonardoImageQuoteBelowEstimatedCost)
	require.Zero(t, balance.calls)
}

func TestLeonardoImageQuoteGuardInsufficientBalance(t *testing.T) {
	for _, available := range []string{"0.004999", "0", "-0.001"} {
		balance := &leonardoImageQuoteBalanceReaderFake{balance: quoteDecimal(available)}
		quote, err := NewLeonardoImageQuoteGuard(quoteResolverFake(), balance).Prepare(context.Background(), quoteRequest("0.005"))
		require.Nil(t, quote)
		require.ErrorIs(t, err, ErrInsufficientBalance)
		require.Equal(t, 1, balance.calls)
	}
}

func TestLeonardoImageQuoteGuardInvalidRequest(t *testing.T) {
	for _, request := range []LeonardoImageQuoteRequest{
		quoteRequest("0"),
		quoteRequest("-0.001"),
		func() LeonardoImageQuoteRequest { r := quoteRequest("0.005"); r.UserID = 0; return r }(),
		func() LeonardoImageQuoteRequest { r := quoteRequest("0.005"); r.UserID = -1; return r }(),
	} {
		resolver := quoteResolverFake()
		balance := &leonardoImageQuoteBalanceReaderFake{}
		quote, err := NewLeonardoImageQuoteGuard(resolver, balance).Prepare(context.Background(), request)
		require.Nil(t, quote)
		require.ErrorIs(t, err, ErrLeonardoImageQuoteInvalid)
		require.Zero(t, resolver.calls)
		require.Zero(t, balance.calls)
	}
}

func TestLeonardoImageQuoteGuardPricingErrors(t *testing.T) {
	for _, pricingErr := range []error{ErrLeonardoImagePricingNotFound, ErrLeonardoImagePricingRequestInvalid} {
		resolver := &leonardoImageQuotePriceResolverFake{err: pricingErr}
		balance := &leonardoImageQuoteBalanceReaderFake{}
		quote, err := NewLeonardoImageQuoteGuard(resolver, balance).Prepare(context.Background(), quoteRequest("0.005"))
		require.Nil(t, quote)
		require.ErrorIs(t, err, pricingErr)
		require.Zero(t, balance.calls)
	}
}

func TestLeonardoImageQuoteGuardInvalidEstimate(t *testing.T) {
	estimates := []*LeonardoImagePriceEstimate{
		nil,
		quoteEstimate("0", "2026-08-01", "source", "exact"),
		quoteEstimate("-0.001", "2026-08-01", "source", "exact"),
		quoteEstimate("0.003", " ", "source", "exact"),
		quoteEstimate("0.003", "2026-08-01", " ", "exact"),
		quoteEstimate("0.003", "2026-08-01", "source", " "),
	}
	for _, estimate := range estimates {
		balance := &leonardoImageQuoteBalanceReaderFake{}
		quote, err := NewLeonardoImageQuoteGuard(&leonardoImageQuotePriceResolverFake{estimate: estimate}, balance).Prepare(context.Background(), quoteRequest("0.005"))
		require.Nil(t, quote)
		require.ErrorIs(t, err, ErrLeonardoImageQuoteInvalid)
		require.Zero(t, balance.calls)
	}
}

func TestLeonardoImageQuoteGuardBalanceReadFailure(t *testing.T) {
	balanceErr := errors.New("balance backend unavailable")
	quote, err := NewLeonardoImageQuoteGuard(quoteResolverFake(), &leonardoImageQuoteBalanceReaderFake{err: balanceErr}).Prepare(context.Background(), quoteRequest("0.005"))
	require.Nil(t, quote)
	require.ErrorIs(t, err, balanceErr)
	require.NotErrorIs(t, err, ErrInsufficientBalance)
}

func TestLeonardoImageQuoteGuardCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resolver := quoteResolverFake()
	balance := &leonardoImageQuoteBalanceReaderFake{}
	quote, err := NewLeonardoImageQuoteGuard(resolver, balance).Prepare(ctx, quoteRequest("0.005"))
	require.Nil(t, quote)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, resolver.calls)
	require.Zero(t, balance.calls)
}

func TestLeonardoImageQuoteGuardConfiguration(t *testing.T) {
	var guard *LeonardoImageQuoteGuard
	_, err := guard.Prepare(context.Background(), quoteRequest("0.005"))
	require.ErrorIs(t, err, ErrLeonardoImageQuoteInvalid)
	_, err = NewLeonardoImageQuoteGuard(nil, &leonardoImageQuoteBalanceReaderFake{}).Prepare(context.Background(), quoteRequest("0.005"))
	require.ErrorIs(t, err, ErrLeonardoImageQuoteInvalid)
	_, err = NewLeonardoImageQuoteGuard(quoteResolverFake(), nil).Prepare(context.Background(), quoteRequest("0.005"))
	require.ErrorIs(t, err, ErrLeonardoImageQuoteInvalid)
}

func TestLeonardoImageQuoteGuardIndependentResults(t *testing.T) {
	guard := NewLeonardoImageQuoteGuard(quoteResolverFake(), &leonardoImageQuoteBalanceReaderFake{balance: quoteDecimal("1")})
	first, err := guard.Prepare(context.Background(), quoteRequest("0.005"))
	require.NoError(t, err)
	second, err := guard.Prepare(context.Background(), quoteRequest("0.005"))
	require.NoError(t, err)
	require.NotSame(t, first, second)
	first.PricingVersion = "changed"
	require.Equal(t, "2026-08-01", second.PricingVersion)
}

func TestLeonardoImageQuoteGuardDecimalPrecision(t *testing.T) {
	quote := "0.003000000000000001"
	_, err := NewLeonardoImageQuoteGuard(quoteResolverFake(), &leonardoImageQuoteBalanceReaderFake{balance: quoteDecimal(quote)}).Prepare(context.Background(), quoteRequest(quote))
	require.NoError(t, err)
	_, err = NewLeonardoImageQuoteGuard(quoteResolverFake(), &leonardoImageQuoteBalanceReaderFake{balance: quoteDecimal("0.003000000000000000")}).Prepare(context.Background(), quoteRequest(quote))
	require.ErrorIs(t, err, ErrInsufficientBalance)
}

func quoteResolverFake() *leonardoImageQuotePriceResolverFake {
	return &leonardoImageQuotePriceResolverFake{estimate: quoteEstimate("0.003", "2026-08-01", "leonardo_authenticated_pricing_calculator", "exact")}
}

func quoteEstimate(cost, version, source, match string) *LeonardoImagePriceEstimate {
	return &LeonardoImagePriceEstimate{EstimatedCostUSD: quoteDecimal(cost), PricingVersion: version, PricingSource: source, MatchType: match}
}

func quoteRequest(customerQuote string) LeonardoImageQuoteRequest {
	return LeonardoImageQuoteRequest{UserID: 1, PricingRequest: leonardoImagePriceRequest(), CustomerQuoteUSD: quoteDecimal(customerQuote)}
}

func quoteDecimal(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}
