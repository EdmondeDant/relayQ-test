package service

import (
	"context"
	"testing"
)

type mediaLeaseAccountRepo struct {
	AccountRepository
	accounts []Account
}

func (r *mediaLeaseAccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return r.accounts, nil
}

func TestMediaAccountLeaserSelectsPriorityAndModel(t *testing.T) {
	unsupported := Account{ID: 1, Platform: PlatformLeonardo, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Priority: 0}
	preferred := Account{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Priority: 1}
	fallback := Account{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Priority: 2}
	leaser := NewMediaAccountLeaser(&mediaLeaseAccountRepo{accounts: []Account{fallback, unsupported, preferred}}, nil)

	lease, err := leaser.Acquire(context.Background(), 1, PlatformOpenAI, "image-model", nil)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Account.ID != preferred.ID {
		t.Fatalf("account = %d", lease.Account.ID)
	}
	lease.Release()
	lease.Release()
}
