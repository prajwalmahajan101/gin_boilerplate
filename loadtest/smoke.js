// Smoke test: a single virtual user runs the full journey a handful of times.
// Proves the API is wired end-to-end before a real load run. Fails (non-zero
// exit) if any request errors or any check fails.
//
//   make load-smoke          # server must be running + migrated
//   BASE_URL=... k6 run loadtest/smoke.js
import { scenario, BASE_URL } from './lib.js';

export const options = {
  vus: 1,
  iterations: 5,
  thresholds: {
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
  },
};

export default function () {
  scenario(BASE_URL);
}
