package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

// The fleet payload is assembled live from each instance's corpus: nodes from
// entities, edges from typed claims PLUS resolved [[wikilink]] references in
// briefs (so entity-only memory stores — which have no typed claims yet — still
// show their real implicit link graph instead of a disconnected cloud).

// reifications bury the concept graph; exclude them from the organism.
var skipPred = map[string]bool{"Predicts": true, "ResolvedAs": true, "Credence": true}

// typed-claim family codes for edge styling (parallel to the HTML).
var family = map[string]int{
	"TheoryOf": 1, "HypothesisExplains": 1, "EarlyFormulationOf": 1,
	"Proposes": 2, "ProposesOrg": 2, "Accepts": 2, "AcceptsOrg": 2,
	"Disputes": 3, "DisputesOrg": 3,
	"BelongsTo": 4, "DerivedFrom": 4, "StructurallyAnalogousTo": 4,
	"Authored": 5, "AuthoredOrg": 5, "CommentaryOn": 5, "AppearsIn": 5,
	"InfluencedBy": 6, "AffiliatedWith": 6, "WorksFor": 6,
}

const famMention = 7 // wikilink-derived edge (latent, not yet a typed claim)

var wikilinkRE = regexp.MustCompile(`\[\[([^\]\|]+)`)

type nodeJSON struct {
	L string `json:"l"`
	C int    `json:"c"`
	D int    `json:"d"`
}

type instanceJSON struct {
	Name     string     `json:"name"`
	Kind     string     `json:"kind"`
	Tier     int        `json:"tier"`
	Entities int        `json:"entities"`
	Claims   int        `json:"claims"`
	Nodes    []nodeJSON `json:"nodes"`
	Edges    [][2]int   `json:"edges"`
	Efam     []int      `json:"efam"`
	Trips    [][3]int   `json:"trips"`
	Spent    int        `json:"spent"`
	Cap      int        `json:"cap"`
	// LastActivity is the unix time (seconds) of the newest metabolism artifact
	// in the corpus — the real "when did this winze last work" signal. 0 means
	// no metabolism has ever run here (e.g. a pure memory store).
	LastActivity int64 `json:"last_activity"`
}

type fleetJSON struct {
	Instances []instanceJSON `json:"instances"`
}

func buildFleet(dirs []string) (fleetJSON, error) {
	targets := resolveTargets(dirs)
	out := fleetJSON{}
	for _, t := range targets {
		inst, err := buildInstance(t.dir, t.tier)
		if err != nil {
			continue // a missing/unparseable instance is skipped, not fatal
		}
		out.Instances = append(out.Instances, inst)
	}
	return out, nil
}

type target struct {
	dir  string
	tier int
}

// resolveTargets: explicit dirs win; else the metabolize registry; else CWD.
func resolveTargets(dirs []string) []target {
	if len(dirs) > 0 {
		ts := make([]target, 0, len(dirs))
		for _, d := range dirs {
			abs, _ := filepath.Abs(d)
			ts = append(ts, target{dir: abs, tier: registryTier(abs)})
		}
		return ts
	}
	if reg := loadRegistryInstances(); len(reg) > 0 {
		return reg
	}
	cwd, _ := os.Getwd()
	return []target{{dir: cwd, tier: 1}}
}

func configPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "winze", "metabolize.json")
}

type regFile struct {
	Instances []struct {
		Dir         string `json:"dir"`
		Tier        int    `json:"tier"`
		BudgetCents int    `json:"budget_cents"`
	} `json:"instances"`
}

func loadRegistryInstances() []target {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil
	}
	var r regFile
	if json.Unmarshal(data, &r) != nil {
		return nil
	}
	ts := make([]target, 0, len(r.Instances))
	for _, in := range r.Instances {
		ts = append(ts, target{dir: in.Dir, tier: in.Tier})
	}
	return ts
}

func registryTier(dir string) int {
	for _, t := range loadRegistryInstances() {
		if t.dir == dir {
			return t.tier
		}
	}
	return 1
}

