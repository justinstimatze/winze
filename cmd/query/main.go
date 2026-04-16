// Command query is the read side of the winze knowledge base.
//
// It parses corpus .go files with go/ast, builds an in-memory index of
// entities, claims, and provenance, and answers queries against it.
//
// Usage:
//
//	go run ./cmd/query "consciousness"              # search entities
//	go run ./cmd/query --theories "apophenia"        # competing theories
//	go run ./cmd/query --claims "Chalmers"           # claims involving entity
//	go run ./cmd/query --provenance "Sagan"          # provenance trail
//	go run ./cmd/query --disputes                    # all disputes
//	go run ./cmd/query --stats                       # KB summary stats
//	go run ./cmd/query --json "consciousness"        # JSON output
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/justinstimatze/winze/internal/astutil"
	"github.com/justinstimatze/winze/internal/defndb"
)

// --- data types ---

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

// --- main ---

func main() {
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(512 << 20) // 512 MiB
	}

	theories := flag.String("theories", "", "show competing theories for a concept")
	claims := flag.String("claims", "", "show all claims involving an entity")
	provenance := flag.String("provenance", "", "show provenance trail for source or entity")
	disputes := flag.Bool("disputes", false, "show all disputes in the KB")
	stats := flag.Bool("stats", false, "show KB summary statistics")
	ask := flag.Bool("ask", false, "natural language query via LLM (needs ANTHROPIC_API_KEY)")
	jsonOut := flag.Bool("json", false, "JSON output")
	flag.Parse()

	dir := "."
	// Find dir argument (non-flag arg that looks like a path)
	args := flag.Args()
	query := ""
	for _, a := range args {
		if a == "." || strings.HasPrefix(a, "/") || strings.HasPrefix(a, "./") {
			dir = a
		} else {
			query = a
		}
	}

	kb, err := buildIndex(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query: %v\n", err)
		os.Exit(1)
	}

	switch {
	case *stats:
		runStats(kb, *jsonOut)
	case *disputes:
		runDisputes(kb, *jsonOut)
	case *theories != "":
		runTheories(kb, *theories, *jsonOut)
	case *claims != "":
		runClaims(kb, *claims, *jsonOut)
	case *provenance != "":
		runProvenance(kb, *provenance, *jsonOut)
	case *ask && query != "":
		runAsk(kb, dir, query)
	case *ask:
		runAskInteractive(kb, dir)
	case query != "":
		runSearch(kb, query, *jsonOut)
	default:
		fmt.Fprintf(os.Stderr, "usage: query [--theories|--claims|--provenance|--disputes|--stats|--ask] [QUERY] [DIR]\n")
		os.Exit(1)
	}
}

// --- query modes ---

func runSearch(kb *kbIndex, query string, jsonOut bool) {
	q := strings.ToLower(query)
	var matches []entityRecord
	for _, e := range kb.Entities {
		if matchEntity(e, q) {
			matches = append(matches, e)
		}
	}

	if jsonOut {
		printJSON(matches)
		return
	}

	if len(matches) == 0 {
		fmt.Printf("No entities matching %q\n", query)
		return
	}

	fmt.Printf("Found %d entities matching %q:\n\n", len(matches), query)
	for _, e := range matches {
		fmt.Printf("  %s (%s)  %s\n", e.VarName, e.RoleType, e.File)
		if e.Name != "" {
			fmt.Printf("    Name: %s\n", e.Name)
		}
		if e.Brief != "" {
			fmt.Printf("    Brief: %s\n", truncate(e.Brief, 200))
		}
		if len(e.Aliases) > 0 {
			fmt.Printf("    Aliases: %s\n", strings.Join(e.Aliases, ", "))
		}

		// Show claims involving this entity
		related := claimsInvolving(kb, e.VarName)
		if len(related) > 0 {
			fmt.Printf("    Claims (%d):\n", len(related))
			for _, c := range related {
				dir := "→"
				other := c.Object
				if c.Object == e.VarName {
					dir = "←"
					other = c.Subject
				}
				fmt.Printf("      %s %s %s  (%s)\n", c.Predicate, dir, other, c.File)
			}
		}
		fmt.Println()
	}
}

