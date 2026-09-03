package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// resolveClients resolves the machine-local {name: path} map codeRefSpanRule
// and codeRefExistenceRule check Client-bearing CodeRefs against. Styled on
// cmd/agent/paths.go's storeRoot() resolution chain: explicit flag first,
// then an env var, then a gitignored local file in the working directory.
//
// Resolution order, most explicit first:
//  1. --clients=name=path,name=path (comma-separated pairs)
//  2. --clients-file=<path> (JSON: {"name": "path", ...})
//  3. $WINZE_CLIENTS_FILE naming a JSON file in the same shape
//  4. .winze-clients.json in the working directory, if present
//  5. none configured: an empty map — every Client-based check skips cleanly.
func resolveClients(flagClients, flagClientsFile string) (map[string]string, error) {
	out := map[string]string{}
	if flagClients != "" {
		for _, pair := range strings.Split(flagClients, ",") {
			name, path, ok := strings.Cut(pair, "=")
			if !ok || name == "" || path == "" {
				return nil, fmt.Errorf("--clients: malformed pair %q, want name=path", pair)
			}
			out[name] = path
		}
	}
	file := flagClientsFile
	if file == "" {
		file = os.Getenv("WINZE_CLIENTS_FILE")
	}
	if file == "" {
		if _, err := os.Stat(".winze-clients.json"); err == nil {
			file = ".winze-clients.json"
		}
	}
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("clients file %s: %w", file, err)
		}
		var fromFile map[string]string
		if err := json.Unmarshal(data, &fromFile); err != nil {
			return nil, fmt.Errorf("clients file %s: %w", file, err)
		}
		for k, v := range fromFile {
			if _, exists := out[k]; !exists { // --clients flag wins over file
				out[k] = v
			}
		}
	}
	return out, nil
}
