import { hit, sanityCheck } from './shared.js';

export const options = {
  scenarios: {
    high: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: 400,
      maxVUs: 2000,
      stages: [
        { duration: '1m',  target: 800 },
        { duration: '3m',  target: 800 },
        { duration: '30s', target: 1200 },
        { duration: '2m',  target: 800 },
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
