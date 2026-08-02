// Package defndb provides typed queries over the corpus: role types, entity
// vars, composite-literal fields, and pragma comments. It is a thin client over
// defn's SQL database (github.com/justinstimatze/defn/db) — the corpus is
// ingested into a local .defn/ SQLite store and queried in-process, no CLI
// binary and no server.
//
// # Why it went back to defn
//
// This package was a defn client, was cut over to an in-process go/ast index
// when defn linked the full Dolt engine (~190 indirect modules, minutes to
// link), and grew a persistent index cache (cache.go, codec.go) to make that
// parse cheap. Every premise of that detour expired: defn is modernc.org/sqlite
// now, pure Go, ~100 modules, sub-second to link. So the cache is gone and this
// is a defn client again — see docs/defn-migration.md for the measurements and
// the plan this closes.
//
// # Freshness
//
// The .go corpus is the source of truth and the write path never trusts the db
// (the build gate parses real files via astutil). This read side keeps the db
// current: New ingests on first use and re-ingests when any .go file has changed
// since the last ingest (db.StaleFiles), so a query always sees the corpus as it
// is on disk, and an unedited corpus pays only the freshness check, not a
// re-ingest.
package defndb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	defnapi "github.com/justinstimatze/defn/db"
)

// ErrNotAvailable indicates the corpus directory could not be opened as a defn
// store. Callers may fall back to direct AST walking.
var ErrNotAvailable = errors.New("defndb: corpus not available")

// Client answers typed queries about a corpus directory over a defn SQLite
// store. New keeps the store in sync with the .go files; the definition index
// used to resolve a literal's source file and var-kind is loaded once, lazily,
// on the first query and cached for the Client's lifetime.
type Client struct {
	db *defnapi.DB

	defOnce sync.Once
	defByID map[int64]defnapi.Definition
	defErr  error
}

// New opens (creating if absent) the .defn store under dir and brings it up to
// date with the corpus, then returns a Client. Returns ErrNotAvailable when dir
// is not a directory or the store cannot be opened; a sync failure (e.g. a
// corpus that does not compile) is returned as a distinct wrapped error, since
// it is a real problem to surface, not a missing dependency to fall back from.
func New(dir string) (*Client, error) {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return nil, ErrNotAvailable
	}
	db, err := defnapi.Open(filepath.Join(dir, ".defn"))
	if err != nil {
		return nil, ErrNotAvailable
	}
	c := &Client{db: db}
	if err := c.ensureFresh(dir); err != nil {
		db.Close()
		return nil, err
	}
	return c, nil
}

// ensureFresh ingests the corpus when the store has never been ingested, or
// re-ingests when any .go file has changed since the last ingest. An unedited
// corpus does no work beyond the stat walk StaleFiles performs.
func (c *Client) ensureFresh(dir string) error {
	// A full re-ingest is a single-writer operation: two overlapping Syncs
	// insert the same definitions and collide on the (module, name, kind,
	// receiver, test) unique index. Serialize the whole check-and-sync with an
	// OS advisory lock so concurrent winze processes — and parallel `go test`
	// binaries sharing one repo-root .defn — take turns. Whoever waits then
	// sees last_ingest set and StaleFiles empty, and skips the redundant Sync.
	unlock, err := lockSync(filepath.Join(dir, ".defn"))
	if err != nil {
		return err
	}
	defer unlock()

	last, err := c.db.GetMeta("last_ingest")
	if err != nil {
		return fmt.Errorf("defndb: read ingest state: %w", err)
	}
	need := last == "" // never ingested: StaleFiles reports nothing on an empty db
	if !need {
		stale, err := c.db.StaleFiles(dir)
		if err != nil {
			return fmt.Errorf("defndb: stale check: %w", err)
		}
		need = len(stale) > 0
	}
	if need {
		if err := c.db.Sync(dir); err != nil {
			return fmt.Errorf("defndb: sync %s: %w", dir, err)
		}
	}
	return nil
}

