-- Your live database already has this constraint (it was created by hand), but a database built
-- from the migrations alone did not. This statement adds it only when it is missing, so it is a
-- no-op on the live database and brings fresh databases to the same schema.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'bookings_staff_id_fkey'
          AND conrelid = 'public.bookings'::regclass
    ) THEN
        ALTER TABLE public.bookings
            ADD CONSTRAINT bookings_staff_id_fkey
            FOREIGN KEY (staff_id) REFERENCES public.staff(id) ON DELETE SET NULL;
    END IF;
END $$;