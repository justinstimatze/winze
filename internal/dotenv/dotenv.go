// Package dotenv loads KEY=VALUE pairs from a .env file at a given
// directory and sets them in the process environment, skipping any key
// that already has a value. Used by cmd/lint, cmd/rot-probe,
// cmd/predicates-suggest, and cmd/add for ANTHROPIC_API_KEY pickup
// without requiring an explicit export.
//
// Behaviour intentionally minimal: no quoting, no expansion, no comments
// beyond lines that start with #. Anything the lint / probe / suggest
// pipelines have actually needed.
package dotenv

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Load reads dir/.env and sets each KEY=VALUE pair in the environment,
// preserving any key that already has a non-empty value. A missing or
// unreadable .env file is silently skipped — .env is an optional
// convenience, not a hard dependency.
func Load(dir string) {
	path := filepath.Join(dir, ".env")
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