func buildInstance(dir string, tier int) (instanceJSON, error) {
	entities, claims, err := corpusparse.ParseCorpus(dir)
	if err != nil {
		return instanceJSON{}, err
	}
	// index entities by var and by normalized name/id for wikilink resolution
	varOf := map[string]int{} // VarName -> entity index
	norm := map[string]int{}  // normalized name/id/var -> entity index
	for i, e := range entities {
		varOf[e.VarName] = i
		for _, k := range []string{e.VarName, e.Name, e.ID} {
			if k != "" {
				norm[normalize(k)] = i
			}
		}
	}

	type edge struct{ a, b, fam int }
	var edges []edge
	seen := map[[2]int]bool{}
	addEdge := func(a, b, fam int) {
		if a == b || a < 0 || b < 0 {
			return
		}
		key := [2]int{a, b}
		if a > b {
			key = [2]int{b, a}
		}
		if seen[key] {
			return
		}
		seen[key] = true
		edges = append(edges, edge{a, b, fam})
	}

	// typed claim edges (reifications excluded)
	typedCount := 0
	for _, c := range claims {
		if skipPred[c.PredicateType] || c.ObjectVar == "" {
			continue
		}
		a, ok1 := varOf[c.SubjectVar]
		b, ok2 := varOf[c.ObjectVar]
		if ok1 && ok2 {
			addEdge(a, b, family[c.PredicateType])
			typedCount++
		}
	}
	// latent wikilink edges from briefs
	for i, e := range entities {
		for _, m := range wikilinkRE.FindAllStringSubmatch(e.Brief, -1) {
			if j, ok := norm[normalize(m[1])]; ok {
				addEdge(i, j, famMention)
			}
		}
	}

	// nodes: those touched by an edge; if none (edgeless store), use all entities
	touched := map[int]bool{}
	for _, e := range edges {
		touched[e.a] = true
		touched[e.b] = true
	}
	var keep []int
	if len(touched) > 0 {
		for i := range entities {
			if touched[i] {
				keep = append(keep, i)
			}
		}
	} else {
		for i := range entities {
			keep = append(keep, i)
		}
	}
	remap := map[int]int{}
	for ni, ei := range keep {
		remap[ei] = ni
	}

	n := len(keep)
	deg := make([]int, n)
	adj := make([][]int, n)
	var reEdges [][2]int
	var efam []int
	for _, e := range edges {
		a, b := remap[e.a], remap[e.b]
		reEdges = append(reEdges, [2]int{a, b})
		efam = append(efam, e.fam)
		deg[a]++
		deg[b]++
		adj[a] = append(adj[a], b)
		adj[b] = append(adj[b], a)
	}

	comm := labelPropagate(n, adj)
	nodes := make([]nodeJSON, n)
	for ni, ei := range keep {
		nodes[ni] = nodeJSON{L: short(entities[ei].VarName), C: comm[ni], D: deg[ni]}
	}

	kind := "memory"
	if typedCount > 0 {
		kind = "epistemology"
	}
	spent, _ := readBudget(dir)
	budgetCap := 300
	if tier == 3 {
		budgetCap = 500
	}
	return instanceJSON{
		Name:         filepath.Base(strings.TrimRight(dir, "/")),
		Kind:         kind,
		Tier:         tier,
		Entities:     len(entities),
		Claims:       typedCount,
		Nodes:        nodes,
		Edges:        reEdges,
		Efam:         efam,
		Trips:        loadTrips(dir, varOf, remap),
		Spent:        spent,
		Cap:          budgetCap,
		LastActivity: newestMetabolismMtime(dir),
	}, nil
}

