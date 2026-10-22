package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const durationBucketCount = 11

var durationBuckets = [durationBucketCount]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type requestLabels struct {
	method string
	route  string
	status int
}

type requestStats struct {
	count   uint64
	errors  uint64
	sumSecs float64
	buckets [durationBucketCount + 1]uint64
}

// Metrics is a small Prometheus text-exposition registry for HTTP server metrics.
// It is instance-scoped so tests and repeated router initialization do not register
// duplicate collectors in a global registry.
type Metrics struct {
	mu       sync.Mutex
	requests map[requestLabels]*requestStats
	inFlight atomic.Int64
}

func NewMetrics() *Metrics {
	return &Metrics{requests: make(map[requestLabels]*requestStats)}
}

// Begin marks a request in flight. The returned function records exactly one
// completion observation and decrements the gauge, including recovered panics.
func (m *Metrics) Begin() func(method, route string, status int, duration time.Duration) {
	m.inFlight.Add(1)
	var once sync.Once
	return func(method, route string, status int, duration time.Duration) {
		once.Do(func() {
			m.inFlight.Add(-1)
			labels := requestLabels{method: normalizeMethod(method), route: normalizeRoute(route), status: normalizeStatus(status)}
			seconds := duration.Seconds()
			m.mu.Lock()
			defer m.mu.Unlock()
			stats := m.requests[labels]
			if stats == nil {
				stats = &requestStats{}
				m.requests[labels] = stats
			}
			stats.count++
			if status >= 400 {
				stats.errors++
			}
			stats.sumSecs += seconds
			for i, bound := range durationBuckets {
				if seconds <= bound {
					stats.buckets[i]++
				}
			}
			stats.buckets[len(durationBuckets)]++ // +Inf bucket
		})
	}
}

// Handler exposes only bounded-cardinality operational metrics.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		m.mu.Lock()
		labels := make([]requestLabels, 0, len(m.requests))
		for key := range m.requests {
			labels = append(labels, key)
		}
		sort.Slice(labels, func(i, j int) bool {
			a, b := labels[i], labels[j]
			if a.method != b.method {
				return a.method < b.method
			}
			if a.route != b.route {
				return a.route < b.route
			}
			return a.status < b.status
		})
		statsCopy := make(map[requestLabels]requestStats, len(m.requests))
		for key, value := range m.requests {
			statsCopy[key] = *value
		}
		m.mu.Unlock()

		fmt.Fprintln(w, "# HELP http_requests_total Total completed HTTP requests.")
		fmt.Fprintln(w, "# TYPE http_requests_total counter")
		for _, key := range labels {
			fmt.Fprintf(w, "http_requests_total%s %d\n", labelSet(key, ""), statsCopy[key].count)
		}
		fmt.Fprintln(w, "# HELP http_errors_total Total HTTP responses with status code 400 or greater.")
		fmt.Fprintln(w, "# TYPE http_errors_total counter")
		for _, key := range labels {
			if errors := statsCopy[key].errors; errors > 0 {
				fmt.Fprintf(w, "http_errors_total%s %d\n", labelSet(key, ""), errors)
			}
		}
		fmt.Fprintln(w, "# HELP http_request_duration_seconds HTTP request duration in seconds.")
		fmt.Fprintln(w, "# TYPE http_request_duration_seconds histogram")
		for _, key := range labels {
			stats := statsCopy[key]
			for i, bound := range durationBuckets {
				fmt.Fprintf(w, "http_request_duration_seconds_bucket%s %d\n", labelSet(key, strconv.FormatFloat(bound, 'g', -1, 64)), stats.buckets[i])
			}
			fmt.Fprintf(w, "http_request_duration_seconds_bucket%s %d\n", labelSet(key, "+Inf"), stats.buckets[len(durationBuckets)])
			fmt.Fprintf(w, "http_request_duration_seconds_sum%s %s\n", labelSet(key, ""), strconv.FormatFloat(stats.sumSecs, 'g', -1, 64))
			fmt.Fprintf(w, "http_request_duration_seconds_count%s %d\n", labelSet(key, ""), stats.count)
		}
		fmt.Fprintln(w, "# HELP http_requests_in_flight Number of HTTP requests currently being handled.")
		fmt.Fprintln(w, "# TYPE http_requests_in_flight gauge")
		fmt.Fprintf(w, "http_requests_in_flight %d\n", m.inFlight.Load())
	})
}

func labelSet(labels requestLabels, bucket string) string {
	parts := []string{
		`method="` + escapeLabel(labels.method) + `"`,
		`route="` + escapeLabel(labels.route) + `"`,
		`status_code="` + strconv.Itoa(labels.status) + `"`,
	}
	if bucket != "" {
		parts = append(parts, `le="`+escapeLabel(bucket)+`"`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func normalizeMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodConnect, http.MethodTrace:
		return method
	default:
		return "OTHER"
	}
}

func normalizeRoute(route string) string {
	if route == "" {
		return "unmatched"
	}
	return route
}

func normalizeStatus(status int) int {
	if status < 100 || status > 599 {
		return 500
	}
	return status
}
