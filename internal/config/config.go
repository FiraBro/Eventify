package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Global validated configuration handle accessible by handlers and services.
var AppConfig *Config

// Backwards-compatible globals expected by internal/db and internal/utils packages.
var (
	JWTSecret string
	DBAddr    string
	SMTPHost  string
	SMTPPort  string
	SMTPUser  string
	SMTPPass  string
)

type Config struct {
	AppEnv              string
	Port                string
	DatabaseURL         string
	RedisURL            string
	JWTSecret           string
	ChapaSecretKey      string
	ChapaWebhookSecret  string
	CORSAllowedOrigins  []string
	MaxRequestBodyBytes int64
	LogLevel            string
	ReadHeaderTimeout   time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	HealthCheckTimeout  time.Duration
	ShutdownTimeout     time.Duration

	// Pprof is only available when explicitly enabled in development.
	PprofEnabled bool
	PprofPort    string

	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
}

// ValidateConfig parses, maps, and rigorously tests environment parameters once at startup.
func ValidateConfig() {
	for _, key := range []string{
		"READ_HEADER_TIMEOUT",
		"READ_TIMEOUT",
		"WRITE_TIMEOUT",
		"IDLE_TIMEOUT",
		"HEALTH_CHECK_TIMEOUT",
		"SHUTDOWN_TIMEOUT",
	} {
		if value := os.Getenv(key); value != "" {
			duration, err := time.ParseDuration(value)
			if err != nil || duration <= 0 {
				fmt.Fprintf(
					os.Stderr,
					"CRITICAL CONFIGURATION ERROR: %s must be a positive Go duration\n",
					key,
				)
				os.Exit(1)
			}
		}
	}

	cfg := &Config{
		AppEnv:              getEnv("APP_ENV", "development"),
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		RedisURL:            getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		ChapaSecretKey:      os.Getenv("CHAPA_SECRET_KEY"),
		ChapaWebhookSecret:  os.Getenv("CHAPA_WEBHOOK_SECRET"),
		MaxRequestBodyBytes: parseEnvInt64("MAX_REQUEST_BODY_BYTES", 2<<20), // 2MB default
		ReadHeaderTimeout:   parseEnvDuration("READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:         parseEnvDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:        parseEnvDuration("WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:         parseEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		HealthCheckTimeout:  parseEnvDuration("HEALTH_CHECK_TIMEOUT", 2*time.Second),
		ShutdownTimeout:     parseEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),

		PprofEnabled: strings.EqualFold(
			strings.TrimSpace(os.Getenv("PPROF_ENABLED")),
			"true",
		),
		PprofPort: getEnv("PPROF_PORT", "6060"),

		LogLevel: strings.ToLower(strings.TrimSpace(getEnv("LOG_LEVEL", "info"))),
		SMTPHost: os.Getenv("SMTP_HOST"),
		SMTPPort: os.Getenv("SMTP_PORT"),
		SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPass: os.Getenv("SMTP_PASS"),
	}

	// Pprof must never be enabled outside development.
	if cfg.PprofEnabled && cfg.AppEnv != "development" {
		fmt.Fprintln(
			os.Stderr,
			"CRITICAL CONFIGURATION ERROR: pprof is only allowed in development",
		)
		os.Exit(1)
	}
	// Validate the profiling port when pprof is enabled.
	if cfg.PprofEnabled {
		port, err := strconv.Atoi(cfg.PprofPort)
		if err != nil || port < 1 || port > 65535 {
			fmt.Fprintln(
				os.Stderr,
				"CRITICAL CONFIGURATION ERROR: PPROF_PORT must be an integer between 1 and 65535",
			)
			os.Exit(1)
		}
	}

	origins := getEnv(
		"CORS_ALLOWED_ORIGINS",
		"http://localhost:3000,http://localhost:5173",
	)
	cfg.CORSAllowedOrigins = strings.Split(origins, ",")

	// Trigger the strict fail-safe validation rules engine.
	if err := cfg.assertSecurityProperties(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ CRITICAL CONFIGURATION ERROR: %v\n", err)
		os.Exit(1)
	}

	// Assign globals cleanly for cross-module accessibility compatibility.
	AppConfig = cfg
	JWTSecret = cfg.JWTSecret
	DBAddr = cfg.DatabaseURL
	SMTPHost = cfg.SMTPHost
	SMTPPort = cfg.SMTPPort
	SMTPUser = cfg.SMTPUser
	SMTPPass = cfg.SMTPPass
}

func (c *Config) assertSecurityProperties() error {
	isProd := c.AppEnv == "production"

	switch strings.ToLower(strings.TrimSpace(c.LogLevel)) {
	case "", "debug", "info", "warn", "warning", "error":
	default:
		return errors.New("LOG_LEVEL must be one of debug, info, warn, or error")
	}

	for name, duration := range map[string]time.Duration{
		"READ_HEADER_TIMEOUT":  c.ReadHeaderTimeout,
		"READ_TIMEOUT":         c.ReadTimeout,
		"WRITE_TIMEOUT":        c.WriteTimeout,
		"IDLE_TIMEOUT":         c.IdleTimeout,
		"HEALTH_CHECK_TIMEOUT": c.HealthCheckTimeout,
		"SHUTDOWN_TIMEOUT":     c.ShutdownTimeout,
	} {
		if duration < 0 {
			return fmt.Errorf("%s must not be negative", name)
		}
	}

	// 1. Enforce Database Connection Strings existence.
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL must be specified to establish a safe baseline connection pool")
	}

	// 2. Enforce explicit, high-entropy JWT secrets in production boundaries.
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET cannot be empty")
	}
	if isProd && len(c.JWTSecret) < 32 {
		return fmt.Errorf(
			"JWT_SECRET token signature key length must be at least 32 characters long in production mode, got %d bytes",
			len(c.JWTSecret),
		)
	}

	// 3. Enforce valid payment gateways parameters if configured or in production.
	if c.ChapaSecretKey == "" && isProd {
		return errors.New("CHAPA_SECRET_KEY is mandatory in production mode to accept gateway processing requests")
	}
	if c.ChapaWebhookSecret == "" && isProd {
		return errors.New("CHAPA_WEBHOOK_SECRET is mandatory in production mode to prevent unauthenticated webhook replays")
	}

	// 4. Validate CORS origins string array configurations format.
	for _, origin := range c.CORSAllowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" && isProd {
			return errors.New("permissive wildcard origin '*' is completely blocked on production endpoints")
		}
		if _, err := url.ParseRequestURI(trimmed); err != nil {
			return fmt.Errorf(
				"invalid CORS origin layout specified in configuration settings: %s",
				origin,
			)
		}
	}

	// 5. Enforce sanity constraints over request body sizing parameters.
	if c.MaxRequestBodyBytes <= 0 || c.MaxRequestBodyBytes > (50<<20) {
		return fmt.Errorf(
			"invalid request body byte constraint range parameter: %d",
			c.MaxRequestBodyBytes,
		)
	}

	return nil
}

// Parsing helpers wrapping configuration fallback boundaries securely.
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func parseEnvInt64(key string, fallback int64) int64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}

	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return fallback
	}

	return val
}

func parseEnvDuration(key string, fallback time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}

	val, err := time.ParseDuration(valStr)
	if err != nil {
		return fallback
	}

	return val
}
