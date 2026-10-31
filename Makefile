VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
PREFIX  ?= $(HOME)/.local
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install uninstall test

build:
	go build -ldflags "$(LDFLAGS)" -o bin/asaas .

install:
	mkdir -p $(PREFIX)/bin
	go build -ldflags "$(LDFLAGS)" -o $(PREFIX)/bin/asaas .

uninstall:
	rm -f $(PREFIX)/bin/asaas

test:
	go test -race ./...
