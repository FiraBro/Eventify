-- A booked slot is the half-open range [scheduled_at, end_at).
-- Existing services get the approved default of 60 minutes.
ALTER TABLE public.services
    ADD COLUMN duration_minutes integer NOT NULL DEFAULT 60
        CONSTRAINT services_duration_minutes_check CHECK (duration_minutes BETWEEN 5 AND 1440);

-- end_at is computed by the server; existing bookings are backfilled from their service.
ALTER TABLE public.bookings ADD COLUMN end_at timestamptz;

UPDATE public.bookings b
SET end_at = b.scheduled_at + make_interval(mins => s.duration_minutes)
FROM public.services s
WHERE s.id = b.service_id;

ALTER TABLE public.bookings
    ALTER COLUMN end_at SET NOT NULL,
    ADD CONSTRAINT bookings_end_after_start CHECK (end_at > scheduled_at);