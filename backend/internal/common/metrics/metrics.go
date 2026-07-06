package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "studyapp"

var (
	initOnce sync.Once
	registry *prometheus.Registry

	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_requests_total",
		Help:      "Total HTTP requests processed.",
	}, []string{"method", "path", "status_code"})

	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	ScanJobsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "scan_jobs_total",
		Help:      "Scan jobs completed by terminal status.",
	}, []string{"status"})

	ScanJobDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "scan_job_duration_seconds",
		Help:      "Scan pipeline duration in seconds.",
		Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
	}, []string{"ocr_provider"})

	ScanCacheHitsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "scan_cache_hits_total",
		Help:      "Quiz content cache hits.",
	})

	ScanCacheMissesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "scan_cache_misses_total",
		Help:      "Quiz content cache misses.",
	})

	AIGenerationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ai_generation_total",
		Help:      "AI quiz generation attempts.",
	}, []string{"provider", "model", "status"})

	AIGenerationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "ai_generation_duration_seconds",
		Help:      "AI generation latency in seconds.",
		Buckets:   []float64{0.5, 1, 2, 5, 10, 30, 60},
	}, []string{"provider"})

	AITokensUsedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ai_tokens_used_total",
		Help:      "Total AI tokens consumed.",
	}, []string{"provider", "model"})

	NotificationSentTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "notification_sent_total",
		Help:      "Notifications processed by channel and outcome.",
	}, []string{"channel", "status"})

	NotificationQueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "notification_queue_depth",
		Help:      "Pending jobs in the notification queue.",
	})

	PaymentEventsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "payment_events_total",
		Help:      "Payment webhook events processed.",
	}, []string{"provider", "status"})
)

// Init registers all application metrics and returns the dedicated registry.
func Init() *prometheus.Registry {
	initOnce.Do(func() {
		registry = prometheus.NewRegistry()
		collectors := []prometheus.Collector{
			HTTPRequestsTotal,
			HTTPRequestDuration,
			ScanJobsTotal,
			ScanJobDuration,
			ScanCacheHitsTotal,
			ScanCacheMissesTotal,
			AIGenerationTotal,
			AIGenerationDuration,
			AITokensUsedTotal,
			NotificationSentTotal,
			NotificationQueueDepth,
			PaymentEventsTotal,
		}
		for _, c := range collectors {
			registry.MustRegister(c)
		}
	})
	return registry
}

// Handler returns an HTTP handler that exposes metrics from the given registry.
func Handler(reg *prometheus.Registry) http.Handler {
	if reg == nil {
		reg = registry
	}
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware records request counts and durations for all routes.
func HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			path := r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
				path = rc.RoutePattern()
			}

			status := strconv.Itoa(rec.status)
			HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
		})
	}
}

// RecordScanJob increments scan job counter for a terminal status.
func RecordScanJob(status string) {
	ScanJobsTotal.WithLabelValues(status).Inc()
}

// ObserveScanJobDuration records scan pipeline duration.
func ObserveScanJobDuration(ocrProvider string, d time.Duration) {
	if ocrProvider == "" {
		ocrProvider = "unknown"
	}
	ScanJobDuration.WithLabelValues(ocrProvider).Observe(d.Seconds())
}

// RecordScanCacheHit increments cache hit counter.
func RecordScanCacheHit() { ScanCacheHitsTotal.Inc() }

// RecordScanCacheMiss increments cache miss counter.
func RecordScanCacheMiss() { ScanCacheMissesTotal.Inc() }

// RecordAIGeneration records AI generation metrics.
func RecordAIGeneration(provider, model, status string, duration time.Duration, tokens int) {
	if provider == "" {
		provider = "unknown"
	}
	if model == "" {
		model = "unknown"
	}
	if status == "" {
		status = "unknown"
	}
	AIGenerationTotal.WithLabelValues(provider, model, status).Inc()
	AIGenerationDuration.WithLabelValues(provider).Observe(duration.Seconds())
	if tokens > 0 {
		AITokensUsedTotal.WithLabelValues(provider, model).Add(float64(tokens))
	}
}

// RecordNotificationSent records a notification delivery outcome.
func RecordNotificationSent(channel, status string) {
	if channel == "" {
		channel = "unknown"
	}
	if status == "" {
		status = "unknown"
	}
	NotificationSentTotal.WithLabelValues(channel, status).Inc()
}

// SetNotificationQueueDepth updates the queue depth gauge.
func SetNotificationQueueDepth(depth float64) {
	NotificationQueueDepth.Set(depth)
}

// RecordPaymentEvent records a payment webhook outcome.
func RecordPaymentEvent(provider, status string) {
	if provider == "" {
		provider = "unknown"
	}
	if status == "" {
		status = "unknown"
	}
	PaymentEventsTotal.WithLabelValues(provider, status).Inc()
}
