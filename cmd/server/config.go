package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Configuration precedence (highest wins):
//  1. command-line flags
//  2. process environment variables
//  3. .env file
//  4. built-in defaults
type options struct {
	addr       string
	dir        string
	cert       string
	key        string
	tlsEnabled bool
}

func loadConfig() options {
	loadDotEnv(envFilePath())

	addr := flag.String("address", getenv("SHARELOCK_ADDRESS", "localhost:8080"), "listen address")
	dir := flag.String("dir", getenv("SHARELOCK_DIR", "./data"), "data directory")
	cert := flag.String("cert", getenv("SHARELOCK_CERT", ""), "TLS cert file (required if -tls=true)")
	key := flag.String("key", getenv("SHARELOCK_KEY", ""), "TLS key file (required if -tls=true)")
	tlsEnabled := flag.Bool("tls", getBoolEnv("SHARELOCK_TLS", true), "enable TLS encryption")
	envFile := flag.String("env", envFilePath(), "path to .env file (optional)")
	flag.Parse()

	log.Printf("config: address=%s dir=%s tls=%t env=%s", *addr, *dir, *tlsEnabled, *envFile)

	return options{
		addr:       *addr,
		dir:        *dir,
		cert:       *cert,
		key:        *key,
		tlsEnabled: *tlsEnabled,
	}
}

// envFilePath resolves the .env location: -env flag > SHARELOCK_ENV_FILE > ".env".
// The flag is scanned manually because it must be known before flag.Parse.
func envFilePath() string {
	path := getenv("SHARELOCK_ENV_FILE", ".env")
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-env=") {
			return strings.TrimPrefix(a, "-env=")
		}
		if (a == "-env" || a == "--env") && i+1 < len(args) {
			return args[i+1]
		}
	}
	return path
}

// loadDotEnv loads the .env file into the process environment without
// overriding variables that are already set. A missing file is not an error.
func loadDotEnv(path string) {
	if path == "" {
		return
	}
	if err := godotenv.Load(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		log.Fatalf("load %s: %v", path, err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getBoolEnv(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Fatalf("invalid %s=%q: %v", key, v, err)
	}
	return b
}
