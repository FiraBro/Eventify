# Booking Contention Test

## Objective

Validate booking creation under concurrent requests and verify that the PostgreSQL exclusion constraint prevents overlapping active bookings for the same staff member.

## Test Environment

- **API:** Local development environment
- **Database:** PostgreSQL
- **Load-testing tool:** k6 v2.3.0
- **Virtual users:** 10
- **Iterations:** 1 per virtual user
- **Benchmark service duration:** 60 minutes
- **Benchmark staff:** One eligible staff member
- **Test date:** 2026-11-02

## Scenarios

### 1. Explicit Staff Assignment

All 10 virtual users submitted a booking request for the same staff member and the same 09:00 UTC time slot.

**Results:**

- Booking created: 1
- Expected conflicts (HTTP 409): 9
- Unexpected responses: 0
- k6 HTTP request failure rate: 90%
- HTTP p95 latency: 9,056.65 ms

**Note:** The application returned the expected booking outcomes. However, this initial run used incorrect k6 expected-response handling, causing HTTP 409 responses to be counted as failed requests.

**Evidence:** `step9-booking-contention-explicit.json`

### 2. Automatic Staff Assignment

All 10 virtual users submitted a booking request for the same service and the same 10:30 UTC time slot, without specifying a staff member.

**Results:**

- Booking created: 1
- Expected conflicts (HTTP 409): 9
- Unexpected responses: 0
- k6 HTTP request failure rate: 0%
- HTTP p95 latency: 58.29 ms

**Note:** The corrected k6 expected-response handling recognized HTTP 201 and HTTP 409 as expected responses.

**Evidence:** `step9-booking-contention-auto.json`

## Database Verification

Both scenarios were verified against the development PostgreSQL database.

- Explicit assignment: one pending booking at 09:00 UTC.
- Automatic assignment: one pending booking at 10:30 UTC.
- Both bookings were assigned to the benchmark staff member.
- Both bookings have a 60-minute duration.

## Conclusion

Both scenarios produced one successful booking and nine expected conflicts, with no unexpected application response statuses.

The automatic-assignment test also recorded zero HTTP request failures under the corrected k6 response handling.

The explicit-assignment test should be repeated with the corrected response handling before making a direct performance comparison.

## Limitations

- Only one benchmark staff member was available.
- Only one date and one time slot were tested per scenario.
- Each virtual user submitted one request.
- These results do not establish capacity limits or performance under sustained production traffic.
