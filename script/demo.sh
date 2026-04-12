#!/bin/bash
set -euo pipefail

# Terminal demo for winze README.
#
# Records three asciinema demos:
#   1. bias-audit.cast   — cognitive bias self-audit
#   2. type-error.cast   — compiler catches knowledge error
#   3. dream.cast        — consolidation cycle
#
# Usage:
#   ./script/demo.sh              # record all demos
#   ./script/demo.sh bias         # record just the bias audit
#   ./script/demo.sh type-error   # record just the type error demo
#   ./script/demo.sh dream        # record just the dream cycle
#
# Requirements: asciinema, Go 1.24+
#
# To convert to GIF: agg demo/bias-audit.cast demo/bias-audit.gif
# To convert to SVG: svg-term --in demo/bias-audit.cast --out demo/bias-audit.svg

# Resolve winze root — works whether run directly or sourced
if [ -n "${WINZE_DIR:-}" ]; then
    cd "$WINZE_DIR"
elif [ -f "$(dirname "${BASH_SOURCE[0]:-$0}")/../go.mod" ]; then
    cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.."
fi
WINZE_DIR="$(pwd)"
export WINZE_DIR

DEMO_DIR="demo"
mkdir -p "$DEMO_DIR"

# Simulate typing for natural-looking recordings
type_cmd() {
    local cmd="$1"
    printf '$ '
    for (( i=0; i<${#cmd}; i++ )); do
        printf '%s' "${cmd:$i:1}"
        sleep 0.04
    done
    echo
    sleep 0.3
    eval "$cmd"
    sleep 1.5
}

record_bias() {
    echo "Recording bias audit demo..."
    asciinema rec "$DEMO_DIR/bias-audit.cast" \
        --cols 100 --rows 35 \
        --title "winze: cognitive bias self-audit" \
        --env WINZE_DIR \
        --command "bash -c 'source $WINZE_DIR/script/demo.sh __run_bias'" \
        --overwrite
    echo "Saved: $DEMO_DIR/bias-audit.cast"
}

__run_bias() {
    cd "$WINZE_DIR"
    sleep 0.5
    type_cmd "go run ./cmd/metabolism --bias ."
    sleep 2
}

record_type_error() {
    echo "Recording type error demo..."
    asciinema rec "$DEMO_DIR/type-error.cast" \
        --cols 100 --rows 25 \
        --title "winze: the compiler catches knowledge errors" \
        --env WINZE_DIR \
        --command "bash -c 'source $WINZE_DIR/script/demo.sh __run_type_error'" \
        --overwrite
    echo "Saved: $DEMO_DIR/type-error.cast"
}

__run_type_error() {
    cd "$WINZE_DIR"
    sleep 0.5
    printf '$ # A valid claim: Carl Sagan (Person) proposes the Baloney Detection Kit\n'
    sleep 1
    printf '$ grep -A4 "SaganProposesBaloney" demon_haunted.go\n'
    sleep 0.3
    grep -A4 "SaganProposesBaloney" demon_haunted.go
    sleep 2

    printf '\n$ # Now break it: use a Concept where a Person is required\n'
    sleep 1

    # Create a broken file in the winze root
    cat > demo_break_tmp.go << 'GOEOF'
package winze

var DemoBroken = Proposes{
	Subject: Consciousness, // Concept, not Person — type error
	Object:  BaloneyDetectionKitThesis,
	Prov:    sagan1995Source,
}
GOEOF

    printf '$ go build ./...\n'
    sleep 0.3
    go build ./... 2>&1 || true
    sleep 2

    rm -f demo_break_tmp.go
    printf '\n$ # The compiler caught the ontological error. Fix it and move on.\n'
    sleep 1.5
}

record_dream() {
    echo "Recording dream cycle demo..."
    asciinema rec "$DEMO_DIR/dream.cast" \
        --cols 100 --rows 45 \
        --title "winze: consolidation cycle (dream)" \
        --env WINZE_DIR \
        --command "bash -c 'source $WINZE_DIR/script/demo.sh __run_dream'" \
        --overwrite
    echo "Saved: $DEMO_DIR/dream.cast"
}

__run_dream() {
    cd "$WINZE_DIR"
    sleep 0.5
    type_cmd "go run ./cmd/metabolism --dream --bias ."
    sleep 2
}

# Internal dispatch for asciinema --command
if [ "${1:-}" = "__run_bias" ]; then __run_bias; exit 0; fi
if [ "${1:-}" = "__run_type_error" ]; then __run_type_error; exit 0; fi
if [ "${1:-}" = "__run_dream" ]; then __run_dream; exit 0; fi

# Main dispatch
case "${1:-all}" in
    bias)       record_bias ;;
    type-error) record_type_error ;;
    dream)      record_dream ;;
    all)
        record_bias
        echo
        record_type_error
        echo
        record_dream
        echo
        echo "All demos recorded in $DEMO_DIR/"
        echo "Embed in README with: [![asciicast](https://asciinema.org/a/ID.svg)](https://asciinema.org/a/ID)"
        echo "Or convert to GIF:    agg $DEMO_DIR/bias-audit.cast $DEMO_DIR/bias-audit.gif"
        ;;
    *)
        echo "Usage: $0 [bias|type-error|dream|all]"
        exit 1
        ;;
esac
