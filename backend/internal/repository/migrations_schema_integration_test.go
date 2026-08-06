//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate(t *testing.T) {
	tx := testTx(t)

	// Re-apply migrations to verify idempotency (no errors, no duplicate rows).
	require.NoError(t, ApplyMigrations(context.Background(), integrationDB))

	// schema_migrations should have at least the current migration set.
	var applied int
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations").Scan(&applied))
	require.GreaterOrEqual(t, applied, 7, "expected schema_migrations to contain applied migrations")

	// users: columns required by repository queries
	requireColumn(t, tx, "users", "username", "character varying", 100, false)
	requireColumn(t, tx, "users", "notes", "text", 0, false)

	// accounts: schedulable and rate-limit fields
	requireColumn(t, tx, "accounts", "notes", "text", 0, true)
	requireColumn(t, tx, "accounts", "schedulable", "boolean", 0, false)
	requireColumn(t, tx, "accounts", "rate_limited_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "rate_limit_reset_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "overload_until", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "session_window_status", "character varying", 20, true)

	// api_keys: key length should be 128
	requireColumn(t, tx, "api_keys", "key", "character varying", 128, false)

	// redeem_codes: subscription fields
	requireColumn(t, tx, "redeem_codes", "group_id", "bigint", 0, true)
	requireColumn(t, tx, "redeem_codes", "validity_days", "integer", 0, false)

	// usage_logs: billing_type used by filters/stats
	requireColumn(t, tx, "usage_logs", "billing_type", "smallint", 0, false)
	requireColumn(t, tx, "usage_logs", "request_type", "smallint", 0, false)
	requireColumn(t, tx, "usage_logs", "openai_ws_mode", "boolean", 0, false)
	requireColumn(t, tx, "usage_logs", "image_input_size", "character varying", 32, true)
	requireColumn(t, tx, "usage_logs", "image_output_size", "character varying", 32, true)
	requireColumn(t, tx, "usage_logs", "image_size_source", "character varying", 16, true)
	requireColumn(t, tx, "usage_logs", "image_size_breakdown", "jsonb", 0, true)
	requireConstraintDefinitionContains(
		t,
		tx,
		"usage_logs",
		"usage_logs_image_size_source_check",
		"image_size_source",
		"'output'",
		"'input'",
		"'default'",
		"'legacy'",
	)
	requireConstraintDefinitionContains(
		t,
		tx,
		"usage_logs",
		"usage_logs_image_billing_size_check",
		"image_count",
		"image_size IS NOT NULL",
		"'1K'",
		"'2K'",
		"'4K'",
		"'mixed'",
	)

	// usage_billing_dedup: billing idempotency narrow table
	var usageBillingDedupRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.usage_billing_dedup')").Scan(&usageBillingDedupRegclass))
	require.True(t, usageBillingDedupRegclass.Valid, "expected usage_billing_dedup table to exist")
	requireColumn(t, tx, "usage_billing_dedup", "request_fingerprint", "character varying", 64, false)
	requireIndex(t, tx, "usage_billing_dedup", "idx_usage_billing_dedup_request_api_key")
	requireIndex(t, tx, "usage_billing_dedup", "idx_usage_billing_dedup_created_at_brin")

	var usageBillingDedupArchiveRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.usage_billing_dedup_archive')").Scan(&usageBillingDedupArchiveRegclass))
	require.True(t, usageBillingDedupArchiveRegclass.Valid, "expected usage_billing_dedup_archive table to exist")
	requireColumn(t, tx, "usage_billing_dedup_archive", "request_fingerprint", "character varying", 64, false)
	requireIndex(t, tx, "usage_billing_dedup_archive", "usage_billing_dedup_archive_pkey")

	// settings table should exist
	var settingsRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.settings')").Scan(&settingsRegclass))
	require.True(t, settingsRegclass.Valid, "expected settings table to exist")

	// security_secrets table should exist
	var securitySecretsRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.security_secrets')").Scan(&securitySecretsRegclass))
	require.True(t, securitySecretsRegclass.Valid, "expected security_secrets table to exist")

	// user_allowed_groups table should exist
	var uagRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.user_allowed_groups')").Scan(&uagRegclass))
	require.True(t, uagRegclass.Valid, "expected user_allowed_groups table to exist")

	// user_subscriptions: deleted_at for soft delete support (migration 012)
	requireColumn(t, tx, "user_subscriptions", "deleted_at", "timestamp with time zone", 0, true)

	// orphan_allowed_groups_audit table should exist (migration 013)
	var orphanAuditRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.orphan_allowed_groups_audit')").Scan(&orphanAuditRegclass))
	require.True(t, orphanAuditRegclass.Valid, "expected orphan_allowed_groups_audit table to exist")

	// account_groups: created_at should be timestamptz
	requireColumn(t, tx, "account_groups", "created_at", "timestamp with time zone", 0, false)

	// user_allowed_groups: created_at should be timestamptz
	requireColumn(t, tx, "user_allowed_groups", "created_at", "timestamp with time zone", 0, false)
}

