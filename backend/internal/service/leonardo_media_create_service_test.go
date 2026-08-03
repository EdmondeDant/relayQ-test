package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type leonardoMediaCreateAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (s *leonardoMediaCreateAccountRepoStub) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, _ string) ([]Account, error) {
	return s.accounts, nil
}

func TestLeonardoMediaCreateServiceQuoteValidation(t *testing.T) {
	require.True(t, validLeonardoMediaQuote(decimal.RequireFromString("0.005")))
	require.False(t, validLeonardoMediaQuote(decimal.Zero))
	require.False(t, validLeonardoMediaQuote(decimal.RequireFromString("0.000000001")))
	require.False(t, validLeonardoMediaQuote(decimal.RequireFromString("1000000000000")))
}

func TestLeonardoMediaCreateServiceAccountValidation(t *testing.T) {
	groupID := int64(2)
	valid := Account{ID: 1, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "leo-key"}, Status: StatusActive, Schedulable: true, GroupIDs: []int64{groupID}}
	tests := []struct {
		name    string
		account *Account
		want    bool
	}{
		{name: "group ids", account: &valid, want: true},
		{name: "account groups", account: func() *Account {
			v := valid
			v.GroupIDs = nil
			v.AccountGroups = []AccountGroup{{GroupID: groupID}}
			return &v
		}(), want: true},
		{name: "groups", account: func() *Account { v := valid; v.GroupIDs = nil; v.Groups = []*Group{{ID: groupID}}; return &v }(), want: true},
		{name: "nil", account: nil},
		{name: "missing id", account: func() *Account { v := valid; v.ID = 0; return &v }()},
		{name: "wrong platform", account: func() *Account { v := valid; v.Platform = PlatformOpenAI; return &v }()},
		{name: "literal api_key account type", account: func() *Account { v := valid; v.Type = "api_key"; return &v }()},
		{name: "disabled", account: func() *Account { v := valid; v.Status = StatusDisabled; return &v }()},
		{name: "unschedulable", account: func() *Account { v := valid; v.Schedulable = false; return &v }()},
		{name: "wrong group", account: func() *Account { v := valid; v.GroupIDs = []int64{groupID + 1}; return &v }()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, validLeonardoMediaAccount(test.account, groupID))
		})
	}
}

func TestLeonardoMediaCreateServiceAccountSelectionMatrix(t *testing.T) {
	valid := Account{ID: 1, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "leo-key"}, Status: StatusActive, Schedulable: true, GroupIDs: []int64{2}}
	literalAPIKey := valid
	literalAPIKey.Type = "api_key"
	input := LeonardoMediaCreateInput{IdempotencyKey: "leo-300a8", UserID: 1, APIKeyID: 1, GroupID: 2, Model: "flux-schnell", Prompt: "cat", Width: 896, Height: 896, Quantity: 1, CustomerQuoteUSD: decimal.RequireFromString("0.005")}
	tests := []struct {
		name     string
		accounts []Account
		wantErr  error
	}{
		{name: "no candidates", wantErr: ErrLeonardoMediaNoAvailableAccount},
		{name: "literal api_key rejected", accounts: []Account{literalAPIKey}, wantErr: ErrLeonardoMediaNoAvailableAccount},
		{name: "one valid among literal api_key", accounts: []Account{literalAPIKey, valid}, wantErr: ErrLeonardoImageCreateNotConfigured},
		{name: "two valid among literal api_key", accounts: []Account{literalAPIKey, valid, func() Account { v := valid; v.ID = 2; return v }()}, wantErr: ErrLeonardoMediaAccountSelectionAmbiguous},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewLeonardoMediaCreateService(&leonardoMediaCreateAccountRepoStub{accounts: test.accounts}, &LeonardoImageCreateOrchestrator{})
			_, err := service.Create(context.Background(), input)
			require.True(t, errors.Is(err, test.wantErr), "got %v", err)
		})
	}
}
