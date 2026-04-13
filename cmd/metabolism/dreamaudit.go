// Dream --bias: cognitive bias auditors that check the KB's own structure
// against the biases cataloged within it.
//
// Each auditor maps a cognitive bias entity already in the KB to a
// deterministic check on KB structure. The KB eats its own dogfood.
//
// Eight deterministic auditors:
//
//  1. Confirmation bias      — is the metabolism corroboration rate suspiciously high?
//  2. Anchoring               — are early-ingested entities disproportionately central?
//  3. Clustering illusion     — do topology clusters map to files rather than concepts?
//  4. Availability heuristic  — is the KB over-indexed on one source type?
//  5. Survivorship bias       — are rejected sources systematically filtered out?
//  6. Framing effect          — do Briefs use evaluative language that biases LLM judges?
//  7. Dunning-Kruger effect   — do simple entities appear healthy because they're under-examined?
//  8. Base rate neglect       — is the predicate distribution so skewed that common edges drown rare signal?
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// BiasReport holds findings from all bias auditors.
type BiasReport struct {
	Auditors []BiasAuditorResult `json:"auditors"`
	Summary  string              `json:"summary"`
}

// BiasAuditorResult is a single auditor's output.
type BiasAuditorResult struct {
	Bias       string  `json:"bias"`        // KB entity variable name
	BiasName   string  `json:"bias_name"`   // human-readable name
	Metric     string  `json:"metric"`      // what was measured
	Value      float64 `json:"value"`       // measured value
	Threshold  float64 `json:"threshold"`   // alerting threshold
	Triggered  bool    `json:"triggered"`   // value exceeds threshold
	Severity   string  `json:"severity"`    // info, warning, critical
	Detail     string  `json:"detail"`      // explanation
	Conclusion string  `json:"conclusion"`  // what this means for the KB
}

func runDreamBias(dir string, jsonOut bool) {
	collectBiasResults(dir, nil, jsonOut, true)
}

// collectBiasResults runs all bias auditors. If topoReport is non-nil,
// reuses it instead of re-running topology (expensive).
// Set printOutput to false when called from dream mode (dream handles display).
func collectBiasResults(dir string, topoReport *TopologyReport, jsonOut bool, printOutput bool) BiasReport {
	var results []BiasAuditorResult

	results = append(results, auditConfirmationBias(dir))
	results = append(results, auditAnchoringBias(dir))
	results = append(results, auditClusteringIllusion(dir))
	results = append(results, auditAvailabilityHeuristic(dir))
	results = append(results, auditSurvivorshipBias(dir))
	results = append(results, auditFramingEffect(dir))
	results = append(results, auditDunningKruger(dir, topoReport))
	results = append(results, auditBaseRateNeglect(dir))

	triggered := 0
	for _, r := range results {
		if r.Triggered {
			triggered++
		}
	}

	report := BiasReport{
		Auditors: results,
		Summary:  fmt.Sprintf("%d of %d bias auditors triggered", triggered, len(results)),
	}

	if !printOutput {
		return report
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return report
	}

	fmt.Printf("[dream-bias] cognitive bias self-audit — %d auditors\n\n", len(results))
	for _, r := range results {
		marker := "  "
		if r.Triggered {
			switch r.Severity {
			case "critical":
				marker = "! "
			case "warning":
				marker = "* "
			default:
				marker = "~ "
			}
		} else {
			marker = "  "
		}
		status := "PASS"
		if r.Triggered {
			status = "TRIGGERED"
		}
		fmt.Printf("  %s%s (%s): %s\n", marker, r.BiasName, r.Bias, status)
		fmt.Printf("      metric: %s = %.2f (threshold: %.2f)\n", r.Metric, r.Value, r.Threshold)
		fmt.Printf("      %s\n", r.Detail)
		if r.Conclusion != "" {
			fmt.Printf("      conclusion: %s\n", r.Conclusion)
		}
		fmt.Println()
	}
	fmt.Printf("[dream-bias] %s\n", report.Summary)
	return report
}

