package config

import (
	"os"
	"testing"
)

func TestConfigValidationMatrix(t *testing.T) {
	// Preserving original environment parameters cleanly before isolation context changes
	origEnv := os.Getenv("APP_ENV")
	origDb := os.Getenv("DATABASE_URL")
	origJwt := os.Getenv("JWT_SECRET")
	defer func() {
		os.Setenv("APP_ENV", origEnv)
		os.Setenv("DATABASE_URL", origDb)
		os.Setenv("JWT_SECRET", origJwt)
	}()

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
				"JWT_SECRET":   "weak-short-prod-secret", // Under 32 characters
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
			os.Clearenv()
			for k, v := range tt.envSetup {
				os.Setenv(k, v)
			}

			cfg := &Config{
				AppEnv:              getEnv("APP_ENV", "development"),
				DatabaseURL:         os.Getenv("DATABASE_URL"),
				JWTSecret:           os.Getenv("JWT_SECRET"),
				MaxRequestBodyBytes: 2 << 20,
			}
			origins := os.Getenv("CORS_ALLOWED_ORIGINS")
			if origins != "" {
				cfg.CORSAllowedOrigins = []string{origins}
			}

			err := cfg.assertSecurityProperties()
			if (err != nil) != tt.expectError {
				t.Errorf("case %q: expected error validation status %t, got error: %v", tt.name, tt.expectError, err)
			}
		})
	}
}