// lockSync takes an exclusive advisory lock on defnDir/sync.lock, blocking
// until it is available, and returns a release function. It guards the
// check-and-reingest in ensureFresh so only one process re-ingests at a time;
// defnDir already exists (defnapi.Open created it) by the time this runs.
func lockSync(defnDir string) (func(), error) {
	f, err := os.OpenFile(filepath.Join(defnDir, "sync.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("defndb: open sync lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("defndb: acquire sync lock: %w", err)
	}
	return func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

// Close releases the underlying database.
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// defs returns the DefID→Definition index, loaded once. It resolves every
// literal field's source file and enclosing-definition kind from a single
// Definitions query rather than a lookup per row.
func (c *Client) defs() (map[int64]defnapi.Definition, error) {
	c.defOnce.Do(func() {
		all, err := c.db.Definitions(defnapi.DefinitionFilter{})
		if err != nil {
			c.defErr = err
			return
		}
		m := make(map[int64]defnapi.Definition, len(all))
		for _, d := range all {
			m[d.ID] = d
		}
		c.defByID = m
	})
	return c.defByID, c.defErr
}

// RoleType is a type that embeds *Entity.
type RoleType struct {
	Name       string
	SourceFile string
}

// LiteralField is a field from a composite literal initializer. TypeName is the
// type of the immediately-enclosing literal, so a Provenance literal inlined
// inside a claim reports a Provenance TypeName, not the claim's predicate type.
// DefName is the enclosing top-level var.
type LiteralField struct {
	DefName    string
	TypeName   string
	FieldName  string
	FieldValue string
	SourceFile string
	Line       int
}

// Pragma represents a parsed pragma comment (e.g., //winze:contested).
type Pragma struct {
	DefName    string
	SourceFile string
	Line       int
	Key        string
	Value      string
}

// SearchResult is a definition found by search.
type SearchResult struct {
	Name       string
	Kind       string
	SourceFile string
	Line       int
}

// VarRoleInfo is a var with its role type and source file.
type VarRoleInfo struct {
	VarName    string
	RoleType   string
	SourceFile string
}

// RoleTypes returns all types embedding *Entity, via the embed ref to Entity.
func (c *Client) RoleTypes() ([]RoleType, error) {
	refs, err := c.db.Refs(defnapi.RefFilter{ToName: "Entity", Kind: "embed"})
	if err != nil {
		return nil, err
	}
	defs, err := c.defs()
	if err != nil {
		return nil, err
	}
	out := make([]RoleType, 0, len(refs))
	for _, r := range refs {
		d, ok := defs[r.FromDef]
		if !ok || d.Kind != "type" {
			continue
		}
		out = append(out, RoleType{Name: d.Name, SourceFile: d.SourceFile})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// RoleTypeSet returns role type names as a set for quick lookup.
func (c *Client) RoleTypeSet() (map[string]bool, error) {
	roles, err := c.RoleTypes()
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(roles))
	for _, r := range roles {
		m[r.Name] = true
	}
	return m, nil
}

// EntityVarsWithRoles returns entity vars with their role types resolved via
// constructor refs (a var whose initializer constructs a role type), in var-name
// order so the index the read paths build from it is stable across runs.
func (c *Client) EntityVarsWithRoles() ([]VarRoleInfo, error) {
	roles, err := c.RoleTypeSet()
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, nil
	}
	refs, err := c.db.Refs(defnapi.RefFilter{Kind: "constructor"})
	if err != nil {
		return nil, err
	}
	defs, err := c.defs()
	if err != nil {
		return nil, err
	}
	var out []VarRoleInfo
	seen := make(map[int64]bool)
	for _, r := range refs {
		if seen[r.FromDef] {
			continue
		}
		to, ok := defs[r.ToDef]
		if !ok || !roles[to.Name] {
			continue
		}
		from, ok := defs[r.FromDef]
		if !ok || from.Kind != "var" {
			continue
		}
		seen[r.FromDef] = true
		out = append(out, VarRoleInfo{VarName: from.Name, RoleType: to.Name, SourceFile: from.SourceFile})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].VarName < out[j].VarName })
	return out, nil
}

// literalFields runs a LiteralFields query, keeps only fields whose enclosing
// definition is a top-level var (corpus entities/claims/provenance all are),
// and returns them in a deterministic source order. It sets both opt-out perf
// flags: DefName and source file are reconstructed from the definition index,
// and ordering is imposed here, so defn's join and sort are pure waste for this
// client.
func (c *Client) literalFields(filter defnapi.LiteralFieldFilter) ([]LiteralField, error) {
	filter.SkipOrderBy = true
	filter.SkipDefName = true
	fields, err := c.db.LiteralFields(filter)
	if err != nil {
		return nil, err
	}
	defs, err := c.defs()
	if err != nil {
		return nil, err
	}
	out := make([]LiteralField, 0, len(fields))
	for _, f := range fields {
		d, ok := defs[f.DefID]
		if !ok || d.Kind != "var" {
			continue
		}
		out = append(out, LiteralField{
			DefName:    d.Name,
			TypeName:   f.TypeName,
			FieldName:  f.FieldName,
			FieldValue: f.FieldValue,
			SourceFile: d.SourceFile,
			Line:       f.Line,
		})
	}
	sortLiteralFields(out)
	return out, nil
}