func TestMigrationsRunner_AuthIdentityAndPaymentSchemaStayAligned(t *testing.T) {
	tx := testTx(t)

	requireColumn(t, tx, "auth_identity_migration_reports", "report_type", "character varying", 80, false)
	requireColumn(t, tx, "users", "signup_source", "character varying", 20, false)
	requireColumnDefaultContains(t, tx, "users", "signup_source", "email")
	requireConstraintDefinitionContains(
		t,
		tx,
		"users",
		"users_signup_source_check",
		"signup_source",
		"'email'",
		"'linuxdo'",
		"'wechat'",
		"'oidc'",
	)

	requireForeignKeyOnDelete(t, tx, "auth_identities", "user_id", "users", "CASCADE")
	requireForeignKeyOnDelete(t, tx, "auth_identity_channels", "identity_id", "auth_identities", "CASCADE")
	requireForeignKeyOnDelete(t, tx, "pending_auth_sessions", "target_user_id", "users", "SET NULL")
	requireForeignKeyOnDelete(t, tx, "identity_adoption_decisions", "pending_auth_session_id", "pending_auth_sessions", "CASCADE")
	requireForeignKeyOnDelete(t, tx, "identity_adoption_decisions", "identity_id", "auth_identities", "SET NULL")

	requireIndex(t, tx, "payment_orders", "paymentorder_out_trade_no")
	requirePartialUniqueIndexDefinition(t, tx, "payment_orders", "paymentorder_out_trade_no", "out_trade_no", "WHERE")
	requireIndexAbsent(t, tx, "payment_orders", "paymentorder_out_trade_no_unique")
}

