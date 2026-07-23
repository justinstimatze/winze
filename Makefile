# winze — build the KB tools.
#
# `go run ./cmd/foo` recompiles on every invocation, which costs ~0.5s before
# the tool does any work. That is fine for batch phases and badly wrong for
# interactive querying, where the tool itself answers in ~25ms — an 18x tax on
# the operation a knowledge base exists to make cheap. Build once, then query
# at native speed.

CMDS := query lint topology metabolism add edit sensor rot-probe predicates-suggest benchmark mcp mem meld metabolize observatory
BIN  := bin

.PHONY: all build install clean test gate jscheck docs-coverage

all: build

## build: compile every command into ./bin
build:
	@mkdir -p $(BIN)
	@for c in $(CMDS); do \
		go build -o $(BIN)/winze-$$c ./cmd/$$c || exit 1; \
	done
	@echo "built: $(BIN)/winze-{$(shell echo $(CMDS) | tr ' ' ',')}"

## install: compile every command into GOBIN (defaults to ~/go/bin)
install:
	@for c in $(CMDS); do \
		go build -o "$${GOBIN:-$$HOME/go/bin}/winze-$$c" ./cmd/$$c || exit 1; \
	done
	@echo "installed winze-* to $${GOBIN:-$$HOME/go/bin}"

## gate: the per-claim consistency gate (what cmd/add runs)
gate:
	go build . && go vet .

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

## clean: remove built binaries
clean:
	rm -rf $(BIN)
