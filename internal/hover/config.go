package hover

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	TOTPSecret string   `json:"totp_secret"`
	Domain      string   `json:"domain"`
	RecordNames []string `json:"record_names"`
	Interval    int      `json:"interval"`

	// SessionFile is not read from JSON — derived from config path
	SessionFile string `json:"-"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("username and password are required in config")
	}
	cfg.SessionFile = filepath.Join(filepath.Dir(path), ".hover-dns.session")
	return &cfg, nil
}
