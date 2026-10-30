# Booking Contention Load Test

## 1. Objective

Validate booking concurrency behavior under simultaneous requests using k6.

The test verifies that multiple requests targeting the same staff member and time slot cannot create overlapping active bookings.

## 2. Test Environment

- API: Local development environment
- Database: PostgreSQL 16.3
- Load tool: k6
- Virtual users: 10
- Iterations: 1 per virtual user
- Service: Performance Benchmark Service
- Staff: Performance Benchmark Staff
- Test date: November 2, 2026
- Booking duration: 60 minutes

## 3. Explicit Staff Assignment

**Scenario:** Ten concurrent requests submit a booking for the same staff member and time slot (09:00–10:00 UTC).

| Metric                   |   Result |
| ------------------------ | -------: |
| Requests                 |       10 |
| Created bookings (201)   |        1 |
| Expected conflicts (409) |        9 |
| Unexpected responses     |        0 |
| k6 failed requests       |       0% |
| Checks passed            |    10/10 |
| HTTP p95                 | 35.86 ms |
| HTTP average             | 29.32 ms |
| HTTP maximum             | 36.88 ms |

The initial explicit-staff run also produced one booking and nine conflicts. However, k6 initially counted HTTP 409 responses as failed requests because the expected-status callback was not configured correctly. The corrected run uses `http.expectedStatuses(201, 409)`.

Result file: `docs/performance/step9-booking-contention-explicit-corrected.json`

## 4. Automatic Staff Assignment

**Scenario:** Ten concurrent requests submit bookings for the same service and time slot (10:30–11:30 UTC), without specifying a staff member.

| Metric                   |   Result |
| ------------------------ | -------: |
| Requests                 |       10 |
| Created bookings (201)   |        1 |
| Expected conflicts (409) |        9 |
| Unexpected responses     |        0 |
| k6 failed requests       |       0% |
| Checks passed            |    10/10 |
| HTTP p95                 | 58.29 ms |

Result file: `docs/performance/step9-booking-contention-auto.json`

## 5. Database Verification

PostgreSQL confirmed one pending booking for each tested slot:

- Explicit assignment: 09:00–10:00 UTC.
- Automatic assignment: 10:30–11:30 UTC.
- Corrected explicit assignment: 11:30–12:30 UTC.

The database exclusion constraint `bookings_no_staff_overlap` prevented competing active bookings from occupying the same staff/time interval.

## 6. Conclusion

Both booking flows successfully handled the controlled contention tests:

- One request successfully created a booking.
- Nine competing requests received the expected conflict response.
- No unexpected HTTP responses occurred.
- The corrected explicit and automatic runs both reported zero failed HTTP requests.

These results demonstrate correct conflict enforcement for the tested scenarios. They do not establish production capacity or performance under large-scale traffic.

## 7. Limitations

- Only one benchmark staff member was available.
- Each scenario tested one time slot.
- Each virtual user submitted one request.
- Tests ran in a local development environment.
- Results should not be interpreted as production-scale capacity measurements.
