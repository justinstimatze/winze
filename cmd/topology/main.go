// Command topology runs structural vulnerability analysis on the winze
// knowledge graph. It walks go/ast to extract entities and claims, builds
// an entity-claim adjacency graph, and identifies structural weaknesses
// that predict which claims are most likely to need revision when tested
// against external signal.
//
// This is the first piece of the epistemic metabolism: the system looking
// at its own structure and identifying where it is thin.
//
// Vulnerability detectors (inspired by slimemold but native to winze's
// typed graph):
//
//  1. single-source       — hypothesis with only one Proposes claim
//  2. uncontested          — hypothesis with Proposes but no Disputes
//  3. thin-provenance      — claims whose shared Provenance has no Quote
//  4. bridge-entity        — entity whose removal disconnects subgraphs
//  5. concentration-risk   — entity referenced by many claims (load-bearing)
//
// Output: JSON to stdout (--json) or human-readable summary.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	jsonOut := flag.Bool("json", false, "output JSON instead of human-readable summary")
	exportKB := flag.Bool("export-kb", false, "export claims as slimemold-compatible KBClaim JSON")
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	if *exportKB {
		if err := runExportKB(dir); err != nil {
			fmt.Fprintf(os.Stderr, "topology: export-kb: %v\n", err)
			os.Exit(1)
		}
		return
	}

	report, err := analyze(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "topology: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "topology: encode: %v\n", err)
			os.Exit(1)
		}
		return
	}

	printReport(report)
}

// --- types ---

type Report struct {
	Entities        int             `json:"entities"`
	Claims          int             `json:"claims"`
	Edges           int             `json:"edges"`
	Clusters        int             `json:"clusters"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	SensorTargets   []SensorTarget  `json:"sensor_targets,omitempty"`
}

// SensorTarget is a topology-derived sensor query suggestion. The topology
// command identifies structurally vulnerable hypotheses and generates
// search terms that would find corroboration or dispute.
type SensorTarget struct {
	Hypothesis  string `json:"hypothesis"`
	Query       string `json:"query"`
	Prediction  string `json:"prediction"`
	VulnType    string `json:"vuln_type"`
	VulnCount   int    `json:"vuln_count"` // number of vulnerability types affecting this hypothesis
}

type Vulnerability struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"` // critical, warning, info
	Entity      string   `json:"entity"`
	Description string   `json:"description"`
	ClaimNames  []string `json:"claim_names,omitempty"`
}

type entityInfo struct {
	name     string
	roleType string
	file     string
}

type claimInfo struct {
	name          string
	predicateType string
	subject       string
	object        string
	file          string
}

type provenanceInfo struct {
	file     string
	hasQuote bool
	origin   string // Provenance.Origin field (e.g. "Wikipedia (zim 2025-12) / Tunguska_event")
}

// --- analysis ---

func analyze(dir string) (*Report, error) {
	entities, err := collectEntities(dir)
	if err != nil {
		return nil, fmt.Errorf("collect entities: %w", err)
	}

	claims, err := collectClaims(dir)
	if err != nil {
		return nil, fmt.Errorf("collect claims: %w", err)
	}

	provenance, err := collectProvenance(dir)
	if err != nil {
		return nil, fmt.Errorf("collect provenance: %w", err)
	}

	// Build adjacency: entity name → set of connected entity names (via shared claims)
	adj := buildAdjacency(claims)
	clusters := countClusters(entities, adj)

	var vulns []Vulnerability
	vulns = append(vulns, findSingleSource(claims)...)
	vulns = append(vulns, findUncontested(claims)...)
	vulns = append(vulns, findThinProvenance(claims, provenance)...)
	vulns = append(vulns, findBridgeEntities(entities, claims, adj)...)
	vulns = append(vulns, findConcentrationRisk(entities, claims)...)

	// Sort: critical first, then warning, then info
	severityOrder := map[string]int{"critical": 0, "warning": 1, "info": 2}
	sort.SliceStable(vulns, func(i, j int) bool {
		return severityOrder[vulns[i].Severity] < severityOrder[vulns[j].Severity]
	})

	// Collect entity metadata for sensor target generation
	metas := collectEntityMetas(dir)

	targets := generateSensorTargets(vulns, metas)

	return &Report{
		Entities:        len(entities),
		Claims:          len(claims),
		Edges:           countEdges(adj),
		Clusters:        clusters,
		Vulnerabilities: vulns,
		SensorTargets:   targets,
	}, nil
}

