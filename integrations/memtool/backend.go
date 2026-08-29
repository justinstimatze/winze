package memtool

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// createdVar mirrors cmd/agent's own helper of the same name over
// winze-add's success line -- duplicated, not imported, because cmd/agent
// is package main and cannot be imported by another Go program.
func createdVar(addOut string) string {
	for _, line := range strings.Split(addOut, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "created entity ")
		if !ok {
			continue
		}
		if name, _, ok := strings.Cut(rest, " "); ok && name != "" {
			return name
		}
	}
	return ""
}

// backend is how the executor reaches winze. The real implementation shells
// out to `winze-agent call`, the same non-MCP-host path the Python Hermes
// integration (integrations/hermes/winze/) already uses -- cmd/agent's
// handlers are unexported in package main, so no Go program can import them
// directly. This is an interface so command logic can be tested without a
// real winze binary or store.
type backend interface {
	// remember writes note as a new entity titled title, bypassing dedup:
	// the memory-tool protocol is deterministic path-addressed CRUD (the
	// caller has already decided this is file X), not an LLM re-asserting a
	// fact it might have already said, which is the case winze's dedup gate
	// exists for.
	remember(note, title string) (varName string, err error)
	// update overwrites varName's Brief with note, through the same gate a
	// direct winze_update call runs.
	update(varName, note string) error
	// fullBrief returns the complete, untruncated Brief of the entity titled
	// title, found via winze's own hybrid recall over an exact title match.
	fullBrief(title string) (brief string, found bool, err error)
}

func (b cliBackend) bin() string {
	if b.BinPath != "" {
		return b.BinPath
	}
	return "winze-agent"
}

func (b cliBackend) call(tool string, args map[string]any) (string, error) {
	payload, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	cmd := exec.Command(b.bin(), "call", tool, string(payload))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("winze-agent call %s: %w", tool, err)
	}
	return string(out), nil
}

func (b cliBackend) fullBrief(title string) (string, bool, error) {
	out, err := b.call("winze_recall", map[string]any{"query": title, "limit": 1, "brief_chars": 1 << 20})
	if err != nil {
		return "", false, err
	}
	var res struct {
		Hits []struct {
			Name  string `json:"name"`
			Brief string `json:"brief"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return "", false, fmt.Errorf("parsing winze_recall output: %w", err)
	}
	if len(res.Hits) == 0 || res.Hits[0].Name != title {
		return "", false, nil
	}
	return res.Hits[0].Brief, true, nil
}

func (b cliBackend) remember(note, title string) (string, error) {
	out, err := b.call("winze_remember", map[string]any{"note": note, "title": title, "force": true})
	if err != nil {
		return "", err
	}
	v := createdVar(out)
	if v == "" {
		return "", fmt.Errorf("winze_remember did not report a created entity:\n%s", out)
	}
	return v, nil
}

func (b cliBackend) update(varName, note string) error {
	_, err := b.call("winze_update", map[string]any{"var": varName, "note": note})
	return err
}

// cliBackend is the real backend: winze-agent call, exactly as documented in
// docs/agent.md.
type cliBackend struct {
	// BinPath is the winze-agent binary to invoke -- a full path or a bare
	// name resolved on $PATH. Empty resolves "winze-agent" on $PATH.
	BinPath string
}