// auditConfirmationBias checks whether the metabolism loop's corroboration
// rate is suspiciously high — suggesting the LLM judge or query design is
// biased toward confirming rather than genuinely testing hypotheses.
//
// KB entity: ConfirmationBias (not yet declared — references the concept)
// Null model: random queries on a general-knowledge corpus should corroborate
// at ~30-50% (many topics are findable in Wikipedia). A rate significantly
// above this suggests confirmation bias in the pipeline.
func auditConfirmationBias(dir string) BiasAuditorResult {
	result := BiasAuditorResult{
		Bias:      "CognitiveBias",
		BiasName:  "Confirmation bias",
		Metric:    "corroboration_rate",
		Threshold: 0.75, // 75% corroboration rate is suspicious
	}

	logPath := filepath.Join(dir, ".metabolism-log.json")
	data, err := os.ReadFile(logPath)
	if err != nil {
		result.Detail = "no metabolism log found"
		return result
	}

	var log struct {
		Cycles []struct {
			Resolution string `json:"resolution"`
			Backend    string `json:"backend"`
			Papers     int    `json:"papers_found"`
		} `json:"cycles"`
	}
	if err := json.Unmarshal(data, &log); err != nil {
		result.Detail = fmt.Sprintf("error parsing log: %v", err)
		return result
	}

	// Count resolutions. Exclude no_signal from the denominator:
	// a cycle that found nothing isn't a genuine test of the hypothesis,
	// so it shouldn't dilute the corroboration rate.
	var corroborated, challenged, irrelevant, noSignal int
	for _, c := range log.Cycles {
		switch c.Resolution {
		case "corroborated":
			corroborated++
		case "challenged":
			challenged++
		case "irrelevant":
			irrelevant++
		case "no_signal":
			noSignal++
		}
	}

	withSignal := corroborated + challenged + irrelevant
	if withSignal == 0 {
		result.Detail = fmt.Sprintf("no cycles with signal (%d no_signal)", noSignal)
		return result
	}

	rate := float64(corroborated) / float64(withSignal)
	result.Value = rate

	result.Detail = fmt.Sprintf("%d/%d signal cycles corroborated, %d challenged, %d irrelevant (+ %d no_signal excluded)",
		corroborated, withSignal, challenged, irrelevant, noSignal)

	// Cross-reference with survivorship: if 0 challenges, the corroboration
	// rate is only measuring corroborated vs irrelevant, never vs challenged.
	crossRef := ""
	if challenged == 0 && irrelevant > 0 {
		dismissRate := float64(irrelevant) / float64(withSignal) * 100
		crossRef = fmt.Sprintf(" Note: zero challenges found — %.0f%% of signal was dismissed as "+
			"irrelevant rather than classified as a challenge (see survivorship bias auditor).", dismissRate)
	}

	if rate > result.Threshold {
		result.Triggered = true
		result.Severity = "warning"
		result.Conclusion = fmt.Sprintf("%.0f%% corroboration rate (among signal cycles) exceeds %.0f%% threshold. "+
			"Possible causes: (a) queries are too broad (anything matches), "+
			"(b) LLM resolution judge is too generous, "+
			"(c) the KB genuinely covers well-documented territory. "+
			"Test: run 5 random-topic queries and measure their corroboration rate as a baseline.%s",
			rate*100, result.Threshold*100, crossRef)
	} else {
		result.Conclusion = fmt.Sprintf("%.0f%% corroboration rate (among signal cycles) is within expected range.%s",
			rate*100, crossRef)
	}

	return result
}

