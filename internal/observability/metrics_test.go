package observability

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsExposePrometheusCountersHistogramAndGauge(t *testing.T) {
	metrics := NewMetrics()
	finish := metrics.Begin()
	finish("GET", "/items/:id", 200, 12*time.Millisecond)
	finish("GET", "/items/:id", 200, 12*time.Millisecond) // completion must be idempotent

	finishError := metrics.Begin()
	finishError("POST", "/items", 500, 40*time.Millisecond)

	response := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	body, err := io.ReadAll(response.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, expected := range []string{
		"# TYPE http_requests_total counter",
		`http_requests_total{method="GET",route="/items/:id",status_code="200"} 1`,
		`http_errors_total{method="POST",route="/items",status_code="500"} 1`,
		`http_request_duration_seconds_bucket{method="GET",route="/items/:id",status_code="200",le="+Inf"} 1`,
		"http_request_duration_seconds_count{method=\"GET\",route=\"/items/:id\",status_code=\"200\"} 1",
		"http_requests_in_flight 0",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("metrics output missing %q:\n%s", expected, text)
		}
	}
}

func TestMetricLabelsNormalizeUnboundedValues(t *testing.T) {
	metrics := NewMetrics()
	finish := metrics.Begin()
	finish("CUSTOM-METHOD-secret", "/users/:id", 500, time.Second)
	response := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	text := response.Body.String()
	if strings.Contains(text, "secret") {
		t.Fatalf("unbounded data leaked into metric labels: %s", text)
	}
	if !strings.Contains(text, `method="OTHER"`) || !strings.Contains(text, `route="/users/:id"`) {
		t.Fatalf("labels were not bounded: %s", text)
	}
}
