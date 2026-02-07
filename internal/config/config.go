package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	API      APIConfig      `yaml:"api"`
	Audit    AuditConfig    `yaml:"audit"`
	Security SecurityConfig `yaml:"security"`
	NetDisk  NetDiskConfig  `yaml:"netdisk"`
	Network  NetworkConfig  `yaml:"network"`
	ShareMgr ShareMgrConfig `yaml:"sharemgr"`
}

type ServerConfig struct {
	ListenAddr string `yaml:"listen_addr"`
	HTTPPort   int    `yaml:"http_port"`
	GRPCPort   int    `yaml:"grpc_port"`
	UDSPath    string `yaml:"uds_path"`
}

type APIConfig struct {
	EnableHTTP bool   `yaml:"enable_http"`
	EnableGRPC bool   `yaml:"enable_grpc"`
	EnableUDS  bool   `yaml:"enable_uds"`
	TLSCert    string `yaml:"tls_cert"`
	TLSKey     string `yaml:"tls_key"`
}

type AuditConfig struct {
	Enabled    bool   `yaml:"enabled"`
	LogPath    string `yaml:"log_path"`
	RemotePush bool   `yaml:"remote_push"`
	RemoteURL  string `yaml:"remote_url"`
}

type SecurityConfig struct {
	EnableMTLS      bool     `yaml:"enable_mtls"`
	TokenAuth       bool     `yaml:"token_auth"`
	AllowedPaths    []string `yaml:"allowed_paths"`
	MaxUploadSize   int64    `yaml:"max_upload_size"`
	RateLimitPerMin int      `yaml:"rate_limit_per_min"`
	RequireConfirm  bool     `yaml:"require_confirm"`
}

type NetDiskConfig struct {
	AllowedHosts       []string `yaml:"allowed_hosts"`
	AllowedMountPoints []string `yaml:"allowed_mount_points"`
	EncryptionKey      string   `yaml:"encryption_key"`
	StateFile          string   `yaml:"state_file"`
}

type NetworkConfig struct {
	ManagementInterface string `yaml:"management_interface"`
	HistoryFile         string `yaml:"history_file"`
	ConfigDir           string `yaml:"config_dir"`
}

type ShareMgrConfig struct {
	AllowedPaths []string `yaml:"allowed_paths"`
	SambaConfig  string   `yaml:"samba_config"`
	NFSConfig    string   `yaml:"nfs_config"`
	BackupDir    string   `yaml:"backup_dir"`
	StateFile    string   `yaml:"state_file"`
}

func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			ListenAddr: "0.0.0.0",
			HTTPPort:   8080,
			GRPCPort:   9090,
			UDSPath:    "/var/run/mingyue-agent/agent.sock",
		},
		API: APIConfig{
			EnableHTTP: true,
			EnableGRPC: true,
			EnableUDS:  true,
		},
		Audit: AuditConfig{
			Enabled:    true,
			LogPath:    "/var/log/mingyue-agent/audit.log",
			RemotePush: false,
		},
		Security: SecurityConfig{
			EnableMTLS:      false,
			TokenAuth:       true,
			AllowedPaths:    []string{"/home", "/data"},
			MaxUploadSize:   10 * 1024 * 1024 * 1024,
			RateLimitPerMin: 1000,
			RequireConfirm:  true,
		},
		NetDisk: NetDiskConfig{
			AllowedHosts:       []string{"*"},
			AllowedMountPoints: []string{"/mnt", "/media"},
			EncryptionKey:      "change-this-to-a-secure-key-32b",
			StateFile:          "/var/lib/mingyue-agent/netdisk-state.json",
		},
		Network: NetworkConfig{
			ManagementInterface: "",
			HistoryFile:         "/var/lib/mingyue-agent/network-history.json",
			ConfigDir:           "/etc/mingyue-agent/network",
		},
		ShareMgr: ShareMgrConfig{
			AllowedPaths: []string{"/home", "/data", "/mnt", "/media"},
			SambaConfig:  "/etc/samba/smb.conf",
			NFSConfig:    "/etc/exports",
			BackupDir:    "/var/lib/mingyue-agent/share-backups",
			StateFile:    "/var/lib/mingyue-agent/share-state.json",
		},
	}
}

func (c *Config) Validate() error {
	if c.Server.HTTPPort < 1 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid http_port: %d", c.Server.HTTPPort)
	}
	if c.Server.GRPCPort < 1 || c.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid grpc_port: %d", c.Server.GRPCPort)
	}
	if (c.API.TLSCert == "") != (c.API.TLSKey == "") {
		return fmt.Errorf("tls_cert and tls_key must both be set")
	}
	if c.API.EnableHTTP && c.API.TLSCert != "" {
		if _, err := os.Stat(c.API.TLSCert); err != nil {
			return fmt.Errorf("tls_cert not found: %w", err)
		}
	}
	if c.API.EnableHTTP && c.API.TLSKey != "" {
		if _, err := os.Stat(c.API.TLSKey); err != nil {
			return fmt.Errorf("tls_key not found: %w", err)
		}
	}
	return nil
}

func (c *Config) SaveExample(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
