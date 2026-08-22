package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// reifySourceRe matches the provenance header runReify writes into
// predictions.go, capturing the cycle count that produced it.
var reifySourceRe = regexp.MustCompile(`// Source: \.metabolism-log\.json \((\d+) cycles`)

// existingReifyCycleCount reads the cycle count a previously-generated
// predictions.go records in its own header. Returns -1 when the file is
// absent or the header is unreadable, which the caller treats as "no
// previous state to compare against".
//
// The generated file carrying its own provenance is what makes the shrink
// guard below possible without any side channel: the file states how many
// cycles produced it, so a reify can compare its input against the output
// it is about to overwrite.
func existingReifyCycleCount(outPath string) int {
	data, err := os.ReadFile(outPath)
	if err != nil {
		return -1
	}
	m := reifySourceRe.FindSubmatch(data)
	if m == nil {
		return -1
	}
	n, err := strconv.Atoi(string(m[1]))
	if err != nil {
		return -1
	}
	return n
}

// runReify reads the metabolism log and generates a Go corpus file that
// encodes the metabolism loop's predictions as first-class KB claims using
// the Predicts/ResolvedAs prediction schema.
//
// The meta-hypothesis: structural fragility (single-source, uncontested)
// predicts that external evidence exists which could strengthen or challenge
// a hypothesis. Only a search that reaches a substantive resolution
// (corroborated or challenged) is reified as an Event — see
// isSubstantiveResolution. A barren search stays in .metabolism-log.json,
// which already records every ask, rather than being promoted into the
// corpus as if it were a finding.
//
// The KB records what it predicted, what it found, and whether its
// predictions resolved — making epistemic performance a first-class
// queryable property of the corpus rather than external bookkeeping.
//
// hypothesisRecord aggregates metabolism cycles for a single hypothesis.
type hypothesisRecord struct {
	name       string
	prediction string
	bestRes    string // best resolution across all cycles
	cycles     int    // total cycles for this hypothesis
	withSignal int    // cycles that found papers
	papers     []PaperSummary
	backends   map[string]bool
	resCounts  map[string]int // resolution → count (for history comments)
	evidence   string         // first non-empty Evidence from any cycle (for KB-internal resolvers)

	// lastResolvedAt is the newest Cycle.ResolvedAt across this hypothesis's
	// cycles, ISO 8601. It dates the resolution-history comment. Stamping
	// time.Now() there instead made the generated file a function of when the
	// reifier ran rather than of the log, so every cycle produced a diff
	// consisting only of the date it had just written — 65 of 67 metabolism
	// commits between 2026-07-23 and 2026-08-22 were exactly that, and they
	// defeated the len(changed)==0 quiet-cycle guard in cmd/metabolize.
	lastResolvedAt string
}

// kbInternalConfig describes one KB-internal prediction-type bucket. Each
// bucket emits its own meta-Hypothesis + per-claim Event/Predicts/ResolvedAs
// trio. Adding a new resolver = adding one entry here.
type kbInternalConfig struct {
	predictionType string // matches Cycle.PredictionType
	metaVar        string // Go var name for the meta-hypothesis
	metaID         string // entity ID
	metaName       string // human-readable name
	metaBrief      string // entity Brief
	sectionHeader  string // human-readable section header in the comment block
	varPrefix      string // prefix for per-claim var names; combined with sanitize(hypName)
	eventIDPrefix  string // entity ID prefix for the per-cycle Event
	eventNameTmpl  string // %s = hypName
	eventBriefTmpl string // %s = hypName
	predictsSuffix string // appended to varBase for the Predicts var name
}

