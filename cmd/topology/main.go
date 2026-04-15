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

	"github.com/justinstimatze/winze/internal/astutil"
	"github.com/justinstimatze/winze/internal/defndb"
)

func main() {
	jsonOut := flag.Bool("json", false, "output JSON instead of human-readable summary")
	exportKB := flag.Bool("export-kb", false, "export claims as slimemold-compatible KBClaim JSON")
	dotOut := flag.Bool("dot", false, "export epistemic support DAG as Graphviz DOT (pipe to dot -Tsvg)")
	why := flag.String("why", "", "trace epistemic support chain for named entity (e.g., --why ChalmersHardProblemThesis)")
	entityCap := flag.Int("entity-cap", 250, "max entities; suppresses breadth targets above this threshold")
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

	if *dotOut {
		if err := runDotExport(dir); err != nil {
			fmt.Fprintf(os.Stderr, "topology: dot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *why != "" {
		if err := runWhy(dir, *why, *jsonOut); err != nil {
			fmt.Fprintf(os.Stderr, "topology: why: %v\n", err)
			os.Exit(1)
		}
		return
	}

	report, err := analyze(dir, *entityCap)
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
	ZimQuery    string `json:"zim_query,omitempty"` // encyclopedia-optimized query (topic name, not author+keywords)
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
	provRef       string // variable name of the Prov field (e.g., "mySource")
}

type supportEdge struct {
	supporter string
	claim     claimInfo
	prov      provenanceInfo
	direction string // "supports", "challenges", "related", "grounds"
}

type provenanceInfo struct {
	file     string
	hasQuote bool
	origin   string // Provenance.Origin field (e.g. "Wikipedia (zim 2025-12) / Tunguska_event")
	quote    string // Provenance.Quote field (exact source text)
	varName  string // the Go variable name
}

// --- analysis ---

func analyze(dir string, entityCap int) (*Report, error) {
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

	targets := generateSensorTargets(vulns, metas, claims, dir, len(entities), entityCap)

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

// --- depth-first target selection ---

// depthTarget represents a contested concept with too few theories.
type depthTarget struct {
	concept  string   // the contested concept (TheoryOf object)
	theories []string // existing hypothesis subjects
}

// findThinContested identifies TheoryOf claims grouped by object with
// exactly 2 subjects. These are contested concepts that need additional
// perspectives — depth targets for the metabolism loop.
func findThinContested(claims []claimInfo) []depthTarget {
	// Group TheoryOf claims by object (the concept being theorized about)
	byObject := map[string][]string{} // concept → list of theory subjects
	for _, c := range claims {
		if c.predicateType == "TheoryOf" {
			// Deduplicate subjects per concept
			subjects := byObject[c.object]
			found := false
			for _, s := range subjects {
				if s == c.subject {
					found = true
					break
				}
			}
			if !found {
				byObject[c.object] = append(subjects, c.subject)
			}
		}
	}

	var targets []depthTarget
	for concept, theories := range byObject {
		if len(theories) == 2 {
			targets = append(targets, depthTarget{concept: concept, theories: theories})
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].concept < targets[j].concept
	})
	return targets
}

// buildDepthQuery constructs a search query for finding additional
// perspectives on a contested concept. Simpler than breadth queries:
// concept name content words + "alternative theory".
func buildDepthQuery(conceptName string, meta entityMeta) string {
	noise := map[string]bool{
		"the": true, "of": true, "and": true, "in": true, "a": true,
		"is": true, "for": true, "to": true, "as": true, "by": true,
		"an": true, "on": true, "or": true, "that": true, "with": true,
	}
	filler := map[string]bool{
		"thesis": true, "hypothesis": true, "framing": true, "typology": true,
		"theory": true, "classification": true, "reframing": true,
		"framework": true, "model": true, "approach": true, "perspective": true,
	}

	// Extract content words from the concept's Name (or var name)
	name := meta.name
	if name == "" {
		name = strings.Join(splitCamelCase(conceptName), " ")
	}

	var parts []string
	for _, w := range strings.Fields(name) {
		lower := strings.ToLower(strings.Trim(w, ".,;:\"'()-/"))
		if len(lower) >= 3 && !noise[lower] && !filler[lower] {
			parts = append(parts, w)
		}
	}

	// Cap concept words at 2 to leave room for discriminator
	if len(parts) > 2 {
		parts = parts[:2]
	}
	parts = append(parts, "alternative", "theory")

	// Cap total at 4
	if len(parts) > 4 {
		parts = parts[:4]
	}
	return strings.Join(parts, " ")
}

// --- sensor target generation ---

// metabolismCycle is a minimal struct for reading the metabolism log.
// Only the fields needed for calibration feedback are included.
type metabolismCycle struct {
	Hypothesis  string `json:"hypothesis"`
	Resolution  string `json:"resolution"`
	PapersFound int    `json:"papers_found"`
}

// loadMetabolismHistory reads .metabolism-log.json and returns the "best"
// resolution per hypothesis. Priority: corroborated > challenged >
// irrelevant > no_signal > "" (unresolved). This lets topology deprioritize
// hypotheses that have already been queried.
func loadMetabolismHistory(dir string) map[string]string {
	data, err := os.ReadFile(filepath.Join(dir, ".metabolism-log.json"))
	if err != nil {
		return nil // no log yet — all hypotheses are fresh
	}

	var log struct {
		Cycles []metabolismCycle `json:"cycles"`
	}
	if err := json.Unmarshal(data, &log); err != nil {
		return nil
	}

	// Resolution priority (higher = more "resolved", less need for re-query)
	priority := map[string]int{
		"":            0,
		"no_signal":   1,
		"irrelevant":  2,
		"challenged":  2,
		"corroborated": 3,
	}

	best := map[string]string{}
	for _, c := range log.Cycles {
		prev := best[c.Hypothesis]
		if priority[c.Resolution] > priority[prev] {
			best[c.Hypothesis] = c.Resolution
		}
	}
	return best
}

// historyBucket assigns a priority bucket for sensor target ranking.
// Lower bucket = higher priority for querying.
//
//	0: never queried (highest priority — fresh hypothesis)
//	1: queried but unresolved
//	2: resolved irrelevant or challenged (signal found but not useful)
//	3: resolved no_signal (query didn't find anything)
//	4: resolved corroborated (needs ingest, not more queries)
func historyBucket(resolution string, queried bool) int {
	if !queried {
		return 0
	}
	switch resolution {
	case "":
		return 1
	case "irrelevant", "challenged":
		return 2
	case "no_signal":
		return 3
	case "corroborated":
		return 4
	default:
		return 1
	}
}

// generateSensorTargets picks the most important targets for sensor queries.
// Depth-first: thin contested concepts (exactly 2 theories) get priority
// over breadth targets (structurally fragile hypotheses). When the entity
// count exceeds entityCap, breadth targets are suppressed entirely — only
// depth targets are emitted.
func generateSensorTargets(vulns []Vulnerability, metas map[string]entityMeta, claims []claimInfo, dir string, entityCount, entityCap int) []SensorTarget {
	// Build proposer lookup: hypothesis var name → proposer var name(s)
	proposerOf := map[string][]string{} // hypothesis → proposer var names
	// Build explains lookup: hypothesis → concept it explains (for ZIM queries)
	explainsOf := map[string]string{} // hypothesis → explained concept var name
	for _, c := range claims {
		if c.predicateType == "Proposes" || c.predicateType == "ProposesOrg" {
			proposerOf[c.object] = append(proposerOf[c.object], c.subject)
		}
		if c.predicateType == "HypothesisExplains" {
			explainsOf[c.subject] = c.object
		}
	}

	// Load metabolism history for calibration feedback
	history := loadMetabolismHistory(dir)

	type candidate struct {
		name      string
		vulnCount int
		bucket    int
		vulnType  string // "thin_contested" or "structural_fragility"
	}
	var candidates []candidate

	// --- Depth targets: thin contested concepts ---
	thinContested := findThinContested(claims)
	for _, dt := range thinContested {
		// Use concept name as the target identifier
		resolution, queried := history[dt.concept]
		bucket := historyBucket(resolution, queried)
		// Depth targets get bucket -1 (always above any breadth target)
		// unless they've been queried — then use normal bucket but still depth-typed
		depthBucket := bucket - 1
		if depthBucket < -1 {
			depthBucket = -1
		}
		candidates = append(candidates, candidate{
			name:      dt.concept,
			vulnCount: len(dt.theories),
			bucket:    depthBucket,
			vulnType:  "thin_contested",
		})
	}

	// --- Breadth targets: structurally fragile hypotheses ---
	// Suppressed when entity count exceeds cap
	if entityCount < entityCap {
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
		for name, types := range vulnTypes {
			resolution, queried := history[name]
			bucket := historyBucket(resolution, queried)
			candidates = append(candidates, candidate{
				name:      name,
				vulnCount: len(types),
				bucket:    bucket,
				vulnType:  "structural_fragility",
			})
		}
	}

	// Rank by: bucket (ascending), then vulnerability count (descending), then name
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].bucket != candidates[j].bucket {
			return candidates[i].bucket < candidates[j].bucket
		}
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
		var query, prediction string

		if c.vulnType == "thin_contested" {
			query = buildDepthQuery(c.name, meta)
			prediction = fmt.Sprintf("contested concept with only %d theories — needs additional perspective", c.vulnCount)
		} else {
			// Enrich with proposer name and explained concept if available
			if proposers, ok := proposerOf[c.name]; ok && len(proposers) > 0 {
				if pm, ok := metas[proposers[0]]; ok && pm.name != "" {
					meta.proposer = pm.name
				}
			}
			if explained, ok := explainsOf[c.name]; ok {
				if em, ok := metas[explained]; ok && em.name != "" {
					meta.explains = em.name
				}
			}
			query = buildSensorQuery(c.name, meta)
			prediction = "single-source"
			if c.vulnCount >= 2 {
				prediction = "single-source AND uncontested — highest revision risk"
			}
		}

		// Build ZIM-optimized query: prefer explained concept name,
		// then entity name — encyclopedias index by topic, not author.
		zimQuery := buildZimQuery(c.name, meta)

		targets = append(targets, SensorTarget{
			Hypothesis: c.name,
			Query:      query,
			ZimQuery:   zimQuery,
			Prediction: prediction,
			VulnType:   c.vulnType,
			VulnCount:  c.vulnCount,
		})
	}
	return targets
}