// auditAnchoringBias checks whether early-ingested entities are
// disproportionately central (high claim count). If the first entities
// added to the KB accumulate more claims over time, the topology may be
// anchored to the seed corpus rather than reflecting domain importance.
//
// Measures: correlation between file creation order and entity degree.
func auditAnchoringBias(dir string) BiasAuditorResult {
	result := BiasAuditorResult{
		Bias:      "AnchoringBias",
		BiasName:  "Anchoring bias",
		Metric:    "degree_vs_age_correlation",
		Threshold: 0.5, // Spearman rank correlation > 0.5 is suspicious
	}

	// Get file creation order from git
	cmd := exec.Command("git", "log", "--format=%H", "--diff-filter=A", "--name-only", "--", "*.go")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		result.Detail = "cannot read git history"
		return result
	}

	// Parse git output: commit hash lines alternate with file paths
	fileOrder := map[string]int{} // filename -> creation rank (lower = older)
	rank := 0
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || len(line) == 40 { // skip blank lines and commit hashes
			continue
		}
		base := filepath.Base(line)
		if strings.HasSuffix(base, ".go") && !isInfraFile(base) && !strings.HasPrefix(line, "cmd/") {
			if _, exists := fileOrder[base]; !exists {
				fileOrder[base] = rank
				rank++
			}
		}
	}

	// Count entity degree (claims as subject or object) per file
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, goFileFilter, parser.ParseComments)
	if err != nil {
		result.Detail = "cannot parse Go files"
		return result
	}

	roleTypes := collectDreamRoleTypes(pkgs)

	// Count entities and claims per file
	type fileStat struct {
		entities int
		claims   int
		rank     int
	}
	fileStats := map[string]*fileStat{}

	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
			base := filepath.Base(fname)
			if isInfraFile(base) {
				continue
			}
			r, ok := fileOrder[base]
			if !ok {
				continue
			}
			fs := &fileStat{rank: r}
			fileStats[base] = fs

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
					if roleTypes[typeName] {
						fs.entities++
					} else if typeName != "Provenance" && typeName != "" {
						fs.claims++
					}
				}
			}
		}
	}

	// Compute Spearman rank correlation between file age rank and claim density.
	// Collect raw values and let spearmanRho handle all ranking (including ties).
	var ageVals, densityVals []float64
	// Iterate in deterministic key order to avoid map iteration non-determinism.
	fileNames := make([]string, 0, len(fileStats))
	for name := range fileStats {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)
	for _, name := range fileNames {
		fs := fileStats[name]
		if fs.entities == 0 {
			continue
		}
		ageVals = append(ageVals, float64(fs.rank))
		densityVals = append(densityVals, float64(fs.claims)/float64(fs.entities))
	}

	if len(ageVals) < 5 {
		result.Detail = fmt.Sprintf("only %d corpus files — too few for correlation analysis", len(ageVals))
		return result
	}

	rho := spearmanRho(ageVals, densityVals)
	result.Value = math.Abs(rho)

	result.Detail = fmt.Sprintf("Spearman rho = %.3f across %d corpus files (negative = older files have higher degree)",
		rho, len(ageVals))

	if math.Abs(rho) > result.Threshold {
		result.Triggered = true
		result.Severity = "info"
		if rho < 0 {
			result.Conclusion = "Older files have disproportionately more claims per entity. " +
				"The KB's graph topology may be anchored to its seed corpus. " +
				"Consider: are newer entities under-connected because they lack claims, " +
				"or because the seed entities are genuinely more important?"
		} else {
			result.Conclusion = "Newer files have disproportionately more claims per entity. " +
				"This is the opposite of anchoring — newer ingest may be over-claiming."
		}
	} else {
		result.Conclusion = "No significant correlation between file age and claim density"
	}

	return result
}

// auditClusteringIllusion checks whether topology clusters are artifacts
// of file organization rather than genuine conceptual groupings. If
// within-cluster entities share files more than cross-cluster entities,
// the structure reflects filing decisions, not domain relationships.
//
// Metric: Jaccard index between file co-occurrence and cluster co-occurrence.
func auditClusteringIllusion(dir string) BiasAuditorResult {
	result := BiasAuditorResult{
		Bias:      "HotHandFallacy", // closest KB entity to clustering illusion
		BiasName:  "Clustering illusion",
		Metric:    "file_cluster_overlap",
		Threshold: 0.7, // 70% overlap means clusters ≈ files
	}

	// Collect entities with file and cluster membership
	entities := collectTripEntities(dir)
	if len(entities) < 10 {
		result.Detail = fmt.Sprintf("only %d entities — too few for clustering analysis", len(entities))
		return result
	}

	// Count how many entity pairs share file vs share cluster
	type entityMeta struct {
		file    string
		cluster int
	}
	meta := map[string]entityMeta{}
	for _, e := range entities {
		meta[e.name] = entityMeta{file: e.file, cluster: e.cluster}
	}

	var sameFile, sameCluster, sameBoth, totalPairs int
	names := make([]string, 0, len(meta))
	for n := range meta {
		names = append(names, n)
	}
	sort.Strings(names)

	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			a, b := meta[names[i]], meta[names[j]]
			totalPairs++
			sf := a.file == b.file
			sc := a.cluster == b.cluster
			if sf {
				sameFile++
			}
			if sc {
				sameCluster++
			}
			if sf && sc {
				sameBoth++
			}
		}
	}

	if sameFile == 0 && sameCluster == 0 {
		result.Detail = "no file or cluster co-occurrence"
		return result
	}

	// Jaccard index: intersection / union
	union := sameFile + sameCluster - sameBoth
	var jaccard float64
	if union > 0 {
		jaccard = float64(sameBoth) / float64(union)
	}

	result.Value = jaccard
	result.Detail = fmt.Sprintf("Jaccard(file, cluster) = %.3f — %d pairs same-file, %d same-cluster, %d both, %d total",
		jaccard, sameFile, sameCluster, sameBoth, totalPairs)

	if jaccard > result.Threshold {
		result.Triggered = true
		result.Severity = "warning"
		result.Conclusion = fmt.Sprintf("%.0f%% overlap between file grouping and topology clusters. "+
			"Clusters may reflect file organization, not conceptual structure. "+
			"Test: move entities between files and re-run topology — if clusters change, they're file artifacts.",
			jaccard*100)
	} else {
		result.Conclusion = fmt.Sprintf("%.0f%% overlap — clusters reflect conceptual structure, not just filing", jaccard*100)
	}

	return result
}

