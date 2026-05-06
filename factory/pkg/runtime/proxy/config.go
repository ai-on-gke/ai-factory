package proxy

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

// ProxySpec defines the desired state of Proxy
type ProxySpec struct {
	ListenAddress string      `json:"listenAddress"`
	Rules         []ProxyRule `json:"rules"`
}

type ProxyRule struct {
	Name         string           `json:"name"`
	AllowedURLs  []string         `json:"allowedURLs"`
	AllowedVerbs []string         `json:"allowedVerbs"`
	Injection    *HeaderInjection `json:"injection,omitempty"`
}

type HeaderInjection struct {
	Header      string `json:"header"`
	Placeholder string `json:"placeholder"`
	SecretFile  string `json:"secretFile"`
	SecretValue string `json:"-"`
}

// ProxyConfig is the top-level configuration object.
type ProxyConfig struct {
	Kind       string    `json:"kind"`
	APIVersion string    `json:"apiVersion"`
	Spec       ProxySpec `json:"spec"`
}

// ParseConfig unmarshals the YAML data and validates the required fields.
func ParseConfig(data []byte) (*ProxyConfig, error) {
	var cfg ProxyConfig
	err := yaml.UnmarshalStrict(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.APIVersion != "factory.ai.gke.io/v1alpha1" {
		return nil, fmt.Errorf("invalid apiVersion: %s", cfg.APIVersion)
	}
	if cfg.Kind != "ProxySpec" {
		return nil, fmt.Errorf("invalid kind: %s", cfg.Kind)
	}

	if cfg.Spec.ListenAddress == "" {
		return nil, fmt.Errorf("listenAddress is required")
	}

	for i, rule := range cfg.Spec.Rules {
		if rule.Name == "" {
			return nil, fmt.Errorf("rule name is required at index %d", i)
		}
		if len(rule.AllowedURLs) == 0 {
			return nil, fmt.Errorf("allowedURLs is required for rule %s", rule.Name)
		}
		if len(rule.AllowedVerbs) == 0 {
			return nil, fmt.Errorf("allowedVerbs is required for rule %s", rule.Name)
		}
		if rule.Injection != nil {
			if rule.Injection.Header == "" {
				return nil, fmt.Errorf("injection header is required for rule %s", rule.Name)
			}
			if rule.Injection.Placeholder == "" {
				return nil, fmt.Errorf("injection placeholder is required for rule %s", rule.Name)
			}
			if strings.Contains(rule.Injection.Placeholder, " ") {
				return nil, fmt.Errorf("injection placeholder for rule %s should not contain spaces (do not include prefixes like 'Bearer ')", rule.Name)
			}
			if rule.Injection.SecretFile == "" {
				return nil, fmt.Errorf("injection secretFile is required for rule %s", rule.Name)
			}
			secretData, err := os.ReadFile(rule.Injection.SecretFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read secret file %q for rule %s: %w", rule.Injection.SecretFile, rule.Name, err)
			}
			cfg.Spec.Rules[i].Injection.SecretValue = strings.TrimRight(string(secretData), "\r\n\t ")
		}
	}

	return &cfg, nil
}

// LoadConfig reads the file from the given path and returns the parsed config.
func LoadConfig(path string) (*ProxyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	return ParseConfig(data)
}
