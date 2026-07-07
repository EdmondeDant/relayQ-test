package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type RetailGrokService struct {
	keyRepo   RetailGrokKeyRepository
	usageRepo RetailGrokUsageLogRepository
	groupRepo GroupRepository
}

func NewRetailGrokService(
	keyRepo RetailGrokKeyRepository,
	usageRepo RetailGrokUsageLogRepository,
	groupRepo GroupRepository,
) *RetailGrokService {
	return &RetailGrokService{
		keyRepo:   keyRepo,
		usageRepo: usageRepo,
		groupRepo: groupRepo,
	}
}

func (s *RetailGrokService) GenerateKey(prefix string) (string, error) {
	if strings.TrimSpace(prefix) == "" {
		prefix = "rgk-"
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate retail key: %w", err)
	}
	return prefix + hex.EncodeToString(buf), nil
}

func (s *RetailGrokService) BatchGenerate(ctx context.Context, adminID int64, req RetailGrokBatchGenerateRequest) (*RetailGrokBatchGenerateResult, error) {
	if req.GroupID <= 0 {
		return nil, ErrRetailGrokGroupRequired
	}
	if req.Count <= 0 || req.Count > 200 {
		return nil, ErrRetailGrokInvalidBatchSize
	}
	if _, err := s.groupRepo.GetByID(ctx, req.GroupID); err != nil {
		return nil, err
	}

	namePrefix := strings.TrimSpace(req.NamePrefix)
	if namePrefix == "" {
		namePrefix = "grok-retail"
	}

	var expiresAt *time.Time
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		value := time.Now().Add(time.Duration(*req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &value
	}

	keys := make([]RetailGrokKey, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		keyValue, err := s.GenerateKey("rgk-")
		if err != nil {
			return nil, err
		}
		key := &RetailGrokKey{
			Key:              keyValue,
			Name:             fmt.Sprintf("%s-%03d", namePrefix, i+1),
			Status:           RetailGrokKeyStatusActive,
			GroupID:          req.GroupID,
			ExpiresAt:        expiresAt,
			TokenLimitTotal:  req.TokenLimitTotal,
			TokenUsedTotal:   0,
			ImageLimitTotal:  req.ImageLimitTotal,
			ImageUsedTotal:   0,
			VideoLimitTotal:  req.VideoLimitTotal,
			VideoUsedTotal:   0,
			CreatedByAdminID: adminID,
		}
		if err := s.keyRepo.Create(ctx, key); err != nil {
			return nil, err
		}
		keys = append(keys, *key)
	}

	return &RetailGrokBatchGenerateResult{Keys: keys}, nil
}

func (s *RetailGrokService) Authenticate(ctx context.Context, rawKey string, allowUsage bool) (*RetailGrokKey, error) {
	key, err := s.keyRepo.GetByKey(ctx, strings.TrimSpace(rawKey))
	if err != nil {
		return nil, err
	}
	if err := key.ValidateAvailability(time.Now(), allowUsage); err != nil {
		return nil, err
	}
	return key, nil
}

func (s *RetailGrokService) GetUsageSummary(ctx context.Context, keyID int64, limit int) (*RetailGrokUsageSummary, error) {
	key, err := s.keyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	logs, err := s.usageRepo.ListByKeyID(ctx, keyID, limit)
	if err != nil {
		return nil, err
	}
	return &RetailGrokUsageSummary{
		Key:        key,
		RecentLogs: logs,
	}, nil
}

func (s *RetailGrokService) ListKeys(ctx context.Context, limit int) ([]RetailGrokKey, error) {
	return s.keyRepo.List(ctx, limit)
}

func (s *RetailGrokService) DeleteKey(ctx context.Context, id int64) error {
	return s.keyRepo.Delete(ctx, id)
}

func (s *RetailGrokService) RecordUsage(ctx context.Context, key *RetailGrokKey, log *RetailGrokUsageLog) error {
	if key == nil || log == nil {
		return nil
	}
	log.RetailGrokKeyID = key.ID
	now := time.Now()
	log.CompletedAt = &now
	if log.TotalTokens == 0 {
		log.TotalTokens = log.InputTokens + log.OutputTokens
	}
	if err := s.usageRepo.Create(ctx, log); err != nil {
		return err
	}
	return s.keyRepo.IncrementUsage(ctx, key.ID, log.TotalTokens, log.ImageCount, log.VideoCount)
}

func (s *RetailGrokService) RecordFailure(ctx context.Context, key *RetailGrokKey, log *RetailGrokUsageLog) error {
	if key == nil || log == nil {
		return nil
	}
	log.RetailGrokKeyID = key.ID
	now := time.Now()
	log.CompletedAt = &now
	return s.usageRepo.Create(ctx, log)
}
