-- The btree_gist extension is left installed on purpose: other objects may use it.
ALTER TABLE public.bookings DROP CONSTRAINT IF EXISTS bookings_no_staff_overlap;