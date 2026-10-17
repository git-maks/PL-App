// Load test for an event peak (spec §1, §12): many players open the app and ~300 scans/minute hit
// one station right after a workshop ends.
//
//   k6 run -e BASE=https://uniquest.example -e ADMIN_COOKIE='__Host-uq_session=…' -e STATION=<uuid> tools/load/k6-event-peak.js
//
// ADMIN_COOKIE is a staff/admin session (log in, copy the cookie) used to read the station's
// current rotating QR payload the same way the station display does.
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE = __ENV.BASE || 'http://localhost:8080';
const API = BASE + '/api/v1';
const H = { 'Content-Type': 'application/json', 'X-Requested-With': 'uniquest', Origin: BASE };

export const options = {
	scenarios: {
		browse: { executor: 'constant-vus', vus: 200, duration: '3m', exec: 'browse' },
		peak: { executor: 'ramping-arrival-rate', startRate: 1, timeUnit: '1s', preAllocatedVUs: 100, stages: [{ target: 5, duration: '30s' }, { target: 5, duration: '2m' }], exec: 'scan' }
	},
	thresholds: { http_req_failed: ['rate<0.01'], 'http_req_duration{kind:api}': ['p(95)<200'] }
};

function guest() {
	const res = http.post(API + '/auth/guest', '{}', { headers: H, tags: { kind: 'api' } });
	check(res, { 'guest created': (r) => r.status === 201 });
	return res;
}

export function browse() {
	guest();
	const ev = http.get(API + '/events/active', { tags: { kind: 'api' } });
	if (ev.status === 200) http.get(API + `/events/${ev.json('id')}/stations`, { tags: { kind: 'api' } });
	http.get(API + '/me/progress', { tags: { kind: 'api' } });
	sleep(5 + Math.random() * 10);
}

let payload = null;
let fetchedAt = 0;
function currentPayload() {
	if (!payload || Date.now() - fetchedAt > 20000) {
		const r = http.get(API + `/staff/stations/${__ENV.STATION}/qr`, { headers: { Cookie: __ENV.ADMIN_COOKIE } });
		payload = r.json('payload');
		fetchedAt = Date.now();
	}
	return payload;
}

export function scan() {
	guest();
	const res = http.post(API + '/scans', JSON.stringify({ payload: currentPayload() }), { headers: H, tags: { kind: 'api' } });
	check(res, { 'scan accepted': (r) => r.status === 200 });
	if (res.status === 200) {
		const r2 = http.put(API + `/scans/${res.json('scan.id')}/rating`, JSON.stringify({ rating: 1 + Math.floor(Math.random() * 5) }), { headers: H, tags: { kind: 'api' } });
		check(r2, { 'rated': (r) => r.status === 200 });
	}
}
