# GhostOperator (GO) - Makefile

BINARY_NAME=ghost.exe
GO_CMD=go

.PHONY: all build test clean run-cli help

all: build test

build:
	$(GO_CMD) build -ldflags "-s -w" -o $(BINARY_NAME) ./cmd/ghost

test:
	$(GO_CMD) test ./internal/...
	$(GO_CMD) test ./pkg/...

clean:
	rm -f $(BINARY_NAME)
	rm -f debug_view.png
	rm -f config.json

run-cli: build
	./$(BINARY_NAME) start

help:
	@echo "GhostOperator v2.0 Makefile"
	@echo "  build    - Compile the production binary"
	@echo "  test     - Run all internal unit tests"
	@echo "  clean    - Remove build artifacts and logs"
	@echo "  run-cli  - Build and start the CGO-free engine"
