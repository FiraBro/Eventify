DROP TABLE IF EXISTS staff_invitations;

DROP INDEX IF EXISTS users_email_case_insensitive_key;

ALTER TABLE staff
    DROP CONSTRAINT IF EXISTS staff_user_id_key,
    DROP CONSTRAINT IF EXISTS staff_user_id_fkey,
    DROP COLUMN IF EXISTS user_id;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_account_status_check,
    DROP COLUMN IF EXISTS account_status;
