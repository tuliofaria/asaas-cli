package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tuliofaria/asaas-cli/pkg/asaasclient"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "nested", "asaas", "config.json"))
}

func TestStoreRoundTripAndPermissions(t *testing.T) {
	s := tempStore(t)

	cfg, err := s.Load()
	if err != nil || len(cfg.Accounts) != 0 {
		t.Fatalf("missing file should load empty, got %+v, %v", cfg, err)
	}

	cfg.ActiveAccount = "sandbox-dev"
	cfg.Accounts["sandbox-dev"] = Account{APIKey: "$aact_hmlg_1", Environment: "sandbox"}
	cfg.Accounts["producao-firma"] = Account{APIKey: "$aact_prod_2", Environment: "production"}
	if err := s.Save(cfg); err != nil {
		t.Fatal(err)
	}

	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.ActiveAccount != "sandbox-dev" || got.Accounts["producao-firma"].APIKey != "$aact_prod_2" {
		t.Errorf("round trip mismatch: %+v", got)
	}

	if runtime.GOOS != "windows" {
		info, _ := os.Stat(s.Path())
		if info.Mode().Perm() != 0o600 {
			t.Errorf("file perm = %v, want 0600", info.Mode().Perm())
		}
		dir, _ := os.Stat(filepath.Dir(s.Path()))
		if dir.Mode().Perm() != 0o700 {
			t.Errorf("dir perm = %v, want 0700", dir.Mode().Perm())
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(s.Path()))
	if len(entries) != 1 {
		t.Errorf("temp files left behind: %v", entries)
	}
}

func TestDir(t *testing.T) {
	if d, _ := Dir(Env{ConfigDir: "/custom"}); d != "/custom" {
		t.Errorf("ASAAS_CONFIG_DIR ignored: %s", d)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	if d, _ := Dir(Env{}); d != filepath.Join(home, ".config", "asaas") {
		t.Errorf("default dir = %s", d)
	}
}

func TestNormalizeAccountName(t *testing.T) {
	if n, err := NormalizeAccountName(" Sandbox-Dev "); err != nil || n != "sandbox-dev" {
		t.Errorf("got %q, %v", n, err)
	}
	for _, bad := range []string{"", "a.b", "-x", "with space", "ação"} {
		if _, err := NormalizeAccountName(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestResolvePriority(t *testing.T) {
	s := tempStore(t)
	cfg, _ := s.Load()
	cfg.ActiveAccount = "prod"
	cfg.Accounts["prod"] = Account{APIKey: "file-prod-key", Environment: "production"}
	cfg.Accounts["sbx"] = Account{APIKey: "file-sbx-key", Environment: "sandbox"}
	if err := s.Save(cfg); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		env        Env
		o          Overrides
		wantKey    string
		wantEnv    asaasclient.Environment
		wantSource string
		wantEnvSrc string
	}{
		{"active account", Env{}, Overrides{}, "file-prod-key", asaasclient.Production, SourceFile, EnvFromAccount},
		{"flag overrides account env", Env{}, Overrides{Sandbox: true}, "file-prod-key", asaasclient.Sandbox, SourceFile, EnvFromFlag},
		{"env var beats file", Env{APIKey: "env-key"}, Overrides{}, "env-key", asaasclient.Production, SourceEnv, EnvFromDefault},
		{"env var with --sandbox", Env{APIKey: "env-key"}, Overrides{Sandbox: true}, "env-key", asaasclient.Sandbox, SourceEnv, EnvFromFlag},
		{"env var with ASAAS_ENV", Env{APIKey: "env-key", Environment: "sandbox"}, Overrides{}, "env-key", asaasclient.Sandbox, SourceEnv, EnvFromVariable},
		{"env var key prefix", Env{APIKey: "$aact_hmlg_x"}, Overrides{}, "$aact_hmlg_x", asaasclient.Sandbox, SourceEnv, EnvFromKeyPrefix},
		{"--account beats env var", Env{APIKey: "env-key"}, Overrides{Account: "SBX"}, "file-sbx-key", asaasclient.Sandbox, SourceFile, EnvFromAccount},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := Resolve(tt.env, s, tt.o)
			if err != nil {
				t.Fatal(err)
			}
			if c.APIKey != tt.wantKey || c.Environment != tt.wantEnv || c.Source != tt.wantSource || c.EnvironmentSource != tt.wantEnvSrc {
				t.Errorf("got key=%s env=%s source=%s envsrc=%s", c.APIKey, c.Environment, c.Source, c.EnvironmentSource)
			}
			if c.BaseURL != c.Environment.BaseURL() {
				t.Errorf("base url = %s", c.BaseURL)
			}
		})
	}
}

func TestResolveErrors(t *testing.T) {
	s := tempStore(t)
	if _, err := Resolve(Env{}, s, Overrides{}); !errors.Is(err, ErrNoCredentials) {
		t.Errorf("want ErrNoCredentials, got %v", err)
	}
	var nf *AccountNotFoundError
	if _, err := Resolve(Env{}, s, Overrides{Account: "ghost"}); !errors.As(err, &nf) {
		t.Errorf("want AccountNotFoundError, got %v", err)
	}
	var ev *EnvVarError
	if _, err := Resolve(Env{APIKey: "k", Environment: "staging"}, s, Overrides{}); !errors.As(err, &ev) {
		t.Errorf("want EnvVarError, got %v", err)
	}
	c, err := Resolve(Env{APIKey: "k", BaseURL: "http://localhost:9/v3"}, s, Overrides{})
	if err != nil || c.BaseURL != "http://localhost:9/v3" {
		t.Errorf("ASAAS_BASE_URL ignored: %+v %v", c, err)
	}
}
