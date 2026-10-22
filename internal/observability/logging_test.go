package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"log/slog"
)

func TestJSONLoggerIncludesRequestIDFromContext(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewJSONLogger("debug", &output)
	if err != nil {
		t.Fatal(err)
	}
	ctx := ContextWithRequestID(context.Background(), "6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	logger.InfoContext(ctx, "test event", "method", "GET")

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("invalid JSON log: %v", err)
	}
	if record["msg"] != "test event" {
		t.Fatalf("unexpected message: %#v", record["msg"])
	}
	if record["request_id"] != "6ba7b810-9dad-11d1-80b4-00c04fd430c8" {
		t.Fatalf("request ID missing: %#v", record)
	}
	if record["method"] != "GET" {
		t.Fatalf("method field missing: %#v", record)
	}
}

func TestNewJSONLoggerRejectsInvalidLevel(t *testing.T) {
	if _, err := NewJSONLogger("verbose", nil); err == nil {
		t.Fatal("expected invalid log level error")
	}
}

func TestJSONLoggerLevelFiltersLowerSeverity(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewJSONLogger("warn", &output)
	if err != nil {
		t.Fatal(err)
	}
	logger.Log(context.Background(), slog.LevelInfo, "hidden")
	logger.Warn("visible")
	if bytes.Contains(output.Bytes(), []byte("hidden")) || !bytes.Contains(output.Bytes(), []byte("visible")) {
		t.Fatalf("unexpected level filtering: %s", output.Bytes())
	}
}