func runTheories(kb *kbIndex, target string, jsonOut bool) {
	q := strings.ToLower(target)

	// Find the target concept entity
	var targetEntity *entityRecord
	for i, e := range kb.Entities {
		if matchEntity(e, q) {
			targetEntity = &kb.Entities[i]
			break
		}
	}

	// Find TheoryOf claims where Object matches
	var theories []claimRecord
	for _, c := range kb.Claims {
		if c.Predicate != "TheoryOf" {
			continue
		}
		if targetEntity != nil && c.Object == targetEntity.VarName {
			theories = append(theories, c)
		} else if strings.Contains(strings.ToLower(c.Object), q) {
			theories = append(theories, c)
		}
	}

	if jsonOut {
		printJSON(theories)
		return
	}

	label := target
	if targetEntity != nil {
		label = targetEntity.VarName
		if targetEntity.Name != "" {
			label = targetEntity.Name
		}
	}

	if len(theories) == 0 {
		fmt.Printf("No competing theories found for %q\n", target)
		return
	}

	fmt.Printf("Competing theories of %s (%d):\n\n", label, len(theories))
	for i, t := range theories {
		fmt.Printf("  %d. %s\n", i+1, t.Subject)
		// Find the hypothesis entity for its Brief
		for _, e := range kb.Entities {
			if e.VarName == t.Subject && e.Brief != "" {
				fmt.Printf("     %s\n", truncate(e.Brief, 200))
				break
			}
		}
		fmt.Printf("     source: %s  (%s)\n\n", t.ProvRef, t.File)
	}
}

func runClaims(kb *kbIndex, target string, jsonOut bool) {
	q := strings.ToLower(target)

	// Find matching entity
	var targetName string
	for _, e := range kb.Entities {
		if matchEntity(e, q) {
			targetName = e.VarName
			break
		}
	}
	if targetName == "" {
		// Fall back to substring match on var names in claims
		targetName = target
	}

	related := claimsInvolving(kb, targetName)

	if jsonOut {
		printJSON(related)
		return
	}

	if len(related) == 0 {
		fmt.Printf("No claims found involving %q\n", target)
		return
	}

	fmt.Printf("Claims involving %s (%d):\n\n", targetName, len(related))
	for _, c := range related {
		role := "subject"
		if c.Object == targetName {
			role = "object"
		}
		fmt.Printf("  %s  %s → %s  (as %s)\n", c.Predicate, c.Subject, c.Object, role)
		fmt.Printf("    prov: %s  (%s)\n\n", c.ProvRef, c.File)
	}
}

func runProvenance(kb *kbIndex, target string, jsonOut bool) {
	q := strings.ToLower(target)
	var matches []provRecord
	for _, p := range kb.Provenance {
		if strings.Contains(strings.ToLower(p.Origin), q) ||
			strings.Contains(strings.ToLower(p.VarName), q) {
			matches = append(matches, p)
		}
	}

	if jsonOut {
		printJSON(matches)
		return
	}

	if len(matches) == 0 {
		fmt.Printf("No provenance matching %q\n", target)
		return
	}

	fmt.Printf("Provenance matching %q (%d):\n\n", target, len(matches))
	for _, p := range matches {
		fmt.Printf("  %s  (%s)\n", p.VarName, p.File)
		fmt.Printf("    Origin: %s\n", p.Origin)
		if p.Quote != "" {
			fmt.Printf("    Quote: %s\n", truncate(p.Quote, 200))
		}
		// Find claims using this provenance
		var refs []string
		for _, c := range kb.Claims {
			if c.ProvRef == p.VarName {
				refs = append(refs, c.VarName)
			}
		}
		if len(refs) > 0 {
			fmt.Printf("    Used by: %s\n", strings.Join(refs, ", "))
		}
		fmt.Println()
	}
}

