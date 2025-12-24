CREATE TABLE IF NOT EXISTS staff_services (
    id UUID PRIMARY KEY,
    staff_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    service_name TEXT NOT NULL
);
