package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestLeonardoMediaCreateServiceQuoteValidation(t *testing.T) {
	require.True(t, validLeonardoMediaQuote(decimal.RequireFromString("0.005")))
	require.False(t, validLeonardoMediaQuote(decimal.Zero))
	require.False(t, validLeonardoMediaQuote(decimal.RequireFromString("0.000000001")))
	require.False(t, validLeonardoMediaQuote(decimal.RequireFromString("1000000000000")))
}

func TestLeonardoMediaCreateServiceAccountValidation(t *testing.T) {
	account := &Account{ID: 1, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, GroupIDs: []int64{2}}
	require.True(t, validLeonardoMediaAccount(account, 2))
	require.False(t, validLeonardoMediaAccount(account, 3))
	account.Platform = PlatformOpenAI
	require.False(t, validLeonardoMediaAccount(account, 2))
}