// buildAdjacency creates an undirected entity graph: two entities are
// connected if they co-occur in any claim (one as Subject, one as Object).
func buildAdjacency(claims []claimInfo) map[string]map[string]bool {
	adj := map[string]map[string]bool{}
	ensure := func(name string) {
		if adj[name] == nil {
			adj[name] = map[string]bool{}
		}
	}
	for _, c := range claims {
		ensure(c.subject)
		ensure(c.object)
		adj[c.subject][c.object] = true
		adj[c.object][c.subject] = true
	}
	return adj
}

func countEdges(adj map[string]map[string]bool) int {
	count := 0
	for _, neighbors := range adj {
		count += len(neighbors)
	}
	return count / 2 // undirected
}

func countClusters(entities []entityInfo, adj map[string]map[string]bool) int {
	visited := map[string]bool{}
	clusters := 0

	bfs := func(start string) {
		queue := []string{start}
		visited[start] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for neighbor := range adj[cur] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
	}

	for _, e := range entities {
		if !visited[e.name] {
			if adj[e.name] != nil {
				bfs(e.name)
				clusters++
			}
		}
	}
	return clusters
}

// --- vulnerability detectors ---

// findSingleSource: hypotheses with only one Proposes/ProposesOrg claim.
// A hypothesis supported by a single advocate is structurally fragile —
// if that source is revised, the hypothesis has no independent support.
func findSingleSource(claims []claimInfo) []Vulnerability {
	// Count proposers per hypothesis (object of Proposes/ProposesOrg)
	proposers := map[string][]string{} // hypothesis → proposer claim names
	for _, c := range claims {
		if c.predicateType == "Proposes" || c.predicateType == "ProposesOrg" {
			proposers[c.object] = append(proposers[c.object], c.name)
		}
	}

	var vulns []Vulnerability
	for hyp, names := range proposers {
		if len(names) == 1 {
			vulns = append(vulns, Vulnerability{
				Type:        "single_source",
				Severity:    "warning",
				Entity:      hyp,
				Description: fmt.Sprintf("Hypothesis %s has only one proposer — no independent corroboration", hyp),
				ClaimNames:  names,
			})
		}
	}
	sort.Slice(vulns, func(i, j int) bool { return vulns[i].Entity < vulns[j].Entity })
	return vulns
}

// findUncontested: hypotheses with Proposes but no Disputes claims.
// A hypothesis nobody disputes hasn't been stress-tested.
func findUncontested(claims []claimInfo) []Vulnerability {
	proposed := map[string]bool{}
	disputed := map[string]bool{}
	for _, c := range claims {
		switch c.predicateType {
		case "Proposes", "ProposesOrg":
			proposed[c.object] = true
		case "Disputes", "DisputesOrg":
			disputed[c.object] = true
		}
	}

	var vulns []Vulnerability
	for hyp := range proposed {
		if !disputed[hyp] {
			vulns = append(vulns, Vulnerability{
				Type:        "uncontested",
				Severity:    "info",
				Entity:      hyp,
				Description: fmt.Sprintf("Hypothesis %s is proposed but never disputed — untested", hyp),
			})
		}
	}
	sort.Slice(vulns, func(i, j int) bool { return vulns[i].Entity < vulns[j].Entity })
	return vulns
}

// findThinProvenance: claims in files whose shared Provenance var has no Quote.
// Without a direct source quote, the claim's audit trail is weaker.
func findThinProvenance(claims []claimInfo, provenance map[string]provenanceInfo) []Vulnerability {
	// Group claims by file
	byFile := map[string][]claimInfo{}
	for _, c := range claims {
		byFile[c.file] = append(byFile[c.file], c)
	}

	var vulns []Vulnerability
	for file, fileClaims := range byFile {
		prov, ok := provenance[file]
		if ok && prov.hasQuote {
			continue // has direct source evidence
		}
		if len(fileClaims) == 0 {
			continue
		}
		names := make([]string, len(fileClaims))
		for i, c := range fileClaims {
			names[i] = c.name
		}
		vulns = append(vulns, Vulnerability{
			Type:        "thin_provenance",
			Severity:    "warning",
			Entity:      file,
			Description: fmt.Sprintf("File %s has %d claims with no direct source quote in provenance", file, len(fileClaims)),
			ClaimNames:  names,
		})
	}
	sort.Slice(vulns, func(i, j int) bool { return vulns[i].Entity < vulns[j].Entity })
	return vulns
}

