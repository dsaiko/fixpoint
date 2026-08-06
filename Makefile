# fixpoint -- agent-agnostic automated code review cycle

BINARY  := fixpoint
# The config the static checks below validate. The run targets each name their
# own, so this is only the default for `make check`.
CONFIG  := fix-code

# Analysis tools are run via `go run` with pinned versions, so no global
# installs are required and CI and local runs use identical tool versions.
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6
STATICCHECK   := go run honnef.co/go/tools/cmd/staticcheck@2025.1.1
GOVULNCHECK   := go run golang.org/x/vuln/cmd/govulncheck@v1.6.0

.PHONY: list all build test test-race cover cover-html vet fmt fmt-check lint staticcheck vulncheck audit tidy tidy-check check check-live \
        fix-code fix-branch fix-pr review-code review-branch review-pr clean clean-logs run help

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

# One target per shipped config, so `make help` lists what can actually be run
# instead of one generic `run` whose behavior depends on a variable.
#
# --trusted-target is asserted here because every one of these targets reviews
# THIS repository, which we wrote. It is a per-invocation flag and never a config
# default -- see the security notes in config/README.md -- so a target pointed at
# somebody else's code would have to say so itself.
#
# The fix- targets run the test suite and vet first: they let an agent edit the
# working tree, and starting that from a tree whose tests already fail makes the
# verify gate's baseline meaningless.

## fix-code: review -> fix -> verify -> commit over the whole project
fix-code: build test vet
	./$(BINARY) fix-code --trusted-target

## fix-branch: the same cycle over only what this branch changed
## Needs an upstream (`git push -u`); without one, pass a base yourself, keeping
## the dots that ask for the merge base:
##   ./fixpoint fix-branch -base-ref 'origin/develop...' --trusted-target
fix-branch: build test vet
	./$(BINARY) fix-branch --trusted-target

## review-code: one review round over the whole project; nothing is modified
review-code: build
	./$(BINARY) review-code --trusted-target

## review-branch: one review round over only what this branch changed
review-branch: build
	./$(BINARY) review-branch --trusted-target

## fix-pr: review -> fix -> verify -> commit over a pull request, answering its
## open conversations. Pass the number as PR=<n>.
##   make fix-pr PR=170
## The most dangerous target here: it edits a tree holding externally-authored
## code, so it asserts -allow-untrusted-fix. Run review-pr first and read it.
fix-pr: build test vet
	@test -n "$(PR)" || { echo "usage: make fix-pr PR=<number>"; exit 2; }
	./$(BINARY) fix-pr -pr $(PR) --allow-untrusted-fix

## review-pr: review a pull request; pass the number as PR=<n>
##   make review-pr PR=170
review-pr: build
	@test -n "$(PR)" || { echo "usage: make review-pr PR=<number>"; exit 2; }
	./$(BINARY) review-pr -pr $(PR) --trusted-target

## run: removed -- name the config you mean (make fix-code, make review-pr PR=n)
run:
	@echo "There is no 'make run': it hid which config was about to spend money."
	@echo
	@$(MAKE) --no-print-directory help

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
## Only "## name: text" lines are listed; a "## " line without a target name is a
## continuation for someone reading the Makefile, and sorting those in among the
## targets made the list unreadable.
help:
	@grep -E '^## [a-z][a-z0-9-]*:' $(MAKEFILE_LIST) | sed 's/^## /  /' | sort
