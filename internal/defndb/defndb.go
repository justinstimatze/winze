// Package defndb provides typed access to defn's SQL-backed code database.
// Uses defn's Go API (github.com/justinstimatze/defn/db) for direct database
// access — no CLI binary or server needed. When the .defn/ database doesn't
// exist, all methods return ErrNotAvailable so callers can fall back to
// direct AST walking.
package defndb

import (
	"errors"
	"os"
	"path/filepath"

	defnapi "github.com/justinstimatze/defn/db"
)

// ErrNotAvailable indicates defn is not usable (missing database or open failure).
var ErrNotAvailable = errors.New("defndb: defn not available")

// Client wraps defn's Go API with winze-specific typed queries.
type Client struct {
	db *defnapi.DB
}

// New creates a Client for the given project directory. Prefers connecting
// to a running Dolt server (via DEFN_DSN or default port 3307) to avoid
// embedding a full Dolt engine (~500 MB). Falls back to embedded if the
// server is unavailable.
// Returns ErrNotAvailable if neither path works.
func New(dir string) (*Client, error) {
	// Try server connection first (much lighter on memory).
	if dsn := os.Getenv("DEFN_DSN"); dsn != "" {
		if db, err := defnapi.Open(dsn); err == nil {
			return &Client{db: db}, nil
		}
	}
	// Try default Gas Town Dolt server.
	if db, err := defnapi.Open("root@tcp(127.0.0.1:3307)/defn"); err == nil {
		return &Client{db: db}, nil
	}
	// Fall back to embedded.
	dbDir := filepath.Join(dir, ".defn")
	if fi, err := os.Stat(dbDir); err != nil || !fi.IsDir() {
		return nil, ErrNotAvailable
	}
	db, err := defnapi.Open(dbDir)
	if err != nil {
		return nil, ErrNotAvailable
	}
	return &Client{db: db}, nil
}

// Close releases database resources.
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// RoleType is a type that embeds *Entity.
type RoleType struct {
	Name       string
	SourceFile string
}

