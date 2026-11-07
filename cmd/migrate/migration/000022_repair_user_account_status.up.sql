-- Repair databases where the earlier invitation migration was recorded as
-- applied but its users.account_status change is missing.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS account_status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_account_status_check'
          AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_account_status_check
            CHECK (account_status IN ('ACTIVE', 'PENDING_INVITATION', 'SUSPENDED'));
    END IF;
END $$;
