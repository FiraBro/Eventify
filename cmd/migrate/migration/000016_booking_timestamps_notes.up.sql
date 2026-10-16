-- 1. Store booking times as real instants. The old columns were "timestamp without time zone",
--    which silently drops the offset of a value such as 2026-11-20T10:00:00+03:00.
--    Existing values were written as UTC, so they keep their meaning.
ALTER TABLE public.bookings
    ALTER COLUMN scheduled_at TYPE timestamptz USING scheduled_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at   TYPE timestamptz USING created_at   AT TIME ZONE 'UTC',
    ALTER COLUMN updated_at   TYPE timestamptz USING updated_at   AT TIME ZONE 'UTC';

-- 2. notes is never NULL (the API always sends a string; a NULL made the read path fail).
UPDATE public.bookings SET notes = '' WHERE notes IS NULL;
ALTER TABLE public.bookings
    ALTER COLUMN notes SET DEFAULT '',
    ALTER COLUMN notes SET NOT NULL;