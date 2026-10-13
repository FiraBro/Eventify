CREATE TABLE IF NOT EXISTS notifications (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        VARCHAR(150) NOT NULL,
    message      TEXT NOT NULL,
    type         VARCHAR(20) NOT NULL DEFAULT 'info'
                 CHECK (type IN ('info','booking','payment','system')),
    reference_id VARCHAR(64) NOT NULL DEFAULT '',
    is_read      BOOLEAN NOT NULL DEFAULT false,
    read_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread  ON notifications(user_id) WHERE is_read = false;