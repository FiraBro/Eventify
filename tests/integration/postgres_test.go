//go:build integration

package integration

import (
	"context"
	"database/sql"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresImage         = "postgres:16.3"
	expectedSchemaVersion = 18
)

func newPostgresDatabase(t *testing.T) *sql.DB {
	t.Helper()

	startupContext, cancelStartup := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelStartup()

	username := "eventify_integration"
	password := uuid.NewString()
	container, err := postgres.Run(
		startupContext,
		postgresImage,
		postgres.WithDatabase("eventify_integration"),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("start disposable PostgreSQL container: %s", redactTestError(err.Error(), "", password))
	}

	var connectionString string
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(cleanupContext); err != nil {
			t.Errorf("terminate disposable PostgreSQL container: %s", redactTestError(err.Error(), connectionString, password))
		}
	})

	connectionString, err = container.ConnectionString(startupContext, "sslmode=disable", "connect_timeout=10")
	if err != nil {
		t.Fatalf("get disposable PostgreSQL connection settings: %s", redactTestError(err.Error(), "", password))
	}

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		t.Fatalf("open disposable PostgreSQL connection: %s", redactTestError(err.Error(), connectionString, password))
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close disposable PostgreSQL connection: %s", redactTestError(err.Error(), connectionString, password))
		}
	})

	pingContext, cancelPing := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelPing()
	if err := db.PingContext(pingContext); err != nil {
		t.Fatalf("connect to disposable PostgreSQL container: %s", redactTestError(err.Error(), connectionString, password))
	}

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate integration test file to resolve migrations")
	}
	migrationsPath := filepath.Join(filepath.Dir(testFile), "..", "..", "cmd", "migrate", "migration")
	migrationsURL := (&url.URL{Scheme: "file", Path: migrationsPath}).String()

	migrator, err := migrate.New(migrationsURL, connectionString)
	if err != nil {
		t.Fatalf("initialize golang-migrate for disposable PostgreSQL: %s", redactTestError(err.Error(), connectionString, password))
	}
	migrationErr := migrator.Up()
	sourceCloseErr, databaseCloseErr := migrator.Close()
	if migrationErr != nil {
		t.Fatalf("apply repository migrations to disposable PostgreSQL: %s", redactTestError(migrationErr.Error(), connectionString, password))
	}
	if sourceCloseErr != nil {
		t.Fatalf("close migration source: %s", redactTestError(sourceCloseErr.Error(), connectionString, password))
	}
	if databaseCloseErr != nil {
		t.Fatalf("close migration database connection: %s", redactTestError(databaseCloseErr.Error(), connectionString, password))
	}

	return db
}

func redactTestError(message, connectionString, password string) string {
	if connectionString != "" {
		message = strings.ReplaceAll(message, connectionString, "[redacted connection string]")
	}
	if password != "" {
		message = strings.ReplaceAll(message, password, "[redacted]")
	}
	if parsed, err := url.Parse(connectionString); err == nil && parsed.User != nil {
		if secret, ok := parsed.User.Password(); ok && secret != "" {
			message = strings.ReplaceAll(message, secret, "[redacted]")
		}
	}
	return message
}

func TestPostgresIntegrationSmoke(t *testing.T) {
	db := newPostgresDatabase(t)

	checkContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var one int
	if err := db.QueryRowContext(checkContext, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("query migrated disposable PostgreSQL database: %v", err)
	}
	if one != 1 {
		t.Fatalf("SELECT 1 returned %d, want 1", one)
	}

	var version int
	var dirty bool
	if err := db.QueryRowContext(checkContext, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
		t.Fatalf("read applied migration version: %v", err)
	}
	if dirty {
		t.Fatal("migration database is marked dirty")
	}
	if version != expectedSchemaVersion {
		t.Fatalf("applied migration version is %d, want %d", version, expectedSchemaVersion)
	}
}