// kbInternalConfigs lists every KB-internal resolver. Order is the section
// order in predictions.go. Adding a new resolver: append a config and ensure
// the resolver writes Cycle{PredictionType: predictionType, Hypothesis: ...,
// Evidence: ..., Resolution: confirmed|refuted}.
var kbInternalConfigs = []kbInternalConfig{
	{
		predictionType: "trip_lint_durability",
		metaVar:        "TripPromotionSurvivesLint",
		metaID:         "trip-promotion-survives-lint",
		metaName:       "Trip-promoted claims survive cmd/lint",
		metaBrief:      "Speculative cross-cluster connections promoted by the trip cycle pass cmd/lint's deterministic rules (value-conflict, orphan-report, provenance-split, brief-check, naming-oracle, contested-concept). Self-resolving: no external sensor, no LLM oracle — the substrate's own rules are the oracle.",
		sectionHeader:  "Meta-hypothesis: trip-promoted claims survive cmd/lint.",
		varPrefix:      "TripLint",
		eventIDPrefix:  "lint-durability-check",
		eventNameTmpl:  "Lint durability check for %s",
		eventBriefTmpl: "cmd/lint run observing whether %s was flagged by any deterministic rule.",
		predictsSuffix: "Survival",
	},
	{
		predictionType: "trip_functional_durability",
		metaVar:        "TripPromotionRespectsFunctionalUniqueness",
		metaID:         "trip-promotion-respects-functional-uniqueness",
		metaName:       "Trip-promoted claims respect //winze:functional uniqueness",
		metaBrief:      "Speculative cross-cluster connections promoted by the trip cycle do not violate functional-predicate uniqueness — for every (Subject, Predicate) where Predicate is //winze:functional, there is at most one Object. Deterministic resolver, no LLM, no API cost.",
		sectionHeader:  "Meta-hypothesis: trip-promoted claims respect functional-predicate uniqueness.",
		varPrefix:      "TripFunctional",
		eventIDPrefix:  "functional-durability-check",
		eventNameTmpl:  "Functional durability check for %s",
		eventBriefTmpl: "//winze:functional pragma check observing whether %s creates a Subject-with-multiple-Objects collision.",
		predictsSuffix: "FunctionalUniqueness",
	},
	{
		predictionType: "trip_llm_durability",
		metaVar:        "TripPromotionPassesContradictionCheck",
		metaID:         "trip-promotion-passes-contradiction-check",
		metaName:       "Trip-promoted claims pass LLM contradiction check",
		metaBrief:      "Speculative cross-cluster connections promoted by the trip cycle do not contradict existing claims in the topology neighborhood, as judged by an LLM with predicate-semantics guidance. Oracle quality is bounded by prompt fidelity to predicates.go.",
		sectionHeader:  "Meta-hypothesis: trip-promoted claims pass LLM contradiction check.",
		varPrefix:      "TripLLM",
		eventIDPrefix:  "llm-contradiction-check",
		eventNameTmpl:  "LLM contradiction check for %s",
		eventBriefTmpl: "LLM neighborhood contradiction check observing whether %s contradicts existing claims.",
		predictsSuffix: "Consistency",
	},
	{
		predictionType: "trip_promotion_attempt",
		metaVar:        "TripPromotionPassesBuildGate",
		metaID:         "trip-promotion-passes-build-gate",
		metaName:       "Trip-promoted claims pass go build/vet/lint",
		metaBrief:      "Speculative cross-cluster connections promoted by the trip cycle compose with the existing typed corpus — entity references resolve, predicate slot types match, and the file passes go build/vet/lint. The compiler is the oracle.",
		sectionHeader:  "Meta-hypothesis: trip-promoted claims pass go build/vet/lint.",
		varPrefix:      "TripBuild",
		eventIDPrefix:  "build-validation",
		eventNameTmpl:  "Build validation for %s",
		eventBriefTmpl: "go build/vet/lint pipeline observing whether %s is structurally well-formed (entities exist, predicate slot types match).",
		predictsSuffix: "Buildability",
	},
}

func kbConfigFor(predictionType string) *kbInternalConfig {
	for i := range kbInternalConfigs {
		if kbInternalConfigs[i].predictionType == predictionType {
			return &kbInternalConfigs[i]
		}
	}
	return nil
}

