# winze — build the KB tools.
#
# `go run ./cmd/foo` recompiles on every invocation, which costs ~0.5s before
# the tool does any work. That is fine for batch phases and badly wrong for
# interactive querying, where the tool itself answers in ~25ms — an 18x tax on
# the operation a knowledge base exists to make cheap. Build once, then query
# at native speed.

CMDS := query lint topology metabolism add edit sensor rot-probe predicates-suggest benchmark mcp agent meld metabolize observatory okf
BIN  := bin

.PHONY: all build install clean test gate jscheck docs-coverage defn-latest

all: build

## build: compile every command into ./bin
build:
	@mkdir -p $(BIN)
	@for c in $(CMDS); do \
		go build -o $(BIN)/winze-$$c ./cmd/$$c || exit 1; \
	done
	@# winze-agent was called winze-mem. Installed hooks reference the binary by
	@# absolute path and fail open, so a missed reference disables memory capture
	@# silently instead of erroring. This shim makes that impossible; delete it
	@# once nothing on the host names the old binary.
	@ln -sfn winze-agent $(BIN)/winze-mem
	@echo "built: $(BIN)/winze-{$(shell echo $(CMDS) | tr ' ' ',')}"

## install: compile every command into GOBIN (defaults to ~/go/bin)
install:
	@for c in $(CMDS); do \
		go build -o "$${GOBIN:-$$HOME/go/bin}/winze-$$c" ./cmd/$$c || exit 1; \
	done
	@echo "installed winze-* to $${GOBIN:-$$HOME/go/bin}"

## gate: the per-claim consistency gate (what cmd/add runs, over the corpus)
gate:
	go build ./corpus && go vet ./corpus

## test: full test suite
test:
	go test ./...

## jscheck: typecheck (tsc checkJs) + lint (biome) the observatory frontend.
## Run `npm --prefix cmd/observatory ci` once to install the tooling.
jscheck:
	npm --prefix cmd/observatory run check

## docs-coverage: fail if any cmd/ binary is named in no doc
docs-coverage:
	go run ./cmd/query --docs-coverage .

## defn-latest: move module + CLI to the newest defn release, then verify.
## winze is defn's dogfooding consumer, so sitting on an old release hides
## regressions rather than avoiding them — the policy is track latest.
## Both halves move together on purpose: the CLI writes the store the library
## reads, so a CLI/library version skew is a schema disagreement waiting to
## surface as a confusing ingest error. CI derives its CLI pin from go.mod for
## the same reason (.github/workflows/ci.yml), which is why this is one command
## and not a CI-side `@latest`.
defn-latest:
	go get github.com/justinstimatze/defn@latest
	go mod tidy
	go install "github.com/justinstimatze/defn/cmd/defn@$$(go list -m -f '{{.Version}}' github.com/justinstimatze/defn)"
	go build ./... && go vet ./... && go test ./...
	@echo "defn at $$(go list -m -f '{{.Version}}' github.com/justinstimatze/defn) — module + CLI in $${GOBIN:-$$HOME/go/bin}"

## clean: remove built binaries
clean:
	rm -rf $(BIN)
