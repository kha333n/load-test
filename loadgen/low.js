import { hit, sanityCheck } from './shared.js';

export const options = {
  scenarios: {
    low: {
      executor: 'constant-arrival-rate',
      rate: 20,
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 30,
      maxVUs: 100,
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
