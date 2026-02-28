import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const projectListDuration = new Trend('project_list_duration');
const uploadDuration = new Trend('upload_duration');
const successfulUploads = new Counter('successful_uploads');

export const options = {
  // Load testing stages
  stages: [
    // Ramp-up: gradually increase to 100 users
    { duration: '5m', target: 100 },
    // Stay at peak for 10 minutes
    { duration: '10m', target: 100 },
    // Spike: suddenly increase to 500 users (test under stress)
    { duration: '2m', target: 500 },
    // Return to normal
    { duration: '5m', target: 100 },
    // Ramp-down
    { duration: '3m', target: 0 },
  ],
  
  // SLA thresholds
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<2000'], // 95% under 500ms, 99% under 2s
    'http_req_failed{static:no}': ['rate<0.1'],   // < 0.1% error rate
    'project_list_duration': ['p(95)<500'],
    errors: ['rate<0.01'],
  },
  
  // Reporting
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(95)', 'p(99)'],
};

const API_URL = __ENV.API_URL || 'http://localhost:8080';
const JWT_TOKEN = __ENV.JWT_TOKEN || 'test-token';

export default function() {
  // Test group 1: Project listing
  group('Project List API', () => {
    const res = http.get(`${API_URL}/projects?limit=20&offset=0`, {
      headers: {
        Authorization: `Bearer ${JWT_TOKEN}`,
      },
      tags: { static: 'no' },
    });
    
    check(res, {
      'status is 200': (r) => r.status === 200,
      'response time < 500ms': (r) => r.timings.duration < 500,
      'has projects': (r) => r.json('data').length > 0,
    }) || errorRate.add(1);
    
    projectListDuration.add(res.timings.duration);
  });
  
  sleep(1);
  
  // Test group 2: Track playback
  group('Track Playback', () => {
    const trackId = 'test-track-id';
    const res = http.post(`${API_URL}/tracks/${trackId}/presigned-download-url`, null, {
      headers: {
        Authorization: `Bearer ${JWT_TOKEN}`,
        'Content-Type': 'application/json',
      },
    });
    
    check(res, {
      'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
      'has signed URL': (r) => r.json('signed_url') !== undefined,
    }) || errorRate.add(1);
  });
  
  sleep(1);
  
  // Test group 3: Health check
  group('Health Check', () => {
    const res = http.get(`${API_URL}/health`);
    check(res, {
      'health status is ok': (r) => r.json('status') === 'ok',
    }) || errorRate.add(1);
  });
  
  sleep(2);
}

// Custom summary function
export function handleSummary(data) {
  console.log('=== Load Test Summary ===');
  console.log(`Success Rate: ${(100 - (data.metrics.errors.values.rate * 100)).toFixed(2)}%`);
  console.log(`P95 Latency: ${Math.round(data.metrics.http_req_duration.values['p(95)']).toFixed(0)}ms`);
  console.log(`P99 Latency: ${Math.round(data.metrics.http_req_duration.values['p(99)']).toFixed(0)}ms`);
  
  return {
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

// Helper: text summary
function textSummary(data, options) {
  let summary = '\n';
  
  for (const [name, metric] of Object.entries(data.metrics)) {
    if (metric.type === 'Trend') {
      summary += `${name}:\n`;
      summary += `  avg: ${Math.round(metric.values.avg)}ms\n`;
      summary += `  p95: ${Math.round(metric.values['p(95)'])}ms\n`;
      summary += `  p99: ${Math.round(metric.values['p(99)'])}ms\n`;
    }
  }
  
  return summary;
}
