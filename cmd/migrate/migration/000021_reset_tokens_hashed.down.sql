DROP TABLE IF EXISTS reset_tokens;

CREATE TABLE reset_tokens (
    email VARCHAR(150) NOT NULL,
    otp VARCHAR(10) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    PRIMARY KEY (email, otp)
);