// buildSensorQuery constructs a search query from entity metadata.
//
// Strategy: author name + concept phrase. This mirrors how researchers
// actually search — "Sagan baloney detection kit" finds the right papers,
// "pseudoscience claims scientific" does not.
//
// Priority order:
//  1. Proposer surname (from Proposes claim graph) — most discriminating single term
//  2. Concept phrase from Entity.Name — extracted as a 2-3 word noun phrase
//  3. Entity.Brief keywords — fallback for thin metadata
//  4. CamelCase var name — last resort
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
		"about": true, "between": true, "through": true,
	}

	// Academic filler — common in hypothesis Name fields but not search-useful.
	academicFiller := map[string]bool{
		"thesis": true, "hypothesis": true, "framing": true, "typology": true,
		"theory": true, "classification": true, "reframing": true,
		"central": true, "claim": true, "argument": true, "proposal": true,
		"framework": true, "model": true, "approach": true, "perspective": true,
	}

	isNoise := func(w string) bool {
		lower := strings.ToLower(strings.Trim(w, ".,;:\"'()-/"))
		return len(lower) < 4 || stopwords[lower]
	}

	// Extract content words from a string, preserving order.
	contentWords := func(s string) []string {
		var out []string
		for _, w := range strings.Fields(s) {
			clean := strings.Trim(w, ".,;:\"'()-/")
			if !isNoise(clean) && !academicFiller[strings.ToLower(clean)] {
				out = append(out, clean)
			}
		}
		return out
	}

	var parts []string

	// 1. Proposer surname — most discriminating term for academic search.
	// Use the last word of the name (surname convention).
	if meta.proposer != "" {
		words := strings.Fields(meta.proposer)
		if len(words) > 0 {
			surname := words[len(words)-1]
			parts = append(parts, surname)
		}
	}

	// Deduplicate helper
	proposerLower := ""
	if len(parts) > 0 {
		proposerLower = strings.ToLower(parts[0])
	}
	dedup := func(words []string) []string {
		var out []string
		for _, w := range words {
			if strings.ToLower(w) != proposerLower {
				out = append(out, w)
			}
		}
		return out
	}

	// 2. Concept phrase. For short Names (≤6 words), use the Name directly.
	// For long Names (full-sentence hypotheses), prefer the CamelCase var name
	// which encodes the concept concisely: "BaloneyDetectionKit" → "Baloney Detection Kit".
	need := 3
	if len(parts) > 0 {
		need = 2
	}

	nameWords := strings.Fields(meta.name)
	if len(nameWords) > 0 && len(nameWords) <= 6 {
		// Short name — use it directly
		cw := dedup(contentWords(meta.name))
		if len(cw) > need {
			cw = cw[:need]
		}
		parts = append(parts, cw...)
	} else {
		// Long name or no name — extract concept from CamelCase var name.
		// Strip common suffixes that are academic filler.
		stripped := varName
		for _, suffix := range []string{"Thesis", "Framing", "Typology", "Reframing"} {
			stripped = strings.TrimSuffix(stripped, suffix)
		}
		cw := dedup(splitCamelCase(stripped))
		// Filter short/noise words
		var good []string
		for _, w := range cw {
			if len(w) >= 3 && !stopwords[strings.ToLower(w)] {
				good = append(good, w)
			}
		}
		if len(good) > need {
			good = good[:need]
		}
		parts = append(parts, good...)
	}

	// 3. Brief supplements if we still don't have enough.
	if len(parts) < 3 && meta.brief != "" {
		seen := map[string]bool{}
		for _, p := range parts {
			seen[strings.ToLower(p)] = true
		}
		for _, w := range contentWords(meta.brief) {
			if !seen[strings.ToLower(w)] {
				parts = append(parts, w)
				seen[strings.ToLower(w)] = true
				if len(parts) >= 4 {
					break
				}
			}
		}
	}

	// 4. CamelCase var name is last resort.
	if len(parts) < 2 {
		for _, w := range splitCamelCase(varName) {
			clean := strings.ToLower(w)
			if len(clean) >= 4 && !stopwords[clean] && !academicFiller[clean] {
				parts = append(parts, w)
			}
		}
	}

	// Cap at 4 terms — enough for specificity, not so many that AND-queries
	// return nothing.
	if len(parts) > 4 {
		parts = parts[:4]
	}
	return strings.Join(parts, " ")
}