func runReify(dir string) error {
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	if len(mlog.Cycles) == 0 {
		fmt.Fprintln(os.Stderr, "metabolism: no cycles logged — nothing to reify")
		return nil
	}

	// Split cycles by prediction type so each gets its own section with
	// its own meta-hypothesis. Empty prediction_type (legacy) is treated
	// as "structural_fragility" — the original sensor-based vocabulary.
	// Sensor records get their own emit (uses backends/papers); each
	// KB-internal type listed in kbInternalConfigs gets its own bucket
	// and shares the generic emit loop.
	sensorRecords := map[string]*hypothesisRecord{}
	var sensorOrder []string
	kbRecords := map[string]map[string]*hypothesisRecord{}
	kbOrder := map[string][]string{}
	for _, cfg := range kbInternalConfigs {
		kbRecords[cfg.predictionType] = map[string]*hypothesisRecord{}
		kbOrder[cfg.predictionType] = nil
	}

	for _, c := range mlog.Cycles {
		pt := c.PredictionType
		if pt == "" {
			pt = "structural_fragility"
		}
		var records map[string]*hypothesisRecord
		isKB := false
		if cfg := kbConfigFor(pt); cfg != nil {
			records = kbRecords[pt]
			isKB = true
		} else {
			records = sensorRecords
		}

		r, ok := records[c.Hypothesis]
		if !ok {
			r = &hypothesisRecord{
				name:       c.Hypothesis,
				prediction: c.Prediction,
				backends:   map[string]bool{},
				resCounts:  map[string]int{},
			}
			records[c.Hypothesis] = r
			if isKB {
				kbOrder[pt] = append(kbOrder[pt], c.Hypothesis)
			} else {
				sensorOrder = append(sensorOrder, c.Hypothesis)
			}
		}
		r.cycles++
		if c.Resolution != "" {
			r.resCounts[c.Resolution]++
		}
		// ISO 8601 sorts lexically, so a string compare is the max.
		if c.ResolvedAt > r.lastResolvedAt {
			r.lastResolvedAt = c.ResolvedAt
		}
		be := c.Backend
		if be == "" {
			be = "arxiv"
		}
		r.backends[be] = true
		if c.PapersFound > 0 {
			r.withSignal++
			for _, p := range c.Papers {
				found := false
				for _, existing := range r.papers {
					if existing.ID == p.ID {
						found = true
						break
					}
				}
				if !found {
					r.papers = append(r.papers, p)
				}
			}
		}
		// Carry Evidence field forward for KB-internal resolvers; it's
		// their only per-cycle evidence.
		if c.Evidence != "" && r.evidence == "" {
			r.evidence = c.Evidence
		}
		r.bestRes = betterResolution(r.bestRes, c.Resolution)
	}

	// Preserve legacy variable names so the rest of the function reads
	// as the sensor-section emitter.
	records := sensorRecords
	order := sensorOrder

	// Count stats
	totalCycles := len(mlog.Cycles)
	uniqueHyps := len(order)
	// A hypothesis counts as resolved only when it reached a substantive
	// verdict — see isSubstantiveResolution. Everything the sensor calls
	// irrelevant or no_signal is activity, not a finding, and is excluded
	// both here and from the emit loop below.
	resolved := 0
	for _, r := range records {
		if isSubstantiveResolution(r.bestRes) {
			resolved++
		}
	}

	// Find date range
	earliest := mlog.Cycles[0].Timestamp
	latest := mlog.Cycles[0].Timestamp
	for _, c := range mlog.Cycles[1:] {
		if c.Timestamp.Before(earliest) {
			earliest = c.Timestamp
		}
		if c.Timestamp.After(latest) {
			latest = c.Timestamp
		}
	}

	// Every date written into the generated file comes from the log, never from
	// the clock. predictions.go is a pure function of .metabolism-log.json, so
	// re-running the reifier on an unchanged log must produce an unchanged file
	// — that is what lets the quiet-cycle guard in cmd/metabolize see "nothing
	// happened" and skip the commit. TestReifyIsDeterministic holds the line.
	newestCycle := latest.Format("2006-01-02")
	outPath := filepath.Join(dir, "predictions.go")

	// Shrink guard. predictions.go is a pure function of the log — the whole
	// file is rebuilt and overwritten, nothing on disk is merged forward. So a
	// log with fewer cycles than the one that produced the current file
	// silently deletes the difference.
	//
	// That is not hypothetical: on 2026-08-05 an unattended cycle regenerated a
	// 376-cycle predictions.go from a 44-cycle log, dropping 3019 lines to 982
	// and taking 375 var declarations of resolution history with it. The corpus
	// had moved into corpus/ and the 4.2 MB log stayed behind at the repo root,
	// so reify read a log that had lost nearly all its history. The only guard
	// then was len(mlog.Cycles) == 0, which an emptier-but-nonempty log walks
	// straight past.
	//
	// A test existed that would have caught the shrinkage and did — it just ran
	// outside the write gate, so nothing acted on it. This check is inside the
	// path that does the writing, which is the whole difference.
	//
	// Set WINZE_REIFY_ALLOW_SHRINK=1 when a smaller log is deliberate (a
	// genuine prune). Fails closed otherwise: refusing to write costs a re-run,
	// writing costs history that only git has.
	if prev := existingReifyCycleCount(outPath); prev > totalCycles && os.Getenv("WINZE_REIFY_ALLOW_SHRINK") == "" {
		fmt.Fprintf(os.Stderr,
			"[reify] REFUSING to write %s: it was generated from %d cycles but this log has only %d.\n",
			filepath.Base(outPath), prev, totalCycles)
		fmt.Fprintf(os.Stderr,
			"[reify] Regenerating would delete the difference. Check that %s is the right log\n",
			filepath.Join(dir, ".metabolism-log.json"))
		fmt.Fprintf(os.Stderr,
			"[reify] (a moved corpus can strand it), or set WINZE_REIFY_ALLOW_SHRINK=1 if the prune is deliberate.\n")
		return fmt.Errorf("refusing to shrink %s", outPath)
	}

	var b strings.Builder

	// Header
	fmt.Fprintf(&b, "package winze\n\n")
	fmt.Fprintf(&b, "// Prediction reification: metabolism loop predictions as first-class KB claims.\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// Generated by: go run ./cmd/metabolism --reify .\n")
	fmt.Fprintf(&b, "// Source: .metabolism-log.json (%d cycles, %d hypotheses, %s to %s)\n",
		totalCycles, uniqueHyps, earliest.Format("2006-01-02"), latest.Format("2006-01-02"))
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// The meta-hypothesis: structural fragility (single-source, uncontested)\n")
	fmt.Fprintf(&b, "// predicts that external evidence exists which could strengthen or\n")
	fmt.Fprintf(&b, "// challenge the hypothesis. Only a search that reaches a substantive\n")
	fmt.Fprintf(&b, "// resolution (corroborated or challenged) is reified as an Event here —\n")
	fmt.Fprintf(&b, "// a search that came back irrelevant or with no signal is activity, not\n")
	fmt.Fprintf(&b, "// a finding, and stays in .metabolism-log.json rather than the corpus.\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// This is the first use of the prediction schema (Predicts, ResolvedAs)\n")
	fmt.Fprintf(&b, "// defined in predicates.go. The forcing function: the metabolism loop\n")
	fmt.Fprintf(&b, "// itself generates falsifiable predictions about the KB's own gaps.\n")

	// Provenance
	fmt.Fprintf(&b, "\nvar metabolismPredictionSource = Provenance{\n")
	fmt.Fprintf(&b, "\tOrigin:     \"winze metabolism log (.metabolism-log.json)\",\n")
	fmt.Fprintf(&b, "\tIngestedAt: %q,\n", newestCycle)
	fmt.Fprintf(&b, "\tIngestedBy: \"winze metabolism --reify\",\n")
	fmt.Fprintf(&b, "\tQuote:      \"%d cycles logged from %s to %s across %d hypotheses. %d resolved.\",\n",
		totalCycles, earliest.Format("2006-01-02"), latest.Format("2006-01-02"), uniqueHyps, resolved)
	fmt.Fprintf(&b, "}\n")

	// Meta-hypothesis entity
	fmt.Fprintf(&b, "\n// ---------------------------------------------------------------------------\n")
	fmt.Fprintf(&b, "// Meta-hypothesis: the metabolism loop's core testable claim.\n")
	fmt.Fprintf(&b, "// Topology-detected structural fragility predicts curation gaps.\n")
	fmt.Fprintf(&b, "// ---------------------------------------------------------------------------\n\n")

	fmt.Fprintf(&b, "var StructuralFragilityPredictsCurationGaps = Hypothesis{&Entity{\n")
	fmt.Fprintf(&b, "\tID:    \"structural-fragility-predicts-curation-gaps\",\n")
	fmt.Fprintf(&b, "\tName:  \"Structural fragility predicts curation gaps\",\n")
	fmt.Fprintf(&b, "\tKind:  \"hypothesis\",\n")
	fmt.Fprintf(&b, "\tBrief: \"Hypotheses that are single-source and/or uncontested in the KB are more likely to have findable external evidence that could strengthen or challenge them. Tested by the metabolism loop.\",\n")
	fmt.Fprintf(&b, "}}\n")

	// Per-hypothesis events, predictions, and resolutions. Only hypotheses
	// that reached a substantive verdict are reified — see
	// isSubstantiveResolution and the doc comment on hypothesisRecord. A
	// hypothesis the loop asked about repeatedly and never got past
	// irrelevant/no_signal is omitted here; it is not lost, it is still every
	// cycle in .metabolism-log.json, which is the record of activity. This
	// file is the record of findings.
	omitted := 0
	for _, hypName := range order {
		r := records[hypName]
		if !isSubstantiveResolution(r.bestRes) {
			omitted++
			continue
		}
		baseName := camelToWords(hypName)
		varBase := strings.TrimSuffix(hypName, "Thesis")
		varBase = strings.TrimSuffix(varBase, "Argument")
		varBase = strings.TrimSuffix(varBase, "Framing")
		// sanitizeIdent strips any punctuation (e.g. the colon in a
		// "goal:Foo" LearningGoal hypName) so it can't leak into a Go
		// identifier. No-op for the CamelCase thesis names that predate
		// learning goals; the KB-internal loop below already does this.
		// Exported: varBase is the leading segment of the SearchOutcome and
		// SearchResolution var names, and a LearningGoal hypName ("goal:Foo")
		// sanitizes to a lowercase-leading identifier. Unexported there means
		// the claim compiles into the file but is unreachable from outside
		// package winze — present in the corpus, absent from the KB.
		varBase = exportedIdent(sanitizeIdent(varBase))
		entityID := camelToKebab(hypName)

		// Backends used (sorted for deterministic emit — map range order
		// would otherwise bounce predictions.go on every reify run).
		var backends []string
		for be := range r.backends {
			backends = append(backends, be)
		}
		sort.Strings(backends)

		// Paper summary for Brief
		briefPapers := ""
		if len(r.papers) > 0 {
			titles := make([]string, 0, 3)
			for i, p := range r.papers {
				if i >= 3 {
					break
				}
				titles = append(titles, p.Title)
			}
			briefPapers = fmt.Sprintf(" Found: %s", strings.Join(titles, "; "))
			if len(r.papers) > 3 {
				briefPapers += fmt.Sprintf(" (+%d more)", len(r.papers)-3)
			}
			briefPapers += "."
		}

		resLabel := "pending"
		if r.bestRes != "" {
			resLabel = r.bestRes
		}

		fmt.Fprintf(&b, "\n// ---------------------------------------------------------------------------\n")
		fmt.Fprintf(&b, "// Evidence search: %s\n", hypName)
		fmt.Fprintf(&b, "// Prediction: %s\n", r.prediction)
		fmt.Fprintf(&b, "// %d cycles (%d with signal), aggregate: %s\n", r.cycles, r.withSignal, resLabel)

		// Resolution history — survives even if .metabolism-log.json is deleted
		if len(r.resCounts) > 0 {
			if r.lastResolvedAt != "" {
				fmt.Fprintf(&b, "// Resolution history (resolved through %s):\n", r.lastResolvedAt)
			} else {
				fmt.Fprintf(&b, "// Resolution history:\n")
			}
			fmt.Fprintf(&b, "//   %d cycles total, %d with signal\n", r.cycles, r.withSignal)
			var trajectory []string
			for _, res := range []string{"corroborated", "challenged", "irrelevant", "no_signal"} {
				if count, ok := r.resCounts[res]; ok {
					trajectory = append(trajectory, fmt.Sprintf("%s ×%d", res, count))
				}
			}
			if len(trajectory) > 0 {
				fmt.Fprintf(&b, "//   Trajectory: %s\n", strings.Join(trajectory, ", "))
			}
		}

		fmt.Fprintf(&b, "// ---------------------------------------------------------------------------\n\n")

		// Event entity
		fmt.Fprintf(&b, "var EvidenceSearch%s = Event{&Entity{\n", varBase)
		fmt.Fprintf(&b, "\tID:    \"evidence-search-%s\",\n", entityID)
		fmt.Fprintf(&b, "\tName:  \"Evidence search for %s\",\n", baseName)
		fmt.Fprintf(&b, "\tKind:  \"event\",\n")
		fmt.Fprintf(&b, "\tBrief: \"Metabolism sensor query across %s for external sources on %s.%s\",\n",
			strings.Join(backends, ", "), baseName, briefPapers)
		fmt.Fprintf(&b, "}}\n\n")

		// Predicts claim
		fmt.Fprintf(&b, "var TopologyPredicts%sSearch = Predicts{\n", varBase)
		fmt.Fprintf(&b, "\tSubject: StructuralFragilityPredictsCurationGaps,\n")
		fmt.Fprintf(&b, "\tObject:  EvidenceSearch%s,\n", varBase)
		fmt.Fprintf(&b, "\tProv:    metabolismPredictionSource,\n")
		fmt.Fprintf(&b, "}\n")

		// ResolvedAs claim (only if resolved). Named outcome var (not inline
		// struct literal) so each outcome is a distinct object in the
		// reference graph — topology no longer collapses identical literals
		// into a single fake high-degree entity.
		if r.bestRes != "" {
			outcome := mapResolution(r.bestRes)
			evidence := buildEvidenceString(r)

			fmt.Fprintf(&b, "\nvar %sSearchOutcome = &ResolutionOutcome{\n", varBase)
			fmt.Fprintf(&b, "\tResult:   %q,\n", outcome)
			fmt.Fprintf(&b, "\tEvidence: %q,\n", evidence)
			fmt.Fprintf(&b, "}\n")

			fmt.Fprintf(&b, "\nvar %sSearchResolution = ResolvedAs{\n", varBase)
			fmt.Fprintf(&b, "\tSubject: EvidenceSearch%s,\n", varBase)
			fmt.Fprintf(&b, "\tObject:  %sSearchOutcome,\n", varBase)
			fmt.Fprintf(&b, "\tProv:    metabolismPredictionSource,\n")
			fmt.Fprintf(&b, "}\n")
		}
	}

	// KB-internal sections — one meta-hypothesis per prediction type. Each
	// promoted claim becomes an Event + Predicts (and ResolvedAs once
	// resolved). Generic emit loop driven by kbInternalConfigs. Not filtered
	// by isSubstantiveResolution: these resolvers write confirmed/refuted once
	// per distinct trip candidate rather than repeating a question, so they
	// are never the repeated-barren-search problem the sensor filter exists
	// for, and a refuted durability check is itself a real finding worth
	// keeping (a promoted claim failed a check).
	kbResolved := map[string]int{}
	for _, cfg := range kbInternalConfigs {
		ord := kbOrder[cfg.predictionType]
		if len(ord) == 0 {
			continue
		}
		recs := kbRecords[cfg.predictionType]

		fmt.Fprintf(&b, "\n// ---------------------------------------------------------------------------\n")
		fmt.Fprintf(&b, "// %s\n", cfg.sectionHeader)
		fmt.Fprintf(&b, "// KB-internal resolver — the metabolism's own oracle, not an external sensor.\n")
		fmt.Fprintf(&b, "// ---------------------------------------------------------------------------\n\n")

		fmt.Fprintf(&b, "var %s = Hypothesis{&Entity{\n", cfg.metaVar)
		fmt.Fprintf(&b, "\tID:    %q,\n", cfg.metaID)
		fmt.Fprintf(&b, "\tName:  %q,\n", cfg.metaName)
		fmt.Fprintf(&b, "\tKind:  \"hypothesis\",\n")
		fmt.Fprintf(&b, "\tBrief: %q,\n", cfg.metaBrief)
		fmt.Fprintf(&b, "}}\n")

		for _, hypName := range ord {
			r := recs[hypName]
			varBase := cfg.varPrefix + sanitizeIdent(hypName)
			entityID := camelToKebab(hypName)

			resLabel := "pending"
			if r.bestRes != "" {
				resLabel = r.bestRes
			}

			fmt.Fprintf(&b, "\n// ---------------------------------------------------------------------------\n")
			fmt.Fprintf(&b, "// %s: %s\n", cfg.metaName, hypName)
			fmt.Fprintf(&b, "// %d cycle(s), aggregate: %s\n", r.cycles, resLabel)
			if r.evidence != "" {
				fmt.Fprintf(&b, "// Evidence: %s\n", r.evidence)
			}
			fmt.Fprintf(&b, "// ---------------------------------------------------------------------------\n\n")

			fmt.Fprintf(&b, "var %sCheck = Event{&Entity{\n", varBase)
			fmt.Fprintf(&b, "\tID:    \"%s-%s\",\n", cfg.eventIDPrefix, entityID)
			fmt.Fprintf(&b, "\tName:  %q,\n", fmt.Sprintf(cfg.eventNameTmpl, hypName))
			fmt.Fprintf(&b, "\tKind:  \"event\",\n")
			fmt.Fprintf(&b, "\tBrief: %q,\n", fmt.Sprintf(cfg.eventBriefTmpl, hypName))
			fmt.Fprintf(&b, "}}\n\n")

			fmt.Fprintf(&b, "var %s%s = Predicts{\n", varBase, cfg.predictsSuffix)
			fmt.Fprintf(&b, "\tSubject: %s,\n", cfg.metaVar)
			fmt.Fprintf(&b, "\tObject:  %sCheck,\n", varBase)
			fmt.Fprintf(&b, "\tProv:    metabolismPredictionSource,\n")
			fmt.Fprintf(&b, "}\n")

			if r.bestRes != "" {
				kbResolved[cfg.predictionType]++
				evidence := r.evidence
				if evidence == "" {
					evidence = fmt.Sprintf("%d cycle(s), resolution: %s", r.cycles, r.bestRes)
				}
				fmt.Fprintf(&b, "\nvar %sCheckOutcome = &ResolutionOutcome{\n", varBase)
				fmt.Fprintf(&b, "\tResult:   %q,\n", mapResolution(r.bestRes))
				fmt.Fprintf(&b, "\tEvidence: %q,\n", evidence)
				fmt.Fprintf(&b, "}\n")

				fmt.Fprintf(&b, "\nvar %sCheckResolution = ResolvedAs{\n", varBase)
				fmt.Fprintf(&b, "\tSubject: %sCheck,\n", varBase)
				fmt.Fprintf(&b, "\tObject:  %sCheckOutcome,\n", varBase)
				fmt.Fprintf(&b, "\tProv:    metabolismPredictionSource,\n")
				fmt.Fprintf(&b, "}\n")
			}
		}
	}

	// gofmt the generated source before writing so it matches the build gate's
	// gofmt -w step and never lands unformatted. If formatting fails the source
	// is not valid Go — surface that rather than writing a broken file, since
	// the whole point of reify is to emit a corpus slice that compiles.
	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return fmt.Errorf("gofmt %s: %w", outPath, err)
	}
	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Printf("[reify] generated %s\n", filepath.Base(outPath))
	fmt.Printf("[reify] structural_fragility: %d hypotheses probed, %d substantive → %d Events + %d Predicts + %d ResolvedAs (%d activity-only omitted from the corpus)\n",
		uniqueHyps, resolved, resolved, resolved, resolved, omitted)
	for _, cfg := range kbInternalConfigs {
		ord := kbOrder[cfg.predictionType]
		if len(ord) == 0 {
			continue
		}
		fmt.Printf("[reify] %s: %d claims → %d Events + %d Predicts + %d ResolvedAs\n",
			cfg.predictionType, len(ord), len(ord), len(ord), kbResolved[cfg.predictionType])
	}

	// Verify it compiles
	fmt.Println("[reify] verifying: go build ./...")
	if !runGate(dir, "go", "build", "./...") {
		return fmt.Errorf("generated file does not compile — check %s", outPath)
	}
	fmt.Println("[reify] ✓ build passed")
	return nil
}

