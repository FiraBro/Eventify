CREATE TABLE IF NOT EXISTS activity_logs (
    id          UUID PRIMARY KEY,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,  -- logs survive account deletion
    role        VARCHAR(50)  NOT NULL DEFAULT '',
    method      VARCHAR(10)  NOT NULL,
    path        TEXT         NOT NULL,
    route       TEXT         NOT NULL DEFAULT '',
    status_code INT          NOT NULL,
    ip          VARCHAR(45)  NOT NULL DEFAULT '',
    duration_ms INT          NOT NULL DEFAULT 0,
    details     TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at ON activity_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_logs_user       ON activity_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activity_logs_status     ON activity_logs(status_code);