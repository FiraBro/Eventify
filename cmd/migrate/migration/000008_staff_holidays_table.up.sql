CREATE TABLE IF NOT EXISTS staff_holidays (
    id UUID PRIMARY KEY,
    staff_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    reason TEXT
);
