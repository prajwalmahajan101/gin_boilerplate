import http from 'k6/http';
import { check, fail } from 'k6';

export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

const JSON_HEADERS = { 'Content-Type': 'application/json' };

// uniq returns a collision-resistant suffix. It avoids __VU/__ITER so it is
// valid in setup() too (those globals exist only in the VU/default context).
function uniq() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

// register creates a unique user and returns a Bearer-auth header set. The
// register endpoint sits behind the strict public (login) rate limit, so load
// tests call this once in setup() rather than per iteration.
export function register(baseURL) {
  const stamp = uniq();
  const res = http.post(
    `${baseURL}/api/v1/auth/register`,
    JSON.stringify({ email: `loadtest-${stamp}@example.com`, password: 'password123' }),
    { headers: JSON_HEADERS },
  );
  check(res, {
    'register 201': (r) => r.status === 201,
    'register success': (r) => r.json('success') === true,
  });
  const token = res.json('data.access_token');
  if (!token) {
    fail(`register returned no access_token (status ${res.status})`);
  }
  return { ...JSON_HEADERS, Authorization: `Bearer ${token}` };
}

// crudCycle runs one create -> list -> get -> update -> delete cycle on the
// protected items API, asserting status on each hop. This is the hot path load
// tests measure.
export function crudCycle(baseURL, auth) {
  const code = `CODE-${uniq()}`;

  let res = http.post(
    `${baseURL}/api/v1/items`,
    JSON.stringify({ name: 'load item', code }),
    { headers: auth },
  );
  check(res, { 'create 201': (r) => r.status === 201 });
  const id = res.json('data.id');
  if (!id) {
    fail(`create returned no id (status ${res.status})`);
  }

  res = http.get(`${baseURL}/api/v1/items`, { headers: auth });
  check(res, { 'list 200': (r) => r.status === 200 });

  res = http.get(`${baseURL}/api/v1/items/${id}`, { headers: auth });
  check(res, { 'get 200': (r) => r.status === 200 });

  res = http.patch(`${baseURL}/api/v1/items/${id}`, JSON.stringify({ name: 'updated' }), { headers: auth });
  check(res, { 'update 200': (r) => r.status === 200 });

  res = http.del(`${baseURL}/api/v1/items/${id}`, null, { headers: auth });
  check(res, { 'delete 200': (r) => r.status === 200 });
}

// scenario is the full journey (register + one CRUD cycle), used by the smoke test.
export function scenario(baseURL) {
  const auth = register(baseURL);
  crudCycle(baseURL, auth);
}
