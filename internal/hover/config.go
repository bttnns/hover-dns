package hover

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the on-disk YAML config. Auth lives at the top level (shared by every
// service); each long-running service has its own nested block with an `enabled`
// toggle. Both services default to disabled: a bare config runs nothing.
type Config struct {
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	TOTPSecret string `yaml:"totp_secret"`

	DDNS DDNSConfig `yaml:"ddns"`
	API  APIConfig  `yaml:"api"`

	// SessionFile is not read from YAML — derived from config path.
	SessionFile string `yaml:"-"`
}

// DDNSConfig configures the dynamic-DNS loop.
type DDNSConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Domain      string   `yaml:"domain"`
	RecordNames []string `yaml:"record_names"`
	Interval    int      `yaml:"interval"`
}

// APIConfig configures the internal HTTP API. The API key itself is NOT stored
// here; it is read from the HOVER_API_KEY env var (a podman secret) at start.
type APIConfig struct {
	Enabled bool   `yaml:"enabled"`
	Listen  string `yaml:"listen"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("username and password are required in config")
	}
	cfg.SessionFile = filepath.Join(filepath.Dir(path), ".hover-dns.session")
	return &cfg, nil
}