// auditAvailabilityHeuristic checks whether the KB is over-indexed on
// sources that were available (Wikipedia ZIM) vs sources that are actually
// important for the domain. A healthy KB should have diverse provenance.
//
// Metric: Herfindahl-Hirschman Index (HHI) of provenance origins.
// HHI = sum of squared market shares. HHI > 0.25 = concentrated.
func auditAvailabilityHeuristic(dir string) BiasAuditorResult {
	result := BiasAuditorResult{
		Bias:      "AvailabilityHeuristic",
		BiasName:  "Availability heuristic",
		Metric:    "provenance_hhi",
		Threshold: 0.25, // HHI > 0.25 = moderately concentrated
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, goFileFilter, parser.ParseComments)
	if err != nil {
		result.Detail = "cannot parse Go files"
		return result
	}

	// Collect provenance origins and classify by source type
	sourceTypes := map[string]int{} // source type -> count
	total := 0

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
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
					if origin == "" {
						continue
					}

					// Classify by source type
					stype := classifyOrigin(origin)
					sourceTypes[stype]++
					total++
				}
			}
		}
	}

	if total == 0 {
		result.Detail = "no provenance records found"
		return result
	}

	// Compute HHI
	var hhi float64
	var breakdown []string
	for stype, count := range sourceTypes {
		share := float64(count) / float64(total)
		hhi += share * share
		breakdown = append(breakdown, fmt.Sprintf("%s: %d (%.0f%%)", stype, count, share*100))
	}
	sort.Strings(breakdown)

	result.Value = hhi
	result.Detail = fmt.Sprintf("HHI = %.3f across %d provenance records: %s",
		hhi, total, strings.Join(breakdown, ", "))

	if hhi > result.Threshold {
		result.Triggered = true
		result.Severity = "info"

		// Find the dominant source
		var dominant string
		var maxShare float64
		for stype, count := range sourceTypes {
			share := float64(count) / float64(total)
			if share > maxShare {
				maxShare = share
				dominant = stype
			}
		}
		result.Conclusion = fmt.Sprintf("Provenance is concentrated: %s accounts for %.0f%%. "+
			"The KB may reflect source availability rather than domain importance. "+
			"Consider adding claims from under-represented source types.",
			dominant, maxShare*100)
	} else {
		result.Conclusion = "Provenance sources are reasonably diverse"
	}

	return result
}