func TestMigrationsRunner_GenerationJobsSchemaAndRepositoryStayAligned(t *testing.T) {
	require.NoError(t, ApplyMigrations(context.Background(), integrationDB))
	require.NoError(t, ApplyMigrations(context.Background(), integrationDB))
	tx := testTx(t)
	prefix := fmt.Sprintf("generation-job-schema-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(context.Background(), "DELETE FROM generation_jobs WHERE public_id LIKE $1", prefix+"%")
		require.NoError(t, err)
	})

	var applied int
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations WHERE filename = '153_create_generation_jobs.sql'").Scan(&applied))
	require.Equal(t, 1, applied)
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations WHERE filename = '154_generation_job_pricing_snapshot.sql'").Scan(&applied))
	require.Equal(t, 1, applied)
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations WHERE filename = '155_leonardo_cost_variance_alert.sql'").Scan(&applied))
	require.Equal(t, 1, applied)

	requireColumn(t, tx, "generation_jobs", "id", "bigint", 0, false)
	requireColumn(t, tx, "generation_jobs", "created_at", "timestamp with time zone", 0, false)
	requireColumn(t, tx, "generation_jobs", "updated_at", "timestamp with time zone", 0, false)
	requireColumn(t, tx, "generation_jobs", "public_id", "character varying", 64, false)
	requireColumn(t, tx, "generation_jobs", "provider", "character varying", 32, false)
	requireColumn(t, tx, "generation_jobs", "modality", "character varying", 32, false)
	requireColumn(t, tx, "generation_jobs", "model", "character varying", 200, false)
	requireColumn(t, tx, "generation_jobs", "upstream_model", "character varying", 200, false)
	requireColumn(t, tx, "generation_jobs", "user_id", "bigint", 0, false)
	requireColumn(t, tx, "generation_jobs", "api_key_id", "bigint", 0, false)
	requireColumn(t, tx, "generation_jobs", "group_id", "bigint", 0, true)
	requireColumn(t, tx, "generation_jobs", "account_id", "bigint", 0, false)
	requireColumn(t, tx, "generation_jobs", "upstream_generation_id", "character varying", 128, true)
	requireColumn(t, tx, "generation_jobs", "status", "character varying", 16, false)
	requireColumn(t, tx, "generation_jobs", "upstream_status", "character varying", 64, true)
	requireColumn(t, tx, "generation_jobs", "request_hash", "character varying", 128, false)
	requireColumn(t, tx, "generation_jobs", "request_payload", "jsonb", 0, false)
	requireColumn(t, tx, "generation_jobs", "result_payload", "jsonb", 0, false)
	requireColumn(t, tx, "generation_jobs", "error_code", "character varying", 64, true)
	requireColumn(t, tx, "generation_jobs", "error_message", "text", 0, true)
	requireColumn(t, tx, "generation_jobs", "output_count", "integer", 0, false)
	requireNumericColumn(t, tx, "generation_jobs", "estimated_upstream_cost_amount", 20, 10, true)
	requireColumn(t, tx, "generation_jobs", "estimated_upstream_cost_unit", "character varying", 32, true)
	requireColumn(t, tx, "generation_jobs", "pricing_snapshot_version", "character varying", 64, true)
	requireColumn(t, tx, "generation_jobs", "pricing_source", "character varying", 128, true)
	requireColumn(t, tx, "generation_jobs", "pricing_match_type", "character varying", 32, true)
	requireNumericColumn(t, tx, "generation_jobs", "actual_upstream_cost_amount", 20, 10, true)
	requireColumn(t, tx, "generation_jobs", "actual_upstream_cost_unit", "character varying", 32, true)
	requireNumericColumn(t, tx, "generation_jobs", "customer_cost", 20, 10, true)
	requireNumericColumn(t, tx, "generation_jobs", "gross_margin", 20, 10, true)
	requireNumericColumn(t, tx, "generation_jobs", "cost_variance", 20, 10, true)
	requireColumn(t, tx, "generation_jobs", "billing_status", "character varying", 16, false)
	requireColumn(t, tx, "generation_jobs", "billing_reference", "character varying", 128, true)
	requireColumn(t, tx, "generation_jobs", "poll_attempts", "integer", 0, false)
	requireColumn(t, tx, "generation_jobs", "next_poll_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "generation_jobs", "last_polled_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "generation_jobs", "submitted_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "generation_jobs", "started_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "generation_jobs", "completed_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "generation_jobs", "failed_at", "timestamp with time zone", 0, true)
	requireColumnDefaultContains(t, tx, "generation_jobs", "id", "nextval")
	requireColumnDefaultContains(t, tx, "generation_jobs", "created_at", "now()")
	requireColumnDefaultContains(t, tx, "generation_jobs", "updated_at", "now()")
	requireColumnDefaultContains(t, tx, "generation_jobs", "status", "created")
	requireColumnDefaultContains(t, tx, "generation_jobs", "request_payload", "{}")
	requireColumnDefaultContains(t, tx, "generation_jobs", "result_payload", "{}")
	requireColumnDefaultContains(t, tx, "generation_jobs", "output_count", "0")
	requireColumnDefaultContains(t, tx, "generation_jobs", "billing_status", "unpriced")
	requireColumnDefaultContains(t, tx, "generation_jobs", "poll_attempts", "0")
	requireConstraintDefinitionContains(t, tx, "generation_jobs", "generation_jobs_status_check", "created", "submitting", "queued", "running", "succeeded", "failed", "cancelled", "unknown")
	requireConstraintDefinitionContains(t, tx, "generation_jobs", "generation_jobs_billing_status_check", "unpriced", "estimated", "reserved", "submitted", "settled", "refunded", "manual_review")
	requireConstraintDefinitionContains(t, tx, "generation_jobs", "generation_jobs_output_count_check", "output_count >= 0")
	requireConstraintDefinitionContains(t, tx, "generation_jobs", "generation_jobs_poll_attempts_check", "poll_attempts >= 0")
	requirePartialUniqueIndexDefinition(t, tx, "generation_jobs", "generation_jobs_public_id_key", "public_id")
	requirePartialUniqueIndexDefinition(t, tx, "generation_jobs", "generationjob_billing_reference", "billing_reference")

	defaultPublicID := prefix + "-defaults"
	_, err := integrationDB.ExecContext(context.Background(), `
INSERT INTO generation_jobs (
	public_id, provider, modality, model, upstream_model, user_id, api_key_id, account_id, request_hash
) VALUES ($1, 'leonardo', 'image', 'flux-schnell', 'flux-schnell', 1, 2, 3, 'defaults-hash')
`, defaultPublicID)
	require.NoError(t, err)
	var status, requestPayload, resultPayload, billingStatus string
	var outputCount, pollAttempts int
	var optionalValuesAreNull bool
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT status, request_payload::text, result_payload::text, output_count, billing_status, poll_attempts,
	group_id IS NULL AND upstream_generation_id IS NULL AND upstream_status IS NULL AND
	error_code IS NULL AND error_message IS NULL AND estimated_upstream_cost_amount IS NULL AND
	estimated_upstream_cost_unit IS NULL AND pricing_snapshot_version IS NULL AND pricing_source IS NULL AND
	pricing_match_type IS NULL AND actual_upstream_cost_amount IS NULL AND actual_upstream_cost_unit IS NULL AND
	customer_cost IS NULL AND gross_margin IS NULL AND cost_variance IS NULL AND billing_reference IS NULL AND
	next_poll_at IS NULL AND last_polled_at IS NULL AND submitted_at IS NULL AND
	started_at IS NULL AND completed_at IS NULL AND failed_at IS NULL
FROM generation_jobs WHERE public_id = $1
`, defaultPublicID).Scan(&status, &requestPayload, &resultPayload, &outputCount, &billingStatus, &pollAttempts, &optionalValuesAreNull))
	require.Equal(t, "created", status)
	require.Equal(t, "{}", requestPayload)
	require.Equal(t, "{}", resultPayload)
	require.Zero(t, outputCount)
	require.Equal(t, "unpriced", billingStatus)
	require.Zero(t, pollAttempts)
	require.True(t, optionalValuesAreNull)
	nullBillingPublicIDs := []string{prefix + "-null-billing-first", prefix + "-null-billing-second"}
	for _, publicID := range nullBillingPublicIDs {
		_, err = integrationDB.ExecContext(context.Background(), `
INSERT INTO generation_jobs (
	public_id, provider, modality, model, upstream_model, user_id, api_key_id, account_id, request_hash
) VALUES ($1, 'leonardo', 'image', 'flux-schnell', 'flux-schnell', 1, 2, 3, 'null-billing-hash')
`, publicID)
		require.NoError(t, err)
	}
	var nullBillingRows int
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
SELECT COUNT(*) FROM generation_jobs
WHERE public_id IN ($1, $2) AND billing_reference IS NULL
`, nullBillingPublicIDs[0], nullBillingPublicIDs[1]).Scan(&nullBillingRows))
	require.Equal(t, 2, nullBillingRows)

	_, err = integrationDB.ExecContext(context.Background(), `
INSERT INTO generation_jobs (
	public_id, provider, modality, model, upstream_model, user_id, api_key_id,
	account_id, status, request_hash, output_count, billing_status, poll_attempts
) VALUES ($1, 'leonardo', 'image', 'flux-schnell', 'flux-schnell', 1, 2, 3, 'invalid', 'hash', 0, 'unpriced', 0)
`, prefix+"-invalid-status")
	require.Error(t, err)
	_, err = integrationDB.ExecContext(context.Background(), `
INSERT INTO generation_jobs (
	public_id, provider, modality, model, upstream_model, user_id, api_key_id,
	account_id, status, request_hash, output_count, billing_status, poll_attempts
) VALUES ($1, 'leonardo', 'image', 'flux-schnell', 'flux-schnell', 1, 2, 3, 'created', 'hash', -1, 'unpriced', 0)
`, prefix+"-negative-output")
	require.Error(t, err)
	_, err = integrationDB.ExecContext(context.Background(), `
INSERT INTO generation_jobs (
	public_id, provider, modality, model, upstream_model, user_id, api_key_id,
	account_id, status, request_hash, output_count, billing_status, poll_attempts
) VALUES ($1, 'leonardo', 'image', 'flux-schnell', 'flux-schnell', 1, 2, 3, 'created', 'hash', 0, 'invalid', 0)
`, prefix+"-invalid-billing")
	require.Error(t, err)
	_, err = integrationDB.ExecContext(context.Background(), `
INSERT INTO generation_jobs (
	public_id, provider, modality, model, upstream_model, user_id, api_key_id,
	account_id, status, request_hash, output_count, billing_status, poll_attempts
) VALUES ($1, 'leonardo', 'image', 'flux-schnell', 'flux-schnell', 1, 2, 3, 'created', 'hash', 0, 'unpriced', -1)
`, prefix+"-negative-poll")
	require.Error(t, err)

	for _, index := range []string{
		"generationjob_provider",
		"generationjob_modality",
		"generationjob_user_id",
		"generationjob_api_key_id",
		"generationjob_group_id",
		"generationjob_account_id",
		"generationjob_upstream_generation_id",
		"generationjob_status",
		"generationjob_next_poll_at",
		"generationjob_status_next_poll_at",
		"generationjob_billing_reference",
		"generationjob_created_at",
	} {
		requireIndex(t, tx, "generation_jobs", index)
	}

	repo := NewGenerationJobRepository(testEntClient(t), integrationDB)
	actualCost := decimal.RequireFromString("0.0030000000")
	customerCost := decimal.RequireFromString("0.0045000000")
	groupID := int64(4)
	upstreamID := "upstream-generation-schema"
	upstreamStatus := "COMPLETE"
	errorCode := "none"
	errorMessage := "none"
	unit := "USD"
	billingReference := prefix + "-billing"
	now := time.Now().UTC().Truncate(time.Microsecond)
	publicID := prefix + "-repository"
	job := &service.GenerationJob{
		PublicID:                 publicID,
		Provider:                 service.PlatformLeonardo,
		Modality:                 "image",
		Model:                    "flux-schnell",
		UpstreamModel:            "flux-schnell",
		UserID:                   1,
		APIKeyID:                 2,
		GroupID:                  &groupID,
		AccountID:                3,
		UpstreamGenerationID:     &upstreamID,
		Status:                   service.GenerationJobStatusSucceeded,
		UpstreamStatus:           &upstreamStatus,
		RequestHash:              "generation-job-schema-request-hash",
		RequestPayload:           map[string]any{"prompt": "schema smoke"},
		ResultPayload:            map[string]any{"state": "succeeded"},
		ErrorCode:                &errorCode,
		ErrorMessage:             &errorMessage,
		OutputCount:              1,
		ActualUpstreamCostAmount: &actualCost,
		ActualUpstreamCostUnit:   &unit,
		CustomerCost:             &customerCost,
		BillingStatus:            service.GenerationJobBillingStatusSubmitted,
		BillingReference:         &billingReference,
		PollAttempts:             2,
		NextPollAt:               &now,
		LastPolledAt:             &now,
		SubmittedAt:              &now,
		StartedAt:                &now,
		CompletedAt:              &now,
		FailedAt:                 &now,
	}
	require.NoError(t, repo.Create(context.Background(), job))
	stored, err := repo.GetByPublicID(context.Background(), publicID)
	require.NoError(t, err)
	require.Equal(t, job.PublicID, stored.PublicID)
	require.Equal(t, job.Provider, stored.Provider)
	require.Equal(t, job.Modality, stored.Modality)
	require.Equal(t, job.Model, stored.Model)
	require.Equal(t, job.UpstreamModel, stored.UpstreamModel)
	require.Equal(t, job.UserID, stored.UserID)
	require.Equal(t, job.APIKeyID, stored.APIKeyID)
	require.Equal(t, job.GroupID, stored.GroupID)
	require.Equal(t, job.AccountID, stored.AccountID)
	require.Equal(t, job.UpstreamGenerationID, stored.UpstreamGenerationID)
	require.Equal(t, job.Status, stored.Status)
	require.Equal(t, job.UpstreamStatus, stored.UpstreamStatus)
	require.Equal(t, job.RequestHash, stored.RequestHash)
	require.Equal(t, job.RequestPayload, stored.RequestPayload)
	require.Equal(t, job.ResultPayload, stored.ResultPayload)
	require.Equal(t, job.ErrorCode, stored.ErrorCode)
	require.Equal(t, job.ErrorMessage, stored.ErrorMessage)
	require.Equal(t, job.OutputCount, stored.OutputCount)
	require.Equal(t, actualCost.String(), stored.ActualUpstreamCostAmount.String())
	require.Equal(t, job.ActualUpstreamCostUnit, stored.ActualUpstreamCostUnit)
	require.Equal(t, customerCost.String(), stored.CustomerCost.String())
	require.Equal(t, job.BillingStatus, stored.BillingStatus)
	require.Equal(t, job.BillingReference, stored.BillingReference)
	require.Equal(t, job.PollAttempts, stored.PollAttempts)
	require.True(t, job.NextPollAt.Equal(*stored.NextPollAt))
	require.True(t, job.LastPolledAt.Equal(*stored.LastPolledAt))
	require.True(t, job.SubmittedAt.Equal(*stored.SubmittedAt))
	require.True(t, job.StartedAt.Equal(*stored.StartedAt))
	require.True(t, job.CompletedAt.Equal(*stored.CompletedAt))
	require.True(t, job.FailedAt.Equal(*stored.FailedAt))
	require.False(t, stored.CreatedAt.IsZero())
	require.False(t, stored.UpdatedAt.IsZero())

	duplicate := *job
	duplicate.ID = 0
	require.Error(t, repo.Create(context.Background(), &duplicate))
	billingDuplicate := *job
	billingDuplicate.ID = 0
	billingDuplicate.PublicID = prefix + "-billing-duplicate"
	require.Error(t, repo.Create(context.Background(), &billingDuplicate))

	settled := *stored
	settled.BillingStatus = service.GenerationJobBillingStatusSettled
	require.NoError(t, repo.CompareAndSwapStatus(context.Background(), publicID, service.GenerationJobStatusSucceeded, &settled))
	require.Equal(t, service.GenerationJobStatusSucceeded, settled.Status)
	require.Equal(t, service.GenerationJobBillingStatusSettled, settled.BillingStatus)
	stored, err = repo.GetByPublicID(context.Background(), publicID)
	require.NoError(t, err)
	require.Equal(t, service.GenerationJobStatusSucceeded, stored.Status)
	require.Equal(t, service.GenerationJobBillingStatusSettled, stored.BillingStatus)
}

