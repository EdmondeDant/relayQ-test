package schema

import (
	"testing"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/entc/load"
	"entgo.io/ent/schema/field"
	"github.com/stretchr/testify/require"
)

func TestGenerationJobSchema(t *testing.T) {
	spec, err := (&load.Config{Path: "."}).Load()
	require.NoError(t, err)

	schemas := map[string]*load.Schema{}
	for _, loadedSchema := range spec.Schemas {
		schemas[loadedSchema.Name] = loadedSchema
	}

	generationJob := requireSchema(t, schemas, "GenerationJob")
	requireSchemaFields(t, generationJob,
		"public_id", "provider", "modality", "model", "upstream_model",
		"user_id", "api_key_id", "group_id", "account_id", "upstream_generation_id",
		"status", "upstream_status", "request_hash", "request_payload", "result_payload",
		"error_code", "error_message", "output_count", "estimated_upstream_cost_amount",
		"estimated_upstream_cost_unit", "pricing_snapshot_version", "pricing_source", "pricing_match_type",
		"actual_upstream_cost_amount", "actual_upstream_cost_unit", "customer_cost", "gross_margin",
		"cost_variance", "billing_status", "billing_reference",
		"poll_attempts", "next_poll_at", "last_polled_at", "submitted_at", "started_at",
		"completed_at", "failed_at", "created_at", "updated_at",
	)

	requireFieldEnums(t, generationJob, "status", "created", "submitting", "queued", "running", "succeeded", "failed", "cancelled", "unknown")
	requireFieldEnums(t, generationJob, "billing_status", "unpriced", "estimated", "reserved", "submitted", "settled", "refunded", "manual_review")
	requirePostgresType(t, generationJob, "request_payload", "jsonb")
	requirePostgresType(t, generationJob, "result_payload", "jsonb")
	require.Equal(t, field.TypeJSON, requireSchemaField(t, generationJob, "request_payload").Info.Type)
	require.Equal(t, field.TypeJSON, requireSchemaField(t, generationJob, "result_payload").Info.Type)

	for _, name := range []string{"estimated_upstream_cost_amount", "estimated_upstream_cost_unit", "actual_upstream_cost_amount", "actual_upstream_cost_unit", "customer_cost", "gross_margin", "cost_variance"} {
		schemaField := requireSchemaField(t, generationJob, name)
		require.True(t, schemaField.Optional)
		require.True(t, schemaField.Nillable)
	}
	for _, name := range []string{"estimated_upstream_cost_amount", "actual_upstream_cost_amount", "customer_cost", "gross_margin", "cost_variance"} {
		schemaField := requireSchemaField(t, generationJob, name)
		require.Equal(t, field.TypeOther, schemaField.Info.Type)
		require.Equal(t, "*decimal.Decimal", schemaField.Info.Ident)
		requirePostgresType(t, generationJob, name, "decimal(20,10)")
		require.Equal(t, "numeric", schemaField.SchemaType[dialect.SQLite])
	}

	for _, name := range []string{"next_poll_at", "last_polled_at", "submitted_at", "started_at", "completed_at", "failed_at"} {
		schemaField := requireSchemaField(t, generationJob, name)
		require.True(t, schemaField.Optional)
		require.True(t, schemaField.Nillable)
		require.Equal(t, "timestamptz", schemaField.SchemaType[dialect.Postgres])
	}

	for _, name := range []string{"model", "upstream_model"} {
		requireStringFieldValid(t, GenerationJob{}.Fields(), name, "flux-schnell")
		requireStringFieldValid(t, GenerationJob{}.Fields(), name, "leonardo/flux-schnell")
		requireStringFieldInvalid(t, GenerationJob{}.Fields(), name, "Flux Schnell")
		requireStringFieldInvalid(t, GenerationJob{}.Fields(), name, "")
		requireStringFieldInvalid(t, GenerationJob{}.Fields(), name, "1dd50843-d653-4516-a8e3-f0238ee453ff")
		requireStringFieldInvalid(t, GenerationJob{}.Fields(), name, "00000000-0000-0000-0000-000000000000")
	}

	requireHasUniqueIndex(t, generationJob, "billing_reference")
	for _, fields := range [][]string{
		{"provider"}, {"modality"}, {"user_id"}, {"api_key_id"}, {"group_id"},
		{"account_id"}, {"upstream_generation_id"}, {"status"}, {"next_poll_at"},
		{"status", "next_poll_at"}, {"created_at"},
	} {
		requireHasIndex(t, generationJob, fields...)
	}
}

func requireFieldEnums(t *testing.T, schema *load.Schema, name string, values ...string) {
	t.Helper()

	schemaField := requireSchemaField(t, schema, name)
	require.Equal(t, field.TypeEnum, schemaField.Info.Type)
	actual := make([]string, 0, len(schemaField.Enums))
	for _, enum := range schemaField.Enums {
		actual = append(actual, enum.V)
	}
	require.Equal(t, values, actual)
}

func requirePostgresType(t *testing.T, schema *load.Schema, name, schemaType string) {
	t.Helper()
	require.Equal(t, schemaType, requireSchemaField(t, schema, name).SchemaType[dialect.Postgres])
}

func requireStringFieldValid(t *testing.T, fields []ent.Field, name, value string) {
	t.Helper()
	for _, validator := range requireStringFieldValidators(t, fields, name) {
		require.NoError(t, validator(value))
	}
}

func requireStringFieldInvalid(t *testing.T, fields []ent.Field, name, value string) {
	t.Helper()
	for _, validator := range requireStringFieldValidators(t, fields, name) {
		if validator(value) != nil {
			return
		}
	}
	require.Failf(t, "validation succeeded", "field %s should reject %q", name, value)
}

func requireStringFieldValidators(t *testing.T, fields []ent.Field, name string) []func(string) error {
	t.Helper()
	for _, entField := range fields {
		descriptor := entField.Descriptor()
		if descriptor.Name != name {
			continue
		}
		validators := make([]func(string) error, 0, len(descriptor.Validators))
		for _, rawValidator := range descriptor.Validators {
			validator, ok := rawValidator.(func(string) error)
			require.True(t, ok)
			validators = append(validators, validator)
		}
		require.NotEmpty(t, validators)
		return validators
	}
	require.Failf(t, "missing field validators", "schema should include field %s", name)
	return nil
}

func requireHasIndex(t *testing.T, schema *load.Schema, fields ...string) {
	t.Helper()

	for _, schemaIndex := range schema.Indexes {
		if len(schemaIndex.Fields) != len(fields) {
			continue
		}
		match := true
		for i := range fields {
			if schemaIndex.Fields[i] != fields[i] {
				match = false
				break
			}
		}
		if match {
			return
		}
	}

	require.Failf(t, "missing index", "schema %s should include index on %v", schema.Name, fields)
}