// findBridgeEntities: entities whose removal would disconnect the graph
// into more components. These are structurally critical — if the entity
// is revised or removed, the knowledge graph fragments.
func findBridgeEntities(entities []entityInfo, claims []claimInfo, adj map[string]map[string]bool) []Vulnerability {
	if len(adj) < 4 {
		return nil
	}

	// Count current clusters
	baseline := countClustersFromAdj(adj)

	var vulns []Vulnerability
	entitySet := map[string]string{} // name → roleType
	for _, e := range entities {
		entitySet[e.name] = e.roleType
	}

	for name := range adj {
		if len(adj[name]) < 2 {
			continue // can't be a bridge with <2 neighbors
		}

		// Temporarily remove this entity and count clusters
		removed := adj[name]
		delete(adj, name)
		for neighbor := range removed {
			delete(adj[neighbor], name)
		}

		after := countClustersFromAdj(adj)

		// Restore
		adj[name] = removed
		for neighbor := range removed {
			if adj[neighbor] == nil {
				adj[neighbor] = map[string]bool{}
			}
			adj[neighbor][name] = true
		}

		if after > baseline {
			severity := "warning"
			if after-baseline >= 2 {
				severity = "critical"
			}
			role := entitySet[name]
			if role == "" {
				role = "non-entity" // e.g. a value struct referenced as object
			}
			vulns = append(vulns, Vulnerability{
				Type:        "bridge_entity",
				Severity:    severity,
				Entity:      name,
				Description: fmt.Sprintf("Bridge entity %s (%s): removal splits graph into %d additional components", name, role, after-baseline),
			})
		}
	}
	sort.Slice(vulns, func(i, j int) bool { return vulns[i].Entity < vulns[j].Entity })
	return vulns
}

func countClustersFromAdj(adj map[string]map[string]bool) int {
	visited := map[string]bool{}
	clusters := 0
	for node := range adj {
		if visited[node] || len(adj[node]) == 0 {
			continue
		}
		queue := []string{node}
		visited[node] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for neighbor := range adj[cur] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		clusters++
	}
	return clusters
}

// findConcentrationRisk: entities referenced by many claims. If an entity
// with high in+out degree has thin provenance or is itself a hypothesis,
// many downstream claims depend on it.
func findConcentrationRisk(entities []entityInfo, claims []claimInfo) []Vulnerability {
	degree := map[string]int{}
	for _, c := range claims {
		degree[c.subject]++
		degree[c.object]++
	}

	// Find outliers: mean + 2*stddev
	if len(degree) < 4 {
		return nil
	}
	var sum, sumSq float64
	for _, d := range degree {
		sum += float64(d)
		sumSq += float64(d * d)
	}
	n := float64(len(degree))
	mean := sum / n
	variance := sumSq/n - mean*mean
	stddev := newtonSqrt(variance)
	threshold := mean + 2*stddev
	if threshold < 5 {
		threshold = 5 // minimum to avoid noise
	}

	entityRoles := map[string]string{}
	for _, e := range entities {
		entityRoles[e.name] = e.roleType
	}

	var vulns []Vulnerability
	for name, d := range degree {
		if float64(d) < threshold {
			continue
		}
		role := entityRoles[name]
		if role == "" {
			role = "value"
		}
		vulns = append(vulns, Vulnerability{
			Type:        "concentration_risk",
			Severity:    "info",
			Entity:      name,
			Description: fmt.Sprintf("High-degree entity %s (%s): %d claim references (threshold %.0f) — load-bearing", name, role, d, threshold),
		})
	}
	sort.Slice(vulns, func(i, j int) bool { return vulns[i].Entity < vulns[j].Entity })
	return vulns
}

func newtonSqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	guess := x / 2
	for i := 0; i < 20; i++ {
		guess = (guess + x/guess) / 2
	}
	return guess
}

// --- sensor target generation ---

