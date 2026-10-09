// Load test: registers one user up front, then ramps a configurable pool of
// virtual users through the protected items CRUD cycle, enforcing the p95
// latency budget (NFR: p95 < 200ms) and a <1% error rate.
//
// The edge rate limiter will throttle a real load run — start the server under
// test with the limits raised, e.g.:
//   RATE_LIMIT_RPM=1000000 LOGIN_RATE_LIMIT_RPM=1000000 ./bin/server
//
//   make load                       # defaults: 20 VUs, 1m
//   make load VUS=50 DURATION=2m
//   BASE_URL=... VUS=50 DURATION=2m k6 run loadtest/load.js
import { register, crudCycle, BASE_URL } from './lib.js';

const VUS = parseInt(__ENV.VUS || '20', 10);
const DURATION = __ENV.DURATION || '1m';

export const options = {
  stages: [
    { duration: '15s', target: VUS }, // ramp up
    { duration: DURATION, target: VUS }, // steady state
    { duration: '10s', target: 0 }, // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'],
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
  },
};

// setup registers a single user once; its return value is shared with every VU.
export function setup() {
  return { auth: register(BASE_URL) };
}

export default function (data) {
  crudCycle(BASE_URL, data.auth);
}
