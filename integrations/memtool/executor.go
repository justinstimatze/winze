package memtool

import (
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// NewExecutor opens (or creates) the path index at indexPath and returns an
// Executor backed by the winze-agent binary at binPath ("" resolves
// "winze-agent" on $PATH).
func NewExecutor(indexPath, binPath string) (*Executor, error) {
	ix, err := loadIndex(indexPath)
	if err != nil {
		return nil, err
	}
	return &Executor{ix: ix, be: cliBackend{BinPath: binPath}}, nil
}

// sliceLines returns lines [start, end] (1-indexed, inclusive) of text, per
// the memory tool's view_range contract.
func sliceLines(text string, start, end int64) string {
	lines := strings.Split(text, "\n")
	if start < 1 {
		start = 1
	}
	if end < 0 || end > int64(len(lines)) {
		end = int64(len(lines))
	}
	if start > end || int(start) > len(lines) {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

// Execute runs one memory-tool command and returns the text for a
// tool_result content block, plus whether it represents an error (maps to
// BetaToolResultBlockParam.IsError).
func (e *Executor) Execute(cmd anthropic.BetaMemoryTool20250818CommandUnion) (result string, isError bool) {
	switch cmd.Command {
	case "view":
		return e.view(cmd.Path, cmd.ViewRange)
	case "create":
		return e.create(cmd.Path, cmd.FileText)
	case "str_replace":
		return e.strReplace(cmd.Path, cmd.OldStr, cmd.NewStr)
	case "insert":
		return e.insert(cmd.Path, cmd.InsertLine, cmd.InsertText)
	case "delete":
		return e.delete(cmd.Path)
	case "rename":
		return e.rename(cmd.OldPath, cmd.NewPath)
	default:
		return fmt.Sprintf("unknown memory command %q", cmd.Command), true
	}
}

func (e *Executor) create(path, fileText string) (string, bool) {
	if entry, ok := e.ix.get(path); ok {
		if err := e.be.update(entry.Var, fileText); err != nil {
			return fmt.Sprintf("error updating %s: %v", path, err), true
		}
		return fmt.Sprintf("File %s updated", path), false
	}
	varName, err := e.be.remember(fileText, path)
	if err != nil {
		return fmt.Sprintf("error creating %s: %v", path, err), true
	}
	e.ix.set(path, indexEntry{Var: varName, CreationTitle: path})
	if err := e.ix.save(); err != nil {
		return fmt.Sprintf("created in winze but failed to persist the index: %v", err), true
	}
	return fmt.Sprintf("File %s created", path), false
}

func (e *Executor) delete(path string) (string, bool) {
	_, direct := e.ix.get(path)
	if !direct && len(e.ix.listPrefix(path)) == 0 {
		return fmt.Sprintf("%s not found", path), true
	}
	e.ix.delete(path)
	if err := e.ix.save(); err != nil {
		return fmt.Sprintf("failed to persist the index: %v", err), true
	}
	return fmt.Sprintf("%s deleted", path), false
}

func (e *Executor) insert(path string, line int64, text string) (string, bool) {
	entry, ok := e.ix.get(path)
	if !ok {
		return fmt.Sprintf("%s not found", path), true
	}
	brief, found, err := e.be.fullBrief(entry.CreationTitle)
	if err != nil {
		return fmt.Sprintf("error reading %s: %v", path, err), true
	}
	if !found {
		return fmt.Sprintf("%s is indexed but winze no longer has it", path), true
	}
	lines := strings.Split(brief, "\n")
	if line < 0 || int(line) > len(lines) {
		return fmt.Sprintf("insert_line %d out of range for %s (%d lines)", line, path, len(lines)), true
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:line]...)
	out = append(out, text)
	out = append(out, lines[line:]...)
	if err := e.be.update(entry.Var, strings.Join(out, "\n")); err != nil {
		return fmt.Sprintf("error updating %s: %v", path, err), true
	}
	return fmt.Sprintf("File %s edited", path), false
}

func (e *Executor) rename(oldPath, newPath string) (string, bool) {
	n := e.ix.rename(oldPath, newPath)
	if n == 0 {
		return fmt.Sprintf("%s not found", oldPath), true
	}
	if err := e.ix.save(); err != nil {
		return fmt.Sprintf("failed to persist the index: %v", err), true
	}
	return fmt.Sprintf("Renamed %s to %s", oldPath, newPath), false
}

func (e *Executor) strReplace(path, oldStr, newStr string) (string, bool) {
	entry, ok := e.ix.get(path)
	if !ok {
		return fmt.Sprintf("%s not found", path), true
	}
	brief, found, err := e.be.fullBrief(entry.CreationTitle)
	if err != nil {
		return fmt.Sprintf("error reading %s: %v", path, err), true
	}
	if !found {
		return fmt.Sprintf("%s is indexed but winze no longer has it", path), true
	}
	n := strings.Count(brief, oldStr)
	if n == 0 {
		return fmt.Sprintf("old_str not found in %s", path), true
	}
	if n > 1 {
		return fmt.Sprintf("old_str matches %d times in %s, must match exactly once", n, path), true
	}
	updated := strings.Replace(brief, oldStr, newStr, 1)
	if err := e.be.update(entry.Var, updated); err != nil {
		return fmt.Sprintf("error updating %s: %v", path, err), true
	}
	return fmt.Sprintf("File %s edited", path), false
}

func (e *Executor) view(path string, viewRange []int64) (string, bool) {
	if entry, ok := e.ix.get(path); ok {
		brief, found, err := e.be.fullBrief(entry.CreationTitle)
		if err != nil {
			return fmt.Sprintf("error reading %s: %v", path, err), true
		}
		if !found {
			return fmt.Sprintf("%s is indexed (%s) but winze no longer has it", path, entry.Var), true
		}
		if len(viewRange) == 2 {
			return sliceLines(brief, viewRange[0], viewRange[1]), false
		}
		return brief, false
	}
	entries := e.ix.listPrefix(path)
	if len(entries) == 0 {
		return fmt.Sprintf("%s not found", path), true
	}
	return "Directory: " + path + "\n" + strings.Join(entries, "\n"), false
}

// Executor executes memory-tool commands against a winze store.
//
// It does not implement anthropic.BetaTool: that interface always declares a
// custom JSON-schema tool, and the native memory tool has no schema of its
// own -- Anthropic's API supplies it server-side for the "memory_20250818"
// type. Wiring this into a live conversation means declaring
// anthropic.BetaMemoryTool20250818Param in the request's tools, then routing
// each returned "memory" tool_use block's input through Execute by hand;
// there is no generic BetaToolRunner path for a schema-less native tool
// (confirmed against the pinned SDK: BetaToolRunner's BetaTool interface
// requires an InputSchema(), which this tool type does not have).
type Executor struct {
	ix *index
	be backend
}
