CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMP NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'cancelled', 'completed')),
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings (user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_service_id ON bookings (service_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings (status);

-- Needed by GET /reports/staff-performance.
-- Nullable on purpose: existing bookings stay valid, and nothing sets it yet.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS staff_id UUID;
CREATE INDEX IF NOT EXISTS idx_bookings_staff_id ON bookings(staff_id);

-- Helpful indexes for the report date filters
CREATE INDEX IF NOT EXISTS idx_bookings_scheduled_at ON bookings(scheduled_at);
