package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/shopspring/decimal"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type MediaProductPrice struct {
	ent.Schema
}

func (MediaProductPrice) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "media_product_prices"}}
}

func (MediaProductPrice) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (MediaProductPrice) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("product_id"),
		field.String("operation").NotEmpty().MaxLen(32),
		field.String("spec_key").NotEmpty().MaxLen(500),
		field.Other("unit_price_usd", &decimal.Decimal{}).SchemaType(map[string]string{dialect.Postgres: "decimal(20,10)", dialect.SQLite: "numeric"}),
		field.String("currency").Default("USD").MaxLen(8),
		field.String("version").NotEmpty().MaxLen(64),
		field.Bool("enabled").Default(true),
	}
}

func (MediaProductPrice) Edges() []ent.Edge {
	return []ent.Edge{edge.From("product", MediaProduct.Type).Ref("prices").Field("product_id").Required().Unique()}
}

func (MediaProductPrice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product_id", "operation", "spec_key", "version").Unique(),
		index.Fields("product_id", "operation", "spec_key", "enabled"),
	}
}
