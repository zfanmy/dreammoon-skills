package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultConfig returns default configuration for our cluster
type Config struct {
	Nodes     []NodeConfig  `yaml:"nodes"`
	Settings  Settings      `yaml:"settings"`
	DataDir   string        `yaml:"data_dir"`
}

type NodeConfig struct {
	Name     string   `yaml:"name"`
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port,omitempty"`
	User     string   `yaml:"user,omitempty"`
	Auth     string   `yaml:"auth"`      // key | password | agent
	KeyPath  string   `yaml:"key_path,omitempty"`
	Local    bool     `yaml:"local,omitempty"`
	Tags     []string `yaml:"tags,omitempty"`
	GPUType  string   `yaml:"gpu_type,omitempty"`
}

type Settings struct {
	ScanTimeout     string            `yaml:"scan_timeout"`
	SSHPoolSize     int               `yaml:"ssh_pool_size"`
	CacheTTL        string            `yaml:"cache_ttl"`
	AlertThresholds AlertThresholds   `yaml:"alert_thresholds"`
}

type AlertThresholds struct {
	CPUPercent    int `yaml:"cpu_percent"`
	MemoryPercent int `yaml:"memory_percent"`
	DiskPercent   int `yaml:"disk_percent"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		DataDir: defaultDataDir(),
		Settings: Settings{
			ScanTimeout: "10s",
			SSHPoolSize: 2,
			CacheTTL:    "30s",
			AlertThresholds: AlertThresholds{
				CPUPercent:    90,
				MemoryPercent: 85,
				DiskPercent:   90,
			},
		},
	}

	// Load from file if exists
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}

	// If no nodes configured, use defaults based on known cluster
	if len(cfg.Nodes) == 0 {
		cfg.Nodes = defaultNodes()
	}

	return cfg, nil
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".env-manager"
	}
	return filepath.Join(home, ".openclaw", "skills", "env-manager", "data")
}

func defaultNodes() []NodeConfig {
	// Return a single generic example node instead of hardcoding personal cluster
	// Users should create their own config.yaml with real nodes
	return []NodeConfig{
		{
			Name:  "example-node",
			Host:  "192.168.1.10",
			Port:  22,
			User:  "user",
			Auth:  "key",
			KeyPath: "~/.ssh/id_rsa",
			Tags:  []string{"amd64", "linux"},
		},
	}
}

func (c *Config) GetNode(name string) (NodeConfig, bool) {
	for _, n := range c.Nodes {
		if n.Name == name {
			return n, true
		}
	}
	return NodeConfig{}, false
}