func requireIndex(t *testing.T, tx *sql.Tx, table, index string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM pg_indexes
	WHERE schemaname = 'public'
	  AND tablename = $1
	  AND indexname = $2
)
`, table, index).Scan(&exists)
	require.NoError(t, err, "query pg_indexes for %s.%s", table, index)
	require.True(t, exists, "expected index %s on %s", index, table)
}

func requireIndexAbsent(t *testing.T, tx *sql.Tx, table, index string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM pg_indexes
	WHERE schemaname = 'public'
	  AND tablename = $1
	  AND indexname = $2
)
`, table, index).Scan(&exists)
	require.NoError(t, err, "query pg_indexes for %s.%s", table, index)
	require.False(t, exists, "expected index %s on %s to be absent", index, table)
}

func requirePartialUniqueIndexDefinition(t *testing.T, tx *sql.Tx, table, index string, fragments ...string) {
	t.Helper()

	var (
		unique bool
		def    string
	)

	err := tx.QueryRowContext(context.Background(), `
SELECT
	i.indisunique,
	pg_get_indexdef(i.indexrelid)
FROM pg_class idx
JOIN pg_index i ON i.indexrelid = idx.oid
JOIN pg_class tbl ON tbl.oid = i.indrelid
JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
WHERE ns.nspname = 'public'
  AND tbl.relname = $1
  AND idx.relname = $2
`, table, index).Scan(&unique, &def)
	require.NoError(t, err, "query index definition for %s.%s", table, index)
	require.True(t, unique, "expected index %s on %s to be unique", index, table)

	for _, fragment := range fragments {
		require.Contains(t, def, fragment, "expected index definition for %s.%s to contain %q", table, index, fragment)
	}
}

