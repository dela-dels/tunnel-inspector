.PHONY: all build build-web build-server test clean dev-web

all: build

build-web:
	cd web && npm run build

build-server:
	go build -ldflags="-s -w" -o bin/tunnel-inspector ./cmd/tunnel-inspector

build: build-web build-server

GOBIN ?= $(shell go env GOPATH)/bin
INSTALL_DIR ?= $(HOME)/.local/bin

install: build
	mkdir -p $(INSTALL_DIR)
	cp bin/tunnel-inspector $(INSTALL_DIR)/tunnel-inspector
	ln -sf $(INSTALL_DIR)/tunnel-inspector $(INSTALL_DIR)/ti
	@echo "Installed tunnel-inspector and ti to $(INSTALL_DIR)"

test:
	go test -v ./...

clean:
	rm -rf bin/ web/dist/ *.db *.db-wal *.db-shm

dev-web:
	cd web && npm run dev
