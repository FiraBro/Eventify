import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = (__ENV.BASE_URL || 'http://127.0.0.1:8080').replace(/\/$/, '');
const SERVICE_ID = __ENV.SERVICE_ID || 'f1000000-0000-4000-8000-000000000001';
const TEST_DATE = __ENV.TEST_DATE || '2026-11-02';
const TOKEN = __ENV.BASELINE_TOKEN;

if (!TOKEN) {
    throw new Error('BASELINE_TOKEN environment variable is required');
}

const authHeaders = {
    headers: {
        Authorization: `Bearer ${TOKEN}`,
    },
};

export const options = {
    vus: 10,
    duration: '1m',

    thresholds: {
        http_req_failed: ['rate<0.01'],
        checks: ['rate>0.99'],
        'http_req_duration{endpoint:availability}': ['p(95)<500'],
        'http_req_duration{endpoint:services}': ['p(95)<500'],
        'http_req_duration{endpoint:settings}': ['p(95)<500'],
    },
};

export default function () {
    // 1. Service availability
    const availability = http.get(
        `${BASE_URL}/api/v1/availability/services/${SERVICE_ID}?date=${TEST_DATE}`,
        {
            ...authHeaders,
            tags: { endpoint: 'availability' },
        }
    );

    check(availability, {
        'availability returns HTTP 200': (r) => r.status === 200,
    });

    // 2. Services list
    const services = http.get(
        `${BASE_URL}/api/v1/services`,
        {
            tags: { endpoint: 'services' },
        }
    );

    check(services, {
        'services returns HTTP 200': (r) => r.status === 200,
    });

    // 3. Settings
    const settings = http.get(
        `${BASE_URL}/api/v1/settings`,
        {
            ...authHeaders,
            tags: { endpoint: 'settings' },
        }
    );

    check(settings, {
        'settings returns HTTP 200': (r) => r.status === 200,
    });

    sleep(1);
}
