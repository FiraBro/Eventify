# Changelog

## [Unreleased]

### Added (API, additive)

- `duration_minutes` (integer, 5-1440, default 60) on services: returned by every service endpoint and
  accepted by `POST /services` and `PATCH /services/:id`.
- `end_at` (RFC 3339, UTC) on bookings. Computed by the server as `scheduled_at + service duration`;
  a value sent by a client is ignored. A booked slot is the half-open range `[scheduled_at, end_at)`.
- `409 Conflict` from `POST /bookings` and `PATCH /bookings/:id` when the chosen staff member already has an
  active (pending or confirmed) booking that overlaps the slot.

### Changed

- Booking `scheduled_at`, `created_at` and `updated_at` are stored as `timestamptz`. JSON output is unchanged (UTC).
- Booking `notes` is never `null` (empty string instead).

### Database (migrations 15-18)

- 15: foreign key `bookings.staff_id -> staff(id)` now recorded in migrations (already present in production).
- 16: `bookings` timestamps to `timestamptz`; `notes` backfilled and `NOT NULL`.
- 17: `services.duration_minutes`, `bookings.end_at` (backfilled).
- 18: `btree_gist` + exclusion constraint `bookings_no_staff_overlap`.

### Not covered yet

- Bookings without a staff member are not protected by the overlap constraint (auto-assignment follows).
- OpenAPI (`docs/`) is not regenerated yet.
