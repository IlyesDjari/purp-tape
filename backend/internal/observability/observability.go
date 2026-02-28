package observability

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Observability manager for distributed tracing and metrics
type ObservabilityManager struct {
	tracer trace.Tracer
	meter  metric.Meter
	log    *slog.Logger
}

// InitializeTracing sets up OpenTelemetry tracing
func InitializeTracing(serviceName, serviceVersion, otlpEndpoint string, log *slog.Logger) (trace.Tracer, func(), error) {
	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
			semconv.ServiceInstanceIDKey.String("purptape-api"),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	// Create OTLP HTTP exporter
	exporter, err := otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint(otlpEndpoint))
	if err != nil {
		return nil, nil, err
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // 10% sampling
	)

	// Set global provider
	otel.SetTracerProvider(tp)

	// Get tracer
	tracer := tp.Tracer(serviceName)

	log.Info("OpenTelemetry tracing initialized", "endpoint", otlpEndpoint)

	// Shutdown function
	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Error("failed to shutdown tracer", "error", err)
		}
	}

	return tracer, shutdown, nil
}

// New creates observability manager
func New(tracer trace.Tracer, log *slog.Logger) *ObservabilityManager {
	return &ObservabilityManager{
		tracer: tracer,
		meter:  otel.Meter("purptape"),
		log:    log,
	}
}

// ============================================
// SLA MONITORING
// ============================================

// SLAMonitor tracks SLA compliance
type SLAMonitor struct {
	latencyBuckets  []time.Duration
	requestCounts   map[string]int64
	durationTotal   map[string]time.Duration
	errorCounts     map[string]int64
	log             *slog.Logger
}

// NewSLAMonitor creates SLA monitor
func NewSLAMonitor(log *slog.Logger) *SLAMonitor {
	return &SLAMonitor{
		latencyBuckets: []time.Duration{
			50 * time.Millisecond,
			100 * time.Millisecond,
			200 * time.Millisecond,
			500 * time.Millisecond,
			1 * time.Second,
			2 * time.Second,
		},
		requestCounts: make(map[string]int64),
		durationTotal: make(map[string]time.Duration),
		errorCounts:   make(map[string]int64),
		log:           log,
	}
}

// RecordRequest records request metrics for SLA tracking
func (sm *SLAMonitor) RecordRequest(endpoint string, duration time.Duration, err error) {
	// Count requests by endpoint
	key := endpoint
	sm.requestCounts[key]++
	sm.durationTotal[key] += duration

	// Count errors
	if err != nil {
		sm.errorCounts[key]++
	}

	// Check SLA violations
	sm.checkSLA(endpoint, duration, err)
}

// checkSLA checks if request violates SLA
func (sm *SLAMonitor) checkSLA(endpoint string, duration time.Duration, err error) {
	// p95 latency SLA: < 500ms
	if duration > 500*time.Millisecond {
		sm.log.Warn("SLA violation: high latency",
			"endpoint", endpoint,
			"duration_ms", duration.Milliseconds(),
			"sla_ms", 500)
	}

	// p99 latency SLA: < 2s
	if duration > 2*time.Second {
		sm.log.Warn("SLA violation: critical latency",
			"endpoint", endpoint,
			"duration_ms", duration.Milliseconds(),
			"sla_ms", 2000)
	}

	// Error rate SLA: < 0.1%
	if sm.requestCounts[endpoint] > 1000 {
		errorRate := float64(sm.errorCounts[endpoint]) / float64(sm.requestCounts[endpoint])
		if errorRate > 0.001 {
			sm.log.Warn("SLA violation: high error rate",
				"endpoint", endpoint,
				"error_rate", errorRate,
				"sla", 0.001)
		}
	}
}

// GetSLAReport returns SLA metrics report
func (sm *SLAMonitor) GetSLAReport() map[string]interface{} {
	report := make(map[string]interface{})

	for endpoint, count := range sm.requestCounts {
		avgDuration := 0.0
		if count > 0 {
			avgDuration = float64(sm.durationTotal[endpoint].Milliseconds()) / float64(count)
		}

		errorRate := 0.0
		if count > 0 {
			errorRate = float64(sm.errorCounts[endpoint]) / float64(count)
		}

		report[endpoint] = map[string]interface{}{
			"request_count": count,
			"error_count":   sm.errorCounts[endpoint],
			"error_rate":    errorRate,
			"avg_latency_ms": avgDuration,
			"sla_compliant": errorRate < 0.001 && avgDuration < 500,
		}
	}

	return report
}