// betterResolution returns the "better" of two resolutions.
// Priority: corroborated > challenged > confirmed > irrelevant > no_signal > refuted > ""
// Sensor-based and KB-internal resolutions interleave, but in practice
// each hypothesis name only sees one vocabulary (they come from different
// prediction types), so interleaving doesn't matter in aggregation —
// it just needs to be total-ordered.
func betterResolution(a, b string) string {
	priority := map[string]int{
		"":             0,
		"refuted":      1,
		"no_signal":    2,
		"irrelevant":   3,
		"confirmed":    4,
		"challenged":   5,
		"corroborated": 6,
	}
	if priority[b] > priority[a] {
		return b
	}
	return a
}

// mapResolution maps a metabolism resolution to a ResolvedAs Result value.
//
// The meta-prediction is "structural fragility predicts that findable
// external evidence exists." Both corroborated and challenged confirm
// this prediction (evidence was found). Irrelevant means the sensor
// found papers but they weren't relevant (sensor miscalibration, not
// prediction failure). No signal means no papers were found at all.
func mapResolution(res string) string {
	switch res {
	case "corroborated":
		return "confirmed"
	case "challenged":
		return "confirmed"
	case "irrelevant":
		return "ambiguous"
	case "no_signal":
		return "refuted"
	case "confirmed":
		// KB-internal resolver (e.g. trip_lint_durability) already
		// uses the ResolutionOutcome vocabulary directly.
		return "confirmed"
	case "refuted":
		return "refuted"
	default:
		return "ambiguous"
	}
}

