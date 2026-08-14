package service

import "testing"

func TestAPIKeyService_RejectsV11AuthSnapshotWithoutManagedMetadata(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy-models-list", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			Version:  11,
			APIKeyID: 1,
			UserID:   2,
			GroupID:  &groupID,
			Status:   StatusActive,
			User: APIKeyAuthUserSnapshot{
				ID:          2,
				Status:      StatusActive,
				Role:        RoleUser,
				Balance:     10,
				Concurrency: 3,
			},
			Group: &APIKeyAuthGroupSnapshot{
				ID:               groupID,
				Name:             "openai",
				Platform:         PlatformOpenAI,
				Status:           StatusActive,
				SubscriptionType: SubscriptionTypeStandard,
				RateMultiplier:   1,
			},
		},
	})

	if err != nil {
		t.Fatalf("expected stale snapshot to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatalf("expected v11 auth snapshot to be rejected after managed metadata was added")
	}
	if apiKey != nil {
		t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
	}
}

func TestAPIKeyService_AuthSnapshotPreservesManagedMetadata(t *testing.T) {
	purpose := "canvas_bootstrap"
	svc := &APIKeyService{}
	apiKey := &APIKey{
		ID:             1,
		UserID:         2,
		Name:           "Infinite Canvas",
		ClientApp:      "infinite-canvas",
		Managed:        true,
		ManagedPurpose: &purpose,
		Status:         StatusActive,
		User: &User{
			ID:          2,
			Status:      StatusActive,
			Role:        RoleUser,
			Balance:     10,
			Concurrency: 3,
		},
	}

	restored := svc.snapshotToAPIKey("canvas-key", svc.snapshotFromAPIKey(t.Context(), apiKey))
	if !IsCanvasAPIKey(restored) {
		t.Fatalf("expected managed Canvas metadata to survive auth cache round trip, got %#v", restored)
	}
}
