package service

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var ErrMediaAccountUnavailable = errors.New("no available media account")

type MediaAccountLease struct {
	Account *Account
	once    sync.Once
	release func()
}

func (l *MediaAccountLease) Release() {
	if l != nil {
		l.once.Do(func() {
			if l.release != nil {
				l.release()
			}
		})
	}
}

type MediaAccountLeaser struct {
	accounts    AccountRepository
	concurrency *ConcurrencyService
}

func NewMediaAccountLeaser(accounts AccountRepository, concurrency *ConcurrencyService) *MediaAccountLeaser {
	return &MediaAccountLeaser{accounts: accounts, concurrency: concurrency}
}

func (s *MediaAccountLeaser) Acquire(ctx context.Context, groupID int64, provider, model string, excluded map[int64]struct{}) (*MediaAccountLease, error) {
	accounts, err := s.accounts.ListSchedulableByGroupIDAndPlatform(ctx, groupID, provider)
	if err != nil {
		return nil, err
	}
	candidates := make([]Account, 0, len(accounts))
	loads := make([]AccountWithConcurrency, 0, len(accounts))
	for i := range accounts {
		if _, skip := excluded[accounts[i].ID]; !skip && accounts[i].IsSchedulable() && accounts[i].IsModelSupported(model) {
			candidates = append(candidates, accounts[i])
			loads = append(loads, AccountWithConcurrency{ID: accounts[i].ID, MaxConcurrency: accounts[i].EffectiveLoadFactor()})
		}
	}
	if len(candidates) == 0 {
		return nil, ErrMediaAccountUnavailable
	}
	loadMap := map[int64]*AccountLoadInfo{}
	if s.concurrency != nil {
		if current, loadErr := s.concurrency.GetAccountsLoadBatch(ctx, loads); loadErr == nil {
			loadMap = current
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := loadMap[candidates[i].ID], loadMap[candidates[j].ID]
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority < candidates[j].Priority
		}
		if left != nil && right != nil && left.LoadRate != right.LoadRate {
			return left.LoadRate < right.LoadRate
		}
		if left != nil && right != nil && left.WaitingCount != right.WaitingCount {
			return left.WaitingCount < right.WaitingCount
		}
		if candidates[i].LastUsedAt != nil && candidates[j].LastUsedAt != nil && !candidates[i].LastUsedAt.Equal(*candidates[j].LastUsedAt) {
			return candidates[i].LastUsedAt.Before(*candidates[j].LastUsedAt)
		}
		return candidates[i].ID < candidates[j].ID
	})
	for i := range candidates {
		if s.concurrency == nil {
			return &MediaAccountLease{Account: &candidates[i]}, nil
		}
		acquired, acquireErr := s.concurrency.AcquireAccountSlot(ctx, candidates[i].ID, candidates[i].Concurrency)
		if acquireErr != nil {
			return nil, acquireErr
		}
		if acquired.Acquired {
			return &MediaAccountLease{Account: &candidates[i], release: acquired.ReleaseFunc}, nil
		}
	}
	return nil, ErrMediaAccountUnavailable
}
