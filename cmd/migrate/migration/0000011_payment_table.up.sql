CREATE TABLE IF NOT EXISTS payments (
    id               UUID PRIMARY KEY,
    booking_id       UUID NOT NULL REFERENCES bookings(id) ON DELETE RESTRICT,
    user_id          UUID NOT NULL REFERENCES users(id)    ON DELETE RESTRICT,
    tx_ref           VARCHAR(100) NOT NULL UNIQUE,
    chapa_reference  VARCHAR(100) NOT NULL DEFAULT '',
    amount           NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    currency         VARCHAR(3)   NOT NULL DEFAULT 'ETB',
    status           VARCHAR(20)  NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','paid','failed','refund_pending','refunded')),
    checkout_url     TEXT NOT NULL DEFAULT '',
    refund_reference VARCHAR(100) NOT NULL DEFAULT '',
    refund_reason    TEXT NOT NULL DEFAULT '',
    paid_at          TIMESTAMPTZ,
    refunded_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payments_booking_id ON payments(booking_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id    ON payments(user_id);

-- A booking can have at most ONE live payment (prevents double checkout / double charge)
CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_one_active_per_booking
    ON payments(booking_id) WHERE status IN ('pending','paid','refund_pending');

-- Helpful indexes for the report date filters
CREATE INDEX IF NOT EXISTS idx_payments_paid_at      ON payments(paid_at);
CREATE INDEX IF NOT EXISTS idx_payments_refunded_at  ON payments(refunded_at);
