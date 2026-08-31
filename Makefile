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
GORELEASER    := go run github.com/goreleaser/goreleaser/v2@v2.12.5

.PHONY: list all build test test-race cover cover-html vet fmt fmt-check lint staticcheck vulncheck audit tidy tidy-check check check-live bench bench-seeds implement-go implement-node implement-web \
        fix-code fix-branch fix-pr review-code review-branch review-pr review-design create-design clean clean-logs run help \
        release-check release-snapshot release-verify

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
# Two DIFFERENT trust assertions appear below, and the difference is the whole
# point of having both. Every target here is built from the bundle under config/,
# which lives inside target.path (this project root), so every one of them needs
# bundle trust: those files are the argv fixpoint execs and the prompts it sends.
#
#   --trusted-bundle says only that. Nothing else consults it.
#   --trusted-target says that AND that the target's content is trusted when the
#   run reaches it -- which additionally permits fix rounds and, in mode pr,
#   downgrades to warnings the refusals that cover what `gh pr checkout` writes
#   (a target-relative agent command, and repository-selected content filters or
#   diff drivers). That claim is true for the -code and -branch targets, whose
#   target is this repository, and FALSE for the -pr targets, whose worktree is
#   filled with the pull request author's files -- so those pass the narrow flag.
#
# Both are per-invocation flags and never config defaults -- see the security
# notes in config/README.md -- so a target pointed at somebody else's code would
# have to say so itself.
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

## review-design: one review round over a design document or a project's
## architecture; pass the target as TARGET=<file-or-dir>
##   make review-design TARGET=docs/DESIGN.md
review-design: build
	@test -n "$(TARGET)" || { echo "usage: make review-design TARGET=<file-or-dir>"; exit 2; }
	./$(BINARY) review-design -target "$(TARGET)" --trusted-bundle

## create-design: draft a design document from an assignment file or directory
##   make create-design TARGET=assignment.md [OUT=path/DESIGN.md]
create-design: build
	@test -n "$(TARGET)" || { echo "usage: make create-design TARGET=<file-or-dir> [OUT=<file>]"; exit 2; }
	./$(BINARY) create-design -target "$(TARGET)" $(if $(OUT),-out "$(OUT)") --trusted-bundle

## implement-go / implement-node / implement-web: build a reviewed design
## into a new project, one task per commit
##   make implement-go TARGET=docs/DESIGN.md OUT=~/src/newproject
implement-go implement-node implement-web: build
	@test -n "$(TARGET)" -a -n "$(OUT)" || { echo "usage: make $@ TARGET=<design.md> OUT=<fresh-dir>"; exit 2; }
	./$(BINARY) $@ -target "$(TARGET)" -out "$(OUT)" --trusted-target --trusted-bundle

## bench: measure one model as a reviewer against the seeded targets;
## methodology and decision rule in bench/README.md
##   make bench MODEL=glm-5.2:cloud [TASK=code|design|all] [N=repeats]
bench:
	@test -n "$(MODEL)" || { echo "usage: make bench MODEL=<ollama-model> [TASK=code|design|all] [N=repeats]"; exit 2; }
	bench/run.sh "$(MODEL)" "$(or $(TASK),all)" "$(or $(N),1)"

## bench-seeds: per-seed hit rate across every scored run -- which seeds still
## discriminate, which are near-dead, which nobody has ever found. No quota.
BENCH_RUST_OUT ?= /tmp/fixpoint-bench-rust
BENCH_JAVA_OUT ?= /tmp/fixpoint-bench-java
BENCH_CS_OUT ?= /tmp/fixpoint-bench-csharp
BENCH_CPP_OUT ?= /tmp/fixpoint-bench-cpp

bench-seeds:
	python3 bench/report.py --seeds

