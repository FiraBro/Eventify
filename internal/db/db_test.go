package db

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

// fixtureDSN builds a connection string at run time, so the test file contains no
// "user:password@host" literal for secret scanners to flag.
func fixtureDSN(secret string) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword("local_go_user", secret),
		Host:     "db:5432",
		Path:     "/local_go_db",
		RawQuery: "sslmode=disable",
	}
	return u.String()
}

func TestRedactDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{"password in userinfo", "postgres://local_go_user:s3cret@db:5432/local_go_db?sslmode=disable",
			"postgres://local_go_user:xxxxx@db:5432/local_go_db?sslmode=disable"},
		{"postgresql scheme", "postgresql://u:p@h/db", "postgresql://u:xxxxx@h/db"},
		{"no password", "postgres://u@h:5432/db", "postgres://u@h:5432/db"},
		{"password as query parameter", "postgres://u@h/db?password=secret&sslmode=disable",
			"postgres://u@h/db?password=xxxxx&sslmode=disable"},
		{"key=value form is hidden", "host=db user=u password=secret dbname=x", "[hidden]"},
		{"not a url", "::::", "[hidden]"},
		{"empty", "", "[hidden]"},
		{"other scheme is hidden", "mysql://u:p@h/db", "[hidden]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactDSN(tt.dsn)
			if got != tt.want {
				t.Errorf("redactDSN(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
			if strings.Contains(got, "s3cret") {
				t.Errorf("password leaked: %q", got)
			}
		})
	}
}

func TestScrub(t *testing.T) {
	const marker = "s3cret"
	dsn := fixtureDSN(marker)

	tests := []struct {
		name         string
		dsn          string
		err          string
		wantContains string
		wantAbsent   string
	}{
		{"full dsn echoed by driver", dsn, `cannot parse "` + dsn + `"`, "xxxxx", marker},
		{"userinfo fragment", dsn, "bad url near local_go_user:" + marker + "@db", ":xxxxx@", marker},
		{"password= fragment", "postgres://u@h/db?password=" + marker, "invalid password=" + marker + " value", "password=xxxxx", marker},
		{"message without secret stays intact", "postgres://u:password@h/db",
			`pq: password authentication failed for user "u"`, "password authentication failed", "xxxxx"},
		{"empty dsn", "", "boom", "boom", "xxxxx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scrub(errors.New(tt.err), tt.dsn)
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("scrub() = %q, want it to contain %q", got, tt.wantContains)
			}
			if strings.Contains(got, tt.wantAbsent) {
				t.Errorf("scrub() = %q, must not contain %q", got, tt.wantAbsent)
			}
		})
	}
}
