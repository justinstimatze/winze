// Package defndb provides typed access to defn's SQL-backed code database.
// It shells out to the defn CLI binary, parses JSON responses, and returns
// typed results. When defn is unavailable (no binary, no .defn/ directory,
// or timeout), all methods return ErrNotAvailable so callers can fall back
// to direct AST walking.
package defndb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ErrNotAvailable indicates defn is not usable (missing binary, missing
// database, timeout, or prior failure in this process).
var ErrNotAvailable = errors.New("defndb: defn not available")

// queryTimeout caps each defn CLI invocation to avoid lock contention
// with a concurrent defn MCP server.
const queryTimeout = 5 * time.Second

// Client wraps defn CLI shell-out with typed queries. A Client caches
// its availability state: after the first failure, all subsequent calls
// return ErrNotAvailable without shelling out.
type Client struct {
	dir     string
	defnBin string

	mu      sync.Mutex
	failed  bool
}

// New creates a Client for the given project directory. Returns
// ErrNotAvailable if the defn binary is not on PATH or the .defn/
// directory does not exist.
func New(dir string) (*Client, error) {
	bin, err := exec.LookPath("defn")
	if err != nil {
		return nil, ErrNotAvailable
	}
	dbDir := filepath.Join(dir, ".defn")
	if fi, err := os.Stat(dbDir); err != nil || !fi.IsDir() {
		return nil, ErrNotAvailable
	}
	return &Client{dir: dir, defnBin: bin}, nil
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
	return c.literalFields(`lf.field_name IN ('Subject','Object','Prov')`)
}

// EntityFields returns Name/Brief/ID/Origin/Quote fields from entity literals.
func (c *Client) EntityFields() ([]LiteralField, error) {
	return c.literalFields(`lf.field_name IN ('Name','Brief','ID','Origin','Quote')`)
}

// LiteralFieldsForType returns all literal fields for the given type name pattern.
func (c *Client) LiteralFieldsForType(typePattern string) ([]LiteralField, error) {
	return c.literalFields(fmt.Sprintf(`lf.type_name LIKE '%%%s%%'`, typePattern))
}

func (c *Client) literalFields(where string) ([]LiteralField, error) {
	sql := fmt.Sprintf(`SELECT d.name AS def_name, lf.type_name, lf.field_name, lf.field_value, d.source_file, lf.line FROM literal_fields lf JOIN definitions d ON lf.def_id=d.id WHERE %s AND d.kind='var'`, where)
	rows, err := c.query(sql)
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
	sql := fmt.Sprintf(`SELECT d.name AS def_name, c.source_file, c.line, c.pragma_key, c.pragma_value FROM comments c LEFT JOIN definitions d ON c.def_id=d.id WHERE c.pragma_key LIKE '%s%%'`, prefix)
	rows, err := c.query(sql)
	if err != nil {
		return nil, err
	}
	out := make([]Pragma, 0, len(rows))
	for _, row := range rows {
		out = append(out, Pragma{
			DefName:    str(row["def_name"]),
			SourceFile: str(row["source_file"]),
			Line:       intVal(row["line"]),
			Key:        str(row["pragma_key"]),
			Value:      str(row["pragma_value"]),
		})
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
	sql := fmt.Sprintf(`SELECT d.name, d.kind, d.source_file, d.line FROM definitions d LEFT JOIN bodies b ON b.def_id=d.id WHERE MATCH(d.doc) AGAINST ('%s') OR MATCH(b.body) AGAINST ('%s') LIMIT 50`, pattern, pattern)
	rows, err := c.query(sql)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, SearchResult{
			Name:       str(row["name"]),
			Kind:       str(row["kind"]),
			SourceFile: str(row["source_file"]),
			Line:       intVal(row["line"]),
		})
	}
	return out, nil
}

// VarsByType returns var definitions whose type matches the given pattern.
func (c *Client) VarsByType(typePattern string) ([]SearchResult, error) {
	sql := fmt.Sprintf(`SELECT DISTINCT d.name, d.kind, d.source_file, d.line FROM definitions d JOIN literal_fields lf ON lf.def_id=d.id WHERE d.kind='var' AND lf.type_name LIKE '%%%s%%' GROUP BY d.id`, typePattern)
	rows, err := c.query(sql)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, SearchResult{
			Name:       str(row["name"]),
			Kind:       str(row["kind"]),
			SourceFile: str(row["source_file"]),
			Line:       intVal(row["line"]),
		})
	}
	return out, nil
}

// VarRoleTypes maps var names to their constructor-referenced role type.
// For a var like `Apophenia = Concept{&Entity{...}}`, this returns
// {"Apophenia": "Concept"} by looking at constructor refs to known role types.
func (c *Client) VarRoleTypes(roleTypeNames []string) (map[string]string, error) {
	if len(roleTypeNames) == 0 {
		return map[string]string{}, nil
	}
	quoted := make([]string, len(roleTypeNames))
	for i, n := range roleTypeNames {
		quoted[i] = "'" + n + "'"
	}
	sql := fmt.Sprintf(`SELECT d1.name AS var_name, d2.name AS role_type, d1.source_file FROM refs r JOIN definitions d1 ON r.from_def=d1.id JOIN definitions d2 ON r.to_def=d2.id WHERE d1.kind='var' AND r.kind='constructor' AND d2.name IN (%s)`, strings.Join(quoted, ","))
	rows, err := c.query(sql)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(rows))
	for _, row := range rows {
		m[str(row["var_name"])] = str(row["role_type"])
	}
	return m, nil
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
	sql := fmt.Sprintf(`SELECT d1.name AS var_name, d2.name AS role_type, d1.source_file FROM refs r JOIN definitions d1 ON r.from_def=d1.id JOIN definitions d2 ON r.to_def=d2.id WHERE d1.kind='var' AND r.kind='constructor' AND d2.name IN (%s)`, strings.Join(quoted, ","))
	rows, err := c.query(sql)
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

// query shells out to defn and parses the JSON response.
func (c *Client) query(sql string) ([]map[string]any, error) {
	c.mu.Lock()
	if c.failed {
		c.mu.Unlock()
		return nil, ErrNotAvailable
	}
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.defnBin, "query", sql)
	cmd.Dir = c.dir
	cmd.Env = append(os.Environ(), "DEFN_DB="+filepath.Join(c.dir, ".defn"))

	out, err := cmd.Output()
	if err != nil {
		c.mu.Lock()
		c.failed = true
		c.mu.Unlock()
		return nil, ErrNotAvailable
	}

	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		c.mu.Lock()
		c.failed = true
		c.mu.Unlock()
		return nil, ErrNotAvailable
	}
	return rows, nil
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
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}