func requireForeignKeyOnDelete(t *testing.T, tx *sql.Tx, table, column, refTable, expected string) {
	t.Helper()

	var actual string
	err := tx.QueryRowContext(context.Background(), `
SELECT CASE c.confdeltype
	WHEN 'a' THEN 'NO ACTION'
	WHEN 'r' THEN 'RESTRICT'
	WHEN 'c' THEN 'CASCADE'
	WHEN 'n' THEN 'SET NULL'
	WHEN 'd' THEN 'SET DEFAULT'
END
FROM pg_constraint c
JOIN pg_class tbl ON tbl.oid = c.conrelid
JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
JOIN pg_class ref_tbl ON ref_tbl.oid = c.confrelid
JOIN pg_attribute attr ON attr.attrelid = tbl.oid AND attr.attnum = ANY(c.conkey)
WHERE ns.nspname = 'public'
  AND c.contype = 'f'
  AND tbl.relname = $1
  AND attr.attname = $2
  AND ref_tbl.relname = $3
LIMIT 1
`, table, column, refTable).Scan(&actual)
	require.NoError(t, err, "query foreign key action for %s.%s -> %s", table, column, refTable)
	require.Equal(t, expected, actual, "unexpected ON DELETE action for %s.%s -> %s", table, column, refTable)
}