// sortLiteralFields imposes the stable (source file, line, field name) order
// that the read paths treat as "index order". defn's rows come back in
// whatever order the store yields once SkipOrderBy drops the ORDER BY; sorting
// here is what makes query output reproducible.
func sortLiteralFields(fs []LiteralField) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.SourceFile != b.SourceFile {
			return a.SourceFile < b.SourceFile
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.FieldName < b.FieldName
	})
}

// ClaimFields returns Subject/Object/Prov fields from claim composite literals.
func (c *Client) ClaimFields() ([]LiteralField, error) {
	return c.literalFields(defnapi.LiteralFieldFilter{FieldNames: []string{"Subject", "Object", "Prov"}})
}

// EntityFields returns Name/Brief/ID/Origin/Quote fields from entity literals.
func (c *Client) EntityFields() ([]LiteralField, error) {
	return c.literalFields(defnapi.LiteralFieldFilter{FieldNames: []string{"Name", "Brief", "ID", "Origin", "Quote"}})
}

// LiteralFieldsForType returns all literal fields whose enclosing literal type
// contains typePattern.
func (c *Client) LiteralFieldsForType(typePattern string) ([]LiteralField, error) {
	return c.literalFields(defnapi.LiteralFieldFilter{TypeName: "%" + typePattern + "%"})
}

// EachField calls fn for every literal field whose name is in names, in index
// order, stopping early if fn returns false.
func (c *Client) EachField(names []string, fn func(*LiteralField) bool) error {
	fields, err := c.literalFields(defnapi.LiteralFieldFilter{FieldNames: names})
	if err != nil {
		return err
	}
	for i := range fields {
		if !fn(&fields[i]) {
			return nil
		}
	}
	return nil
}

// EachFieldOfType calls fn for every literal field whose enclosing literal type
// contains typePattern, in index order.
func (c *Client) EachFieldOfType(typePattern string, fn func(*LiteralField) bool) error {
	fields, err := c.literalFields(defnapi.LiteralFieldFilter{TypeName: "%" + typePattern + "%"})
	if err != nil {
		return err
	}
	for i := range fields {
		if !fn(&fields[i]) {
			return nil
		}
	}
	return nil
}

// EachFieldWithValuePrefix calls fn for every literal field among the named
// fields whose FieldValue begins with valuePrefix — an anchored value lookup
// pushed to SQL (field_value LIKE 'prefix%', an indexed range scan via defn's
// likePrefixRange). The single-entity read paths use it to find the claims that
// reference one var without scanning the whole claim table.
func (c *Client) EachFieldWithValuePrefix(valuePrefix string, names []string, fn func(*LiteralField) bool) error {
	fields, err := c.literalFields(defnapi.LiteralFieldFilter{FieldNames: names, Value: valuePrefix + "%"})
	if err != nil {
		return err
	}
	for i := range fields {
		if !fn(&fields[i]) {
			return nil
		}
	}
	return nil
}

// EachFieldForDefs calls fn for every literal field among the named fields whose
// enclosing var is in defNames, in index order. The assembly half of a
// single-entity lookup: pass one finds the referencing vars, this fetches their
// full field sets. defn's LiteralFieldFilter has no def-set predicate, so the
// membership test is applied here; at corpus scale the fetched set is small.
func (c *Client) EachFieldForDefs(defNames map[string]bool, names []string, fn func(*LiteralField) bool) error {
	fields, err := c.literalFields(defnapi.LiteralFieldFilter{FieldNames: names})
	if err != nil {
		return err
	}
	for i := range fields {
		if !defNames[fields[i].DefName] {
			continue
		}
		if !fn(&fields[i]) {
			return nil
		}
	}
	return nil
}

// Pragmas returns all pragma comments whose key starts with prefix.
func (c *Client) Pragmas(prefix string) ([]Pragma, error) {
	pragmas, err := c.db.Pragmas(prefix + "%")
	if err != nil {
		return nil, err
	}
	out := make([]Pragma, 0, len(pragmas))
	for _, p := range pragmas {
		// defn links each pragma doc-comment to the definition it sits directly
		// above, so DefName is populated at the DB layer (verified 2026-08-02
		// against defn 2ec0eaaa). No reconstruction needed here.
		out = append(out, Pragma{
			DefName:    p.DefName,
			SourceFile: p.SourceFile,
			Line:       p.Line,
			Key:        p.Key,
			Value:      p.Value,
		})
	}
	return out, nil
}

// Search returns definitions whose name contains pattern (case-insensitive).
func (c *Client) Search(pattern string) ([]SearchResult, error) {
	defs, err := c.db.Definitions(defnapi.DefinitionFilter{Name: "%" + strings.Trim(pattern, "%") + "%"})
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(defs))
	for _, d := range defs {
		out = append(out, SearchResult{Name: d.Name, Kind: d.Kind, SourceFile: d.SourceFile, Line: d.StartLine})
	}
	return out, nil
}
