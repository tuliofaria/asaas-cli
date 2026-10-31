package asaasclient

import (
	"fmt"
	"strings"
)

// Environment identifies which Asaas deployment a request targets.
type Environment string

const (
	Production Environment = "production"
	Sandbox    Environment = "sandbox"
)

// Base URLs of the Asaas API v3.
const (
	ProductionBaseURL = "https://api.asaas.com/v3"
	SandboxBaseURL    = "https://api-sandbox.asaas.com/v3"
)

// API key prefixes issued by Asaas. Keys generated in the sandbox
// ("homologação") start with $aact_hmlg_, production keys with $aact_prod_.
const (
	productionKeyPrefix = "$aact_prod_"
	sandboxKeyPrefix    = "$aact_hmlg_"
)

// ParseEnvironment converts user input into an Environment.
func ParseEnvironment(s string) (Environment, error) {
	switch Environment(strings.ToLower(strings.TrimSpace(s))) {
	case Production:
		return Production, nil
	case Sandbox:
		return Sandbox, nil
	}
	return "", fmt.Errorf("invalid environment %q: must be %q or %q", s, Production, Sandbox)
}

// Valid reports whether e is a known environment.
func (e Environment) Valid() bool {
	return e == Production || e == Sandbox
}

// BaseURL returns the API v3 base URL for the environment.
func (e Environment) BaseURL() string {
	if e == Sandbox {
		return SandboxBaseURL
	}
	return ProductionBaseURL
}

// EnvironmentFromKey infers the environment from the API key prefix.
// The boolean is false when the key has no recognizable prefix.
func EnvironmentFromKey(key string) (Environment, bool) {
	switch {
	case strings.HasPrefix(key, productionKeyPrefix):
		return Production, true
	case strings.HasPrefix(key, sandboxKeyPrefix):
		return Sandbox, true
	}
	return "", false
}

// MaskKey hides the secret part of an API key, keeping only the
// non-secret environment prefix (when present) and the last 4 characters.
func MaskKey(key string) string {
	if key == "" {
		return ""
	}
	prefix := ""
	rest := key
	for _, p := range []string{productionKeyPrefix, sandboxKeyPrefix} {
		if strings.HasPrefix(key, p) {
			prefix, rest = p, key[len(p):]
			break
		}
	}
	if len(rest) < 12 {
		return prefix + "****"
	}
	if prefix == "" {
		prefix = rest[:4]
	}
	return prefix + "****" + rest[len(rest)-4:]
}
