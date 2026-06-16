package main

import (
	"flag"
	"log"

	"github.com/cs161-staff/project2-starter-code/internal/server"
)

func main() {
	addr := flag.String("address", "localhost:8080", "listen address")
	dir := flag.String("dir", "./data", "data directory")
	cert := flag.String("cert", "", "TLS cert file")
	key := flag.String("key", "", "TLS key file")
	flag.Parse()

	if *cert == "" || *key == "" {
		log.Fatal("-cert and -key are required")
	}

	srv, err := server.New(server.Config{
		Addr:    *addr,
		DataDir: *dir,
		Cert:    *cert,
		Key:     *key,
	})
	if err != nil {
		log.Fatalf("server init: %v", err)
	}

	log.Fatal(srv.Run())
}