func requireConstraintDefinitionContains(t *testing.T, tx *sql.Tx, table, constraint string, fragments ...string) {
	t.Helper()

	var def string
	err := tx.QueryRowContext(context.Background(), `
SELECT pg_get_constraintdef(c.oid)
FROM pg_constraint c
JOIN pg_class tbl ON tbl.oid = c.conrelid
JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
WHERE ns.nspname = 'public'
  AND tbl.relname = $1
  AND c.conname = $2
`, table, constraint).Scan(&def)
	require.NoError(t, err, "query constraint definition for %s.%s", table, constraint)

	for _, fragment := range fragments {
		require.Contains(t, def, fragment, "expected constraint definition for %s.%s to contain %q", table, constraint, fragment)
	}
}

func requireColumnDefaultContains(t *testing.T, tx *sql.Tx, table, column string, fragments ...string) {
	t.Helper()

	var columnDefault sql.NullString
	err := tx.QueryRowContext(context.Background(), `
SELECT column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = $1
  AND column_name = $2
`, table, column).Scan(&columnDefault)
	require.NoError(t, err, "query column_default for %s.%s", table, column)
	require.True(t, columnDefault.Valid, "expected column_default for %s.%s", table, column)

	for _, fragment := range fragments {
		require.Contains(t, columnDefault.String, fragment, "expected default for %s.%s to contain %q", table, column, fragment)
	}
}