## bench-check: validate both seed manifests against the frozen targets -- no model, no quota
## Every anchor must resolve to exactly one line, and every seed must recover
## its own finding instead of losing it to a neighbouring seed. Run it after
## any manifest or target edit; a broken manifest scores a model low and calls
## it a measurement.
bench-check:
	python3 bench/score.py --check bench/manifest-go.yaml >/dev/null
	python3 bench/score.py --check bench/manifest-rust.yaml >/dev/null
	python3 bench/score.py --check bench/manifest-java.yaml >/dev/null
	python3 bench/score.py --check bench/manifest-typescript.yaml >/dev/null
	python3 bench/score.py --check bench/manifest-csharp.yaml >/dev/null
	python3 bench/score.py --check bench/manifest-cpp.yaml >/dev/null
	python3 bench/score.py --check bench/manifest-python.yaml >/dev/null
	python3 bench/score.py --check bench/manifest-design.yaml >/dev/null
	python3 bench/score.py --selftest bench/manifest-go.yaml
	python3 bench/score.py --selftest bench/manifest-rust.yaml
	python3 bench/score.py --selftest bench/manifest-java.yaml
	python3 bench/score.py --selftest bench/manifest-typescript.yaml
	python3 bench/score.py --selftest bench/manifest-csharp.yaml
	python3 bench/score.py --selftest bench/manifest-cpp.yaml
	python3 bench/score.py --selftest bench/manifest-python.yaml
	python3 bench/score.py --selftest bench/manifest-design.yaml
	cd bench/testdata/target-go && go build ./...
	# --offline and an out-of-tree CARGO_TARGET_DIR: the target has no
	# dependencies, and a target/ directory inside the tree would be handed to a
	# reviewer along with the code.
	cd bench/testdata/target-rust && CARGO_TARGET_DIR=$(BENCH_RUST_OUT) cargo build --offline -q
	@mkdir -p $(BENCH_JAVA_OUT)
	cd bench/testdata/target-java && javac -nowarn -d $(BENCH_JAVA_OUT) $$(find src -name '*.java')
	# strict type-check only; the seeds are all defects tsc cannot catch
	cd bench/testdata/target-typescript && tsc -p .
	cd bench/testdata/target-csharp && dotnet build -v q --nologo -o $(BENCH_CS_OUT) >/dev/null
	@mkdir -p $(BENCH_CPP_OUT)
	cd bench/testdata/target-cpp && clang++ -std=c++20 -w -o $(BENCH_CPP_OUT)/linkd *.cpp
	# IMPORT, not just compile: py_compile accepts a dataclass with a mutable
	# default that `import` rejects outright, and a target that cannot be
	# imported is broken rather than seeded.
	cd bench/testdata/target-python && python3 -c 'import store, auth, cache, config, export, worker'
	@rm -rf bench/testdata/target-python/__pycache__
	@echo "bench: manifests and targets OK"

## review-branch: one review round over only what this branch changed
review-branch: build
	./$(BINARY) review-branch --trusted-target

## fix-pr: review -> fix -> verify -> commit over a pull request; POST=1 also answers its conversations
## Pass the number as PR=<n>.
##   make fix-pr PR=170
##   make fix-pr PR=170 POST=1
## POST=1 appends -post, the flag every reply path is gated on: without it the
## triage still runs and the answers are written into the run directory, but
## nothing reaches the pull request. Replies go out under your identity, so
## sending them is opt-in here for the same reason -post is a flag and not a
## config key. POST must be exactly 1 or unset; any other value is refused
## rather than read as true, so an inherited or mistyped POST cannot post.
## The most dangerous target here: it edits a tree holding externally-authored
## code, so it asserts -allow-untrusted-fix -- the flag that accepts the PR
## author's content, and the one the checkout guards are about. Run review-pr
## first and read it. -trusted-bundle is the separate, narrower claim: the target
## path defaults to this project root, so the bundle under config/ is
## project-supplied policy and the run is refused without it. Those are OUR files,
## read before the checkout replaced the tree.
fix-pr: build test vet
	@test -n "$(PR)" || { echo "usage: make fix-pr PR=<number> [POST=1]"; exit 2; }
	@test -z "$(POST)" || test "$(POST)" = 1 || { echo "POST must be 1 or unset, got '$(POST)'; replies are not sent"; exit 2; }
	./$(BINARY) fix-pr -pr $(PR) --allow-untrusted-fix --trusted-bundle $(if $(filter 1,$(POST)),-post)