// buildEvidenceString creates the Evidence field for a ResolvedAs claim.
func buildEvidenceString(r *hypothesisRecord) string {
	switch r.bestRes {
	case "corroborated":
		if len(r.papers) > 0 {
			titles := make([]string, 0, 3)
			for i, p := range r.papers {
				if i >= 3 {
					break
				}
				titles = append(titles, p.Title)
			}
			return fmt.Sprintf("%d cycles, %d with signal. Corroborated: found %d unique sources including %s.",
				r.cycles, r.withSignal, len(r.papers), strings.Join(titles, "; "))
		}
		return fmt.Sprintf("%d cycles, %d with signal. Resolution: corroborated.", r.cycles, r.withSignal)
	case "challenged":
		return fmt.Sprintf("%d cycles, %d with signal. Resolution: challenged — found evidence contradicting the hypothesis.", r.cycles, r.withSignal)
	case "irrelevant":
		return fmt.Sprintf("%d cycles, %d with signal. Resolution: irrelevant — papers found but not relevant to the hypothesis. Sensor query may need refinement.", r.cycles, r.withSignal)
	case "no_signal":
		return fmt.Sprintf("%d cycles, 0 with signal. Resolution: no signal — no relevant sources found in any backend.", r.cycles)
	default:
		return fmt.Sprintf("%d cycles, %d with signal. Resolution: %s.", r.cycles, r.withSignal, r.bestRes)
	}
}