// auditSurvivorshipBias checks whether the metabolism loop's rejection
// pipeline is systematically filtering out valid contrarian perspectives.
// Sources resolved as "irrelevant" might contain legitimate challenges
// that the pipeline's quality criteria can't recognize.
//
// Metric: irrelevant-to-challenged ratio. If the pipeline finds lots of
// sources but almost none are classified as challenges, it may be
// filtering out valid dissent.
func auditSurvivorshipBias(dir string) BiasAuditorResult {
	result := BiasAuditorResult{
		Bias:      "CognitiveBias", // general reference
		BiasName:  "Survivorship bias",
		Metric:    "irrelevant_to_challenged_ratio",
		Threshold: 5.0, // more than 5:1 irrelevant:challenged is suspicious
	}

	logPath := filepath.Join(dir, ".metabolism-log.json")
	data, err := os.ReadFile(logPath)
	if err != nil {
		result.Detail = "no metabolism log found"
		return result
	}

	var log struct {
		Cycles []struct {
			Resolution string `json:"resolution"`
			Papers     int    `json:"papers_found"`
		} `json:"cycles"`
	}
	if err := json.Unmarshal(data, &log); err != nil {
		result.Detail = fmt.Sprintf("error parsing log: %v", err)
		return result
	}

	var irrelevant, challenged, resolved int
	for _, c := range log.Cycles {
		switch c.Resolution {
		case "irrelevant":
			irrelevant++
			resolved++
		case "challenged":
			challenged++
			resolved++
		case "corroborated", "no_signal":
			resolved++
		}
	}

	if resolved == 0 {
		result.Detail = "no resolved cycles"
		return result
	}

	var ratio float64
	var ratioStr string
	if challenged > 0 {
		ratio = float64(irrelevant) / float64(challenged)
		ratioStr = fmt.Sprintf("%.1f:1", ratio)
	} else if irrelevant > 0 {
		ratio = float64(irrelevant + 1) // penalize zero-challenge case
		ratioStr = fmt.Sprintf("%d:0 (no challenges ever found)", irrelevant)
	} else {
		ratioStr = "n/a (no irrelevant or challenged cycles)"
	}

	result.Value = ratio
	result.Detail = fmt.Sprintf("%d irrelevant, %d challenged out of %d resolved (ratio: %s)",
		irrelevant, challenged, resolved, ratioStr)

	if challenged == 0 && irrelevant > 3 {
		result.Triggered = true
		result.Severity = "warning"
		result.Conclusion = fmt.Sprintf("Zero challenges found despite %d irrelevant resolutions. "+
			"The metabolism pipeline may be systematically classifying challenges as irrelevant. "+
			"Review: are any 'irrelevant' sources actually presenting contrarian positions that "+
			"the resolution judge dismissed?", irrelevant)
	} else if ratio > result.Threshold {
		result.Triggered = true
		result.Severity = "info"
		result.Conclusion = fmt.Sprintf("%.0f:1 irrelevant-to-challenged ratio suggests the pipeline "+
			"may filter out valid dissent. Not necessarily wrong — the domain may genuinely "+
			"have few challenges — but worth auditing a sample of 'irrelevant' resolutions.",
			ratio)
	} else {
		result.Conclusion = "Challenge/irrelevant ratio is within expected range"
	}

	return result
}