// RoleTypes returns all types embedding *Entity via the embed ref kind.
func (c *Client) RoleTypes() ([]RoleType, error) {
	// Use Refs API to find embed relationships, then resolve type names.
	refs, err := c.db.Refs(defnapi.RefFilter{ToName: "Entity", Kind: "embed"})
	if err != nil {
		return nil, err
	}
	out := make([]RoleType, 0, len(refs))
	for _, r := range refs {
		def, err := c.db.DefinitionByID(r.FromDef)
		if err != nil || def.Kind != "type" {
			continue
		}
		out = append(out, RoleType{Name: def.Name, SourceFile: def.SourceFile})
	}
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

// LiteralField is a field from a composite literal initializer.
type LiteralField struct {
	DefName    string
	TypeName   string
	FieldName  string
	FieldValue string
	SourceFile string
	Line       int
}

func convertLiteralFields(fields []defnapi.LiteralField, defs map[int64]*defnapi.Definition) []LiteralField {
	out := make([]LiteralField, len(fields))
	for i, f := range fields {
		defName := f.DefName
		// If DefName wasn't populated by the API, try the defs map.
		if defName == "" {
			if d, ok := defs[f.DefID]; ok {
				defName = d.Name
			}
		}
		sf := ""
		if d, ok := defs[f.DefID]; ok {
			sf = d.SourceFile
		}
		out[i] = LiteralField{
			DefName:    defName,
			TypeName:   f.TypeName,
			FieldName:  f.FieldName,
			FieldValue: f.FieldValue,
			SourceFile: sf,
			Line:       f.Line,
		}
	}
	return out
}

// ClaimFields returns Subject/Object/Prov fields from claim composite literals.
func (c *Client) ClaimFields() ([]LiteralField, error) {
	return c.literalFields(defnapi.LiteralFieldFilter{
		FieldNames: []string{"Subject", "Object", "Prov"},
	})
}

// EntityFields returns Name/Brief/ID/Origin/Quote fields from entity literals.
func (c *Client) EntityFields() ([]LiteralField, error) {
	return c.literalFields(defnapi.LiteralFieldFilter{
		FieldNames: []string{"Name", "Brief", "ID", "Origin", "Quote"},
	})
}

// LiteralFieldsForType returns all literal fields for the given type name pattern.
func (c *Client) LiteralFieldsForType(typePattern string) ([]LiteralField, error) {
	return c.literalFields(defnapi.LiteralFieldFilter{
		TypeName: "%" + typePattern + "%",
	})
}

func (c *Client) literalFields(filter defnapi.LiteralFieldFilter) ([]LiteralField, error) {
	fields, err := c.db.LiteralFields(filter)
	if err != nil {
		return nil, err
	}
	// Build def lookup for SourceFile (DefName comes from the API now).
	defIDs := make(map[int64]bool, len(fields))
	for _, f := range fields {
		defIDs[f.DefID] = true
	}
	defs := make(map[int64]*defnapi.Definition, len(defIDs))
	for id := range defIDs {
		if d, err := c.db.DefinitionByID(id); err == nil {
			defs[id] = d
		}
	}
	// Filter to var definitions only (matches previous behavior).
	var filtered []defnapi.LiteralField
	for _, f := range fields {
		if d, ok := defs[f.DefID]; ok && d.Kind == "var" {
			filtered = append(filtered, f)
		}
	}
	return convertLiteralFields(filtered, defs), nil
}

// Pragma represents a parsed pragma comment (e.g., //winze:contested).
type Pragma struct {
	DefName    string
	SourceFile string
	Line       int
	Key        string
	Value      string
}

// Pragmas returns all pragma comments matching the given key prefix.
func (c *Client) Pragmas(prefix string) ([]Pragma, error) {
	pragmas, err := c.db.Pragmas(prefix + "%")
	if err != nil {
		return nil, err
	}
	out := make([]Pragma, len(pragmas))
	for i, p := range pragmas {
		out[i] = Pragma{
			DefName:    p.DefName,
			SourceFile: p.SourceFile,
			Line:       p.Line,
			Key:        p.Key,
			Value:      p.Value,
		}
	}
	return out, nil
}

// SearchResult is a definition found by fulltext search.
type SearchResult struct {
	Name       string
	Kind       string
	SourceFile string
	Line       int
}

// Search runs a FULLTEXT search across definitions and bodies.
func (c *Client) Search(pattern string) ([]SearchResult, error) {
	defs, err := c.db.Search(pattern)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(defs))
	for _, d := range defs {
		out = append(out, SearchResult{
			Name:       d.Name,
			Kind:       d.Kind,
			SourceFile: d.SourceFile,
			Line:       d.StartLine,
		})
	}
	return out, nil
}

// VarRoleInfo is a var with its role type and source file.
type VarRoleInfo struct {
	VarName    string
	RoleType   string
	SourceFile string
}

// EntityVarsWithRoles returns entity vars with their role types resolved via constructor refs.
func (c *Client) EntityVarsWithRoles() ([]VarRoleInfo, error) {
	roles, err := c.RoleTypes()
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, nil
	}
	roleNames := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleNames[r.Name] = true
	}
	// Find constructor refs to any role type.
	refs, err := c.db.Refs(defnapi.RefFilter{Kind: "constructor"})
	if err != nil {
		return nil, err
	}
	var out []VarRoleInfo
	seen := make(map[int64]bool)
	for _, r := range refs {
		if seen[r.FromDef] {
			continue
		}
		toDef, err := c.db.DefinitionByID(r.ToDef)
		if err != nil || !roleNames[toDef.Name] {
			continue
		}
		fromDef, err := c.db.DefinitionByID(r.FromDef)
		if err != nil || fromDef.Kind != "var" {
			continue
		}
		seen[r.FromDef] = true
		out = append(out, VarRoleInfo{
			VarName:    fromDef.Name,
			RoleType:   toDef.Name,
			SourceFile: fromDef.SourceFile,
		})
	}
	return out, nil
}

