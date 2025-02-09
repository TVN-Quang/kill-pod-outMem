import http from 'k6/http';
import { sleep } from 'k6';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '40s', target: 100 }, // Ramp-up to 10 users over 1 minute
    { duration: '3m', target: 90 }, // Stay at 10 users for 3 minutes
    { duration: '1m', target: 0 },  // Ramp-down to 0 users over 1 minute
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests must complete below 500ms
  },
};

export default function () {
  let res = http.get('http://127.0.0.1:62368/api/delay');
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
  sleep(1);
}
