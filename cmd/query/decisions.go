package main

// The decision-log read view: which choices are current and which were
// superseded. A decision isn't a separate ontological category — it's a
// memory with a lifecycle, and the lifecycle is the typed part: a `Supersedes`
// claim (winze-memory's predicate, written via winze_link) says memory B
// replaces memory A. The compiler gates the graph — a Supersedes pointing at a
// deleted memory won't build — so "what's the current decision on X" is a query
// over real edges, not a grep over prose that may lie.
//
// This reuses Supersedes rather than adding a Decision store: decisions are the
// subset of project-memory that gets superseded, so they live where the memory
// already is. No new store means no cross-store duplication of the same
// decision — the coin-time-dedup failure a second store would invite.

import (
	"fmt"
	"sort"
)

type decisionChain struct {
	Current     string   `json:"current"` // var name of the un-superseded head
	CurrentName string   `json:"current_name"`
	Superseded  []string `json:"superseded"` // names, newest-superseded first
}

// decisionChains builds the Supersedes graph into chains: each current (un-
// superseded) decision followed by the decisions it replaced, transitively.
// Longest-history chains first. Pure over kb so it is directly testable.
func decisionChains(kb *kbIndex) []decisionChain {
	newerOf := map[string]string{} // A -> B where B Supersedes A
	olderOf := map[string]string{} // B -> A where B Supersedes A
	inGraph := map[string]bool{}
	for _, c := range kb.Claims {
		if c.Predicate != "Supersedes" || c.Subject == "" || c.Object == "" {
			continue
		}
		// Convention: Subject supersedes Object (Subject is the newer decision).
		newerOf[c.Object] = c.Subject
		olderOf[c.Subject] = c.Object
		inGraph[c.Subject] = true
		inGraph[c.Object] = true
	}
	if len(inGraph) == 0 {
		return nil
	}

	name := entityNamer(kb)

	// Heads are entities in the graph that nothing supersedes (not an Object of
	// any Supersedes) — the current decisions.
	var heads []string
	for v := range inGraph {
		if _, superseded := newerOf[v]; !superseded {
			heads = append(heads, v)
		}
	}
	sort.Slice(heads, func(i, j int) bool { return name(heads[i]) < name(heads[j]) })

	var chains []decisionChain
	for _, h := range heads {
		ch := decisionChain{Current: h, CurrentName: name(h)}
		// Walk down: h supersedes olderOf[h] supersedes ... Guard against a
		// cycle (a Supersedes loop is a data error) with a visited set.
		seen := map[string]bool{h: true}
		for cur := h; ; {
			older, ok := olderOf[cur]
			if !ok || seen[older] {
				break
			}
			ch.Superseded = append(ch.Superseded, name(older))
			seen[older] = true
			cur = older
		}
		chains = append(chains, ch)
	}
	sort.Slice(chains, func(i, j int) bool {
		if len(chains[i].Superseded) != len(chains[j].Superseded) {
			return len(chains[i].Superseded) > len(chains[j].Superseded) // longest history first
		}
		return chains[i].CurrentName < chains[j].CurrentName
	})
	return chains
}

// runDecisions renders the decision log: current decisions and what each
// superseded. A store with no Supersedes claims prints a clean pointer.
func runDecisions(kb *kbIndex, jsonOut bool) {
	chains := decisionChains(kb)
	if len(chains) == 0 {
		if jsonOut {
			printJSON(map[string]any{"chains": []decisionChain{}, "count": 0})
			return
		}
		fmt.Println("decisions: no Supersedes claims — nothing has been superseded yet")
		fmt.Println("  (record one with winze_link(from=Newer, to=Older, relation=\"Supersedes\", rationale=…))")
		return
	}

	if jsonOut {
		printJSON(map[string]any{"chains": chains, "count": len(chains)})
		return
	}

	fmt.Printf("decisions — %d current, tracing what each superseded:\n\n", len(chains))
	for _, ch := range chains {
		fmt.Printf("  ● %s\n", ch.CurrentName)
		for i, s := range ch.Superseded {
			lead := "    ↑ supersedes"
			if i > 0 {
				lead = "    ↑ which superseded"
			}
			fmt.Printf("%s  %s\n", lead, s)
		}
	}
}

// entityNamer returns a func mapping a var name to its display Name, falling
// back to the var name when the entity carries no Name.
func entityNamer(kb *kbIndex) func(string) string {
	byVar := make(map[string]string, len(kb.Entities))
	for _, e := range kb.Entities {
		if e.Name != "" {
			byVar[e.VarName] = e.Name
		}
	}
	return func(v string) string {
		if n, ok := byVar[v]; ok {
			return n
		}
		return v
	}
}
