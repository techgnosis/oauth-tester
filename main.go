package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"oauth-tester/internal/config"
	"oauth-tester/internal/crypto"
	"oauth-tester/internal/server"
	"oauth-tester/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	keys, err := crypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate keys: %w", err)
	}
	log.Printf("Generated RSA key pair (kid=%s)", keys.KID)

	db, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer db.Close()

	mux := server.New(&server.Deps{
		Config: cfg,
		Store:  db,
		Keys:   keys,
	})

	log.Printf("Listening on %s (TLS)", cfg.Addr)
	return http.ListenAndServeTLS(cfg.Addr, cfg.CertFile, cfg.KeyFile, mux)
}
