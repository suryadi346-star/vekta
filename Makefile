BINARY     := vekta
BUILD_DIR  := .
CONFIG     := configs/app.json
GO         := go
GOFLAGS    := CGO_ENABLED=1

.PHONY: all build test bench run clean lint

all: build

## build: compile binary ke root project
build:
	$(GOFLAGS) $(GO) build -o $(BINARY) ./cmd/server/

## build-termux: compile untuk Termux/Android (arm64)
build-termux:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 $(GO) build -o $(BINARY)-arm64 ./cmd/server/

## test: jalankan semua unit test
test:
	$(GO) test ./... -race -count=1

## bench: jalankan benchmark cache
bench:
	$(GO) test -bench=. -benchmem -run='^$$' ./internal/cache/...

## run: build lalu jalankan server
run: build
	./$(BINARY)

## run-dev: jalankan langsung tanpa build step (lebih cepat saat dev)
run-dev:
	$(GOFLAGS) $(GO) run ./cmd/server/

## clean: hapus binary dan cache db
clean:
	rm -f $(BINARY) $(BINARY)-arm64
	rm -f cache/*.db cache/*.db-shm cache/*.db-wal

## lint: jalankan go vet
lint:
	$(GO) vet ./...

## tidy: sync go.mod dan go.sum
tidy:
	$(GO) mod tidy

## help: tampilkan semua target
help:
	@grep -E '^##' Makefile | sed 's/## //'
