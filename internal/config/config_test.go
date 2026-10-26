package config

import (
	"os"
	"testing"
)

func TestConfigValidationMatrix(t *testing.T) {
	tests := []struct {
		name        string
		envSetup    map[string]string
		expectError bool
	}{
		{
			name: "Valid environment parameters load successfully",
			envSetup: map[string]string{
				"APP_ENV":      "development",
				"DATABASE_URL": "postgres://localhost:5432/db",
				"JWT_SECRET":   "short-dev-secret",
			},
			expectError: false,
		},
		{
			name: "Missing critical database parameter fails cleanly",
			envSetup: map[string]string{
				"APP_ENV":    "development",
				"JWT_SECRET": "some-secret-key-string",
			},
			expectError: true,
		},
		{
			name: "Production environment rejects short weak keys",
			envSetup: map[string]string{
				"APP_ENV":      "production",
				"DATABASE_URL": "postgres://localhost:5432/db",
				"JWT_SECRET":   "weak-short-prod-secret", // #nosec G101 -- Deliberately weak test fixture.
			},
			expectError: true,
		},
		{
			name: "Production environment catches illegal wildcard CORS values",
			envSetup: map[string]string{
				"APP_ENV":              "production",
				"DATABASE_URL":         "postgres://localhost:5432/db",
				"JWT_SECRET":           "super-long-cryptographically-secure-string-key-value",
				"CORS_ALLOWED_ORIGINS": "*",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set every environment variable read by this test so inherited
			// developer or CI environment values cannot affect the result.
			for _, key := range []string{
				"APP_ENV",
				"DATABASE_URL",
				"JWT_SECRET",
				"CORS_ALLOWED_ORIGINS",
			} {
				t.Setenv(key, "")
			}
			for key, value := range tt.envSetup {
				t.Setenv(key, value)
			}

			cfg := &Config{
				AppEnv:              getEnv("APP_ENV", "development"),
				DatabaseURL:         os.Getenv("DATABASE_URL"),
				JWTSecret:           os.Getenv("JWT_SECRET"),
				MaxRequestBodyBytes: 2 << 20,
			}

			if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
				cfg.CORSAllowedOrigins = []string{origins}
			}

			err := cfg.assertSecurityProperties()
			if (err != nil) != tt.expectError {
				t.Errorf(
					"case %q: expected error validation status %t, got error: %v",
					tt.name,
					tt.expectError,
					err,
				)
			}
		})
	}
}
func TestConfigRejectsUnsupportedLogLevel(t *testing.T) {
	cfg := &Config{
		AppEnv:      "development",
		DatabaseURL: "postgres://localhost:5432/db",
		JWTSecret:   "development-secret",
		LogLevel:    "verbose",
	}
	if err := cfg.assertSecurityProperties(); err == nil {
		t.Fatal("expected unsupported log level to be rejected")
	}
}

func TestConfigRejectsNegativeTimeout(t *testing.T) {
	cfg := &Config{
		AppEnv:      "development",
		DatabaseURL: "postgres://localhost:5432/db",
		JWTSecret:   "development-secret",
		ReadTimeout: -1,
	}
	if err := cfg.assertSecurityProperties(); err == nil {
		t.Fatal("expected negative timeout to be rejected")
	}
}
