import { hit, sanityCheck } from './shared.js';

// Saturation. Drives the cluster to its connection-pool / CPU ceiling and back.
// Expect HPA to top out at maxReplicas; watch pool depth + non-2xx rate climb.
export const options = {
  setupTimeout: '5m',
  scenarios: {
    high: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      // Sized for ~5500 RPS × ~250ms WAN-included latency = ~1400 in-flight requests.
      // 3000 VUs gives plenty of headroom without burning a minute on init.
      preAllocatedVUs: 1500,
      maxVUs: 3000,
      stages: [
        { duration: '1m',  target: 3500 },
        { duration: '3m',  target: 3500 },
        { duration: '30s', target: 5500 },
        { duration: '2m',  target: 3500 },
        { duration: '30s', target: 0 },
      ],
    },
  },
  summaryTrendStats: ['min', 'avg', 'med', 'p(95)', 'p(99)', 'max'],
};

export function setup() {
  sanityCheck();
}

export default function () {
  hit();
}
