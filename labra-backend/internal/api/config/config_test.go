package config

import "testing"

func TestLoadValidConfig(t *testing.T) {
	cfg, err := Load(func(key string) string {
		vals := map[string]string{
			"APP_ENV":            "dev",
			"API_HOST":           "0.0.0.0",
			"API_PORT":           "9090",
			"DB_URL":             "file:test.db",
			"LOG_LEVEL":          "debug",
			"JWT_ISSUER":         "https://issuer.example.com",
			"JWT_AUDIENCE":       "labra-api",
			"JWT_SIGNING_SECRET": "super-secret",
		}
		return vals[key]
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Environment != "dev" {
		t.Fatalf("expected env dev, got %q", cfg.Environment)
	}
	if cfg.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.ListenAddress() != "0.0.0.0:9090" {
		t.Fatalf("unexpected listen addr: %s", cfg.ListenAddress())
	}
}

func TestLoadRejectsMissingDBURL(t *testing.T) {
	_, err := Load(func(key string) string {
		if key == "APP_ENV" {
			return "local"
		}
		return ""
	})
	if err == nil {
		t.Fatalf("expected error for missing DB_URL")
	}
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	_, err := Load(func(key string) string {
		vals := map[string]string{
			"APP_ENV": "qa",
			"DB_URL":  "file:test.db",
		}
		return vals[key]
	})
	if err == nil {
		t.Fatalf("expected invalid APP_ENV error")
	}
}