// auditFramingEffect checks whether entity Briefs use evaluative language
// that frames hypotheses rather than describing them. Loaded terms like
// "groundbreaking", "controversial", "seminal", "flawed" predispose LLM
// judges (and human readers) toward or against an entity before examining
// its claims.
//
// Metric: fraction of Briefs containing evaluative terms.
// KB entity: CognitiveBias (framing effect is a documented bias)
func auditFramingEffect(dir string) BiasAuditorResult {
	result := BiasAuditorResult{
		Bias:      "CognitiveBias",
		BiasName:  "Framing effect",
		Metric:    "evaluative_brief_fraction",
		Threshold: 0.15, // more than 15% of Briefs using loaded language
	}

	// Evaluative terms that frame rather than describe.
	// Split into positive and negative to report framing direction.
	positiveFraming := []string{
		"groundbreaking", "seminal", "landmark", "revolutionary",
		"brilliant", "influential", "pioneering", "definitive",
		"canonical", "masterful", "profound", "celebrated",
	}
	negativeFraming := []string{
		"controversial", "flawed", "debunked", "discredited",
		"pseudoscientific", "misleading", "simplistic",
		"outdated", "questionable",
	}

	// Technical terms that look evaluative but are descriptive in context.
	// Ambiguous words (naive, rejected, failed) are omitted from the word
	// lists entirely rather than excluded — too context-dependent.
	technicalExclusions := map[string][]string{
		"controversial": {"controversial claim that", "controversial in"},
		"simplistic":    {"simplistic model", "simplistic view"},
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, goFileFilter, parser.ParseComments)
	if err != nil {
		result.Detail = "cannot parse Go files"
		return result
	}

	roleTypes := collectDreamRoleTypes(pkgs)

	totalBriefs := 0
	positiveCount := 0
	negativeCount := 0
	var examples []string

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

					brief := extractEntityBrief(cl)
					if brief == "" {
						continue
					}

					totalBriefs++
					lower := strings.ToLower(brief)
					matched := false
					for _, term := range positiveFraming {
						if containsWord(lower, term, technicalExclusions) {
							positiveCount++
							matched = true
							if len(examples) < 3 {
								examples = append(examples, fmt.Sprintf("%s: +%q", vs.Names[0].Name, term))
							}
							break
						}
					}
					if !matched {
						for _, term := range negativeFraming {
							if containsWord(lower, term, technicalExclusions) {
								negativeCount++
								if len(examples) < 3 {
									examples = append(examples, fmt.Sprintf("%s: -%q", vs.Names[0].Name, term))
								}
								break
							}
						}
					}
				}
			}
		}
	}

	if totalBriefs == 0 {
		result.Detail = "no Briefs found"
		return result
	}

	framedBriefs := positiveCount + negativeCount
	fraction := float64(framedBriefs) / float64(totalBriefs)
	result.Value = fraction

	exampleStr := ""
	if len(examples) > 0 {
		exampleStr = " — e.g., " + strings.Join(examples, "; ")
	}

	direction := "neutral"
	if positiveCount > negativeCount*2 {
		direction = "skews positive (pro-framing)"
	} else if negativeCount > positiveCount*2 {
		direction = "skews negative (anti-framing)"
	}

	result.Detail = fmt.Sprintf("%d/%d Briefs (%.0f%%) contain evaluative language "+
		"(%d positive, %d negative — %s)%s",
		framedBriefs, totalBriefs, fraction*100,
		positiveCount, negativeCount, direction, exampleStr)

	if fraction > result.Threshold {
		result.Triggered = true
		result.Severity = "info"
		result.Conclusion = fmt.Sprintf("%.0f%% of Briefs use evaluative framing (%s). "+
			"These terms may predispose the LLM contradiction checker toward or against "+
			"entities. Consider: does the Brief describe what the entity IS, or evaluate "+
			"how good/bad it is?", fraction*100, direction)
	} else {
		result.Conclusion = fmt.Sprintf("%.0f%% evaluative framing (%s) — within acceptable range",
			fraction*100, direction)
	}

	return result
}

