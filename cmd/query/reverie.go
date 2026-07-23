package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// runReverie takes an associative walk over the typed-claim graph: from a seed
// entity (or a random one) it drifts along real edges to a distant, connected
// place, biased toward rarer neighbours so the destination surprises. Unlike a
// trip (an LLM *inventing* a cross-cluster edge), a reverie only ever traverses
// claims that already exist — it cannot fabricate a connection. Code, not model:
// fast, free, and un-inventable. It's the KB daydreaming down its own links.

type reverieHop struct {
	Predicate string `json:"predicate"`
	Dir       string `json:"dir"` // "->" subject→object, "<-" object→subject
	To        string `json:"to"`
	ToName    string `json:"to_name"`
}

type reverieResult struct {
	Start     string       `json:"start"`
	StartName string       `json:"start_name"`
	Path      []reverieHop `json:"path"`
	Dest      string       `json:"dest"`
	DestName  string       `json:"dest_name"`
	DestBrief string       `json:"dest_brief,omitempty"`
}

func runReverie(kb *kbIndex, seed string, jsonOut bool) {
	type edge struct{ pred, to, dir string }
	adj := map[string][]edge{}
	deg := map[string]int{}
	for _, c := range kb.Claims {
		if c.Subject == "" || c.Object == "" || c.Subject == c.Object {
			continue // unary or self-claims aren't edges to wander
		}
		adj[c.Subject] = append(adj[c.Subject], edge{c.Predicate, c.Object, "->"})
		adj[c.Object] = append(adj[c.Object], edge{c.Predicate, c.Subject, "<-"})
		deg[c.Subject]++
		deg[c.Object]++
	}
	if len(adj) == 0 {
		fmt.Println("reverie: the graph has no edges to wander")
		return
	}
	ent := map[string]entityRecord{}
	for _, e := range kb.Entities {
		ent[e.VarName] = e
	}
	name := func(v string) string {
		if e, ok := ent[v]; ok && e.Name != "" {
			return e.Name
		}
		return v
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// choose a start
	start := ""
	if strings.TrimSpace(seed) != "" {
		q := strings.ToLower(seed)
		for _, e := range kb.Entities {
			if matchEntity(e, q) && len(adj[e.VarName]) > 0 {
				start = e.VarName
				break
			}
		}
		if start == "" {
			fmt.Printf("reverie: no connected entity matches %q\n", seed)
			return
		}
	} else {
		keys := make([]string, 0, len(adj))
		for v := range adj {
			keys = append(keys, v)
		}
		start = keys[rng.Intn(len(keys))]
	}

	// walk: up to N hops, no immediate backtrack, no revisits, weighted toward
	// rarer (lower-degree) neighbours so the drift finds the surprising edge.
	const steps = 5
	visited := map[string]bool{start: true}
	cur, prev := start, ""
	var path []reverieHop
	for i := 0; i < steps; i++ {
		var cands []edge
		for _, h := range adj[cur] {
			if h.to == prev || visited[h.to] {
				continue
			}
			cands = append(cands, h)
		}
		if len(cands) == 0 {
			break
		}
		weights := make([]float64, len(cands))
		total := 0.0
		for j, h := range cands {
			w := 1.0 / float64(1+deg[h.to]) // rarer neighbour → heavier
			weights[j] = w
			total += w
		}
		r := rng.Float64() * total
		pick := cands[len(cands)-1]
		for j, h := range cands {
			if r -= weights[j]; r <= 0 {
				pick = h
				break
			}
		}
		path = append(path, reverieHop{pick.pred, pick.dir, pick.to, name(pick.to)})
		prev, cur = cur, pick.to
		visited[cur] = true
	}

	res := reverieResult{Start: start, StartName: name(start), Path: path, Dest: cur, DestName: name(cur)}
	if e, ok := ent[cur]; ok {
		res.DestBrief = firstSentence(e.Brief, 180)
	}

	if jsonOut {
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
		return
	}

	if len(path) == 0 {
		fmt.Printf("reverie · %s sits at a dead end — no onward edge to follow\n", res.StartName)
		return
	}
	fmt.Printf("reverie · a walk from %s\n\n", res.StartName)
	fmt.Printf("  %s\n", res.StartName)
	indent := "  "
	for _, h := range path {
		indent += "   "
		arrow := "──▶"
		if h.Dir == "<-" {
			arrow = "◀──"
		}
		fmt.Printf("%s└─ %s %s  %s\n", indent, h.Predicate, arrow, h.ToName)
	}
	fmt.Printf("\narrived at %s", res.DestName)
	if res.DestBrief != "" {
		fmt.Printf("\n  %s", res.DestBrief)
	}
	fmt.Printf("\n  (%d hops · every edge is a real claim, nothing invented)\n", len(path))
}

// firstSentence trims a brief to its first sentence, or maxChars, for display.
func firstSentence(s string, maxChars int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if i := strings.IndexAny(s, ".!?"); i > 0 && i < maxChars {
		return s[:i+1]
	}
	if len(s) > maxChars {
		return strings.TrimSpace(s[:maxChars]) + "…"
	}
	return s
}
