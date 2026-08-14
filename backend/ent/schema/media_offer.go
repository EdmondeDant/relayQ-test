package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type MediaOffer struct {
	ent.Schema
}

func (MediaOffer) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "media_offers"}}
}

func (MediaOffer) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (MediaOffer) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("product_id"),
		field.String("provider").NotEmpty().MaxLen(32),
		field.Int64("source_group_id"),
		field.String("upstream_model").NotEmpty().MaxLen(200),
		field.Bool("enabled").Default(true),
		field.Int("priority").Default(0).NonNegative(),
		field.JSON("operations", []string{}).Default(func() []string { return []string{} }).SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("capabilities", map[string]any{}).Default(func() map[string]any { return map[string]any{} }).SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("cost_rules", map[string]any{}).Default(func() map[string]any { return map[string]any{} }).SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.String("cost_source").NotEmpty().MaxLen(500),
		field.String("cost_version").NotEmpty().MaxLen(64),
		field.Time("verified_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("expires_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (MediaOffer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("product", MediaProduct.Type).Ref("offers").Field("product_id").Required().Unique(),
		edge.From("source_group", Group.Type).Ref("media_source_offers").Field("source_group_id").Required().Unique(),
	}
}

func (MediaOffer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product_id", "provider", "source_group_id", "upstream_model").Unique(),
		index.Fields("product_id", "enabled", "expires_at", "priority"),
		index.Fields("source_group_id"),
	}
}