// newestMetabolismMtime returns the unix seconds of the most recently modified
// metabolism artifact in dir, or 0 if none exist. This is the honest "last
// worked" clock — no metabolism files means the winze has never metabolized.
func newestMetabolismMtime(dir string) int64 {
	var newest int64
	names := []string{
		".metabolism-log.json", ".metabolism-budget.json",
		".metabolism-calibration.jsonl", ".metabolism-trip-isolated.jsonl",
		".metabolism-topology-state.json",
	}
	for _, n := range names {
		if fi, err := os.Stat(filepath.Join(dir, n)); err == nil {
			if t := fi.ModTime().Unix(); t > newest {
				newest = t
			}
		}
	}
	// trip-cycle files land as metabolism_cycle*.go
	if matches, _ := filepath.Glob(filepath.Join(dir, "metabolism_cycle*.go")); matches != nil {
		for _, m := range matches {
			if fi, err := os.Stat(m); err == nil {
				if t := fi.ModTime().Unix(); t > newest {
					newest = t
				}
			}
		}
	}
	return newest
}

// labelPropagate finds organic communities, then renumbers so the nine largest
// keep distinct ids and everything else collapses to a single "other" bucket.
func labelPropagate(n int, adj [][]int) []int {
	label := make([]int, n)
	for i := range label {
		label[i] = i
	}
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	for pass := 0; pass < 12; pass++ {
		// deterministic sweep (no RNG — server output should be stable per corpus)
		changed := 0
		for _, v := range order {
			if len(adj[v]) == 0 {
				continue
			}
			cnt := map[int]int{}
			for _, u := range adj[v] {
				cnt[label[u]]++
			}
			best, bestC := label[v], -1
			for l, c := range cnt {
				if c > bestC || (c == bestC && l < best) {
					best, bestC = l, c
				}
			}
			if label[v] != best {
				label[v] = best
				changed++
			}
		}
		if changed == 0 {
			break
		}
	}
	size := map[int]int{}
	for _, l := range label {
		size[l]++
	}
	ranked := make([]int, 0, len(size))
	for l := range size {
		ranked = append(ranked, l)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if size[ranked[i]] != size[ranked[j]] {
			return size[ranked[i]] > size[ranked[j]]
		}
		return ranked[i] < ranked[j]
	})
	remap := map[int]int{}
	for i, l := range ranked {
		if i < 9 {
			remap[l] = i
		} else {
			remap[l] = 9
		}
	}
	out := make([]int, n)
	for i, l := range label {
		out[i] = remap[l]
	}
	return out
}

func loadTrips(dir string, varOf map[string]int, remap map[int]int) [][3]int {
	data, err := os.ReadFile(filepath.Join(dir, ".metabolism-trip-isolated.jsonl"))
	if err != nil {
		return nil
	}
	type rec struct {
		A, B, EA, EB string
		Score        int
	}
	var out [][3]int
	seen := map[[2]int]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw map[string]json.RawMessage
		if json.Unmarshal([]byte(line), &raw) != nil {
			continue
		}
		get := func(keys ...string) string {
			for _, k := range keys {
				if v, ok := raw[k]; ok {
					var s string
					if json.Unmarshal(v, &s) == nil {
						return s
					}
				}
			}
			return ""
		}
		a := get("entity_a", "a")
		b := get("entity_b", "b")
		var score int
		if v, ok := raw["score"]; ok {
			json.Unmarshal(v, &score)
		}
		ia, ok1 := varOf[a]
		ib, ok2 := varOf[b]
		if !ok1 || !ok2 {
			continue
		}
		na, ok3 := remap[ia]
		nb, ok4 := remap[ib]
		if !ok3 || !ok4 || na == nb {
			continue
		}
		key := [2]int{na, nb}
		if na > nb {
			key = [2]int{nb, na}
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, [3]int{na, nb, score})
	}
	sort.Slice(out, func(i, j int) bool { return out[i][2] > out[j][2] })
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func readBudget(dir string) (int, string) {
	data, err := os.ReadFile(filepath.Join(dir, ".metabolism-budget.json"))
	if err != nil {
		return 0, ""
	}
	var b struct {
		SpentCents int    `json:"spent_cents"`
		Month      string `json:"month"`
	}
	if json.Unmarshal(data, &b) != nil {
		return 0, ""
	}
	return b.SpentCents, b.Month
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]`)

func normalize(s string) string { return nonAlnum.ReplaceAllString(strings.ToLower(s), "") }

func short(s string) string {
	if len(s) > 26 {
		return s[:26]
	}
	return s
}
