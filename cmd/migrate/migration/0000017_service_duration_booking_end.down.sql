-- Remove constraints first
ALTER TABLE public.bookings DROP CONSTRAINT IF EXISTS bookings_end_after_start;

-- Drop the columns added in the up migration
ALTER TABLE public.bookings DROP COLUMN IF EXISTS end_at;
ALTER TABLE public.services DROP COLUMN IF EXISTS duration_minutes;
