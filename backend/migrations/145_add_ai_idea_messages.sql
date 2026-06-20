-- 轻量 AI 创业想法留言板
CREATE TABLE IF NOT EXISTS ai_idea_messages (
    id BIGSERIAL PRIMARY KEY,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    author_name VARCHAR(120) NOT NULL,
    title VARCHAR(120) NOT NULL,
    content TEXT NOT NULL,
    admin_reply TEXT DEFAULT NULL,
    admin_reply_by BIGINT DEFAULT NULL REFERENCES users(id) ON DELETE SET NULL,
    admin_reply_at TIMESTAMPTZ DEFAULT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    CONSTRAINT chk_ai_idea_messages_status CHECK (status IN ('active', 'user_deleted', 'admin_deleted'))
);

CREATE INDEX IF NOT EXISTS idx_ai_idea_messages_status_created_at
    ON ai_idea_messages(status, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ai_idea_messages_author_id_created_at
    ON ai_idea_messages(author_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

COMMENT ON TABLE ai_idea_messages IS '联系页 AI 创业想法留言板';
COMMENT ON COLUMN ai_idea_messages.author_name IS '作者名快照，减少列表读取时的关联开销';
COMMENT ON COLUMN ai_idea_messages.admin_reply IS '管理员单条官方回复';
COMMENT ON COLUMN ai_idea_messages.status IS 'active, user_deleted, admin_deleted';