// buildZimQuery constructs a search query optimized for encyclopedia lookup.
// Encyclopedias index by topic name, not by author. Strategy:
//  1. If the hypothesis explains a concept, use that concept's name
//  2. Otherwise, use the entity name directly (for short names)
//  3. Fallback: split the CamelCase var name into words
func buildZimQuery(varName string, meta entityMeta) string {
	// If this hypothesis explains a concept, that concept name is the best
	// ZIM query — "Tunguska event" finds the article, "Whipple" does not.
	if meta.explains != "" {
		return meta.explains
	}

	// For short entity names, use the name directly
	nameWords := strings.Fields(meta.name)
	if len(nameWords) > 0 && len(nameWords) <= 6 {
		return meta.name
	}

	// For long names (full-sentence hypotheses), extract topic from var name
	stripped := varName
	for _, suffix := range []string{"Thesis", "Framing", "Typology", "Reframing", "Hypothesis"} {
		stripped = strings.TrimSuffix(stripped, suffix)
	}
	words := splitCamelCase(stripped)
	var clean []string
	noise := map[string]bool{
		"the": true, "of": true, "and": true, "in": true, "a": true,
		"is": true, "for": true, "to": true, "as": true, "by": true,
	}
	for _, w := range words {
		if len(w) >= 3 && !noise[strings.ToLower(w)] {
			clean = append(clean, w)
		}
	}
	if len(clean) > 4 {
		clean = clean[:4]
	}
	return strings.Join(clean, " ")
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
	// Try defndb first.
	if client, err := defndb.New(dir); err == nil {
		if metas, err := collectEntityMetasDefn(client); err == nil {
			return metas
		}
	}
	return collectEntityMetasAST(dir)
}

