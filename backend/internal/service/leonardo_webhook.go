package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

var (
	ErrLeonardoWebhookUnauthorized  = errors.New("leonardo webhook is unauthorized")
	ErrLeonardoWebhookInvalid       = errors.New("leonardo webhook is invalid")
	ErrLeonardoWebhookNotConfigured = errors.New("leonardo webhook is not configured")
)

type LeonardoWebhookEvent struct {
	ID                   int64
	AccountID            int64
	EventKey             string
	EventType            string
	UpstreamGenerationID string
	Payload              []byte
	AttemptCount         int
}

type LeonardoWebhookEventRepository interface {
	CreatePending(context.Context, *LeonardoWebhookEvent) (bool, error)
	ClaimPending(context.Context, int) ([]*LeonardoWebhookEvent, error)
	MarkProcessed(context.Context, int64) error
	MarkFailed(context.Context, int64, int) error
}

type LeonardoWebhookService struct {
	accounts AccountRepository
	events   LeonardoWebhookEventRepository
}

func NewLeonardoWebhookService(accounts AccountRepository, events LeonardoWebhookEventRepository) *LeonardoWebhookService {
	return &LeonardoWebhookService{accounts: accounts, events: events}
}

func (s *LeonardoWebhookService) Receive(ctx context.Context, accountID int64, routeToken, authorization string, body []byte) (bool, error) {
	if s == nil || s.accounts == nil || s.events == nil {
		return false, ErrLeonardoWebhookNotConfigured
	}
	if accountID <= 0 || len(body) == 0 || !json.Valid(body) {
		return false, ErrLeonardoWebhookInvalid
	}
	account, err := s.accounts.GetByID(ctx, accountID)
	if err != nil || account == nil || account.Platform != PlatformLeonardo {
		return false, ErrLeonardoWebhookUnauthorized
	}
	expectedRoute, _ := account.Extra["webhook_route_token"].(string)
	expectedSecret, _ := account.Extra["webhook_secret"].(string)
	providedSecret := ""
	if scheme, value, ok := strings.Cut(strings.TrimSpace(authorization), " "); ok && strings.EqualFold(scheme, "Bearer") {
		providedSecret = strings.TrimSpace(value)
	}
	if !constantTimeStringEqual(strings.TrimSpace(routeToken), strings.TrimSpace(expectedRoute)) || !constantTimeStringEqual(providedSecret, strings.TrimSpace(expectedSecret)) {
		return false, ErrLeonardoWebhookUnauthorized
	}
	var payload map[string]any
	if err = json.Unmarshal(body, &payload); err != nil {
		return false, ErrLeonardoWebhookInvalid
	}
	eventType, _ := payload["type"].(string)
	upstreamID := leonardoWebhookGenerationID(payload)
	redacted, err := json.Marshal(redactLeonardoWebhookValue(payload))
	if err != nil {
		return false, ErrLeonardoWebhookInvalid
	}
	sum := sha256.Sum256(body)
	event := &LeonardoWebhookEvent{AccountID: accountID, EventKey: hex.EncodeToString(sum[:]), EventType: strings.TrimSpace(eventType), UpstreamGenerationID: upstreamID, Payload: redacted}
	created, err := s.events.CreatePending(ctx, event)
	return !created, err
}

func constantTimeStringEqual(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	leftHash, rightHash := sha256.Sum256([]byte(left)), sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftHash[:], rightHash[:]) == 1
}

func leonardoWebhookGenerationID(payload map[string]any) string {
	data, _ := payload["data"].(map[string]any)
	object, _ := data["object"].(map[string]any)
	id, _ := object["id"].(string)
	return strings.TrimSpace(id)
}

func redactLeonardoWebhookValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "authorization") || strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") {
				out[key] = "[REDACTED]"
			} else {
				out[key] = redactLeonardoWebhookValue(item)
			}
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			out[index] = redactLeonardoWebhookValue(item)
		}
		return out
	default:
		return value
	}
}