// camelToKebab converts CamelCase to kebab-case.
var camelSplitRe = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func camelToKebab(s string) string {
	kebab := camelSplitRe.ReplaceAllString(s, "${1}-${2}")
	return strings.ToLower(kebab)
}

// sanitizeIdent keeps only alphanumerics from s. Trip-promoted claim vars
// are already valid Go identifiers, but we pass them through defensively
// so reified var names never introduce a syntax error if a future code
// path produces a name with punctuation.
func sanitizeIdent(s string) string {
	var out []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		}
	}
	return string(out)
}

// camelToWords converts CamelCase to space-separated words.
func camelToWords(s string) string {
	var words []string
	var current []rune
	for _, r := range s {
		if unicode.IsUpper(r) && len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return strings.Join(words, " ")
}

// exportedIdent upper-cases the first rune so a generated var name is
// reachable from outside package winze.
//
// It matters because an unexported claim still compiles, still passes the
// build gate, and still reads as present in the corpus file — it is simply
// invisible to every consumer of the package. Two ResolvedAs claims shipped
// that way before this existed, from a LearningGoal hypName ("goal:Foo")
// whose sanitized form keeps its lowercase lead.
func exportedIdent(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// isSubstantiveResolution reports whether a resolution is a finding worth
// promoting into the corpus, rather than a record of the loop having asked
// and gotten nothing back. Scoped to the sensor vocabulary
// (structural_fragility) deliberately: KB-internal resolvers
// (trip_lint_durability and friends) write confirmed/refuted once per
// distinct trip candidate, never repeating the same question, so their
// volume is never the "same three questions every hour" problem this exists
// to fix — they are left unfiltered.
//
// Measured against 624 logged cycles on 2026-08-22: only 18 of 114 sensor
// hypotheses ever reached corroborated or challenged. The other 96 are the
// loop re-asking a handful of questions repeatedly and getting irrelevant or
// no_signal back — activity, not knowledge, and the reason 160 of 390 corpus
// entities were the loop describing its own searches rather than findings.
func isSubstantiveResolution(res string) bool {
	return res == "corroborated" || res == "challenged"
}
