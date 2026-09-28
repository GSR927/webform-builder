import http from "k6/http";
import { check } from "k6";

export const options = {
 scenarios: {
   burst: {
     executor: "ramping-arrival-rate",
     startRate: 5,
     timeUnit: "1s",
     preAllocatedVUs: 20,
     maxVUs: 100,
     stages: [
       { target: 5, duration: "10s" },
       { target: 20, duration: "20s" },
       { target: 5, duration: "10s" }
     ]
   }
 },
 thresholds: {
   http_req_duration: ["p(95)<800"],
   http_req_failed: ["rate<0.05"]
 }
};

const BASE_URL = __ENV.BASE_URL || "http://host.docker.internal:8080";
const FORM_ID = __ENV.FORM_ID || "3";
const VERSION_ID = __ENV.VERSION_ID || "1";

export default function () {
 const res = http.post(
   `${BASE_URL}/f/${FORM_ID}/submit`,
   JSON.stringify({
     versionId: Number(VERSION_ID),
     payload: {
       name: `Load Test User ${__VU}-${__ITER}`,
       email: `loadtest+${__VU}_${__ITER}@example.com`
     },
     hp: ""
   }),
   { headers: { "Content-Type": "application/json" }, timeout: "10s" }
 );

 check(res, {
   "accepted or rate-limited": (r) => r.status === 202 || r.status === 429
 });
}