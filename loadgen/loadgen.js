// k6 load generator for the submission hot path.
// Run: k6 run loadgen/loadgen.js
// Demonstrates a burst profile: steady baseline, then a sudden spike well
// beyond steady-state, then recovery -- the scenario the architecture is
// specifically designed to absorb without dropping submissions.

import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: {
    steady_state: {
      executor: "constant-arrival-rate",
      rate: 20,
      timeUnit: "1s",
      duration: "30s",
      preAllocatedVUs: 20,
      maxVUs: 50,
      startTime: "0s",
    },
    sudden_burst: {
      executor: "ramping-arrival-rate",
      startRate: 20,
      timeUnit: "1s",
      stages: [
        { target: 500, duration: "10s" }, // sudden spike, 25x baseline
        { target: 500, duration: "20s" }, // sustain the spike
        { target: 20, duration: "10s" },  // recover
      ],
      preAllocatedVUs: 100,
      maxVUs: 600,
      startTime: "30s",
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<800"], // API should stay responsive even under burst
    http_req_failed: ["rate<0.01"],   // near-zero failed/lost submissions
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const FORM_ID = __ENV.FORM_ID || "1";
const VERSION_ID = __ENV.VERSION_ID || "1";

export default function () {
  const payload = JSON.stringify({
    versionId: Number(VERSION_ID),
    payload: {
      name: `Load Test User ${__VU}-${__ITER}`,
      email: `loadtest+${__VU}_${__ITER}@example.com`,
    },
    hp: "",
  });

  const res = http.post(`${BASE_URL}/f/${FORM_ID}/submit`, payload, {
    headers: { "Content-Type": "application/json" },
  });

  check(res, {
    "accepted (202)": (r) => r.status === 202,
  });

  sleep(0.05);
}