// auditDunningKruger checks whether structurally simple entities appear
// disproportionately "healthy" in topology analysis. Simple entities have
// fewer attack surfaces — they can't be flagged as single-source if they
// only have one claim, and they can't have contradictions if they have no
// cross-references. The topology may rate them as well-supported when
// they're actually just under-examined. This is the Dunning-Kruger analog:
// low complexity → apparent competence.
//
// Metric: fraction of low-complexity entities (≤2 claim references) that
// have zero topology vulnerabilities. If near 100%, simple entities are
// systematically escaping scrutiny.
//
// topoReport may be nil; if so, topology is run fresh.
func auditDunningKruger(dir string, topoReport *TopologyReport) BiasAuditorResult {
	result := BiasAuditorResult{
		Bias:      "DunningKrugerEffect",
		BiasName:  "Dunning-Kruger effect",
		Metric:    "low_complexity_zero_vuln_rate",
		Threshold: 0.90, // >90% of simple entities have zero vulns
	}

	// Get topology vulnerabilities per entity
	var report TopologyReport
	if topoReport != nil {
		report = *topoReport
	} else {
		_, r, err := runTopology(dir)
		if err != nil {
			result.Detail = fmt.Sprintf("topology error: %v", err)
			return result
		}
		report = r
	}

	vulnCount := map[string]int{}
	for _, v := range report.Vulnerabilities {
		vulnCount[v.Entity]++
	}

	// Get claim reference counts per entity from AST
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, goFileFilter, parser.ParseComments)
	if err != nil {
		result.Detail = "cannot parse Go files"
		return result
	}

	roleTypes := collectDreamRoleTypes(pkgs)

	// Collect entity names
	entityNames := map[string]bool{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
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
					if roleTypes[compositeTypeName(cl)] {
						entityNames[vs.Names[0].Name] = true
					}
				}
			}
		}
	}

	// Count claims referencing each entity (as subject or object)
	claimRefs := map[string]int{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
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
					if roleTypes[typeName] || typeName == "Provenance" || typeName == "" {
						continue
					}
					for _, elt := range cl.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := kv.Key.(*ast.Ident)
						if !ok {
							continue
						}
						if key.Name == "Subject" || key.Name == "Object" {
							ref := exprIdent(kv.Value)
							if ref != "" && entityNames[ref] {
								claimRefs[ref]++
							}
						}
					}
				}
			}
		}
	}

	// Split entities into low-complexity (1-2 refs) and high-complexity (>2 refs).
	// Exclude entities with 0 refs — these are entities whose references
	// exprIdent can't resolve, not truly isolated nodes. Including them
	// inflates the low-complexity bucket with unmeasurable entities.
	var lowTotal, lowZeroVuln, highTotal, highZeroVuln, unresolvedRefs int
	for name := range entityNames {
		refs := claimRefs[name]
		if refs == 0 {
			unresolvedRefs++
			continue
		}
		vulns := vulnCount[name]
		if refs <= 2 {
			lowTotal++
			if vulns == 0 {
				lowZeroVuln++
			}
		} else {
			highTotal++
			if vulns == 0 {
				highZeroVuln++
			}
		}
	}

	if lowTotal < 5 {
		result.Detail = fmt.Sprintf("only %d low-complexity entities — too few for analysis", lowTotal)
		return result
	}

	lowRate := float64(lowZeroVuln) / float64(lowTotal)
	result.Value = lowRate

	var highRate float64
	if highTotal > 0 {
		highRate = float64(highZeroVuln) / float64(highTotal)
	}

	excludedNote := ""
	if unresolvedRefs > 0 {
		excludedNote = fmt.Sprintf(" (%d entities excluded: unresolved refs)", unresolvedRefs)
	}
	result.Detail = fmt.Sprintf("low-complexity (1-2 refs): %d/%d (%.0f%%) zero vulns; "+
		"high-complexity (>2 refs): %d/%d (%.0f%%) zero vulns%s",
		lowZeroVuln, lowTotal, lowRate*100,
		highZeroVuln, highTotal, highRate*100, excludedNote)

	gap := lowRate - highRate
	if lowRate > result.Threshold {
		result.Triggered = true
		if gap > 0.3 {
			result.Severity = "warning"
		} else {
			result.Severity = "info"
		}
		result.Conclusion = fmt.Sprintf("%.0f%% of low-complexity entities have zero vulnerabilities "+
			"(vs %.0f%% of high-complexity). Simple entities may appear healthy because "+
			"they're under-examined, not because they're well-supported. The topology's "+
			"vulnerability detectors can't flag what they can't see.",
			lowRate*100, highRate*100)
	} else {
		result.Conclusion = fmt.Sprintf("%.0f%% of low-complexity entities have zero vulns "+
			"(vs %.0f%% high-complexity, gap: %.0f points). Below threshold but the gap "+
			"is structural: topology detectors need connections to work, so simple entities "+
			"escape scrutiny by default.",
			lowRate*100, highRate*100, gap*100)
	}

	return result
}

