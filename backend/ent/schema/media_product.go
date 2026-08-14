package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type MediaProduct struct {
	ent.Schema
}

func (MediaProduct) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "media_products"}}
}

func (MediaProduct) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (MediaProduct) Fields() []ent.Field {
	return []ent.Field{
		field.String("public_model").NotEmpty().MaxLen(200),
		field.Enum("modality").Values("image", "video"),
		field.Bool("enabled").Default(false),
		field.String("description").Optional().Nillable(),
	}
}

func (MediaProduct) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("prices", MediaProductPrice.Type),
		edge.To("offers", MediaOffer.Type),
		edge.From("groups", Group.Type).Ref("media_products"),
	}
}

func (MediaProduct) Indexes() []ent.Index {
	return []ent.Index{index.Fields("public_model", "modality").Unique(), index.Fields("enabled")}
}
