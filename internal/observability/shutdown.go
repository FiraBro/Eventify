package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ShutdownHTTPServer stops accepting requests, waits up to timeout for active
// requests, then cancels application contexts and force-closes connections if
// graceful shutdown cannot finish in time.
func ShutdownHTTPServer(server *http.Server, timeout time.Duration, cancelApplication context.CancelFunc, logger *slog.Logger) error {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if cancelApplication == nil {
		cancelApplication = func() {}
	}
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	err := server.Shutdown(ctx)
	cancel()
	if err != nil {
		logger.Error("graceful HTTP shutdown failed; forcing close", "error", err)
		cancelApplication()
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			logger.Error("forced HTTP server close failed", "error", closeErr)
			return errors.Join(fmt.Errorf("graceful HTTP shutdown: %w", err), fmt.Errorf("force-close HTTP server: %w", closeErr))
		}
		return fmt.Errorf("graceful HTTP shutdown: %w", err)
	}
	cancelApplication()
	logger.Info("HTTP server drained successfully")
	return nil
}
