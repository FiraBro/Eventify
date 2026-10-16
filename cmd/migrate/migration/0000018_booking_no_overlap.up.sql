-- btree_gist lets a GiST exclusion constraint compare uuid values with "=".
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- A staff member cannot have two ACTIVE bookings that overlap in time.
--   * '[)' ranges: a booking ending at 11:00 and another starting at 11:00 do not conflict.
--   * cancelled and completed bookings do not hold a slot.
--   * bookings without a staff member are not covered (they are handled by auto-assignment).
-- The database enforces this atomically, so two concurrent requests for the same slot cannot both succeed.
ALTER TABLE public.bookings
    ADD CONSTRAINT bookings_no_staff_overlap
    EXCLUDE USING gist (staff_id WITH =, tstzrange(scheduled_at, end_at, '[)') WITH &&)
    WHERE (staff_id IS NOT NULL AND status IN ('pending', 'confirmed'));