## review-pr: review a pull request; pass the number as PR=<n>
##   make review-pr PR=170
## -trusted-bundle and NOT -trusted-target: the only thing this run needs to trust
## is fixpoint's own config/ bundle, resolved before `gh pr checkout` ran. The
## branch itself is externally authored, so the pr-mode refusals over what the
## checkout writes stay armed -- which is what makes this the target to run first
## on a fork's pull request.
review-pr: build
	@test -n "$(PR)" || { echo "usage: make review-pr PR=<number>"; exit 2; }
	./$(BINARY) review-pr -pr $(PR) --trusted-bundle

## run: removed -- name the config you mean (make fix-code, make review-pr PR=n)
# Exits 2, the usage-error code the PR= guards above use. `run` was the
# documented entry point for the whole review -> fix -> verify -> commit cycle,
# so a wrapper or CI step still calling it must not read this explanation as a
# completed run -- that is the same confusion the review exit codes exist to
# prevent.
run:
	@echo "There is no 'make run': it hid which config was about to spend money."
	@echo
	@$(MAKE) --no-print-directory help
	@exit 2

## clean: remove the built binary and coverage artifacts
clean:
	rm -f $(BINARY) coverage.out

## clean-logs: remove all run artifacts (logs/ is the pre-rename location)
clean-logs:
	rm -rf .fixpoint logs

## list: show the task configs available on the bundle search path
list: build
	./$(BINARY) --list

## release-check: validate .goreleaser.yaml without building anything
release-check:
	$(GORELEASER) check

## release-snapshot: build the full release locally -- archives and Linux
## packages -- without tagging or publishing. Output lands in dist/.
## This is how a packaging change is verified before a tag exists, because a
## tag that produces a broken archive cannot be taken back.
release-snapshot:
	$(GORELEASER) release --snapshot --clean --skip=publish
	@echo
	@echo "dist/ contains:" && ls dist/*.tar.gz dist/*.deb dist/*.rpm 2>/dev/null

## release-verify: unpack the snapshot archive and prove it runs -- the bundle
## resolves and a config is listed. The binary carries no fallback
## configuration, so an archive with a mis-shaped bundle installs a tool that
## refuses to run; both packagers flattened it on the first attempt.
release-verify: release-snapshot
	@set -e; \
	archive=$$(ls dist/fixpoint_*_$$(uname -s)_$$(uname -m | sed 's/x86_64/x86_64/;s/aarch64/arm64/;s/arm64/arm64/').tar.gz 2>/dev/null | head -1); \
	if [ -z "$$archive" ]; then echo "no archive for this platform in dist/"; exit 1; fi; \
	work=$$(mktemp -d); tar xzf "$$archive" -C "$$work"; \
	test -d "$$work/config/agents" || { echo "FAIL: bundle has no agents/"; exit 1; }; \
	test -d "$$work/config/prompts" || { echo "FAIL: bundle has no prompts/"; exit 1; }; \
	cd $$(mktemp -d) && "$$work/fixpoint" -version && "$$work/fixpoint" --list | grep -q review-code; \
	echo "OK: $$archive resolves its bundle and lists its configs"

## help: list all targets with their descriptions
## Only "## name: text" lines are listed; a "## " line without a target name is a
## continuation for someone reading the Makefile, and sorting those in among the
## targets made the list unreadable.
help:
	@grep -E '^## [a-z][a-z0-9-]*:' $(MAKEFILE_LIST) | sed 's/^## /  /' | sort
