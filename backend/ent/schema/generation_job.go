package schema

import (
	"fmt"
	"regexp"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

var generationModelSlug = regexp.MustCompile(`^[a-z0-9]+(?:[._/-][a-z0-9]+)*$`)

func validateGenerationModelSlug(value string) error {
	if _, err := uuid.Parse(value); err == nil || !generationModelSlug.MatchString(value) {
		return fmt.Errorf("invalid generation model slug %q", value)
	}
	return nil
}

type GenerationJob struct {
	ent.Schema
}

func (GenerationJob) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "generation_jobs"},
	}
}

func (GenerationJob) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (GenerationJob) Fields() []ent.Field {
	return []ent.Field{
		field.String("public_id").
			NotEmpty().
			MaxLen(64).
			Unique(),
		field.String("provider").
			NotEmpty().
			MaxLen(32),
		field.String("modality").
			NotEmpty().
			MaxLen(32),
		field.String("model").
			NotEmpty().
			MaxLen(200).
			Validate(validateGenerationModelSlug),
		field.String("upstream_model").
			NotEmpty().
			MaxLen(200).
			Validate(validateGenerationModelSlug),
		field.Int64("user_id"),
		field.Int64("api_key_id"),
		field.Int64("group_id").
			Optional().
			Nillable(),
		field.Int64("product_id").Optional().Nillable(),
		field.Int64("offer_id").Optional().Nillable(),
		field.Int64("source_group_id").Optional().Nillable(),
		field.String("operation").Optional().Nillable().MaxLen(32),
		field.String("customer_price_version").Optional().Nillable().MaxLen(64),
		field.Int64("account_id"),
		field.String("upstream_generation_id").
			Optional().
			Nillable().
			MaxLen(128),
		field.Enum("status").
			Values("created", "submitting", "queued", "running", "succeeded", "failed", "cancelled", "unknown").
			Default("created"),
		field.String("upstream_status").
			Optional().
			Nillable().
			MaxLen(64),
		field.String("request_hash").
			NotEmpty().
			MaxLen(128),
		field.JSON("request_payload", map[string]any{}).
			Default(func() map[string]any { return map[string]any{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("result_payload", map[string]any{}).
			Default(func() map[string]any { return map[string]any{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("error_code").
			Optional().
			Nillable().
			MaxLen(64),
		field.String("error_message").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Int("output_count").
			Default(0).
			NonNegative(),
		field.Other("estimated_upstream_cost_amount", &decimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)", dialect.SQLite: "numeric"}),
		field.String("estimated_upstream_cost_unit").Optional().Nillable().MaxLen(32),
		field.String("pricing_snapshot_version").Optional().Nillable().MaxLen(64),
		field.String("pricing_source").Optional().Nillable().MaxLen(128),
		field.String("pricing_match_type").Optional().Nillable().MaxLen(32),
		field.Other("actual_upstream_cost_amount", &decimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)", dialect.SQLite: "numeric"}),
		field.String("actual_upstream_cost_unit").
			Optional().
			Nillable().
			MaxLen(32),
		field.Other("customer_cost", &decimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)", dialect.SQLite: "numeric"}),
		field.Other("gross_margin", &decimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)", dialect.SQLite: "numeric"}),
		field.Other("cost_variance", &decimal.Decimal{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)", dialect.SQLite: "numeric"}),
		field.Enum("billing_status").
			Values("unpriced", "estimated", "reserved", "submitted", "settled", "refunded", "manual_review").
			Default("unpriced"),
		field.String("billing_reference").
			Optional().
			Nillable().
			MaxLen(128),
		field.Int("poll_attempts").
			Default(0).
			NonNegative(),
		field.Time("next_poll_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("last_polled_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("submitted_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("started_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("completed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("failed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (GenerationJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("provider"),
		index.Fields("modality"),
		index.Fields("user_id"),
		index.Fields("api_key_id"),
		index.Fields("group_id"),
		index.Fields("product_id"),
		index.Fields("offer_id"),
		index.Fields("source_group_id"),
		index.Fields("account_id"),
		index.Fields("upstream_generation_id"),
		index.Fields("status"),
		index.Fields("next_poll_at"),
		index.Fields("status", "next_poll_at"),
		index.Fields("billing_reference").Unique(),
		index.Fields("created_at"),
	}
}
