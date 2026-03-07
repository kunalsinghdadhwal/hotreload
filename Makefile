# hotreload — CLI hot-reload tool for Go
# Usage: make [target]
#   build    Build the hotreload binary
#   run      Build and run hotreload against testserver
#   demo     Alias for run (for demo recordings)
#   test     Run all tests with race detection
#   coverage Generate test coverage report
#   clean    Remove build artifacts
#   lint     Run linters

.PHONY: build run demo test coverage clean lint

build:
	@echo "Building hotreload..."
	@mkdir -p ./bin
	go build -o ./bin/hotreload ./cmd/hotreload

run: build
	@echo "Running hotreload with testserver..."
	./bin/hotreload --root ./testserver --build "go build -o ./bin/testserver ./testserver" --exec "./bin/testserver"

demo: run

test:
	@echo "Running tests..."
	go test ./... -v -race

coverage:
	@echo "Generating coverage report..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	@echo "Cleaning..."
	rm -rf ./bin coverage.out

lint:
	@echo "Running go vet..."
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running golangci-lint..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping"; \
	fi
