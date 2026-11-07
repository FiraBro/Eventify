-- Reset tokens are short-lived, so invalidate any tokens created by the old
-- schema while replacing plaintext OTP storage with one hashed token per email.
DROP TABLE IF EXISTS reset_tokens;

CREATE TABLE reset_tokens (
    email VARCHAR(150) PRIMARY KEY,
    token_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE
);
