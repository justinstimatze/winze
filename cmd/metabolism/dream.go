// Dream cycle: NREM-like consolidation without new ingest.
//
// Orchestrates existing analysis tools (topology, lint, adit) and synthesizes
// their output into a maintenance report. No external sensor queries. No new
// entities. The dream cycle processes what's already there.
//
// Analogy: slow-wave sleep consolidation. Replay existing KB structure,
// identify what's tangled, over-connected, or fragile, and surface
// refactoring opportunities.
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// --- dream cycle types ---

// DreamReport is the consolidation analysis output.
type DreamReport struct {
	// Topology summary
	Entities int `json:"entities"`
	Claims   int `json:"claims"`
	Clusters int `json:"clusters"`

	// Consolidation targets by category
	BridgeEntities    []DreamFinding `json:"bridge_entities"`
	ConcentrationRisk []DreamFinding `json:"concentration_risk"`
	FileBalance       []DreamFinding `json:"file_balance"`
	ProvenanceSplits  []DreamFinding `json:"provenance_splits"`
	BriefQuality      []DreamFinding `json:"brief_quality"`
	AditScores        []DreamFinding `json:"adit_scores,omitempty"`
	BiasAudit         *BiasReport    `json:"bias_audit,omitempty"`

	// Summary counts
	TotalFindings int `json:"total_findings"`
}

// DreamFinding is a single consolidation opportunity.
type DreamFinding struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"` // critical, warning, info
	Entity      string `json:"entity,omitempty"`
	File        string `json:"file,omitempty"`
	Description string `json:"description"`
}

func runDream(dir string, includeBias bool, jsonOut bool) {
	// 1. Topology analysis
	_, report, err := runTopology(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dream: topology: %v\n", err)
		os.Exit(1)
	}

	var dream DreamReport
	dream.Entities = report.Entities
	dream.Claims = report.Claims
	dream.Clusters = report.Clusters

	// 2. Extract bridge entities and concentration risk from topology
	for _, v := range report.Vulnerabilities {
		switch v.Type {
		case "bridge_entity":
			dream.BridgeEntities = append(dream.BridgeEntities, DreamFinding{
				Category:    "bridge_entity",
				Severity:    v.Severity,
				Entity:      v.Entity,
				Description: v.Description,
			})
		case "concentration_risk":
			dream.ConcentrationRisk = append(dream.ConcentrationRisk, DreamFinding{
				Category:    "concentration_risk",
				Severity:    v.Severity,
				Entity:      v.Entity,
				Description: v.Description,
			})
		}
	}

	// 3. File balance analysis — entity and claim distribution across corpus files
	dream.FileBalance = analyzeFileBalance(dir)

	// 4. Provenance split detection — same source cited differently across files
	dream.ProvenanceSplits = analyzeProvenanceSplits(dir)

	// 5. Brief quality — entities with very short, very long, or vague Briefs
	dream.BriefQuality = analyzeBriefQuality(dir)

	// 6. Adit scores (if available)
	dream.AditScores = runAditScoring(dir)

	// 7. Bias audit (if requested)
	if includeBias {
		var results []BiasAuditorResult
		results = append(results, auditConfirmationBias(dir))
		results = append(results, auditAnchoringBias(dir))
		results = append(results, auditClusteringIllusion(dir))
		results = append(results, auditAvailabilityHeuristic(dir))
		results = append(results, auditSurvivorshipBias(dir))
		results = append(results, auditFramingEffect(dir))
		results = append(results, auditDunningKruger(dir))
		results = append(results, auditBaseRateNeglect(dir))
		triggered := 0
		for _, r := range results {
			if r.Triggered {
				triggered++
			}
		}
		dream.BiasAudit = &BiasReport{
			Auditors: results,
			Summary:  fmt.Sprintf("%d of %d bias auditors triggered", triggered, len(results)),
		}
	}

	// Count total
	dream.TotalFindings = len(dream.BridgeEntities) + len(dream.ConcentrationRisk) +
		len(dream.FileBalance) + len(dream.ProvenanceSplits) +
		len(dream.BriefQuality) + len(dream.AditScores)

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(dream)
		return
	}

	// --- text output ---
	fmt.Printf("[dream] consolidation report — %d entities, %d claims, %d clusters\n\n",
		dream.Entities, dream.Claims, dream.Clusters)

	printDreamSection("bridge entities (load-bearing, need strengthening)", dream.BridgeEntities)
	printDreamSection("concentration risk (over-connected, refactoring candidates)", dream.ConcentrationRisk)
	printDreamSection("file balance (entity distribution)", dream.FileBalance)
	printDreamSection("provenance splits (same source, different citation)", dream.ProvenanceSplits)
	printDreamSection("brief quality", dream.BriefQuality)
	printDreamSection("adit scores (agent-writability)", dream.AditScores)

	if dream.BiasAudit != nil {
		fmt.Printf("  bias self-audit (%s):\n", dream.BiasAudit.Summary)
		for _, r := range dream.BiasAudit.Auditors {
			status := "PASS"
			marker := "  "
			if r.Triggered {
				status = "TRIGGERED"
				marker = "* "
			}
			fmt.Printf("    %s%s: %s (%.2f vs %.2f threshold)\n",
				marker, r.BiasName, status, r.Value, r.Threshold)
		}
		fmt.Println()
	}

	fmt.Printf("\n[dream] %d findings total\n", dream.TotalFindings)
	if dream.TotalFindings == 0 {
		fmt.Println("[dream] KB is well-consolidated — no maintenance needed")
	}
}