func collectEntityMetasDefn(client *defndb.Client) (map[string]entityMeta, error) {
	// EntityFields returns Name/Brief fields from Entity literals.
	// We need to know which vars are entities (have constructor ref to a role type).
	varRoles, err := client.EntityVarsWithRoles()
	if err != nil {
		return nil, err
	}
	entityVars := map[string]bool{}
	for _, vr := range varRoles {
		entityVars[vr.VarName] = true
	}

	fields, err := client.EntityFields()
	if err != nil {
		return nil, err
	}
	metas := map[string]entityMeta{}
	for _, f := range fields {
		if !entityVars[f.DefName] {
			continue
		}
		meta := metas[f.DefName]
		val := strings.Trim(f.FieldValue, "\"")
		switch f.FieldName {
		case "Name":
			meta.name = val
		case "Brief":
			meta.brief = val
		}
		if meta.name != "" || meta.brief != "" {
			metas[f.DefName] = meta
		}
	}
	return metas, nil
}

func collectEntityMetasAST(dir string) map[string]entityMeta {
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
	name     string // Entity.Name field (human-readable, often a full sentence for hypotheses)
	brief    string // Entity.Brief field (supplementary description)
	proposer string // Name of the person/org who proposed this (from Proposes claims)
	explains string // Name of concept this hypothesis explains (from HypothesisExplains)
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
	if client, err := defndb.New(dir); err == nil {
		if ents, err := collectEntitiesDefn(client); err == nil {
			return ents, nil
		}
	}
	return collectEntitiesAST(dir)
}

func collectEntitiesDefn(client *defndb.Client) ([]entityInfo, error) {
	varRoles, err := client.EntityVarsWithRoles()
	if err != nil {
		return nil, err
	}
	out := make([]entityInfo, 0, len(varRoles))
	for _, vr := range varRoles {
		out = append(out, entityInfo{
			name:     vr.VarName,
			roleType: vr.RoleType,
			file:     filepath.Base(vr.SourceFile),
		})
	}
	return out, nil
}

