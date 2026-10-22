package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var registerHealthDriver sync.Once

func healthTestDB(t *testing.T, mode string) *sql.DB {
	t.Helper()
	registerHealthDriver.Do(func() { sql.Register("observability-health-test", healthTestDriver{}) })
	db, err := sql.Open("observability-health-test", mode)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type healthTestDriver struct{}

func (healthTestDriver) Open(name string) (driver.Conn, error) {
	return healthTestConn{mode: name}, nil
}

type healthTestConn struct{ mode string }

func (healthTestConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (healthTestConn) Close() error                        { return nil }
func (healthTestConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }
func (c healthTestConn) Ping(ctx context.Context) error {
	switch c.mode {
	case "fail":
		return errors.New("database offline")
	case "block":
		<-ctx.Done()
		return ctx.Err()
	default:
		return nil
	}
}

func TestHealthzDoesNotDependOnDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/healthz", Healthz)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK || response.Body.String() != `{"status":"ok"}` {
		t.Fatalf("unexpected liveness response: %d %s", response.Code, response.Body.String())
	}
}

func TestReadyzRequiresPostgresButNotRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", Readyz(healthTestDB(t, "ok"), time.Second))
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK || response.Body.String() != `{"status":"ready"}` {
		t.Fatalf("unexpected readiness response: %d %s", response.Code, response.Body.String())
	}
}

func TestReadyzReturnsUnavailableOnDatabaseFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", Readyz(healthTestDB(t, "fail"), time.Second))
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable || response.Body.String() != `{"status":"not_ready"}` {
		t.Fatalf("unexpected readiness response: %d %s", response.Code, response.Body.String())
	}
}

func TestReadyzUsesBoundedDependencyTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", Readyz(healthTestDB(t, "block"), 15*time.Millisecond))
	start := time.Now()
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", response.Code)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("readiness check exceeded bounded timeout: %s", time.Since(start))
	}
}

func TestReadyzHandlesMissingDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/readyz", Readyz(nil, time.Second))
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", response.Code)
	}
}
