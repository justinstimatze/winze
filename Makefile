# winze — build the KB tools.
#
# `go run ./cmd/foo` recompiles on every invocation, which costs ~0.5s before
# the tool does any work. That is fine for batch phases and badly wrong for
# interactive querying, where the tool itself answers in ~25ms — an 18x tax on
# the operation a knowledge base exists to make cheap. Build once, then query
# at native speed.

CMDS := query lint topology metabolism add edit sensor rot-probe predicates-suggest benchmark mcp mem meld metabolize
BIN  := bin

.PHONY: all build install clean test gate

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

## clean: remove built binaries
clean:
	rm -rf $(BIN)
