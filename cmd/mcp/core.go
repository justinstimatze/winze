package main

// Core query functions shared by MCP handlers and A2A dispatch.
// Each returns a map[string]any suitable for JSON serialization.

import "strings"

func (h *handler) coreSearch(query string) any {
	q := strings.ToLower(query)
	var results []map[string]any
	for _, e := range h.kb.Entities {
		if !matchEntity(e, q) {
			continue
		}
		rec := map[string]any{
			"var_name":  e.VarName,
			"role_type": e.RoleType,
			"file":      e.File,
		}
		if e.Name != "" {
			rec["name"] = e.Name
		}
		if e.Brief != "" {
			rec["brief"] = e.Brief
		}
		if e.ID != "" {
			rec["id"] = e.ID
		}
		related := claimsInvolving(h.kb, e.VarName)
		if len(related) > 0 {
			rec["claims"] = related
		}
		results = append(results, rec)
	}
	if len(results) == 0 {
		return map[string]any{"matches": 0, "message": "No entities matching " + query}
	}
	return map[string]any{"matches": len(results), "entities": results}
}

func (h *handler) coreTheories(concept string) any {
	q := strings.ToLower(concept)

	var targetEntity *entityRecord
	for i, e := range h.kb.Entities {
		if matchEntity(e, q) {
			targetEntity = &h.kb.Entities[i]
			break
		}
	}

	var theories []map[string]any
	for _, c := range h.kb.Claims {
		if c.Predicate != "TheoryOf" {
			continue
		}
		match := false
		if targetEntity != nil && c.Object == targetEntity.VarName {
			match = true
		} else if strings.Contains(strings.ToLower(c.Object), q) {
			match = true
		}
		if !match {
			continue
		}
		t := map[string]any{
			"hypothesis": c.Subject,
			"concept":    c.Object,
			"prov":       c.ProvRef,
			"file":       c.File,
		}
		for _, e := range h.kb.Entities {
			if e.VarName == c.Subject && e.Brief != "" {
				t["brief"] = e.Brief
				break
			}
		}
		theories = append(theories, t)
	}

	if len(theories) == 0 {
		return map[string]any{"count": 0, "message": "No competing theories found for " + concept}
	}

	label := concept
	if targetEntity != nil && targetEntity.Name != "" {
		label = targetEntity.Name
	}
	return map[string]any{"concept": label, "count": len(theories), "theories": theories}
}

func (h *handler) coreClaims(entity string) any {
	q := strings.ToLower(entity)
	var targetName string
	for _, e := range h.kb.Entities {
		if matchEntity(e, q) {
			targetName = e.VarName
			break
		}
	}
	if targetName == "" {
		targetName = entity
	}

	related := claimsInvolving(h.kb, targetName)
	if len(related) == 0 {
		return map[string]any{"count": 0, "message": "No claims found involving " + entity}
	}
	return map[string]any{"entity": targetName, "count": len(related), "claims": related}
}

func (h *handler) coreProvenance(query string) any {
	q := strings.ToLower(query)
	var results []map[string]any
	for _, p := range h.kb.Provenance {
		if !strings.Contains(strings.ToLower(p.Origin), q) &&
			!strings.Contains(strings.ToLower(p.VarName), q) {
			continue
		}
		rec := map[string]any{
			"var_name": p.VarName,
			"origin":   p.Origin,
			"file":     p.File,
		}
		if p.Quote != "" {
			rec["quote"] = p.Quote
		}
		var refs []string
		for _, c := range h.kb.Claims {
			if c.ProvRef == p.VarName {
				refs = append(refs, c.VarName)
			}
		}
		if len(refs) > 0 {
			rec["used_by"] = refs
		}
		results = append(results, rec)
	}
	if len(results) == 0 {
		return map[string]any{"count": 0, "message": "No provenance matching " + query}
	}
	return map[string]any{"count": len(results), "provenance": results}
}

func (h *handler) coreDisputes() any {
	var disputes []claimRecord
	for _, c := range h.kb.Claims {
		if c.Predicate == "Disputes" || c.Predicate == "DisputesOrg" {
			disputes = append(disputes, c)
		}
	}
	if len(disputes) == 0 {
		return map[string]any{"count": 0, "message": "No disputes in the KB."}
	}
	return map[string]any{"count": len(disputes), "disputes": disputes}
}

func (h *handler) coreStats() any {
	roleCounts := map[string]int{}
	for _, e := range h.kb.Entities {
		roleCounts[e.RoleType]++
	}

	predCounts := map[string]int{}
	for _, c := range h.kb.Claims {
		predCounts[c.Predicate]++
	}

	files := map[string]bool{}
	for _, e := range h.kb.Entities {
		files[e.File] = true
	}

	disputes := 0
	for _, c := range h.kb.Claims {
		if c.Predicate == "Disputes" || c.Predicate == "DisputesOrg" {
			disputes++
		}
	}

	theoryTargets := map[string]int{}
	for _, c := range h.kb.Claims {
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

	return map[string]any{
		"entities":           len(h.kb.Entities),
		"claims":             len(h.kb.Claims),
		"predicates":         len(predCounts),
		"provenance":         len(h.kb.Provenance),
		"files":              len(files),
		"disputes":           disputes,
		"contested_concepts": contested,
		"role_counts":        roleCounts,
		"predicate_counts":   sortedCounts(predCounts),
	}
}
