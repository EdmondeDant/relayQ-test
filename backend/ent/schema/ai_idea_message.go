package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// IdeaMessage holds the schema definition for the AI idea board message entity.
type IdeaMessage struct {
	ent.Schema
}

func (IdeaMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ai_idea_messages"},
	}
}

func (IdeaMessage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (IdeaMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("author_id").
			Positive().
			Comment("留言作者用户ID"),
		field.String("author_name").
			MaxLen(120).
			NotEmpty().
			Comment("作者名快照"),
		field.String("title").
			MaxLen(120).
			NotEmpty().
			Comment("留言标题"),
		field.String("content").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			NotEmpty().
			Comment("留言正文"),
		field.String("admin_reply").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable().
			Comment("管理员单条官方回复"),
		field.Int64("admin_reply_by").
			Optional().
			Nillable().
			Comment("回复管理员用户ID"),
		field.Time("admin_reply_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("管理员回复时间"),
		field.String("status").
			MaxLen(20).
			Default("active").
			Comment("状态: active, user_deleted, admin_deleted"),
	}
}

func (IdeaMessage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "created_at"),
		index.Fields("author_id", "created_at"),
	}
}
