ALTER TABLE refresh_tokens
    DROP COLUMN IF EXISTS parent_tag,
    DROP COLUMN IF EXISTS is_used;