func runDisputes(kb *kbIndex, jsonOut bool) {
	var disputes []claimRecord
	for _, c := range kb.Claims {
		if c.Predicate == "Disputes" || c.Predicate == "DisputesOrg" {
			disputes = append(disputes, c)
		}
	}

	if jsonOut {
		printJSON(disputes)
		return
	}

	if len(disputes) == 0 {
		fmt.Println("No disputes in the KB.")
		return
	}

	fmt.Printf("Disputes (%d):\n\n", len(disputes))
	for _, d := range disputes {
		fmt.Printf("  %s disputes %s\n", d.Subject, d.Object)
		fmt.Printf("    prov: %s  (%s)\n\n", d.ProvRef, d.File)
	}
}

func runStats(kb *kbIndex, jsonOut bool) {
	// Count by role type
	roleCounts := map[string]int{}
	for _, e := range kb.Entities {
		roleCounts[e.RoleType]++
	}

	// Count by predicate
	predCounts := map[string]int{}
	for _, c := range kb.Claims {
		predCounts[c.Predicate]++
	}

	// Count files
	files := map[string]bool{}
	for _, e := range kb.Entities {
		files[e.File] = true
	}

	// Count disputed
	disputes := 0
	for _, c := range kb.Claims {
		if c.Predicate == "Disputes" || c.Predicate == "DisputesOrg" {
			disputes++
		}
	}

	// Count TheoryOf targets (contested concepts)
	theoryTargets := map[string]int{}
	for _, c := range kb.Claims {
		if c.Predicate == "TheoryOf" {
			theoryTargets[c.Object]++
		}
	}
	contested := 0
	for _, count := range theoryTargets {
		if count >= 2 {
			contested++
		}
	}

	if jsonOut {
		printJSON(map[string]any{
			"entities":    len(kb.Entities),
			"claims":      len(kb.Claims),
			"provenance":  len(kb.Provenance),
			"files":       len(files),
			"disputes":    disputes,
			"contested":   contested,
			"role_counts": roleCounts,
			"pred_counts": predCounts,
		})
		return
	}

	fmt.Printf("KB Summary:\n\n")
	fmt.Printf("  %d entities across %d files\n", len(kb.Entities), len(files))
	fmt.Printf("  %d claims (%d predicates)\n", len(kb.Claims), len(predCounts))
	fmt.Printf("  %d provenance sources\n", len(kb.Provenance))
	fmt.Printf("  %d disputes\n", disputes)
	fmt.Printf("  %d contested concepts (2+ theories)\n\n", contested)

	fmt.Println("  Entities by role:")
	sortedRoles := sortedKeys(roleCounts)
	for _, r := range sortedRoles {
		fmt.Printf("    %-20s %d\n", r, roleCounts[r])
	}

	fmt.Println("\n  Claims by predicate:")
	sortedPreds := sortedKeys(predCounts)
	for _, p := range sortedPreds {
		fmt.Printf("    %-25s %d\n", p, predCounts[p])
	}
}

// --- index building ---

func buildIndex(dir string) (*kbIndex, error) {
	// Try defndb first for faster indexed access.
	if client, err := defndb.New(dir); err == nil {
		defer client.Close()
		if kb, err := buildIndexDefn(client, dir); err == nil {
			return kb, nil
		}
	}
	return buildIndexAST(dir)
}