// auditBaseRateNeglect checks whether the KB's predicate distribution is
// so skewed that common predicates (BelongsTo, InfluencedBy) drown out
// rare but high-signal predicates (Disputes, Refutes). When one predicate
// type dominates, pattern-matching (LLM or human) may treat all
// connections as equally likely, ignoring that a Disputes edge is far
// more informative than a BelongsTo edge.
//
// Metric: entropy of predicate distribution. Low entropy = concentrated.
func auditBaseRateNeglect(dir string) BiasAuditorResult {
	result := BiasAuditorResult{
		Bias:      "CognitiveBias",
		BiasName:  "Base rate neglect",
		Metric:    "predicate_entropy",
		Threshold: 3.0, // Shannon entropy below 3.0 bits = concentrated (max ~5 for this KB)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, goFileFilter, parser.ParseComments)
	if err != nil {
		result.Detail = "cannot parse Go files"
		return result
	}

	roleTypes := collectDreamRoleTypes(pkgs)

	// Count predicate types (claim composite types that aren't roles or provenance)
	predCounts := map[string]int{}
	total := 0

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
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
					if typeName == "" || typeName == "Provenance" || roleTypes[typeName] {
						continue
					}
					predCounts[typeName]++
					total++
				}
			}
		}
	}

	if total == 0 || len(predCounts) < 2 {
		result.Detail = "not enough predicate types for analysis"
		return result
	}

	// Compute Shannon entropy
	var entropy float64
	for _, count := range predCounts {
		p := float64(count) / float64(total)
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	// Max entropy for this many types
	maxEntropy := math.Log2(float64(len(predCounts)))

	result.Value = entropy

	// Build distribution summary
	type predInfo struct {
		name  string
		count int
	}
	var preds []predInfo
	for name, count := range predCounts {
		preds = append(preds, predInfo{name, count})
	}
	sort.Slice(preds, func(i, j int) bool { return preds[i].count > preds[j].count })

	var top3 []string
	for i, p := range preds {
		if i >= 3 {
			break
		}
		top3 = append(top3, fmt.Sprintf("%s: %d (%.0f%%)", p.name, p.count, float64(p.count)/float64(total)*100))
	}

	result.Detail = fmt.Sprintf("entropy = %.2f bits (max %.2f for %d types, %d claims) — top: %s",
		entropy, maxEntropy, len(predCounts), total, strings.Join(top3, ", "))

	if entropy < result.Threshold {
		result.Triggered = true
		result.Severity = "info"

		// Find the dominant predicate
		dominant := preds[0]
		result.Conclusion = fmt.Sprintf("Predicate distribution is concentrated (%.1f bits vs %.1f max). "+
			"%s accounts for %.0f%% of claims. Rare predicates like Disputes carry more "+
			"epistemic weight but may be undervalued in automated analysis because they "+
			"occur less frequently. Consider: is the LLM contradiction checker treating "+
			"a Disputes edge as seriously as a BelongsTo edge?",
			entropy, maxEntropy, dominant.name, float64(dominant.count)/float64(total)*100)
	} else {
		result.Conclusion = fmt.Sprintf("Predicate distribution has reasonable entropy (%.1f bits) — "+
			"diverse enough that base rates aren't drowning signal", entropy)
	}

	return result
}

// --- helpers ---

// containsWord checks if text contains the evaluative term, excluding
// known technical uses where the term is descriptive, not evaluative.
func containsWord(text, term string, exclusions map[string][]string) bool {
	if !strings.Contains(text, term) {
		return false
	}
	// Check if this is a technical use
	if excl, ok := exclusions[term]; ok {
		for _, pattern := range excl {
			if strings.Contains(text, pattern) {
				return false
			}
		}
	}
	return true
}

// classifyOrigin categorizes a provenance origin string into a source type.
func classifyOrigin(origin string) string {
	lower := strings.ToLower(origin)
	switch {
	case strings.Contains(lower, "wikipedia") || strings.Contains(lower, "zim"):
		return "wikipedia"
	case strings.Contains(lower, "arxiv"):
		return "arxiv"
	case strings.Contains(lower, "doi.org") || strings.Contains(lower, "doi:"):
		return "doi"
	case strings.Contains(lower, "isbn"):
		return "book"
	case strings.Contains(lower, "manual") || strings.Contains(lower, "direct"):
		return "manual"
	default:
		return "other"
	}
}

// spearmanRho computes Spearman's rank correlation coefficient.
// Assumes inputs are already rank values (or can be used as-is for ranking).
func spearmanRho(x, y []float64) float64 {
	n := len(x)
	if n != len(y) || n < 3 {
		return 0
	}

	// Rank both arrays
	xRanks := assignRanks(x)
	yRanks := assignRanks(y)

	// Compute using Pearson's on ranks
	var sumD2 float64
	for i := range xRanks {
		d := xRanks[i] - yRanks[i]
		sumD2 += d * d
	}

	nf := float64(n)
	return 1 - (6*sumD2)/(nf*(nf*nf-1))
}

// assignRanks returns rank values (1-based, averaged for ties).
func assignRanks(vals []float64) []float64 {
	n := len(vals)
	type indexed struct {
		val float64
		idx int
	}
	items := make([]indexed, n)
	for i, v := range vals {
		items[i] = indexed{val: v, idx: i}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].val < items[j].val })

	ranks := make([]float64, n)
	i := 0
	for i < n {
		j := i
		for j < n && items[j].val == items[i].val {
			j++
		}
		avgRank := float64(i+j+1) / 2.0 // average rank for ties
		for k := i; k < j; k++ {
			ranks[items[k].idx] = avgRank
		}
		i = j
	}
	return ranks
}
