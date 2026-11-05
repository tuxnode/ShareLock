BIN     := sharelock
BIN_SRV := sharelock-server
CERT    := cert.pem
KEY     := key.pem

.PHONY: all build client server clean test vet bench gen-cert help \
        test-app test-encryption test-unit test-handler test-store test-integration \
        bench-store bench-handler bench-integration

all: build gen-cert

build: client server

client:
	go build -o $(BIN) ./cmd/sharelock

server:
	go build -o $(BIN_SRV) ./cmd/sharelock-server

gen-cert:
	@if [ ! -f $(CERT) ] || [ ! -f $(KEY) ]; then \
		openssl req -x509 -newkey rsa:2048 \
			-keyout $(KEY) -out $(CERT) \
			-days 365 -nodes \
			-subj "/CN=localhost" 2>/dev/null; \
		echo "generated $(CERT) / $(KEY)"; \
	else \
		echo "certificate already exists"; \
	fi

# --- Full test suite ---
test:
	go test ./... -count=1

test-app:
	go test -v -count=1 ./internal/client/app_test/

test-encryption:
	go test -v -count=1 ./internal/client/encryption_test/

test-unit:
	go test -v -count=1 ./internal/client/encryption/...

test-handler:
	go test -v -count=1 ./internal/server/handler/

test-store:
	go test -v -count=1 ./internal/server/store/

test-integration:
	go test -v -count=1 ./internal/integration_test/ -timeout=120s

test-userlib:
	go test -v -count=1 ./internal/userlib/

# --- Benchmarks ---
bench:
	go test ./... -bench=. -benchtime=1s

bench-store:
	go test -v -bench=. -benchtime=1s ./internal/server/store/

bench-handler:
	go test -v -bench=. -benchtime=1s ./internal/server/handler/

bench-integration:
	go test -v -bench=. -benchtime=1s -timeout=120s ./internal/integration_test/

# --- Code quality ---
vet:
	go vet ./...

# --- Cleanup ---
clean:
	rm -f $(BIN) $(BIN_SRV) $(CERT) $(KEY)
	go clean ./...

help:
	@echo "Usage:"
	@echo ""
	@echo "Build:"
	@echo "  make all        build binaries + generate dev certificate"
	@echo "  make build      build client and server binaries"
	@echo "  make client     build client binary only"
	@echo "  make server     build server binary only"
	@echo "  make gen-cert   generate self-signed TLS certificate"
	@echo ""
	@echo "Test:"
	@echo "  make test              run all tests"
	@echo "  make test-app          app client tests (-v)"
	@echo "  make test-encryption   encryption integration tests (-v)"
	@echo "  make test-unit         encryption unit tests (-v)"
	@echo "  make test-handler      handler protocol tests (-v)"
	@echo "  make test-store        KV store tests (-v)"
	@echo "  make test-integration  server TLS integration tests (-v)"
	@echo "  make test-userlib      userlib tests (-v)"
	@echo ""
	@echo "Benchmark:"
	@echo "  make bench             run all benchmarks"
	@echo "  make bench-store       KV store benchmarks (-v)"
	@echo "  make bench-handler     handler benchmarks (-v)"
	@echo "  make bench-integration TLS integration benchmarks (-v)"
	@echo ""
	@echo "Other:"
	@echo "  make vet        run go vet"
	@echo "  make clean      remove binaries and certificates"
