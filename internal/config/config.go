// Package config manages the multi-account file stored at
// ~/.config/asaas/config.json and the ASAAS_* environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

const (
	dirName  = "asaas"
	fileName = "config.json"

	dirPerm  = 0o700
	filePerm = 0o600
)

// Account is one entry of the "accounts" map.
type Account struct {
	APIKey      string `mapstructure:"api_key" json:"api_key"`
	Environment string `mapstructure:"environment" json:"environment"`
}

// Config mirrors the JSON file:
//
//	{
//	  "active_account": "sandbox-dev",
//	  "accounts": {
//	    "sandbox-dev": { "api_key": "...", "environment": "sandbox" }
//	  }
//	}
type Config struct {
	ActiveAccount string             `mapstructure:"active_account" json:"active_account"`
	Accounts      map[string]Account `mapstructure:"accounts" json:"accounts"`
}

// Env holds the ASAAS_* environment variables, read through Viper.
type Env struct {
	APIKey      string // ASAAS_API_KEY
	Environment string // ASAAS_ENV ("sandbox" | "production")
	BaseURL     string // ASAAS_BASE_URL (override, mainly for proxies/tests)
	ConfigDir   string // ASAAS_CONFIG_DIR (override of the config directory)
}

// LoadEnv reads the ASAAS_* variables from the process environment.
func LoadEnv() Env {
	v := viper.New()
	v.SetEnvPrefix("ASAAS")
	v.AutomaticEnv()
	return Env{
		APIKey:      strings.TrimSpace(v.GetString("api_key")),
		Environment: strings.TrimSpace(v.GetString("env")),
		BaseURL:     strings.TrimSpace(v.GetString("base_url")),
		ConfigDir:   strings.TrimSpace(v.GetString("config_dir")),
	}
}

// Dir returns the configuration directory, in order of precedence:
// $ASAAS_CONFIG_DIR, $XDG_CONFIG_HOME/asaas, ~/.config/asaas.
func Dir(env Env) (string, error) {
	if env.ConfigDir != "" {
		return env.ConfigDir, nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, dirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", &FileError{Op: "locate home directory", Err: err}
	}
	return filepath.Join(home, ".config", dirName), nil
}

// Store reads and writes the config file.
type Store struct {
	path string
}

// NewStore returns a Store for an explicit file path.
func NewStore(path string) *Store { return &Store{path: path} }

// DefaultStore returns the Store at Dir(env)/config.json.
func DefaultStore(env Env) (*Store, error) {
	dir, err := Dir(env)
	if err != nil {
		return nil, err
	}
	return NewStore(filepath.Join(dir, fileName)), nil
}

// Path is the absolute location of the config file.
func (s *Store) Path() string { return s.path }

// Load reads the config file. A missing or empty file yields an empty Config.
func (s *Store) Load() (*Config, error) {
	cfg := &Config{Accounts: map[string]Account{}}

	info, err := os.Stat(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, &FileError{Op: "stat", Path: s.path, Err: err}
	}
	if info.Size() == 0 {
		return cfg, nil
	}

	v := viper.New()
	v.SetConfigFile(s.path)
	v.SetConfigType("json")
	if err := v.ReadInConfig(); err != nil {
		return nil, &FileError{Op: "read", Path: s.path, Err: err}
	}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, &FileError{Op: "decode", Path: s.path, Err: err}
	}
	if cfg.Accounts == nil {
		cfg.Accounts = map[string]Account{}
	}
	return cfg, nil
}

// Save atomically writes cfg with 0600 permissions, creating the directory
// (0700) when needed.
func (s *Store) Save(cfg *Config) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return &FileError{Op: "create directory", Path: dir, Err: err}
	}

	accounts := make(map[string]any, len(cfg.Accounts))
	for name, acc := range cfg.Accounts {
		accounts[name] = map[string]any{"api_key": acc.APIKey, "environment": acc.Environment}
	}

	v := viper.New()
	v.SetConfigType("json")
	v.SetConfigPermissions(filePerm)
	v.Set("active_account", cfg.ActiveAccount)
	v.Set("accounts", accounts)

	// Write to a temp file in the same directory, then rename, so a crash
	// never leaves a truncated config behind.
	tmp := filepath.Join(dir, fmt.Sprintf(".config-%d.tmp.json", os.Getpid()))
	if err := v.WriteConfigAs(tmp); err != nil {
		_ = os.Remove(tmp)
		return &FileError{Op: "write", Path: s.path, Err: err}
	}
	if err := os.Chmod(tmp, filePerm); err != nil {
		_ = os.Remove(tmp)
		return &FileError{Op: "chmod", Path: s.path, Err: err}
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return &FileError{Op: "write", Path: s.path, Err: err}
	}
	return nil
}

var accountNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// NormalizeAccountName lower-cases and validates an account nickname.
// Names are case-insensitive because Viper lower-cases map keys.
func NormalizeAccountName(name string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	if !accountNameRe.MatchString(n) {
		return "", fmt.Errorf("invalid account name %q: use 1-64 characters from [a-z0-9_-], starting with a letter or digit", name)
	}
	return n, nil
}

// FileError reports a failure reading or writing the config file.
type FileError struct {
	Op   string
	Path string
	Err  error
}

func (e *FileError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("config: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("config: %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *FileError) Unwrap() error { return e.Err }