// generateSensorTargets picks the most vulnerable hypotheses and generates
// search queries that would find corroboration or dispute on arXiv.
// Hypotheses that are BOTH single-source AND uncontested are prioritized.
func generateSensorTargets(vulns []Vulnerability, metas map[string]entityMeta) []SensorTarget {
	// Count vulnerability types per hypothesis
	vulnTypes := map[string]map[string]bool{}
	for _, v := range vulns {
		if v.Type != "single_source" && v.Type != "uncontested" {
			continue
		}
		if vulnTypes[v.Entity] == nil {
			vulnTypes[v.Entity] = map[string]bool{}
		}
		vulnTypes[v.Entity][v.Type] = true
	}

	// Rank by number of vulnerability types (2 = both single-source + uncontested)
	type candidate struct {
		name      string
		vulnCount int
	}
	var candidates []candidate
	for name, types := range vulnTypes {
		candidates = append(candidates, candidate{name, len(types)})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].vulnCount != candidates[j].vulnCount {
			return candidates[i].vulnCount > candidates[j].vulnCount
		}
		return candidates[i].name < candidates[j].name
	})

	// Take top 5
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}

	var targets []SensorTarget
	for _, c := range candidates {
		meta := metas[c.name]
		query := buildSensorQuery(c.name, meta)
		prediction := "single-source"
		if c.vulnCount >= 2 {
			prediction = "single-source AND uncontested — highest revision risk"
		}
		targets = append(targets, SensorTarget{
			Hypothesis: c.name,
			Query:      query,
			Prediction: prediction,
			VulnType:   "structural_fragility",
			VulnCount:  c.vulnCount,
		})
	}
	return targets
}

// buildSensorQuery extracts key terms from entity metadata to construct
// an arXiv search query. Priority: Name (natural language) > Brief > CamelCase var name.
func buildSensorQuery(varName string, meta entityMeta) string {
	stopwords := map[string]bool{
		"the": true, "of": true, "and": true, "in": true, "a": true,
		"is": true, "for": true, "to": true, "as": true, "by": true,
		"an": true, "on": true, "at": true, "or": true, "that": true,
		"this": true, "with": true, "from": true, "are": true, "was": true,
		"were": true, "been": true, "have": true, "has": true, "had": true,
		"its": true, "his": true, "her": true, "their": true, "our": true,
		"which": true, "can": true, "not": true, "but": true, "all": true,
		"there": true, "exists": true, "every": true, "each": true,
		"some": true, "many": true, "most": true, "more": true, "only": true,
		"such": true, "also": true, "than": true, "may": true, "would": true,
		"does": true, "into": true, "when": true, "how": true, "what": true,
		"thesis": true, "hypothesis": true, "framing": true, "typology": true,
		"theory": true, "classification": true, "reframing": true,
		"central": true, "about": true, "between": true, "through": true,
	}

	var queryTerms []string
	seen := map[string]bool{}
	addTerm := func(w string) bool {
		lower := strings.ToLower(strings.Trim(w, ".,;:\"'()-/"))
		if len(lower) < 4 || stopwords[lower] || seen[lower] {
			return false
		}
		seen[lower] = true
		queryTerms = append(queryTerms, lower)
		return true
	}

	// Entity.Name is primary — for hypotheses it's a full sentence
	// describing the intellectual claim in natural language.
	// For long names, key concepts cluster at the end (academic sentences
	// start with "There exists..." / "The claim that..." filler), so we
	// reverse the word order before picking terms.
	if meta.name != "" {
		words := strings.Fields(meta.name)
		if len(words) > 10 {
			for i, j := 0, len(words)-1; i < j; i, j = i+1, j-1 {
				words[i], words[j] = words[j], words[i]
			}
		}
		for _, w := range words {
			addTerm(w)
		}
	}

	// Entity.Brief supplements with additional context.
	if len(queryTerms) < 3 && meta.brief != "" {
		for _, w := range strings.Fields(meta.brief) {
			addTerm(w)
		}
	}

	// CamelCase variable name is last resort — internal naming,
	// not academic vocabulary.
	if len(queryTerms) < 3 {
		for _, w := range splitCamelCase(varName) {
			addTerm(w)
		}
	}

	// 3 terms is the sweet spot for arXiv AND-queries: specific enough
	// to be relevant, broad enough to actually find papers.
	if len(queryTerms) > 3 {
		queryTerms = queryTerms[:3]
	}
	return strings.Join(queryTerms, " ")
}

