package db

import (
	"database/sql"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/FiraBro/local-go/internal/config"
	// Registers the "postgres" driver used by sql.Open.
	_ "github.com/lib/pq"
)

const redacted = "xxxxx"

func InitDB() *sql.DB {
	slog.Info("connecting to PostgreSQL", "addr", redactDSN(config.DBAddr))

	db, err := sql.Open("postgres", config.DBAddr)
	if err != nil {
		slog.Error("failed to open PostgreSQL connection", "error", scrub(err, config.DBAddr))
		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		slog.Error("PostgreSQL ping failed", "error", scrub(err, config.DBAddr))
		os.Exit(1)
	}

	slog.Info("PostgreSQL connected successfully")
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
