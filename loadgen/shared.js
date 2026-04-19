import http from 'k6/http';
import { check, fail } from 'k6';

export const TARGET = __ENV.TARGET || 'http://load-test.local:30080';

// Endpoint mix per requirements §"Load scenarios".
const ENDPOINTS = [
  { path: '/test/compute',    weight: 10 },
  { path: '/test/cache-hit',  weight: 20 },
  { path: '/test/cache-miss', weight: 10 },
  { path: '/test/db-read',    weight: 20 },
  { path: '/test/db-write',   weight: 15 },
  { path: '/test/combined',   weight: 25 },
];

const TOTAL_WEIGHT = ENDPOINTS.reduce((s, e) => s + e.weight, 0);

function pickEndpoint() {
  let r = Math.random() * TOTAL_WEIGHT;
  for (const e of ENDPOINTS) {
    if ((r -= e.weight) <= 0) return e.path;
  }
  return ENDPOINTS[ENDPOINTS.length - 1].path;
}

export function hit() {
  const path = pickEndpoint();
  const res = http.get(`${TARGET}${path}`, { tags: { endpoint: path } });
  check(res, {
    'status 2xx': (r) => r.status >= 200 && r.status < 300,
  }, { endpoint: path });
}

export function sanityCheck() {
  const r = http.get(`${TARGET}/healthz`);
  if (r.status !== 200) fail(`healthz failed: ${r.status} from ${TARGET}`);
}
