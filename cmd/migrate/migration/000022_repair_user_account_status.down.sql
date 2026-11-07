ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_account_status_check,
    DROP COLUMN IF EXISTS account_status;
