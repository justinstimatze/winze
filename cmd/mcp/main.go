// Command mcp serves the winze knowledge base as an MCP server (stdio)
// and/or A2A server (HTTP JSON-RPC 2.0).
//
// Usage:
//
//	go run ./cmd/mcp .                       # MCP over stdio
//	go run ./cmd/mcp --http :8090 .          # HTTP with A2A endpoint
//	go run ./cmd/mcp --http :8090 --secret $TOKEN .  # A2A with auth
//
// MCP registration (.mcp.json):
//
//	{ "mcpServers": { "winze": { "command": "/path/to/winze-mcp", "args": ["/path/to/kb"] } } }
//
// A2A usage:
//
//	curl -X POST http://localhost:8090/a2a \
//	  -d '{"jsonrpc":"2.0","id":1,"method":"agent/info","params":{}}'
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/justinstimatze/winze/internal/defndb"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// --- data types (shared with cmd/query) ---

type entityRecord struct {
	VarName  string   `json:"var_name"`
	RoleType string   `json:"role_type"`
	ID       string   `json:"id,omitempty"`
	Name     string   `json:"name,omitempty"`
	Brief    string   `json:"brief,omitempty"`
	Aliases  []string `json:"aliases,omitempty"`
	File     string   `json:"file"`
}

type claimRecord struct {
	VarName   string `json:"var_name"`
	Predicate string `json:"predicate"`
	Subject   string `json:"subject"`
	Object    string `json:"object"`
	ProvRef   string `json:"prov_ref,omitempty"`
	File      string `json:"file"`
}

type provRecord struct {
	VarName string `json:"var_name"`
	Origin  string `json:"origin"`
	Quote   string `json:"quote,omitempty"`
	File    string `json:"file"`
}

type kbIndex struct {
	Entities   []entityRecord
	Claims     []claimRecord
	Provenance []provRecord
	RoleTypes  map[string]bool
}

const version = "0.1.0"

func main() {
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(512 << 20)
	}

	httpAddr := flag.String("http", "", "HTTP address for A2A server (e.g. :8090). If empty, runs MCP over stdio.")
	secret := flag.String("secret", "", "Bearer token for A2A auth. If empty, no auth required (dev mode). Also reads WINZE_A2A_SECRET env var.")
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "winze: %v\n", err)
		os.Exit(1)
	}

	kb, err := buildIndex(absDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "winze: failed to build index: %v\n", err)
		os.Exit(1)
	}

	h := &handler{kb: kb, dir: absDir}

	if *httpAddr != "" {
		// HTTP mode: A2A JSON-RPC endpoint
		apiSecret := *secret
		if apiSecret == "" {
			apiSecret = os.Getenv("WINZE_A2A_SECRET")
		}

		mux := http.NewServeMux()
		mux.HandleFunc("POST /a2a", a2aHandler(h, apiSecret))
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(h.coreStats()) //nolint:errcheck
		})

		mode := "dev (no auth)"
		if apiSecret != "" {
			mode = "auth required"
		}
		fmt.Fprintf(os.Stderr, "winze %s: A2A server on %s [%s] (dir: %s)\n", version, *httpAddr, mode, absDir)
		if err := http.ListenAndServe(*httpAddr, mux); err != nil {
			fmt.Fprintf(os.Stderr, "winze HTTP error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Stdio mode: MCP server
	s := server.NewMCPServer("winze", version,
		server.WithToolCapabilities(true),
	)

	s.AddTool(mcp.NewTool("search",
		mcp.WithDescription("Search winze KB entities by name, brief, or alias. Returns matching entities with their claims."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search term (case-insensitive substring match against entity name, brief, ID, aliases)")),
	), h.handleSearch)

	s.AddTool(mcp.NewTool("theories",
		mcp.WithDescription("Show competing theories for a concept. Returns all TheoryOf claims where the object matches the query, with hypothesis briefs and provenance."),
		mcp.WithString("concept", mcp.Required(), mcp.Description("Concept name to find competing theories for (e.g. 'consciousness', 'apophenia')")),
	), h.handleTheories)

	s.AddTool(mcp.NewTool("claims",
		mcp.WithDescription("Show all claims involving an entity (as subject or object). Useful for understanding an entity's relationships in the KB."),
		mcp.WithString("entity", mcp.Required(), mcp.Description("Entity name to find claims for (e.g. 'Chalmers', 'FreeEnergyPrinciple')")),
	), h.handleClaims)

	s.AddTool(mcp.NewTool("provenance",
		mcp.WithDescription("Show provenance trail for a source or entity. Returns origin URLs/citations, source quotes, and which claims reference each provenance record."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Source or entity name to search provenance for (e.g. 'Sagan', 'arXiv')")),
	), h.handleProvenance)

	s.AddTool(mcp.NewTool("disputes",
		mcp.WithDescription("Show all active disputes in the KB. Returns Disputes and DisputesOrg claims with subject, object, and provenance."),
	), h.handleDisputes)

	s.AddTool(mcp.NewTool("stats",
		mcp.WithDescription("KB summary statistics: entity count, claim count, predicate count, provenance sources, disputes, contested concepts, breakdowns by role type and predicate."),
	), h.handleStats)

	fmt.Fprintf(os.Stderr, "winze %s: MCP server starting (dir: %s)\n", version, absDir)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "winze MCP error: %v\n", err)
		os.Exit(1)
	}
}

// --- handler ---

type handler struct {
	kb  *kbIndex
	dir string
}

// MCP handlers — thin wrappers around core functions.

func (h *handler) handleSearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := req.GetArguments()["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query: required string argument"), nil
	}
	return mcp.NewToolResultText(jsonString(h.coreSearch(query))), nil
}

