// Package config parses server configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	IssuerURL string // e.g. https://oauth-tester.example.com
	ClientID  string
	CertFile  string // defaults to cert.pem
	KeyFile   string // defaults to key.pem
	DBPath    string // hardcoded to oauth-tester.db
	Addr      string // listen address, :443
}

func Load() (*Config, error) {
	issuer := os.Getenv("ISSUER_URL")
	if issuer == "" {
		return nil, fmt.Errorf("ISSUER_URL environment variable is required")
	}

	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("CLIENT_ID environment variable is required")
	}

	certFile := os.Getenv("TLS_CERT")
	if certFile == "" {
		certFile = "cert.pem"
	}

	keyFile := os.Getenv("TLS_KEY")
	if keyFile == "" {
		keyFile = "key.pem"
	}

	return &Config{
		IssuerURL: issuer,
		ClientID:  clientID,
		CertFile:  certFile,
		KeyFile:   keyFile,
		DBPath:    "oauth-tester.db",
		Addr:      ":443",
	}, nil
}