func splitCamelCase(s string) []string {
	var words []string
	var current []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' && len(current) > 0 {
			words = append(words, string(current))
			current = current[:0]
		}
		current = append(current, c)
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

// collectEntityMetas walks .go files and extracts Name+Brief from entity
// composite literals that embed *Entity.
func collectEntityMetas(dir string) map[string]entityMeta {
	fset := token.NewFileSet()
	metas := map[string]entityMeta{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return metas
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
					meta := extractEntityMeta(cl)
					if meta.name != "" || meta.brief != "" {
						metas[nameIdent.Name] = meta
					}
				}
			}
		}
	}
	return metas
}

type entityMeta struct {
	name  string // Entity.Name field (human-readable, often a full sentence for hypotheses)
	brief string // Entity.Brief field (supplementary description)
}

// extractEntityMeta finds Name and Brief fields from an entity composite
// literal. Handles: RoleType{&Entity{Name: "...", Brief: "..."}} where
// the first positional element is a &Entity{...} unary expression.
func extractEntityMeta(cl *ast.CompositeLit) entityMeta {
	var meta entityMeta
	extractFields := func(elts []ast.Expr) {
		for _, elt := range elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			switch key.Name {
			case "Name":
				meta.name = basicLitString(kv.Value)
			case "Brief":
				meta.brief = basicLitString(kv.Value)
			}
		}
	}

	for _, elt := range cl.Elts {
		// Direct fields
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.Ident); ok {
				switch key.Name {
				case "Name":
					meta.name = basicLitString(kv.Value)
				case "Brief":
					meta.brief = basicLitString(kv.Value)
				}
			}
			continue
		}
		// &Entity{Name: "...", Brief: "..."}
		ue, ok := elt.(*ast.UnaryExpr)
		if !ok {
			continue
		}
		inner, ok := ue.X.(*ast.CompositeLit)
		if !ok {
			continue
		}
		extractFields(inner.Elts)
	}
	return meta
}

func basicLitString(e ast.Expr) string {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	// Simple unquoting: strip surrounding quotes
	s := lit.Value
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return s
}

// --- AST extraction (same patterns as cmd/lint) ---

func collectEntities(dir string) ([]entityInfo, error) {
	roleTypes, err := collectRoleTypes(dir)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	var out []entityInfo

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
					if roleTypes[typeIdent.Name] {
						out = append(out, entityInfo{
							name:     nameIdent.Name,
							roleType: typeIdent.Name,
							file:     e.Name(),
						})
					}
				}
			}
		}
	}
	return out, nil
}

func collectRoleTypes(dir string) (map[string]bool, error) {
	fset := token.NewFileSet()
	roles := map[string]bool{}

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
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					continue
				}
				if embedsEntityPointer(st) {
					roles[ts.Name.Name] = true
				}
			}
		}
	}
	return roles, nil
}

func embedsEntityPointer(st *ast.StructType) bool {
	for _, field := range st.Fields.List {
		if len(field.Names) != 0 {
			continue
		}
		star, ok := field.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		ident, ok := star.X.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "Entity" {
			return true
		}
	}
	return false
}

func collectClaims(dir string) ([]claimInfo, error) {
	fset := token.NewFileSet()
	var out []claimInfo

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
					predType, subj, obj, ok := extractClaim(cl)
					if !ok {
						continue
					}
					out = append(out, claimInfo{
						name:          nameIdent.Name,
						predicateType: predType,
						subject:       subj,
						object:        obj,
						file:          e.Name(),
					})
				}
			}
		}
	}
	return out, nil
}

func extractClaim(cl *ast.CompositeLit) (predType, subject, object string, ok bool) {
	typeIdent, typeOK := cl.Type.(*ast.Ident)
	if !typeOK {
		return "", "", "", false
	}
	var haveSubject, haveObject bool
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
			haveSubject = true
			subject = exprString(kv.Value)
		case "Object":
			haveObject = true
			object = exprString(kv.Value)
		}
	}
	if !haveSubject || !haveObject {
		return "", "", "", false
	}
	return typeIdent.Name, subject, object, true
}