func (h *handler) handleTheories(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	concept, ok := req.GetArguments()["concept"].(string)
	if !ok || concept == "" {
		return mcp.NewToolResultError("concept: required string argument"), nil
	}
	return mcp.NewToolResultText(jsonString(h.coreTheories(concept))), nil
}

func (h *handler) handleClaims(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	entity, ok := req.GetArguments()["entity"].(string)
	if !ok || entity == "" {
		return mcp.NewToolResultError("entity: required string argument"), nil
	}
	return mcp.NewToolResultText(jsonString(h.coreClaims(entity))), nil
}

func (h *handler) handleProvenance(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := req.GetArguments()["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query: required string argument"), nil
	}
	return mcp.NewToolResultText(jsonString(h.coreProvenance(query))), nil
}

func (h *handler) handleDisputes(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(jsonString(h.coreDisputes())), nil
}

func (h *handler) handleStats(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(jsonString(h.coreStats())), nil
}

// --- index building (same as cmd/query, defn-only) ---

func buildIndex(dir string) (*kbIndex, error) {
	client, err := defndb.New(dir)
	if err != nil {
		return nil, fmt.Errorf("defndb: %w", err)
	}
	defer client.Close()

	roleTypes, err := client.RoleTypeSet()
	if err != nil {
		return nil, err
	}

	kb := &kbIndex{RoleTypes: roleTypes}

	varRoles, err := client.EntityVarsWithRoles()
	if err != nil {
		return nil, err
	}
	varRoleMap := map[string]string{}
	varFileMap := map[string]string{}
	for _, vr := range varRoles {
		varRoleMap[vr.VarName] = vr.RoleType
		varFileMap[vr.VarName] = filepath.Base(vr.SourceFile)
	}

	eFields, err := client.EntityFields()
	if err != nil {
		return nil, err
	}
	entityMap := map[string]*entityRecord{}
	for _, f := range eFields {
		rt, ok := varRoleMap[f.DefName]
		if !ok {
			continue
		}
		rec, ok := entityMap[f.DefName]
		if !ok {
			rec = &entityRecord{VarName: f.DefName, RoleType: rt, File: varFileMap[f.DefName]}
			entityMap[f.DefName] = rec
		}
		val := strings.Trim(f.FieldValue, "\"")
		switch f.FieldName {
		case "Name":
			rec.Name = val
		case "Brief":
			rec.Brief = val
		case "ID":
			rec.ID = val
		}
	}
	for varName, rt := range varRoleMap {
		if _, ok := entityMap[varName]; !ok {
			entityMap[varName] = &entityRecord{VarName: varName, RoleType: rt, File: varFileMap[varName]}
		}
	}
	for _, rec := range entityMap {
		kb.Entities = append(kb.Entities, *rec)
	}

	cFields, err := client.ClaimFields()
	if err != nil {
		return nil, err
	}
	claimMap := map[string]*claimRecord{}
	for _, f := range cFields {
		base := filepath.Base(f.SourceFile)
		typeParts := strings.Split(f.TypeName, ".")
		typeName := typeParts[len(typeParts)-1]
		rec, ok := claimMap[f.DefName]
		if !ok {
			rec = &claimRecord{VarName: f.DefName, Predicate: typeName, File: base}
			claimMap[f.DefName] = rec
		}
		val := strings.Trim(f.FieldValue, "\"")
		switch f.FieldName {
		case "Subject":
			rec.Subject = val
		case "Object":
			rec.Object = val
		case "Prov":
			rec.ProvRef = val
		}
	}
	for _, rec := range claimMap {
		if rec.Subject != "" && rec.Object != "" {
			kb.Claims = append(kb.Claims, *rec)
		}
	}

	provFields, err := client.LiteralFieldsForType("Provenance")
	if err != nil {
		return nil, err
	}
	provMap := map[string]*provRecord{}
	for _, f := range provFields {
		base := filepath.Base(f.SourceFile)
		rec, ok := provMap[f.DefName]
		if !ok {
			rec = &provRecord{VarName: f.DefName, File: base}
			provMap[f.DefName] = rec
		}
		val := strings.Trim(f.FieldValue, "\"")
		switch f.FieldName {
		case "Origin":
			rec.Origin = val
		case "Quote":
			rec.Quote = val
		}
	}
	for _, rec := range provMap {
		kb.Provenance = append(kb.Provenance, *rec)
	}

	return kb, nil
}

// --- helpers ---

func matchEntity(e entityRecord, q string) bool {
	if strings.Contains(strings.ToLower(e.VarName), q) ||
		strings.Contains(strings.ToLower(e.Name), q) ||
		strings.Contains(strings.ToLower(e.Brief), q) ||
		strings.Contains(strings.ToLower(e.ID), q) {
		return true
	}
	for _, a := range e.Aliases {
		if strings.Contains(strings.ToLower(a), q) {
			return true
		}
	}
	return false
}

func claimsInvolving(kb *kbIndex, varName string) []claimRecord {
	var out []claimRecord
	for _, c := range kb.Claims {
		if c.Subject == varName || c.Object == varName {
			out = append(out, c)
		}
	}
	return out
}

func jsonString(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func sortedCounts(m map[string]int) []map[string]any {
	type kv struct {
		Key   string
		Count int
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})
	var out []map[string]any
	for _, s := range sorted {
		out = append(out, map[string]any{"predicate": s.Key, "count": s.Count})
	}
	return out
}