func printDreamSection(title string, findings []DreamFinding) {
	if len(findings) == 0 {
		return
	}
	fmt.Printf("  %s:\n", title)
	for _, f := range findings {
		marker := " "
		switch f.Severity {
		case "critical":
			marker = "!"
		case "warning":
			marker = "*"
		case "info":
			marker = "·"
		}
		location := ""
		if f.File != "" {
			location = f.File + ": "
		}
		if f.Entity != "" && f.File != "" {
			location = f.File + " / " + f.Entity + ": "
		} else if f.Entity != "" {
			location = f.Entity + ": "
		}
		fmt.Printf("    %s %s%s\n", marker, location, f.Description)
	}
	fmt.Println()
}

// analyzeFileBalance checks entity/claim distribution across corpus files.
// Files with very few entities are candidates for merging; files with many
// are candidates for splitting.
func analyzeFileBalance(dir string) []DreamFinding {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, goFileFilter, parser.ParseComments)
	if err != nil {
		return nil
	}

	type fileStats struct {
		entities int
		claims   int
	}
	stats := map[string]*fileStats{}

	// Collect role types for entity detection
	roleTypes := collectDreamRoleTypes(pkgs)

	// Count entities and claims per file
	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
			base := filepath.Base(fname)
			// Skip non-corpus files
			if isInfraFile(base) {
				continue
			}
			s := &fileStats{}
			stats[base] = s

			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Values) == 0 {
						continue
					}
					cl, ok := vs.Values[0].(*ast.CompositeLit)
					if !ok {
						continue
					}
					typeName := compositeTypeName(cl)
					if typeName == "" {
						continue
					}
					if roleTypes[typeName] {
						s.entities++
					} else if typeName != "Provenance" {
						// Non-entity, non-provenance composites are claims
						s.claims++
					}
				}
			}
		}
	}

	var findings []DreamFinding

	// Compute stats for thresholds
	var entityCounts []int
	for _, s := range stats {
		if s.entities > 0 || s.claims > 0 {
			entityCounts = append(entityCounts, s.entities)
		}
	}
	if len(entityCounts) < 2 {
		return nil
	}

	sort.Ints(entityCounts)
	median := entityCounts[len(entityCounts)/2]

	for fname, s := range stats {
		if s.entities == 0 && s.claims == 0 {
			continue
		}
		// Small files: candidates for merging
		if s.entities > 0 && s.entities <= 2 && median > 5 {
			findings = append(findings, DreamFinding{
				Category:    "file_balance",
				Severity:    "info",
				File:        fname,
				Description: fmt.Sprintf("only %d entities — candidate for merging with related file", s.entities),
			})
		}
		// Large files: candidates for splitting
		if s.entities > median*3 && s.entities > 20 {
			findings = append(findings, DreamFinding{
				Category:    "file_balance",
				Severity:    "warning",
				File:        fname,
				Description: fmt.Sprintf("%d entities (median %d) — candidate for splitting by sub-topic", s.entities, median),
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		return findings[i].File < findings[j].File
	})
	return findings
}

// analyzeProvenanceSplits finds provenance variables that reference the same
// origin but are declared in different files (or same file with different names).
func analyzeProvenanceSplits(dir string) []DreamFinding {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, goFileFilter, parser.ParseComments)
	if err != nil {
		return nil
	}

	type provEntry struct {
		varName string
		file    string
		origin  string
	}
	var provs []provEntry

	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
			base := filepath.Base(fname)
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Values) == 0 {
						continue
					}
					cl, ok := vs.Values[0].(*ast.CompositeLit)
					if !ok {
						continue
					}
					if compositeTypeName(cl) != "Provenance" {
						continue
					}
					origin := extractStringField(cl, "Origin")
					if origin != "" {
						provs = append(provs, provEntry{
							varName: vs.Names[0].Name,
							file:    base,
							origin:  origin,
						})
					}
				}
			}
		}
	}

	// Group by normalized origin
	byOrigin := map[string][]provEntry{}
	for _, p := range provs {
		key := strings.ToLower(strings.TrimSpace(p.origin))
		byOrigin[key] = append(byOrigin[key], p)
	}

	var findings []DreamFinding
	for _, entries := range byOrigin {
		if len(entries) <= 1 {
			continue
		}
		// Check if they're in different files or have different var names
		files := map[string]bool{}
		var names []string
		for _, e := range entries {
			files[e.file] = true
			names = append(names, e.varName)
		}
		if len(files) > 1 {
			findings = append(findings, DreamFinding{
				Category:    "provenance_split",
				Severity:    "warning",
				Description: fmt.Sprintf("origin %q cited in %d files (%s) — consolidate to one shared provenance", entries[0].origin, len(files), strings.Join(names, ", ")),
			})
		}
	}
	return findings
}