func collectEntitiesAST(dir string) ([]entityInfo, error) {
	roleTypes, err := collectRoleTypesAST(dir)
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
	if client, err := defndb.New(dir); err == nil {
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

func collectClaims(dir string) ([]claimInfo, error) {
	if client, err := defndb.New(dir); err == nil {
		if claims, err := collectClaimsDefn(client); err == nil {
			return claims, nil
		}
	}
	return collectClaimsAST(dir)
}

func collectClaimsDefn(client *defndb.Client) ([]claimInfo, error) {
	fields, err := client.ClaimFields()
	if err != nil {
		return nil, err
	}
	type partial struct {
		name, predType, subject, object, file, provRef string
		hasSubject, hasObject                          bool
	}
	m := map[string]*partial{}
	for _, f := range fields {
		typeParts := strings.Split(f.TypeName, ".")
		typeName := typeParts[len(typeParts)-1]
		p, ok := m[f.DefName]
		if !ok {
			p = &partial{name: f.DefName, predType: typeName, file: filepath.Base(f.SourceFile)}
			m[f.DefName] = p
		}
		val := strings.Trim(f.FieldValue, "\"")
		switch f.FieldName {
		case "Subject":
			p.subject = val
			p.hasSubject = true
		case "Object":
			p.object = val
			p.hasObject = true
		case "Prov":
			p.provRef = val
		}
	}
	var out []claimInfo
	for _, p := range m {
		if !p.hasSubject || !p.hasObject {
			continue
		}
		out = append(out, claimInfo{
			name:          p.name,
			predicateType: p.predType,
			subject:       p.subject,
			object:        p.object,
			file:          p.file,
			provRef:       p.provRef,
		})
	}
	return out, nil
}

func collectClaimsAST(dir string) ([]claimInfo, error) {
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
					predType, subj, obj, prov, ok := extractClaimFull(cl)
					if !ok {
						continue
					}
					out = append(out, claimInfo{
						name:          nameIdent.Name,
						predicateType: predType,
						subject:       subj,
						object:        obj,
						file:          e.Name(),
						provRef:       prov,
					})
				}
			}
		}
	}
	return out, nil
}

func extractClaim(cl *ast.CompositeLit) (predType, subject, object string, ok bool) {
	predType, subject, object, _, ok = extractClaimFull(cl)
	return
}

func extractClaimFull(cl *ast.CompositeLit) (predType, subject, object, provRef string, ok bool) {
	typeIdent, typeOK := cl.Type.(*ast.Ident)
	if !typeOK {
		return "", "", "", "", false
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
		case "Prov":
			provRef = exprString(kv.Value)
		}
	}
	if !haveSubject || !haveObject {
		return "", "", "", "", false
	}
	return typeIdent.Name, subject, object, provRef, true
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
// field is non-empty. Returns map keyed by FILE name (for backward compat
// with thin-provenance detector).
func collectProvenance(dir string) (map[string]provenanceInfo, error) {
	byFile, _, err := collectProvenanceFull(dir)
	return byFile, err
}