func requireColumn(t *testing.T, tx *sql.Tx, table, column, dataType string, maxLen int, nullable bool) {
	t.Helper()

	var row struct {
		DataType string
		MaxLen   sql.NullInt64
		Nullable string
	}

	err := tx.QueryRowContext(context.Background(), `
SELECT
  data_type,
  character_maximum_length,
  is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = $1
  AND column_name = $2
`, table, column).Scan(&row.DataType, &row.MaxLen, &row.Nullable)
	require.NoError(t, err, "query information_schema.columns for %s.%s", table, column)
	require.Equal(t, dataType, row.DataType, "data_type mismatch for %s.%s", table, column)

	if maxLen > 0 {
		require.True(t, row.MaxLen.Valid, "expected maxLen for %s.%s", table, column)
		require.Equal(t, int64(maxLen), row.MaxLen.Int64, "maxLen mismatch for %s.%s", table, column)
	}

	if nullable {
		require.Equal(t, "YES", row.Nullable, "nullable mismatch for %s.%s", table, column)
	} else {
		require.Equal(t, "NO", row.Nullable, "nullable mismatch for %s.%s", table, column)
	}
}

func requireNumericColumn(t *testing.T, tx *sql.Tx, table, column string, precision, scale int64, nullable bool) {
	t.Helper()

	var row struct {
		DataType  string
		Precision sql.NullInt64
		Scale     sql.NullInt64
		Nullable  string
	}
	err := tx.QueryRowContext(context.Background(), `
SELECT data_type, numeric_precision, numeric_scale, is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = $1
  AND column_name = $2
`, table, column).Scan(&row.DataType, &row.Precision, &row.Scale, &row.Nullable)
	require.NoError(t, err, "query numeric column for %s.%s", table, column)
	require.Equal(t, "numeric", row.DataType)
	require.Equal(t, precision, row.Precision.Int64)
	require.Equal(t, scale, row.Scale.Int64)
	if nullable {
		require.Equal(t, "YES", row.Nullable)
	} else {
		require.Equal(t, "NO", row.Nullable)
	}
}