func buildIndexDefn(client *defndb.Client, dir string) (*kbIndex, error) {
	roleTypes, err := client.RoleTypeSet()
	if err != nil {
		return nil, err
	}

	kb := &kbIndex{RoleTypes: roleTypes}

	// Get var-to-role-type mapping via constructor refs
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

	// Entity fields (Name, Brief, ID)
	eFields, err := client.EntityFields()
	if err != nil {
		return nil, err
	}
	entityMap := map[string]*entityRecord{}
	for _, f := range eFields {
		rt, ok := varRoleMap[f.DefName]
		if !ok {
			continue // not an entity var
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
	for _, rec := range entityMap {
		kb.Entities = append(kb.Entities, *rec)
	}

	// Claim fields (Subject, Object, Prov)
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

	// Provenance fields (Origin, Quote)
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

func buildIndexAST(dir string) (*kbIndex, error) {
	roleTypes, err := collectRoleTypesAST(dir)
	if err != nil {
		return nil, err
	}

	kb := &kbIndex{RoleTypes: roleTypes}
	fset := token.NewFileSet()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, nameIdent := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					cl, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					typeIdent, ok := cl.Type.(*ast.Ident)
					if !ok {
						continue
					}
					typeName := typeIdent.Name

					switch {
					case roleTypes[typeName]:
						kb.Entities = append(kb.Entities, extractEntity(nameIdent.Name, typeName, e.Name(), cl))
					case typeName == "Provenance":
						kb.Provenance = append(kb.Provenance, extractProv(nameIdent.Name, e.Name(), cl))
					default:
						if c, ok := extractClaim(nameIdent.Name, typeName, e.Name(), cl); ok {
							kb.Claims = append(kb.Claims, c)
						}
					}
				}
			}
		}
	}
	return kb, nil
}

func collectRoleTypes(dir string) (map[string]bool, error) {
	if client, err := defndb.New(dir); err == nil {
		defer client.Close()
		if roles, err := client.RoleTypeSet(); err == nil {
			return roles, nil
		}
	}
	return collectRoleTypesAST(dir)
}

func collectRoleTypesAST(dir string) (map[string]bool, error) {
	pkgs, _, err := astutil.ParseCorpus(dir)
	if err != nil {
		return nil, err
	}
	return astutil.CollectRoleTypes(pkgs), nil
}

// --- extraction helpers ---

func extractEntity(varName, roleType, file string, cl *ast.CompositeLit) entityRecord {
	rec := entityRecord{VarName: varName, RoleType: roleType, File: file}

	// Entity might be direct fields or nested via &Entity{...}
	fields := cl.Elts
	for _, elt := range cl.Elts {
		if ue, ok := elt.(*ast.UnaryExpr); ok {
			if inner, ok := ue.X.(*ast.CompositeLit); ok {
				fields = inner.Elts
				break
			}
		}
	}

	for _, elt := range fields {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "ID":
			rec.ID = unquote(kv.Value)
		case "Name":
			rec.Name = unquote(kv.Value)
		case "Brief":
			rec.Brief = resolveStringExpr(kv.Value)
		case "Aliases":
			rec.Aliases = extractStringSlice(kv.Value)
		}
	}
	return rec
}

func extractClaim(varName, typeName, file string, cl *ast.CompositeLit) (claimRecord, bool) {
	var subject, object, provRef string
	var hasSubject, hasObject bool
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "Subject":
			hasSubject = true
			subject = exprName(kv.Value)
		case "Object":
			hasObject = true
			object = exprName(kv.Value)
		case "Prov":
			provRef = exprName(kv.Value)
		}
	}
	if !hasSubject || !hasObject {
		return claimRecord{}, false
	}
	return claimRecord{
		VarName:   varName,
		Predicate: typeName,
		Subject:   subject,
		Object:    object,
		ProvRef:   provRef,
		File:      file,
	}, true
}

func extractProv(varName, file string, cl *ast.CompositeLit) provRecord {
	rec := provRecord{VarName: varName, File: file}
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "Origin":
			rec.Origin = unquote(kv.Value)
		case "Quote":
			rec.Quote = unquote(kv.Value)
		}
	}
	return rec
}

// --- search helpers ---

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

// --- AST helpers ---

func exprName(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.UnaryExpr:
		return exprName(v.X)
	case *ast.CompositeLit:
		return "<inline>"
	default:
		return fmt.Sprintf("<%T>", e)
	}
}

func unquote(e ast.Expr) string          { return astutil.Unquote(e) }
func resolveStringExpr(e ast.Expr) string { return astutil.ResolveStringExpr(e) }