// collectProvenanceFull returns provenance keyed both by file and by var name.
func collectProvenanceFull(dir string) (byFile map[string]provenanceInfo, byVar map[string]provenanceInfo, err error) {
	fset := token.NewFileSet()
	byFile = map[string]provenanceInfo{}
	byVar = map[string]provenanceInfo{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
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
					quote := ""
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
							q := basicLitString(kv.Value)
							if q != "" {
								hasQuote = true
								quote = q
							}
						case "Origin":
							origin = basicLitString(kv.Value)
						}
					}
					varName := vs.Names[i].Name
					info := provenanceInfo{
						file:     e.Name(),
						hasQuote: hasQuote,
						origin:   origin,
						quote:    quote,
						varName:  varName,
					}
					byVar[varName] = info

					// Update file-level: if any provenance in this file has a quote, mark it
					existing := byFile[e.Name()]
					existing.file = e.Name()
					existing.hasQuote = existing.hasQuote || hasQuote
					if origin != "" {
						existing.origin = origin
					}
					byFile[e.Name()] = existing
				}
			}
		}
	}
	return byFile, byVar, nil
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
// runWhy traces the epistemic support chain for a named entity.
// Walks the directional support DAG backward, collecting provenance
// at each step, annotating with topology vulnerability data.
//
// Usage: go run ./cmd/topology --why ChalmersHardProblemThesis .
func runWhy(dir, entityName string, jsonOut bool) error {
	entities, err := collectEntities(dir)
	if err != nil {
		return fmt.Errorf("collect entities: %w", err)
	}

	// Verify entity exists
	entityMap := map[string]entityInfo{}
	for _, e := range entities {
		entityMap[e.name] = e
	}
	root, exists := entityMap[entityName]
	if !exists {
		// Try fuzzy match
		var candidates []string
		lower := strings.ToLower(entityName)
		for _, e := range entities {
			if strings.Contains(strings.ToLower(e.name), lower) {
				candidates = append(candidates, e.name)
			}
		}
		if len(candidates) == 0 {
			return fmt.Errorf("entity %q not found", entityName)
		}
		if len(candidates) == 1 {
			entityName = candidates[0]
			root = entityMap[entityName]
		} else {
			fmt.Fprintf(os.Stderr, "entity %q not found. Did you mean:\n", entityName)
			for _, c := range candidates {
				fmt.Fprintf(os.Stderr, "  %s\n", c)
			}
			return fmt.Errorf("ambiguous entity name")
		}
	}

	claims, err := collectClaims(dir)
	if err != nil {
		return fmt.Errorf("collect claims: %w", err)
	}

	_, provByVar, err := collectProvenanceFull(dir)
	if err != nil {
		return fmt.Errorf("collect provenance: %w", err)
	}

	// Get vulnerability data
	report, err := analyze(dir, 250)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}
	vulnsByEntity := map[string][]Vulnerability{}
	for _, v := range report.Vulnerabilities {
		vulnsByEntity[v.Entity] = append(vulnsByEntity[v.Entity], v)
	}

	// Predicate direction classification (same as --dot and dreamaudit.go)
	subjectGrounds := map[string]bool{
		"Proposes": true, "ProposesOrg": true,
		"Authored": true, "AuthoredOrg": true,
		"TheoryOf": true, "HypothesisExplains": true,
		"CommentaryOn": true,
	}
	objectGrounds := map[string]bool{
		"DerivedFrom": true, "InfluencedBy": true,
		"BelongsTo": true,
	}
	contraEdge := map[string]bool{
		"Disputes": true, "DisputesOrg": true,
	}

	// Build support edges: supportedBy[entity] = list of (supporter, claim)
	edges := map[string][]supportEdge{}

	for _, c := range claims {
		if c.subject == entityName || c.object == entityName {
			prov := provByVar[c.provRef]
			if subjectGrounds[c.predicateType] {
				if c.object == entityName {
					edges[entityName] = append(edges[entityName], supportEdge{
						supporter: c.subject, claim: c, prov: prov, direction: "supports",
					})
				}
			} else if objectGrounds[c.predicateType] {
				if c.subject == entityName {
					edges[entityName] = append(edges[entityName], supportEdge{
						supporter: c.object, claim: c, prov: prov, direction: "supports",
					})
				}
			} else if contraEdge[c.predicateType] {
				if c.object == entityName {
					edges[entityName] = append(edges[entityName], supportEdge{
						supporter: c.subject, claim: c, prov: prov, direction: "challenges",
					})
				} else {
					edges[entityName] = append(edges[entityName], supportEdge{
						supporter: c.object, claim: c, prov: prov, direction: "challenges",
					})
				}
			} else {
				other := c.object
				if c.object == entityName {
					other = c.subject
				}
				edges[entityName] = append(edges[entityName], supportEdge{
					supporter: other, claim: c, prov: prov, direction: "related",
				})
			}
		}
	}

	// Also collect claims where this entity is Subject grounding others
	var grounds []supportEdge
	for _, c := range claims {
		if c.subject == entityName && subjectGrounds[c.predicateType] {
			prov := provByVar[c.provRef]
			grounds = append(grounds, supportEdge{
				supporter: c.object, claim: c, prov: prov, direction: "grounds",
			})
		}
		if c.object == entityName && objectGrounds[c.predicateType] {
			prov := provByVar[c.provRef]
			grounds = append(grounds, supportEdge{
				supporter: c.subject, claim: c, prov: prov, direction: "grounds",
			})
		}
	}

	if jsonOut {
		type whyJSON struct {
			Entity  string        `json:"entity"`
			Role    string        `json:"role_type"`
			File    string        `json:"file"`
			Brief   string        `json:"brief,omitempty"`
			Vulns   int           `json:"vulnerabilities"`
			Support []interface{} `json:"support_chain"`
		}
		// Collect brief
		metas := collectEntityMetas(dir)
		brief := ""
		if m, ok := metas[entityName]; ok {
			brief = m.brief
		}
		w := whyJSON{
			Entity: entityName,
			Role:   root.roleType,
			File:   root.file,
			Brief:  brief,
			Vulns:  len(vulnsByEntity[entityName]),
		}
		for _, e := range edges[entityName] {
			entry := map[string]interface{}{
				"direction": e.direction,
				"entity":    e.supporter,
				"predicate": e.claim.predicateType,
				"claim":     e.claim.name,
			}
			if e.prov.origin != "" {
				entry["origin"] = e.prov.origin
			}
			if e.prov.quote != "" {
				entry["quote"] = e.prov.quote
			}
			w.Support = append(w.Support, entry)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(w)
	}

	// Text output
	metas := collectEntityMetas(dir)
	brief := ""
	if m, ok := metas[entityName]; ok {
		brief = m.brief
	}

	fmt.Printf("%s (%s)\n", entityName, root.roleType)
	if brief != "" {
		fmt.Printf("  %s\n", brief)
	}
	fmt.Printf("  file: %s\n", root.file)

	vulns := vulnsByEntity[entityName]
	if len(vulns) > 0 {
		fmt.Printf("  vulnerabilities: %d\n", len(vulns))
		for _, v := range vulns {
			fmt.Printf("    [%s] %s\n", v.Severity, v.Type)
		}
	}

	incoming := edges[entityName]
	if len(incoming) > 0 {
		// Group by direction
		var supports, challenges, related []supportEdge
		for _, e := range incoming {
			switch e.direction {
			case "supports":
				supports = append(supports, e)
			case "challenges":
				challenges = append(challenges, e)
			default:
				related = append(related, e)
			}
		}

		if len(supports) > 0 {
			fmt.Println("\n  supported by:")
			for _, e := range supports {
				printWhyEdge(e, entityMap)
			}
		}

		if len(challenges) > 0 {
			fmt.Println("\n  challenged by:")
			for _, e := range challenges {
				printWhyEdge(e, entityMap)
			}
		}

		if len(related) > 0 {
			fmt.Println("\n  related:")
			for _, e := range related {
				printWhyEdge(e, entityMap)
			}
		}
	}

	if len(grounds) > 0 {
		fmt.Println("\n  grounds:")
		for _, e := range grounds {
			ei := entityMap[e.supporter]
			fmt.Printf("    → %s (%s) via %s\n", e.supporter, ei.roleType, e.claim.predicateType)
		}
	}

	// Summary
	fmt.Printf("\n  %d sources, %d supports, %d challenges, %d vulnerabilities\n",
		countUniqueSources(incoming, provByVar),
		len(supports(incoming)), len(challengeEdges(incoming)), len(vulns))

	return nil
}