func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		return v.Value
	case *ast.UnaryExpr:
		return v.Op.String() + exprString(v.X)
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	default:
		return fmt.Sprintf("<expr@%T>", e)
	}
}

// collectProvenance walks .go files looking for Provenance composite
// literals assigned to package-level vars. Records whether the Quote
// field is non-empty. This is a heuristic: most corpus files declare a
// single shared provenance var that all claims in the file reference.
func collectProvenance(dir string) (map[string]provenanceInfo, error) {
	fset := token.NewFileSet()
	prov := map[string]provenanceInfo{}

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
				for i := range vs.Names {
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
					if typeIdent.Name != "Provenance" {
						continue
					}
					hasQuote := false
					origin := ""
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
						case "Quote":
							lit, ok := kv.Value.(*ast.BasicLit)
							if ok && lit.Kind == token.STRING && len(lit.Value) > 2 {
								hasQuote = true
							}
						case "Origin":
							origin = basicLitString(kv.Value)
						}
					}
					// Update: if any provenance in this file has a quote, mark it
					existing, exists := prov[e.Name()]
					if !exists || hasQuote {
						existing.file = e.Name()
						existing.hasQuote = existing.hasQuote || hasQuote
						if origin != "" {
							existing.origin = origin
						}
						prov[e.Name()] = existing
					}
				}
			}
		}
	}
	return prov, nil
}

// --- slimemold KB export ---

// KBClaim is the slimemold-compatible claim format for the analyze_kb action.
type KBClaim struct {
	ID            string `json:"id"`
	PredicateType string `json:"predicate_type"`
	Subject       string `json:"subject"`
	Object        string `json:"object"`
	HasQuote      bool   `json:"has_quote"`
	ProvenanceURL string `json:"provenance_url,omitempty"`
}

// KBExportPayload is the full slimemold MCP request for analyze_kb.
type KBExportPayload struct {
	Action   string    `json:"action"`
	Project  string    `json:"project"`
	KBClaims []KBClaim `json:"kb_claims"`
}

// runExportKB walks the winze corpus and emits a slimemold-compatible
// analyze_kb payload. Uses the same AST extraction as topology analysis
// but outputs KBClaim format instead of vulnerability reports.
func runExportKB(dir string) error {
	claims, err := collectClaims(dir)
	if err != nil {
		return fmt.Errorf("collect claims: %w", err)
	}

	provenance, err := collectProvenance(dir)
	if err != nil {
		return fmt.Errorf("collect provenance: %w", err)
	}

	var kbClaims []KBClaim
	for _, c := range claims {
		prov := provenance[c.file]
		kbClaims = append(kbClaims, KBClaim{
			ID:            c.name,
			PredicateType: c.predicateType,
			Subject:       c.subject,
			Object:        c.object,
			HasQuote:      prov.hasQuote,
			ProvenanceURL: prov.origin,
		})
	}

	payload := KBExportPayload{
		Action:   "analyze_kb",
		Project:  "winze",
		KBClaims: kbClaims,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// --- output ---

func printReport(r *Report) {
	fmt.Printf("[topology] %d entities, %d claims, %d edges, %d clusters\n",
		r.Entities, r.Claims, r.Edges, r.Clusters)

	if len(r.Vulnerabilities) == 0 {
		fmt.Println("  no vulnerabilities found")
		return
	}

	// Count by severity
	counts := map[string]int{}
	for _, v := range r.Vulnerabilities {
		counts[v.Severity]++
	}
	fmt.Printf("  vulnerabilities: %d critical, %d warning, %d info\n",
		counts["critical"], counts["warning"], counts["info"])

	// Group by type
	byType := map[string][]Vulnerability{}
	for _, v := range r.Vulnerabilities {
		byType[v.Type] = append(byType[v.Type], v)
	}

	typeOrder := []string{"bridge_entity", "single_source", "thin_provenance", "concentration_risk", "uncontested"}
	for _, t := range typeOrder {
		vulns, ok := byType[t]
		if !ok {
			continue
		}
		fmt.Printf("\n  [%s] (%d)\n", t, len(vulns))
		for _, v := range vulns {
			fmt.Printf("    %s: %s\n", v.Severity, v.Description)
		}
	}
}
