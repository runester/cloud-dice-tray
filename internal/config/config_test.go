package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	path := writeConfig(t, `
server:
  listen_address: ":9090"
  base_url: "https://dice.example.com"
database:
  path: "/tmp/dice.db"
security:
  session_secret_file: "/tmp/secret"
  admin_emails:
    - "admin@example.com"
`)
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Server.ListenAddress != ":9090" || got.Database.Path != "/tmp/dice.db" {
		t.Fatalf("unexpected config: %+v", got)
	}
	if len(got.Security.AdminEmails) != 1 || got.Security.AdminEmails[0] != "admin@example.com" {
		t.Fatalf("unexpected admins: %v", got.Security.AdminEmails)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := writeConfig(t, "server:\n  listen_address: ':8080'\n  mystery: true\n")
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "field mystery not found") {
		t.Fatalf("got %v, want unknown-field error", err)
	}
}

func TestLoadRequiresListenAddress(t *testing.T) {
	path := writeConfig(t, "server:\n  listen_address: ''\n")
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "server.listen_address is required") {
		t.Fatalf("got %v, want required-field error", err)
	}
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
