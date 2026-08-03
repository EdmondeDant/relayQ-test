package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

var (
	ErrLeonardoMediaCreateNotConfigured       = infraerrors.InternalServer("LEONARDO_MEDIA_CREATE_NOT_CONFIGURED", "Leonardo media create service is not configured")
	ErrLeonardoMediaCreateInputInvalid        = infraerrors.BadRequest("LEONARDO_MEDIA_CREATE_INPUT_INVALID", "Leonardo media create input is invalid")
	ErrLeonardoMediaNoAvailableAccount        = infraerrors.ServiceUnavailable("LEONARDO_MEDIA_NO_AVAILABLE_ACCOUNT", "no available Leonardo account")
	ErrLeonardoMediaAccountSelectionAmbiguous = infraerrors.ServiceUnavailable("LEONARDO_MEDIA_ACCOUNT_SELECTION_AMBIGUOUS", "multiple Leonardo accounts require scheduler selection")
	ErrLeonardoMediaCreateResultInvalid       = infraerrors.InternalServer("LEONARDO_MEDIA_CREATE_RESULT_INVALID", "Leonardo media create result is invalid")
)

var leonardoMediaMaxQuote = decimal.RequireFromString("999999999999.99999999")

type LeonardoMediaCreateInput struct {
	IdempotencyKey   string
	UserID           int64
	APIKeyID         int64
	GroupID          int64
	Model            string
	Prompt           string
	Public           bool
	Width            int
	Height           int
	Quantity         int
	CustomerQuoteUSD decimal.Decimal
}

type LeonardoMediaCreateResult struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Modality  string `json:"modality"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type LeonardoMediaCreateService struct {
	accounts     AccountRepository
	orchestrator *LeonardoImageCreateOrchestrator
}

func NewLeonardoMediaCreateService(accounts AccountRepository, orchestrator *LeonardoImageCreateOrchestrator) *LeonardoMediaCreateService {
	return &LeonardoMediaCreateService{accounts: accounts, orchestrator: orchestrator}
}

func (s *LeonardoMediaCreateService) Create(ctx context.Context, input LeonardoMediaCreateInput) (*LeonardoMediaCreateResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.accounts == nil || s.orchestrator == nil {
		return nil, ErrLeonardoMediaCreateNotConfigured
	}
	model, prompt := strings.TrimSpace(input.Model), strings.TrimSpace(input.Prompt)
	if input.UserID <= 0 || input.APIKeyID <= 0 || input.GroupID <= 0 || model == "" || prompt == "" || len(prompt) > 4000 || input.Width <= 0 || input.Height <= 0 || input.Quantity <= 0 || !validLeonardoMediaQuote(input.CustomerQuoteUSD) {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	key, err := NormalizeIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	publicSum := sha256.Sum256([]byte("leonardo_media_create\n" + strconv.FormatInt(input.UserID, 10) + "\n" + key))
	publicID := "gen_rq_" + hex.EncodeToString(publicSum[:16])
	fingerprint, err := json.Marshal(struct {
		Model            string `json:"model"`
		Modality         string `json:"modality"`
		Prompt           string `json:"prompt"`
		Public           bool   `json:"public"`
		Width            int    `json:"width"`
		Height           int    `json:"height"`
		Quantity         int    `json:"quantity"`
		CustomerQuoteUSD string `json:"customer_quote_usd"`
	}{model, "image", prompt, input.Public, input.Width, input.Height, input.Quantity, input.CustomerQuoteUSD.String()})
	if err != nil {
		return nil, ErrLeonardoMediaCreateInputInvalid
	}
	hash := sha256.Sum256(fingerprint)
	accounts, err := s.accounts.ListSchedulableByGroupIDAndPlatform(ctx, input.GroupID, PlatformLeonardo)
	if err != nil {
		return nil, err
	}
	valid := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if validLeonardoMediaAccount(&account, input.GroupID) {
			valid = append(valid, account)
		}
	}
	if len(valid) == 0 {
		return nil, ErrLeonardoMediaNoAvailableAccount
	}
	if len(valid) > 1 {
		return nil, ErrLeonardoMediaAccountSelectionAmbiguous
	}
	job, err := s.orchestrator.Create(ctx, LeonardoImageCreateRequest{PublicID: publicID, UserID: input.UserID, APIKeyID: input.APIKeyID, GroupID: &input.GroupID, AccountID: valid[0].ID, RequestHash: hex.EncodeToString(hash[:]), Model: model, Prompt: prompt, Width: input.Width, Height: input.Height, Quantity: input.Quantity, Public: input.Public, CustomerQuoteUSD: input.CustomerQuoteUSD})
	if err != nil {
		return nil, err
	}
	if job == nil || strings.TrimSpace(job.PublicID) == "" {
		return nil, ErrLeonardoMediaCreateResultInvalid
	}
	createdAt := int64(0)
	if !job.CreatedAt.IsZero() {
		createdAt = job.CreatedAt.Unix()
	}
	return &LeonardoMediaCreateResult{ID: job.PublicID, Object: "media.generation", Provider: PlatformLeonardo, Model: job.Model, Modality: "image", Status: string(job.Status), CreatedAt: createdAt}, nil
}

func validLeonardoMediaQuote(value decimal.Decimal) bool {
	return value.Sign() > 0 && value.Exponent() >= -8 && value.Cmp(leonardoMediaMaxQuote) <= 0
}

func validLeonardoMediaAccount(account *Account, groupID int64) bool {
	if account == nil || account.ID <= 0 || account.Platform != PlatformLeonardo || account.Type != AccountTypeAPIKey || !account.IsSchedulable() {
		return false
	}
	for _, id := range account.GroupIDs {
		if id == groupID {
			return true
		}
	}
	for _, relation := range account.AccountGroups {
		if relation.GroupID == groupID {
			return true
		}
	}
	for _, group := range account.Groups {
		if group != nil && group.ID == groupID {
			return true
		}
	}
	return false
}
