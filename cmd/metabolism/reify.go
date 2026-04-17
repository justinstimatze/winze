package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// runReify reads the metabolism log and generates a Go corpus file that
// encodes the metabolism loop's predictions as first-class KB claims using
// the Predicts/ResolvedAs prediction schema.
//
// The meta-hypothesis: structural fragility (single-source, uncontested)
// predicts that external evidence exists which could strengthen or challenge
// a hypothesis. Each evidence search is an Event, and its resolution records
// whether the prediction was confirmed.
//
// This is the sophotech move: the KB becomes self-aware about its own
// epistemic performance. It knows what it predicted, what it found, and
// whether its predictions were right.
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

func runReify(dir string) {
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	if len(mlog.Cycles) == 0 {
		fmt.Fprintln(os.Stderr, "metabolism: no cycles logged — nothing to reify")
		return
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
	resolved := 0
	for _, r := range records {
		if r.bestRes != "" {
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

	today := time.Now().Format("2006-01-02")
	outPath := filepath.Join(dir, "predictions.go")

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
	fmt.Fprintf(&b, "// challenge the hypothesis. Each evidence search is an Event, and its\n")
	fmt.Fprintf(&b, "// resolution records whether the prediction was confirmed.\n")
	fmt.Fprintf(&b, "//\n")
	fmt.Fprintf(&b, "// This is the first use of the prediction schema (Predicts, ResolvedAs)\n")
	fmt.Fprintf(&b, "// defined in predicates.go. The forcing function: the metabolism loop\n")
	fmt.Fprintf(&b, "// itself generates falsifiable predictions about the KB's own gaps.\n")

	// Provenance
	fmt.Fprintf(&b, "\nvar metabolismPredictionSource = Provenance{\n")
	fmt.Fprintf(&b, "\tOrigin:     \"winze metabolism log (.metabolism-log.json)\",\n")
	fmt.Fprintf(&b, "\tIngestedAt: %q,\n", today)
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

	// Per-hypothesis events, predictions, and resolutions
	for _, hypName := range order {
		r := records[hypName]
		baseName := camelToWords(hypName)
		varBase := strings.TrimSuffix(hypName, "Thesis")
		varBase = strings.TrimSuffix(varBase, "Argument")
		varBase = strings.TrimSuffix(varBase, "Framing")
		entityID := camelToKebab(hypName)

		// Backends used
		var backends []string
		for be := range r.backends {
			backends = append(backends, be)
		}

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
			fmt.Fprintf(&b, "// Resolution history (reified %s):\n", today)
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
	// resolved). Generic emit loop driven by kbInternalConfigs.
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

	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "metabolism: write %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("[reify] generated %s\n", filepath.Base(outPath))
	fmt.Printf("[reify] structural_fragility: %d hypotheses → %d Events + %d Predicts + %d ResolvedAs\n",
		uniqueHyps, uniqueHyps, uniqueHyps, resolved)
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
		fmt.Fprintf(os.Stderr, "[reify] generated file does not compile — check %s\n", outPath)
		os.Exit(1)
	}
	fmt.Println("[reify] ✓ build passed")
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