// analyzeBriefQuality checks entity Brief fields for quality issues.
func analyzeBriefQuality(dir string) []DreamFinding {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, goFileFilter, parser.ParseComments)
	if err != nil {
		return nil
	}

	roleTypes := collectDreamRoleTypes(pkgs)
	var findings []DreamFinding

	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
			base := filepath.Base(fname)
			if isInfraFile(base) {
				continue
			}
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Values) == 0 {
						continue
					}
					cl, ok := vs.Values[0].(*ast.CompositeLit)
					if !ok {
						continue
					}
					typeName := compositeTypeName(cl)
					if !roleTypes[typeName] {
						continue
					}

					varName := vs.Names[0].Name
					brief := extractEntityBrief(cl)

					if brief == "" {
						findings = append(findings, DreamFinding{
							Category:    "brief_quality",
							Severity:    "warning",
							File:        base,
							Entity:      varName,
							Description: "missing Brief field",
						})
					} else if len(brief) < 20 {
						findings = append(findings, DreamFinding{
							Category:    "brief_quality",
							Severity:    "info",
							File:        base,
							Entity:      varName,
							Description: fmt.Sprintf("Brief is very short (%d chars) — may lack distinguishing detail", len(brief)),
						})
					} else if len(brief) > 300 {
						findings = append(findings, DreamFinding{
							Category:    "brief_quality",
							Severity:    "info",
							File:        base,
							Entity:      varName,
							Description: fmt.Sprintf("Brief is very long (%d chars) — consider tightening", len(brief)),
						})
					}
				}
			}
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity < findings[j].Severity
		}
		return findings[i].File < findings[j].File
	})
	return findings
}

// runAditScoring runs adit score-file on each corpus .go file if adit CLI is available.
func runAditScoring(dir string) []DreamFinding {
	// Check if adit CLI is available
	if _, err := exec.LookPath("adit"); err != nil {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var findings []DreamFinding
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || isInfraFile(e.Name()) {
			continue
		}

		fpath := filepath.Join(dir, e.Name())
		cmd := exec.Command("adit", "score-file", fpath, "--json")
		out, err := cmd.Output()
		if err != nil {
			continue
		}

		var result struct {
			Score    float64 `json:"score"`
			Findings []struct {
				Rule    string `json:"rule"`
				Message string `json:"message"`
			} `json:"findings"`
		}
		if err := json.Unmarshal(out, &result); err != nil {
			continue
		}

		if result.Score < 70 {
			severity := "info"
			if result.Score < 50 {
				severity = "warning"
			}
			desc := fmt.Sprintf("adit score %.0f/100", result.Score)
			if len(result.Findings) > 0 {
				var rules []string
				for _, f := range result.Findings {
					rules = append(rules, f.Rule)
				}
				desc += " (" + strings.Join(rules, ", ") + ")"
			}
			findings = append(findings, DreamFinding{
				Category:    "adit_score",
				Severity:    severity,
				File:        e.Name(),
				Description: desc,
			})
		}
	}
	return findings
}

// --- dream helpers ---

func goFileFilter(info os.FileInfo) bool {
	return strings.HasSuffix(info.Name(), ".go")
}

func isInfraFile(name string) bool {
	infra := map[string]bool{
		"schema.go":       true,
		"roles.go":        true,
		"predicates.go":   true,
		"design_roles.go": true,
	}
	// Also skip test files and cmd directories
	return infra[name] || strings.HasSuffix(name, "_test.go")
}

// collectDreamRoleTypes finds types that embed *Entity (role types).
func collectDreamRoleTypes(pkgs map[string]*ast.Package) map[string]bool {
	roleTypes := map[string]bool{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range st.Fields.List {
						if len(field.Names) > 0 {
							continue // named field
						}
						// Check for *Entity
						star, ok := field.Type.(*ast.StarExpr)
						if !ok {
							continue
						}
						ident, ok := star.X.(*ast.Ident)
						if ok && ident.Name == "Entity" {
							roleTypes[ts.Name.Name] = true
						}
					}
				}
			}
		}
	}
	return roleTypes
}

// extractStringField extracts a string field value from a composite literal.
// Uses resolveStringExpr to handle concatenated strings ("a" + "b").
func extractStringField(cl *ast.CompositeLit, fieldName string) string {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != fieldName {
			continue
		}
		return resolveStringExpr(kv.Value)
	}
	return ""
}

// compositeTypeName and extractEntityBrief are defined in ingest.go.