func printWhyEdge(e supportEdge, entityMap map[string]entityInfo) {
	ei := entityMap[e.supporter]
	fmt.Printf("    ← %s: %s (%s)\n", e.claim.predicateType, e.supporter, ei.roleType)
	if e.prov.origin != "" {
		fmt.Printf("      prov: %s\n", e.prov.origin)
	}
	if e.prov.quote != "" {
		q := e.prov.quote
		if len(q) > 120 {
			q = q[:117] + "..."
		}
		fmt.Printf("      quote: %q\n", q)
	}
}

func countUniqueSources(edges []supportEdge, provByVar map[string]provenanceInfo) int {
	origins := map[string]bool{}
	for _, e := range edges {
		if e.prov.origin != "" {
			origins[e.prov.origin] = true
		}
	}
	return len(origins)
}

func supports(edges []supportEdge) []supportEdge {
	var out []supportEdge
	for _, e := range edges {
		if e.direction == "supports" {
			out = append(out, e)
		}
	}
	return out
}

func challengeEdges(edges []supportEdge) []supportEdge {
	var out []supportEdge
	for _, e := range edges {
		if e.direction == "challenges" {
			out = append(out, e)
		}
	}
	return out
}

// runDotExport outputs the epistemic support DAG as Graphviz DOT.
// Nodes are entities colored by cluster and shaped by role type.
// Edges are directed support edges inferred from predicate semantics.
// Contra edges (Disputes) are dashed. Vulnerability annotations are tooltips.
//
// Usage: go run ./cmd/topology --dot . | dot -Tsvg > dag.svg
func runDotExport(dir string) error {
	entities, err := collectEntities(dir)
	if err != nil {
		return fmt.Errorf("collect entities: %w", err)
	}

	claims, err := collectClaims(dir)
	if err != nil {
		return fmt.Errorf("collect claims: %w", err)
	}

	// Build undirected adjacency for cluster detection
	adj := buildAdjacency(claims)

	// Assign clusters via BFS
	clusterOf := map[string]int{}
	clusterID := 0
	visited := map[string]bool{}
	for _, e := range entities {
		if visited[e.name] || adj[e.name] == nil {
			continue
		}
		queue := []string{e.name}
		visited[e.name] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			clusterOf[cur] = clusterID
			for neighbor := range adj[cur] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		clusterID++
	}

	// Predicate direction classification
	subjectGrounds := map[string]bool{
		"Proposes": true, "ProposesOrg": true,
		"Authored": true, "AuthoredOrg": true,
		"TheoryOf": true, "HypothesisExplains": true,
		"CommentaryOn": true,
	}
	objectGrounds := map[string]bool{
		"DerivedFrom": true, "InfluencedBy": true,
		"BelongsTo": true,
	}
	contraEdge := map[string]bool{
		"Disputes": true, "DisputesOrg": true,
	}

	// Role type → shape
	shapeOf := map[string]string{
		"Person":       "ellipse",
		"Hypothesis":   "diamond",
		"Concept":      "box",
		"Event":        "hexagon",
		"Organization": "house",
		"Place":        "tab",
		"CreativeWork": "note",
	}

	// Cluster colors (cycle through)
	colors := []string{
		"#e8f4f8", "#f8e8e8", "#e8f8e8", "#f8f4e8",
		"#f0e8f8", "#e8f0f8", "#f8e8f0", "#f4f8e8",
		"#e8e8f8", "#f8f0e8", "#e8f8f0", "#f0f8e8",
		"#f8e8f4", "#e8f8f4", "#f4e8f8", "#f8f4f0",
	}

	// Collect vulnerability counts per entity for annotation
	report, err := analyze(dir, 250)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}
	vulnCount := map[string]int{}
	for _, v := range report.Vulnerabilities {
		vulnCount[v.Entity]++
	}

	// Group entities by cluster
	byCluster := map[int][]entityInfo{}
	for _, e := range entities {
		c := clusterOf[e.name]
		byCluster[c] = append(byCluster[c], e)
	}

	// Output DOT
	fmt.Println("digraph winze {")
	fmt.Println("  rankdir=LR;")
	fmt.Println("  graph [fontname=\"Helvetica\" fontsize=10];")
	fmt.Println("  node [fontname=\"Helvetica\" fontsize=9 style=filled];")
	fmt.Println("  edge [fontname=\"Helvetica\" fontsize=7];")
	fmt.Println()

	// Cluster subgraphs
	clusterIDs := make([]int, 0, len(byCluster))
	for id := range byCluster {
		clusterIDs = append(clusterIDs, id)
	}
	sort.Ints(clusterIDs)

	for _, cid := range clusterIDs {
		ents := byCluster[cid]
		if len(ents) < 2 {
			// Singletons outside any cluster subgraph
			for _, e := range ents {
				shape := shapeOf[e.roleType]
				if shape == "" {
					shape = "box"
				}
				vuln := vulnCount[e.name]
				penwidth := "1"
				if vuln >= 3 {
					penwidth = "3"
				} else if vuln >= 1 {
					penwidth = "2"
				}
				fmt.Printf("  %s [shape=%s fillcolor=\"#f0f0f0\" penwidth=%s label=%q];\n",
					dotID(e.name), shape, penwidth, dotLabel(e))
			}
			continue
		}
		color := colors[cid%len(colors)]
		fmt.Printf("  subgraph cluster_%d {\n", cid)
		fmt.Printf("    style=filled; color=%q; label=\"\";\n", color)
		for _, e := range ents {
			shape := shapeOf[e.roleType]
			if shape == "" {
				shape = "box"
			}
			vuln := vulnCount[e.name]
			penwidth := "1"
			if vuln >= 3 {
				penwidth = "3"
			} else if vuln >= 1 {
				penwidth = "2"
			}
			fmt.Printf("    %s [shape=%s fillcolor=\"white\" penwidth=%s label=%q];\n",
				dotID(e.name), shape, penwidth, dotLabel(e))
		}
		fmt.Println("  }")
		fmt.Println()
	}

	// Edges
	entitySet := map[string]bool{}
	for _, e := range entities {
		entitySet[e.name] = true
	}

	for _, c := range claims {
		if !entitySet[c.subject] || !entitySet[c.object] {
			continue
		}

		if subjectGrounds[c.predicateType] {
			// Subject → Object (subject grounds object)
			fmt.Printf("  %s -> %s [label=%q];\n",
				dotID(c.subject), dotID(c.object), c.predicateType)
		} else if objectGrounds[c.predicateType] {
			// Object → Subject (object grounds subject)
			fmt.Printf("  %s -> %s [label=%q];\n",
				dotID(c.object), dotID(c.subject), c.predicateType)
		} else if contraEdge[c.predicateType] {
			// Dashed contra edge
			fmt.Printf("  %s -> %s [label=%q style=dashed color=red];\n",
				dotID(c.subject), dotID(c.object), c.predicateType)
		} else {
			// Undirected (spatial, unary-like with two slots, etc.)
			fmt.Printf("  %s -> %s [label=%q dir=none color=gray];\n",
				dotID(c.subject), dotID(c.object), c.predicateType)
		}
	}

	fmt.Println("}")
	return nil
}

// dotID sanitizes an entity name for use as a Graphviz node ID.
func dotID(name string) string {
	// Graphviz IDs must be quoted if they contain special chars.
	// Go variable names are safe but quote anyway for consistency.
	return fmt.Sprintf("%q", name)
}

// dotLabel creates a short node label: "Name\n(RoleType)"
func dotLabel(e entityInfo) string {
	// Convert CamelCase to spaces for readability
	name := e.name
	if len(name) > 25 {
		name = name[:22] + "..."
	}
	return name + "\n(" + e.roleType + ")"
}

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
