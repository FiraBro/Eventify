import http, { setResponseCallback } from "k6/http";
import { check } from "k6";
import { Counter } from "k6/metrics";

const BASE_URL = (__ENV.BASE_URL || "http://127.0.0.1:8080").replace(/\/$/, "");
const SERVICE_ID = __ENV.SERVICE_ID || "f1000000-0000-4000-8000-000000000001";
const STAFF_ID = __ENV.STAFF_ID || "f2000000-0000-4000-8000-000000000001";
const TEST_DATE = __ENV.TEST_DATE || "2026-11-02";
const TEST_TIME = __ENV.TEST_TIME || "09:00:00Z";
const MODE = __ENV.MODE || "explicit";
const TOKEN = __ENV.BASELINE_TOKEN;

// Treat successful creation and expected booking conflicts as expected HTTP responses.
setResponseCallback(http.expectedStatuses(201, 409));

if (!["explicit", "auto"].includes(MODE)) {
  throw new Error("MODE must be explicit or auto");
}

const created = new Counter("booking_created");
const conflicts = new Counter("booking_conflicts");
const unexpected = new Counter("booking_unexpected_status");

export const options = {
  scenarios: {
    booking_contention: {
      executor: "per-vu-iterations",
      vus: 10,
      iterations: 1,
      maxDuration: "30s",
    },
  },
  thresholds: {
    booking_created: ["count==1"],
    booking_conflicts: ["count==9"],
    booking_unexpected_status: ["count==0"],
  },
};

export default function () {
  if (!TOKEN) {
    throw new Error("BASELINE_TOKEN environment variable is required");
  }

  const payload = {
    service_id: SERVICE_ID,
    scheduled_at: `${TEST_DATE}T${TEST_TIME}`,
    notes: `Performance contention test: ${MODE}`,
  };

  if (MODE === "explicit") {
    payload.staff_id = STAFF_ID;
  }

  const response = http.post(
    `${BASE_URL}/api/v1/bookings`,
    JSON.stringify(payload),
    {
      headers: {
        Authorization: `Bearer ${TOKEN}`,
        "Content-Type": "application/json",
      },
      tags: {
        endpoint: "booking_create",
        mode: MODE,
      },
    },
  );

  if (response.status === 201) {
    created.add(1);
  } else if (response.status === 409) {
    conflicts.add(1);
  } else {
    unexpected.add(1);
  }

  check(response, {
    "booking returns 201 or 409": (r) => r.status === 201 || r.status === 409,
  });
}
