package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newGenerationJobRepoSQLite(t *testing.T) (*generationJobRepository, *dbent.Client) {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", t.Name()))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE generation_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			public_id TEXT NOT NULL UNIQUE,
			provider TEXT NOT NULL,
			modality TEXT NOT NULL,
			model TEXT NOT NULL,
			upstream_model TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			api_key_id INTEGER NOT NULL,
			group_id INTEGER NULL,
			account_id INTEGER NOT NULL,
			upstream_generation_id TEXT NULL,
			status TEXT NOT NULL,
			upstream_status TEXT NULL,
			request_hash TEXT NOT NULL,
			request_payload JSON NOT NULL,
			result_payload JSON NOT NULL,
			error_code TEXT NULL,
			error_message TEXT NULL,
			output_count INTEGER NOT NULL,
			actual_upstream_cost_amount NUMERIC NULL,
			actual_upstream_cost_unit TEXT NULL,
			customer_cost NUMERIC NULL,
			billing_status TEXT NOT NULL,
			billing_reference TEXT NULL,
			poll_attempts INTEGER NOT NULL,
			next_poll_at DATETIME NULL,
			last_polled_at DATETIME NULL,
			submitted_at DATETIME NULL,
			started_at DATETIME NULL,
			completed_at DATETIME NULL,
			failed_at DATETIME NULL
		)
	`)
	require.NoError(t, err)
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })
	return &generationJobRepository{client: client}, client
}

func TestGenerationJobRepositoryCreateAndLookupFullMapping(t *testing.T) {
	repo, _ := newGenerationJobRepoSQLite(t)
	ctx := context.Background()
	groupID := int64(3)
	upstreamID := "upstream-123"
	upstreamStatus := "PENDING"
	errorCode := "waiting"
	errorMessage := "not ready"
	unit := "USD"
	billingReference := "bill-123"
	actualCost := decimal.RequireFromString("0.1234567890")
	customerCost := decimal.RequireFromString("0.2345678901")
	now := time.Now().UTC().Truncate(time.Second)
	job := &service.GenerationJob{
		PublicID:                 "job-full-mapping",
		Provider:                 "leonardo",
		Modality:                 "image",
		Model:                    "flux-schnell",
		UpstreamModel:            "flux-schnell",
		UserID:                   1,
		APIKeyID:                 2,
		GroupID:                  &groupID,
		AccountID:                4,
		UpstreamGenerationID:     &upstreamID,
		Status:                   service.GenerationJobStatusQueued,
		UpstreamStatus:           &upstreamStatus,
		RequestHash:              "request-hash",
		RequestPayload:           map[string]any{"prompt": "cat", "private": true},
		ResultPayload:            map[string]any{"url": "https://example.invalid/image.png"},
		ErrorCode:                &errorCode,
		ErrorMessage:             &errorMessage,
		OutputCount:              1,
		ActualUpstreamCostAmount: &actualCost,
		ActualUpstreamCostUnit:   &unit,
		CustomerCost:             &customerCost,
		BillingStatus:            service.GenerationJobBillingStatusSubmitted,
		BillingReference:         &billingReference,
		PollAttempts:             2,
		NextPollAt:               timePointer(now.Add(time.Minute)),
		LastPolledAt:             timePointer(now.Add(-time.Minute)),
		SubmittedAt:              timePointer(now.Add(-2 * time.Minute)),
		StartedAt:                timePointer(now.Add(-time.Minute)),
		CompletedAt:              timePointer(now.Add(time.Minute)),
		FailedAt:                 timePointer(now.Add(2 * time.Minute)),
	}

	require.NoError(t, repo.Create(ctx, job))
	require.NotZero(t, job.ID)
	require.False(t, job.CreatedAt.IsZero())
	require.False(t, job.UpdatedAt.IsZero())

	byPublicID, err := repo.GetByPublicID(ctx, job.PublicID)
	require.NoError(t, err)
	byUpstreamID, err := repo.GetByUpstreamGenerationID(ctx, upstreamID)
	require.NoError(t, err)
	require.Equal(t, byPublicID, byUpstreamID)
	require.Equal(t, job.PublicID, byPublicID.PublicID)
	require.Equal(t, job.Provider, byPublicID.Provider)
	require.Equal(t, job.Modality, byPublicID.Modality)
	require.Equal(t, job.Model, byPublicID.Model)
	require.Equal(t, job.UpstreamModel, byPublicID.UpstreamModel)
	require.Equal(t, job.UserID, byPublicID.UserID)
	require.Equal(t, job.APIKeyID, byPublicID.APIKeyID)
	require.Equal(t, job.GroupID, byPublicID.GroupID)
	require.Equal(t, job.AccountID, byPublicID.AccountID)
	require.Equal(t, job.UpstreamGenerationID, byPublicID.UpstreamGenerationID)
	require.Equal(t, job.Status, byPublicID.Status)
	require.Equal(t, job.UpstreamStatus, byPublicID.UpstreamStatus)
	require.Equal(t, job.RequestHash, byPublicID.RequestHash)
	require.Equal(t, job.RequestPayload, byPublicID.RequestPayload)
	require.Equal(t, job.ResultPayload, byPublicID.ResultPayload)
	require.Equal(t, job.ErrorCode, byPublicID.ErrorCode)
	require.Equal(t, job.ErrorMessage, byPublicID.ErrorMessage)
	require.Equal(t, job.OutputCount, byPublicID.OutputCount)
	require.Equal(t, actualCost.String(), byPublicID.ActualUpstreamCostAmount.String())
	require.Equal(t, job.ActualUpstreamCostUnit, byPublicID.ActualUpstreamCostUnit)
	require.Equal(t, customerCost.String(), byPublicID.CustomerCost.String())
	require.Equal(t, job.BillingStatus, byPublicID.BillingStatus)
	require.Equal(t, job.BillingReference, byPublicID.BillingReference)
	require.Equal(t, job.PollAttempts, byPublicID.PollAttempts)
	require.True(t, job.NextPollAt.Equal(*byPublicID.NextPollAt))
	require.True(t, job.LastPolledAt.Equal(*byPublicID.LastPolledAt))
	require.True(t, job.SubmittedAt.Equal(*byPublicID.SubmittedAt))
	require.True(t, job.StartedAt.Equal(*byPublicID.StartedAt))
	require.True(t, job.CompletedAt.Equal(*byPublicID.CompletedAt))
	require.True(t, job.FailedAt.Equal(*byPublicID.FailedAt))
}

func TestGenerationJobRepositoryLookupNotFound(t *testing.T) {
	repo, _ := newGenerationJobRepoSQLite(t)
	ctx := context.Background()

	_, err := repo.GetByPublicID(ctx, "missing")
	require.ErrorIs(t, err, service.ErrGenerationJobNotFound)
	_, err = repo.GetByUpstreamGenerationID(ctx, "missing")
	require.ErrorIs(t, err, service.ErrGenerationJobNotFound)
}

func TestGenerationJobRepositoryCompareAndSwapStatus(t *testing.T) {
	repo, _ := newGenerationJobRepoSQLite(t)
	ctx := context.Background()
	job := minimalGenerationJob("job-cas")
	require.NoError(t, repo.Create(ctx, job))

	upstreamID := "upstream-cas"
	startedAt := time.Now().UTC().Truncate(time.Second)
	update := *job
	update.Status = service.GenerationJobStatusRunning
	update.UpstreamGenerationID = &upstreamID
	update.ResultPayload = map[string]any{"progress": "50"}
	update.PollAttempts = 4
	update.StartedAt = &startedAt
	require.NoError(t, repo.CompareAndSwapStatus(ctx, job.PublicID, service.GenerationJobStatusQueued, &update))
	require.Equal(t, service.GenerationJobStatusRunning, update.Status)
	require.Equal(t, upstreamID, *update.UpstreamGenerationID)
	require.Equal(t, 4, update.PollAttempts)

	stale := update
	stale.Status = service.GenerationJobStatusSucceeded
	err := repo.CompareAndSwapStatus(ctx, job.PublicID, service.GenerationJobStatusQueued, &stale)
	require.ErrorIs(t, err, service.ErrGenerationJobConflict)
	stored, err := repo.GetByPublicID(ctx, job.PublicID)
	require.NoError(t, err)
	require.Equal(t, service.GenerationJobStatusRunning, stored.Status)
	require.Equal(t, map[string]any{"progress": "50"}, stored.ResultPayload)

	missing := update
	missing.Status = service.GenerationJobStatusSucceeded
	err = repo.CompareAndSwapStatus(ctx, "missing", service.GenerationJobStatusRunning, &missing)
	require.ErrorIs(t, err, service.ErrGenerationJobNotFound)
}

func TestGenerationJobRepositoryRejectsTerminalRegression(t *testing.T) {
	repo, _ := newGenerationJobRepoSQLite(t)
	ctx := context.Background()
	job := minimalGenerationJob("job-terminal")
	job.Status = service.GenerationJobStatusSucceeded
	require.NoError(t, repo.Create(ctx, job))

	update := *job
	update.Status = service.GenerationJobStatusRunning
	err := repo.CompareAndSwapStatus(ctx, job.PublicID, service.GenerationJobStatusSucceeded, &update)
	require.ErrorIs(t, err, service.ErrGenerationJobConflict)
}

func TestGenerationJobRepositoryUnknownForcesManualReviewAndNilCosts(t *testing.T) {
	repo, _ := newGenerationJobRepoSQLite(t)
	ctx := context.Background()
	job := minimalGenerationJob("job-unknown")
	job.Status = service.GenerationJobStatusSubmitting
	require.NoError(t, repo.Create(ctx, job))

	cost := decimal.RequireFromString("9.99")
	unit := "USD"
	update := *job
	update.Status = service.GenerationJobStatusUnknown
	update.ActualUpstreamCostAmount = &cost
	update.ActualUpstreamCostUnit = &unit
	update.CustomerCost = &cost
	update.BillingStatus = service.GenerationJobBillingStatusSettled
	require.NoError(t, repo.CompareAndSwapStatus(ctx, job.PublicID, service.GenerationJobStatusSubmitting, &update))

	require.Equal(t, "submission_unknown", *update.ErrorCode)
	require.Equal(t, service.GenerationJobBillingStatusManualReview, update.BillingStatus)
	require.Nil(t, update.ActualUpstreamCostAmount)
	require.Nil(t, update.ActualUpstreamCostUnit)
	require.Nil(t, update.CustomerCost)
}

func minimalGenerationJob(publicID string) *service.GenerationJob {
	return &service.GenerationJob{
		PublicID:       publicID,
		Provider:       "leonardo",
		Modality:       "image",
		Model:          "flux-schnell",
		UpstreamModel:  "flux-schnell",
		UserID:         1,
		APIKeyID:       2,
		AccountID:      3,
		Status:         service.GenerationJobStatusQueued,
		RequestHash:    "request-hash",
		RequestPayload: map[string]any{},
		ResultPayload:  map[string]any{},
		BillingStatus:  service.GenerationJobBillingStatusUnpriced,
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
