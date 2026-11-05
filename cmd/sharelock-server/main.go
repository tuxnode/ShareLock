package main

import (
	"log"

	"github.com/tuxnode/ShareLock/internal/server"
)

var version = "dev"

func main() {
	log.Printf("sharelock-server %s", version)

	cfg := loadConfig()

	if cfg.tlsEnabled && (cfg.cert == "" || cfg.key == "") {
		log.Fatal("-cert and -key are required when -tls=true")
	}

	srv, err := server.New(server.Config{
		Addr:       cfg.addr,
		DataDir:    cfg.dir,
		Cert:       cfg.cert,
		Key:        cfg.key,
		TLSEnabled: cfg.tlsEnabled,
	})
	if err != nil {
		log.Fatalf("server init: %v", err)
	}

	log.Fatal(srv.Run())
}
