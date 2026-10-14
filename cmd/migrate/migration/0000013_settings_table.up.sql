CREATE TABLE IF NOT EXISTS settings (
    id                        SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),  -- single-row table
    business_name             VARCHAR(100) NOT NULL DEFAULT 'My Business',
    contact_email             VARCHAR(150) NOT NULL DEFAULT '',
    contact_phone             VARCHAR(30)  NOT NULL DEFAULT '',
    currency                  VARCHAR(3)   NOT NULL DEFAULT 'ETB',
    timezone                  VARCHAR(64)  NOT NULL DEFAULT 'UTC',
    min_booking_notice_hours  INT NOT NULL DEFAULT 1  CHECK (min_booking_notice_hours >= 0),
    max_booking_advance_days  INT NOT NULL DEFAULT 90 CHECK (max_booking_advance_days >= 1),
    cancellation_window_hours INT NOT NULL DEFAULT 24 CHECK (cancellation_window_hours >= 0),
    updated_by                UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;