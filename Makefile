# fixpoint -- agent-agnostic automated code review cycle

BINARY  := fixpoint
CONFIG  := full-review

# Analysis tools are run via `go run` with pinned versions, so no global
# installs are required and CI and local runs use identical tool versions.
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6
STATICCHECK   := go run honnef.co/go/tools/cmd/staticcheck@2025.1.1
GOVULNCHECK   := go run golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: list all build test test-race cover cover-html vet fmt fmt-check lint staticcheck vulncheck audit tidy tidy-check check check-live run review-only clean clean-logs help

all: build

## build: compile the fixpoint binary
build:
	go build -o $(BINARY) ./cmd/fixpoint

## test: run unit tests
test:
	go test ./...

## test-race: run unit tests with the race detector
test-race:
	go test -race ./...

## cover: run unit tests and print per-function coverage
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## cover-html: run unit tests and open the coverage report in a browser
cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

## vet: run go vet
vet:
	go vet ./...

## fmt: format all Go sources
fmt:
	gofmt -l -w .

## fmt-check: fail if any Go source is not gofmt-formatted (for CI)
fmt-check:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "not gofmt-formatted:"; echo "$$out"; exit 1; fi

## lint: run golangci-lint over the whole module
lint:
	$(GOLANGCI_LINT) run ./...

## staticcheck: run staticcheck over the whole module
staticcheck:
	$(STATICCHECK) ./...

## vulncheck: scan dependencies and calls for known vulnerabilities
vulncheck:
	$(GOVULNCHECK) ./...

## audit: everything CI runs -- format check, vet, lint, staticcheck, race tests, vulnerability scan
audit: fmt-check vet lint staticcheck test-race vulncheck

## tidy: sync go.mod/go.sum
tidy:
	go mod tidy

## tidy-check: fail if go.mod/go.sum are not tidy (for CI)
tidy-check:
	go mod tidy -diff

## check: static validation of the configuration (no agents invoked)
check: build
	./$(BINARY) --check $(CONFIG) --trusted-target

## check-live: static validation + ping every configured agent
check-live: build
	./$(BINARY) --check-live $(CONFIG) --trusted-target

## run: run the full review->fix cycle per the configuration
## The trust gate is a per-invocation flag, never a config default: we assert it
## here because this target reviews fixpoint's own repository.
run: build test vet
	./$(BINARY) $(CONFIG) --trusted-target

## review-only: run a single review round; the coder is never invoked
review-only: build
	./$(BINARY) review-only --trusted-target

## clean: remove the built binary and coverage artifacts
clean:
	rm -f $(BINARY) coverage.out

## clean-logs: remove all run artifacts (logs/ is the pre-rename location)
clean-logs:
	rm -rf .fixpoint logs

## list: show the task configs available on the bundle search path
list: build
	./$(BINARY) --list

## help: list all targets with their descriptions
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | sort
