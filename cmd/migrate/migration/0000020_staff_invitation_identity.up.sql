-- Keep staff authentication in the existing users identity table. Existing
-- users remain active, while new staff identities can be gated until accepted.
ALTER TABLE users
    ADD COLUMN account_status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    ADD CONSTRAINT users_account_status_check
        CHECK (account_status IN ('ACTIVE', 'PENDING_INVITATION', 'SUSPENDED'));

ALTER TABLE staff
    ADD COLUMN user_id UUID NULL,
    ADD CONSTRAINT staff_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT staff_user_id_key UNIQUE (user_id);

-- Login normalizes email casing; enforce the same invariant under concurrent
-- registration and staff invitation requests.
CREATE UNIQUE INDEX users_email_case_insensitive_key ON users (LOWER(email));

CREATE TABLE staff_invitations (
    id UUID PRIMARY KEY,
    staff_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX staff_invitations_staff_id_idx
    ON staff_invitations (staff_id);

-- Resend revokes the previous invitation before inserting its replacement.
CREATE UNIQUE INDEX staff_invitations_one_open_per_staff_idx
    ON staff_invitations (staff_id)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;