func extractStringSlice(e ast.Expr) []string {
	cl, ok := e.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	var out []string
	for _, elt := range cl.Elts {
		if s := unquote(elt); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// --- output helpers ---

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v) //nolint:errcheck
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- ask mode (LLM-powered natural language queries) ---

func runAsk(kb *kbIndex, dir, question string) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		loadDotEnv(dir)
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "query: --ask requires ANTHROPIC_API_KEY (set in env or .env)\n")
		os.Exit(1)
	}

	answer, err := askLLM(kb, apiKey, question)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query: ask: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(answer)
}

func runAskInteractive(kb *kbIndex, dir string) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		loadDotEnv(dir)
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "query: --ask requires ANTHROPIC_API_KEY (set in env or .env)\n")
		os.Exit(1)
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	kbContext := buildKBContext(kb)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("winze query (type 'quit' to exit)")
	fmt.Println()
	for {
		fmt.Print("? ")
		if !scanner.Scan() {
			break
		}
		q := strings.TrimSpace(scanner.Text())
		if q == "" {
			continue
		}
		if q == "quit" || q == "exit" {
			break
		}
		answer, err := askWithClient(client, kbContext, q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			continue
		}
		fmt.Println(answer)
		fmt.Println()
	}
}

func buildKBContext(kb *kbIndex) string {
	var b strings.Builder
	b.WriteString("You are answering questions about the winze knowledge base.\n")
	b.WriteString("The KB tracks the epistemology of minds — how minds build, validate, and fail at modeling reality.\n\n")

	b.WriteString("## Entities\n\n")
	for _, e := range kb.Entities {
		b.WriteString(fmt.Sprintf("- %s (%s)", e.VarName, e.RoleType))
		if e.Name != "" && e.Name != e.VarName {
			b.WriteString(fmt.Sprintf(" — %s", e.Name))
		}
		if e.Brief != "" {
			b.WriteString(fmt.Sprintf(": %s", e.Brief))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n## Claims (Subject → Object via Predicate)\n\n")
	for _, c := range kb.Claims {
		b.WriteString(fmt.Sprintf("- %s: %s → %s (%s, prov: %s)\n",
			c.Predicate, c.Subject, c.Object, c.File, c.ProvRef))
	}

	b.WriteString("\n## Provenance Sources\n\n")
	for _, p := range kb.Provenance {
		b.WriteString(fmt.Sprintf("- %s: %s", p.VarName, p.Origin))
		if p.Quote != "" {
			b.WriteString(fmt.Sprintf(" [quote: %s]", truncate(p.Quote, 150)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n## Disputes\n\n")
	for _, c := range kb.Claims {
		if c.Predicate == "Disputes" || c.Predicate == "DisputesOrg" {
			b.WriteString(fmt.Sprintf("- %s disputes %s (prov: %s)\n", c.Subject, c.Object, c.ProvRef))
		}
	}
	return b.String()
}

// maxKBContextChars is the approximate character limit for KB context sent to LLM.
// Haiku's context is ~200k tokens; at ~4 chars/token, 400k chars is safe.
const maxKBContextChars = 400_000

func askLLM(kb *kbIndex, apiKey, question string) (string, error) {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	kbCtx := buildKBContext(kb)
	return askWithClient(client, kbCtx, question)
}

func askWithClient(client anthropic.Client, kbContext, question string) (string, error) {
	if len(kbContext) > maxKBContextChars {
		return "", fmt.Errorf("KB context too large (%d chars, max %d) — reduce entity count or use defn MCP for large KBs", len(kbContext), maxKBContextChars)
	}

	prompt := kbContext +
		"\nAnswer the question using ONLY the KB data above. " +
		"Cite entity names and file locations. Be specific and concise. " +
		"If the KB doesn't contain enough information to answer, say so. " +
		"Do not invent claims not in the data." +
		"\n\nQuestion: " + question

	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 2048,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", err
	}

	var answer strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			answer.WriteString(block.Text)
		}
	}
	return answer.String(), nil
}

func loadDotEnv(dir string) {
	path := filepath.Join(dir, ".env")
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
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
		key := strings.TrimSpace(parts[0])
		if os.Getenv(key) == "" {
			os.Setenv(key, strings.TrimSpace(parts[1]))
		}
	}
}
