package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/FiraBro/local-go/internal/observability"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func testObservabilityRouter(output *bytes.Buffer) (*gin.Engine, *observability.Metrics) {
	gin.SetMode(gin.TestMode)
	logger, _ := observability.NewJSONLogger("info", output)
	metrics := observability.NewMetrics()
	r := gin.New()
	r.Use(RequestID(), HTTPObservability(logger, metrics), Recovery(logger))
	return r, metrics
}

func TestRequestIDAcceptedGeneratedAndReturned(t *testing.T) {
	var output bytes.Buffer
	r, _ := testObservabilityRouter(&output)
	r.GET("/items/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"id": c.Param("id")}) })
	r.POST("/submit", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	validID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	request := httptest.NewRequest(http.MethodGet, "/items/123?token=do-not-log", nil)
	request.Header.Set(RequestIDHeader, validID)
	request.Header.Set("Authorization", "Bearer never-log-this")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Header().Get(RequestIDHeader) != validID {
		t.Fatalf("valid request ID was not propagated: %q", response.Header().Get(RequestIDHeader))
	}

	request = httptest.NewRequest(http.MethodGet, "/items/456", nil)
	request.Header.Set(RequestIDHeader, "bad\r\nrequest-id")
	response = httptest.NewRecorder()
	r.ServeHTTP(response, request)
	generated := response.Header().Get(RequestIDHeader)
	if _, err := uuid.Parse(generated); err != nil {
		t.Fatalf("invalid ID was not replaced with UUID: %q", generated)
	}
	if generated == "bad\r\nrequest-id" {
		t.Fatal("invalid incoming request ID was accepted")
	}

	bodyRequest := httptest.NewRequest(http.MethodPost, "/submit?signature=hidden", strings.NewReader("password=body-secret"))
	bodyRequest.Header.Set("Cookie", "session=hidden-cookie")
	bodyRequest.Header.Set("X-API-Key", "hidden-api-key")
	bodyResponse := httptest.NewRecorder()
	r.ServeHTTP(bodyResponse, bodyRequest)
	if strings.Contains(output.String(), "do-not-log") || strings.Contains(output.String(), "never-log-this") || strings.Contains(output.String(), "/items/123?") || strings.Contains(output.String(), "body-secret") || strings.Contains(output.String(), "hidden-cookie") || strings.Contains(output.String(), "hidden-api-key") || strings.Contains(output.String(), "signature=hidden") {
		t.Fatalf("sensitive request details leaked to logs: %s", output.String())
	}
	if !strings.Contains(output.String(), `"route":"/items/:id"`) || !strings.Contains(output.String(), validID) {
		t.Fatalf("expected route pattern and correlation ID in logs: %s", output.String())
	}
	if count := strings.Count(output.String(), `"msg":"http request completed"`); count != 3 {
		t.Fatalf("got %d completion logs for 3 requests; want exactly one per request", count)
	}
}

func TestRequestIDsAreIsolatedAcrossConcurrentRequests(t *testing.T) {
	var output bytes.Buffer
	r, _ := testObservabilityRouter(&output)
	r.GET("/correlate", func(c *gin.Context) {
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(c.Writer, c.GetString("request_id"))
	})

	const count = 64
	ids := make(chan string, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response := httptest.NewRecorder()
			r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/correlate", nil))
			ids <- response.Header().Get(RequestIDHeader)
		}()
	}
	wg.Wait()
	close(ids)
	seen := make(map[string]bool)
	for id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			t.Errorf("invalid generated request ID %q", id)
		}
		if seen[id] {
			t.Errorf("duplicate request ID %q", id)
		}
		seen[id] = true
	}
	if len(seen) != count {
		t.Fatalf("got %d unique IDs, want %d", len(seen), count)
	}
}

func TestPanicRecoveryDoesNotExposePanicDetails(t *testing.T) {
	var output bytes.Buffer
	r, metrics := testObservabilityRouter(&output)
	r.GET("/panic", func(*gin.Context) { panic("private internal detail") })
	request := httptest.NewRequest(http.MethodGet, "/panic?api_key=secret", nil)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", response.Code)
	}
	if strings.Contains(response.Body.String(), "private internal detail") || strings.Contains(response.Body.String(), "goroutine") {
		t.Fatalf("panic internals leaked to client: %s", response.Body.String())
	}
	if response.Header().Get(RequestIDHeader) == "" {
		t.Fatal("panic response lacks request ID")
	}
	if !strings.Contains(output.String(), "panic recovered") || !strings.Contains(output.String(), response.Header().Get(RequestIDHeader)) {
		t.Fatalf("panic log missing correlation ID: %s", output.String())
	}
	metricResponse := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(metricResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(metricResponse.Body.String(), `status_code="500"`) {
		t.Fatalf("panic not included in metrics: %s", metricResponse.Body.String())
	}
}

func TestMetricsEndpointIsNotSelfInstrumented(t *testing.T) {
	var output bytes.Buffer
	r, metrics := testObservabilityRouter(&output)
	r.GET("/metrics", func(c *gin.Context) { c.String(http.StatusOK, "metrics") })
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("got %d", response.Code)
	}
	metricsResponse := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if strings.Contains(metricsResponse.Body.String(), `route="/metrics"`) {
		t.Fatal("metrics endpoint should not count itself")
	}
	var logRecord map[string]any
	if output.Len() > 0 {
		if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &logRecord); err != nil {
			t.Fatal(err)
		}
	}
}