// ============================================
// TRACING MIDDLEWARE
// ============================================

// TracingMiddleware adds distributed tracing to requests
func (om *ObservabilityManager) TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start span
		ctx, span := om.tracer.Start(r.Context(), r.Method+" "+r.URL.Path,
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.target", r.URL.Path),
				attribute.String("http.host", r.Host),
			),
		)
		defer span.End()

		// Record request details
		span.SetAttributes(
			attribute.String("http.client_ip", getClientIP(r)),
			attribute.String("user_agent", r.UserAgent()),
		)

		// Track latency
		start := time.Now()

		// Continue with request
		next.ServeHTTP(w, r.WithContext(ctx))

		// Record latency
		duration := time.Since(start)
		span.SetAttributes(attribute.Int64("http.duration_ms", duration.Milliseconds()))
	})
}

// ============================================
// REQUESTS COUNTING & ALERTING
// ============================================

// AlertThresholds for SLA alerting
type AlertThresholds struct {
	LatencyP95Ms       int64
	LatencyP99Ms       int64
	ErrorRatePercent   float64
	AvailabilityPercent float64
}

// DefaultAlertThresholds returns default SLA thresholds
func DefaultAlertThresholds() AlertThresholds {
	return AlertThresholds{
		LatencyP95Ms:       500,
		LatencyP99Ms:       2000,
		ErrorRatePercent:   0.1,
		AvailabilityPercent: 99.95,
	}
}

// AlertingClient sends alerts when SLA violated
type AlertingClient struct {
	thresholds AlertThresholds
	log        *slog.Logger
	alertChan  chan Alert
}

// Alert represents SLA alert
type Alert struct {
	Severity  string `json:"severity"` // critical, warning, info
	Title     string `json:"title"`
	Message   string `json:"message"`
	Endpoint  string `json:"endpoint"`
	Timestamp time.Time `json:"timestamp"`
	Metric    string `json:"metric"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
}

// NewAlertingClient creates alerting client
func NewAlertingClient(thresholds AlertThresholds, log *slog.Logger) *AlertingClient {
	return &AlertingClient{
		thresholds: thresholds,
		log:        log,
		alertChan:  make(chan Alert, 1000),
	}
}

// SendAlert sends alert (to Datadog, Sentry, etc)
func (ac *AlertingClient) SendAlert(alert Alert) {
	select {
	case ac.alertChan <- alert:
		ac.log.Warn("alert generated",
			"severity", alert.Severity,
			"title", alert.Title,
			"endpoint", alert.Endpoint)
	default:
		ac.log.Error("alert queue full", "title", alert.Title)
	}
}

// AlertIfLatencyViolation checks and alerts if latency SLA violated
func (ac *AlertingClient) AlertIfLatencyViolation(endpoint string, latencyMs int64) {
	if latencyMs > ac.thresholds.LatencyP99Ms {
		ac.SendAlert(Alert{
			Severity:  "critical",
			Title:     "Critical Latency SLA Violation",
			Message:   "p99 latency exceeded 2000ms",
			Endpoint:  endpoint,
			Timestamp: time.Now(),
			Metric:    "latency_p99",
			Value:     float64(latencyMs),
			Threshold: float64(ac.thresholds.LatencyP99Ms),
		})
	} else if latencyMs > ac.thresholds.LatencyP95Ms {
		ac.SendAlert(Alert{
			Severity:  "warning",
			Title:     "Latency SLA Violation",
			Message:   "p95 latency exceeded 500ms",
			Endpoint:  endpoint,
			Timestamp: time.Now(),
			Metric:    "latency_p95",
			Value:     float64(latencyMs),
			Threshold: float64(ac.thresholds.LatencyP95Ms),
		})
	}
}

// AlertIfErrorRateViolation checks and alerts if error rate exceeds threshold
func (ac *AlertingClient) AlertIfErrorRateViolation(endpoint string, errorRatePercent float64) {
	if errorRatePercent > ac.thresholds.ErrorRatePercent {
		ac.SendAlert(Alert{
			Severity:  "critical",
			Title:     "High Error Rate",
			Message:   "Error rate exceeded 0.1%",
			Endpoint:  endpoint,
			Timestamp: time.Now(),
			Metric:    "error_rate",
			Value:     errorRatePercent,
			Threshold: ac.thresholds.ErrorRatePercent,
		})
	}
}

// ============================================
// HELPER FUNCTIONS
// ============================================

func getClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
