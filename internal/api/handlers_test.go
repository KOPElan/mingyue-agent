package api

import (
	"testing"

	"github.com/KOPElan/mingyue-agent/internal/config"
)

func TestBuildAPIURLsDisabled(t *testing.T) {
	cfg := &config.Config{
		API: config.APIConfig{
			EnableHTTP: false,
		},
	}

	if urls := buildAPIURLs(cfg, "host"); urls != nil {
		t.Fatalf("expected no URLs when HTTP disabled, got %v", urls)
	}
}

func TestBuildAPIURLsUsesHostnameAndTLS(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			ListenAddr: "0.0.0.0",
			HTTPPort:   8443,
		},
		API: config.APIConfig{
			EnableHTTP: true,
			TLSCert:    "/tmp/cert.pem",
			TLSKey:     "/tmp/key.pem",
		},
	}

	urls := buildAPIURLs(cfg, "agent-host")
	if len(urls) != 1 {
		t.Fatalf("expected single URL, got %v", urls)
	}

	expected := "https://agent-host:8443/api/v1"
	if urls[0] != expected {
		t.Fatalf("expected %q, got %q", expected, urls[0])
	}
}
