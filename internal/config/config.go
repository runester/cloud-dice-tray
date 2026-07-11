// Package config loads cloud-dice-tray application configuration.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

const DefaultPath = "config.yaml"

// Config is the application's YAML configuration.
type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Security Security `yaml:"security"`
}

type Server struct {
	ListenAddress string `yaml:"listen_address"`
	BaseURL       string `yaml:"base_url"`
}

type Database struct {
	Path string `yaml:"path"`
}

type Security struct {
	SessionSecretFile string   `yaml:"session_secret_file"`
	AdminEmails       []string `yaml:"admin_emails"`
}

func Default() Config {
	return Config{Server: Server{ListenAddress: "127.0.0.1:8080", BaseURL: "http://127.0.0.1:8080"}}
}

// Load reads a YAML file strictly, rejecting unknown fields. Empty optional
// fields are retained for future increments that use the database and sessions.
func Load(path string) (Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	configuration := Default()
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)
	if err := decoder.Decode(&configuration); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	if configuration.Server.ListenAddress == "" {
		return Config{}, errors.New("server.listen_address is required")
	}
	return configuration, nil
}
