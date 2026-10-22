package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/config"
	// Registers the "postgres" driver used by sql.Open.
	_ "github.com/lib/pq"
)

const redacted = "xxxxx"

func OpenDB() (*sql.DB, error) {
	slog.Info("connecting to PostgreSQL", "addr", redactDSN(config.DBAddr))
	db, err := sql.Open("postgres", config.DBAddr)
	if err != nil {
		safeErr := errors.New(scrub(err, config.DBAddr))
		slog.Error("failed to open PostgreSQL connection", "error", safeErr)
		return nil, safeErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		safeErr := errors.New(scrub(err, config.DBAddr))
		slog.Error("PostgreSQL ping failed", "error", safeErr)
		return nil, safeErr
	}

	slog.Info("PostgreSQL connected successfully")
	return db, nil
}

// InitDB is retained for compatibility with older internal callers. New startup
// code should use OpenDB so it can unwind initialization without exiting here.
func InitDB() *sql.DB {
	db, err := OpenDB()
	if err != nil {
		os.Exit(1)
	}
	return db
}

// redactDSN returns a version of the connection string that is safe to log.
// Anything that is not a postgres:// URL is hidden completely.
func redactDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return "[hidden]"
	}

	if u.User != nil {
		if _, ok := u.User.Password(); ok {
			u.User = url.UserPassword(u.User.Username(), redacted)
		}
	}

	if q := u.Query(); q.Has("password") {
		q.Set("password", redacted)
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// scrub returns err's message with the password removed. Driver errors can echo parts of
// the connection string. Only specific patterns are replaced (never a bare substring), so a
// password such as "password" cannot mangle messages like "password authentication failed".
func scrub(err error, dsn string) string {
	msg := err.Error()
	if dsn == "" {
		return msg
	}

	msg = strings.ReplaceAll(msg, dsn, redactDSN(dsn))

	u, parseErr := url.Parse(dsn)
	if parseErr != nil {
		return msg
	}
	var secrets []string
	if u.User != nil {
		if p, ok := u.User.Password(); ok && p != "" {
			secrets = append(secrets, p)
		}
	}
	if p := u.Query().Get("password"); p != "" {
		secrets = append(secrets, p)
	}
	for _, p := range secrets {
		msg = strings.ReplaceAll(msg, ":"+p+"@", ":"+redacted+"@")
		msg = strings.ReplaceAll(msg, "password="+p, "password="+redacted)
	}
	return msg
}
