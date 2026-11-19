package config

import (
	"strings"
	"testing"
)

func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoad(t *testing.T) {
	const dsn = "postgres://u:p@localhost:5432/db?sslmode=disable"

	tests := []struct {
		name     string
		vars     map[string]string
		wantPort string
		wantErr  string // substring of the expected error, empty means success
	}{
		{"defaults the port to 8080", map[string]string{"DATABASE_URL": dsn}, "8080", ""},
		{"uses PORT when set", map[string]string{"DATABASE_URL": dsn, "PORT": "9000"}, "9000", ""},
		{"database url is required", map[string]string{}, "", "DATABASE_URL"},
		{"port must be a number", map[string]string{"DATABASE_URL": dsn, "PORT": "abc"}, "", "PORT"},
		{"port must not be zero", map[string]string{"DATABASE_URL": dsn, "PORT": "0"}, "", "PORT"},
		{"port must be in range", map[string]string{"DATABASE_URL": dsn, "PORT": "70000"}, "", "PORT"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := load(env(tc.vars))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want one mentioning %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Port != tc.wantPort || cfg.DatabaseURL != dsn {
				t.Errorf("got %+v", cfg)
			}
		})
	}
}
