-- 000004_create_reset_tokens_table.up.sql
CREATE TABLE reset_tokens (
    email VARCHAR(150) NOT NULL,
    otp VARCHAR(10) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    PRIMARY KEY (email, otp)
);

-- 000004_create_reset_tokens_table.down.sql
