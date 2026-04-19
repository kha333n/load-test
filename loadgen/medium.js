import { hit, sanityCheck } from './shared.js';

// Designed to push HPA past its CPU threshold and exercise multi-pod cross-node spread.
export const options = {
  scenarios: {
    medium: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: 400,
      maxVUs: 2500,
      stages: [
        { duration: '2m', target: 1500 },
        { duration: '6m', target: 1500 },
        { duration: '1m', target: 0 },
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
