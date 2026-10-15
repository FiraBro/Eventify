package repositories

import (
	"database/sql"
	"errors"
	"log/slog"
)

// rollbackTx is meant for `defer rollbackTx(tx)` right after BeginTx.
// After a successful Commit, Rollback returns sql.ErrTxDone, which is expected and ignored.
// Any other rollback failure is logged instead of being silently dropped.
func rollbackTx(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		slog.Error("transaction rollback failed", "error", err)
	}
}
