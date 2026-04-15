// Package defndb provides typed access to defn's SQL-backed code database.
// Uses defn's Go API (github.com/justinstimatze/defn/db) for direct database
// access — no CLI binary or server needed. When the .defn/ database doesn't
// exist, all methods return ErrNotAvailable so callers can fall back to
// direct AST walking.
package defndb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	defnapi "github.com/justinstimatze/defn/db"
)

// ErrNotAvailable indicates defn is not usable (missing database or open failure).
var ErrNotAvailable = errors.New("defndb: defn not available")

// Client wraps defn's Go API with winze-specific typed queries.
type Client struct {
	db *defnapi.DB
}

// New creates a Client for the given project directory. Returns
// ErrNotAvailable if the .defn/ directory does not exist or cannot be opened.
func New(dir string) (*Client, error) {
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
	rows, err := c.query(`SELECT d.name, d.source_file FROM definitions d WHERE d.kind='type' AND d.id IN (SELECT r.from_def FROM refs r WHERE r.kind='embed' AND r.to_def=(SELECT id FROM definitions WHERE name='Entity' AND kind='type'))`)
	if err != nil {
		return nil, err
	}
	out := make([]RoleType, 0, len(rows))
	for _, row := range rows {
		out = append(out, RoleType{
			Name:       str(row["name"]),
			SourceFile: str(row["source_file"]),
		})
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

// ClaimFields returns Subject/Object/Prov fields from claim composite literals.
func (c *Client) ClaimFields() ([]LiteralField, error) {
	return c.literalFieldsSQL(`lf.field_name IN ('Subject','Object','Prov')`)
}

// EntityFields returns Name/Brief/ID/Origin/Quote fields from entity literals.
func (c *Client) EntityFields() ([]LiteralField, error) {
	return c.literalFieldsSQL(`lf.field_name IN ('Name','Brief','ID','Origin','Quote')`)
}

// LiteralFieldsForType returns all literal fields for the given type name pattern.
func (c *Client) LiteralFieldsForType(typePattern string) ([]LiteralField, error) {
	return c.literalFieldsSQL(fmt.Sprintf(`lf.type_name LIKE '%%%s%%'`, typePattern))
}

func (c *Client) literalFieldsSQL(where string) ([]LiteralField, error) {
	rows, err := c.query(fmt.Sprintf(`SELECT d.name AS def_name, lf.type_name, lf.field_name, lf.field_value, d.source_file, lf.line FROM literal_fields lf JOIN definitions d ON lf.def_id=d.id WHERE %s AND d.kind='var'`, where))
	if err != nil {
		return nil, err
	}
	out := make([]LiteralField, 0, len(rows))
	for _, row := range rows {
		out = append(out, LiteralField{
			DefName:    str(row["def_name"]),
			TypeName:   str(row["type_name"]),
			FieldName:  str(row["field_name"]),
			FieldValue: str(row["field_value"]),
			SourceFile: str(row["source_file"]),
			Line:       intVal(row["line"]),
		})
	}
	return out, nil
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
	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}
	quoted := make([]string, len(roleNames))
	for i, n := range roleNames {
		quoted[i] = "'" + n + "'"
	}
	rows, err := c.query(fmt.Sprintf(`SELECT d1.name AS var_name, d2.name AS role_type, d1.source_file FROM refs r JOIN definitions d1 ON r.from_def=d1.id JOIN definitions d2 ON r.to_def=d2.id WHERE d1.kind='var' AND r.kind='constructor' AND d2.name IN (%s)`, strings.Join(quoted, ",")))
	if err != nil {
		return nil, err
	}
	out := make([]VarRoleInfo, 0, len(rows))
	for _, row := range rows {
		out = append(out, VarRoleInfo{
			VarName:    str(row["var_name"]),
			RoleType:   str(row["role_type"]),
			SourceFile: str(row["source_file"]),
		})
	}
	return out, nil
}

// query delegates to defn's Go API.
func (c *Client) query(sql string) ([]map[string]any, error) {
	return c.db.Query(sql)
}

func str(v any) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	if s == "<nil>" {
		return ""
	}
	return strings.TrimSpace(s)
}

func intVal(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case uint64:
		return int(n)
	}
	return 0
}
