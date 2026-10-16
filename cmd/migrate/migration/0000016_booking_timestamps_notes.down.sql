ALTER TABLE public.bookings
    ALTER COLUMN notes DROP NOT NULL,
    ALTER COLUMN notes DROP DEFAULT;

ALTER TABLE public.bookings
    ALTER COLUMN scheduled_at TYPE timestamp USING scheduled_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at   TYPE timestamp USING created_at   AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at   TYPE timestamp USING updated_at   AT TIME ZONE 'UTC';