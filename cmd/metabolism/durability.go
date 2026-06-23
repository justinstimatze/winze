package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justinstimatze/winze/internal/defndb"
)

// runDurability re-checks KB-internal verdicts against the current corpus
// and reports drift. This is the A3 mechanism: turning calibrate into a
// moving time-series statistic by detecting which historical verdicts
// have flipped under the current corpus + current oracle code.
//
// Resolver semantics per prediction type:
//   - trip_lint_durability: identical re-run of cmd/lint with per-var
//     substring match.
//   - trip_functional_durability: identical re-run of the functional-pragma
//     collision check against current claim graph.
//   - trip_promotion_attempt: PROXY. The original oracle was the promotion
//     pipeline's entity-existence + slot-type validation, which we cannot
//     replay for rejected attempts (candidate data is gone). We substitute
//     a presence+build check: does the claim var exist and does `go build`
//     still succeed? Accepted promotions with missing vars or build errors
//     flip to refuted; rejected promotions register as unresolvable.
//
// Skipped: trip_llm_durability (API cost + stochasticity makes "flip"
// noisy until results cache). Sensor cycles are a separate concern —
// their oracle is external (arXiv/ZIM).
//
// Read-only by default. Pass writeLog=true to append recheck entries
// with suffixed PredictionType ("_recheck") so calibrate picks them up.
func runDurability(dir string, jsonOut, writeLog bool) {
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	latest := latestVerdicts(mlog.Cycles)
	if len(latest) == 0 {
		fmt.Println("[durability] no KB-internal resolver entries in log")
		return
	}
	// digests carries the inputs (corpus+oracle) recorded at the last actual
	// run, including _recheck entries — the gate keys on this, not `latest`.
	digests := latestRecheckState(mlog.Cycles)

	commit := oracleCommit(dir)

	// Batch per prediction type
	var lintVars, functionalVars, buildVars []string
	for key := range latest {
		switch key.predictionType {
		case "trip_lint_durability":
			lintVars = append(lintVars, key.hypothesis)
		case "trip_functional_durability":
			functionalVars = append(functionalVars, key.hypothesis)
		case "trip_promotion_attempt":
			buildVars = append(buildVars, key.hypothesis)
		}
	}
	sort.Strings(lintVars)
	sort.Strings(functionalVars)
	sort.Strings(buildVars)

	var results []durabilityResult
	if len(lintVars) > 0 {
		results = append(results, recheckLint(dir, lintVars, latest, digests, commit)...)
	}
	if len(functionalVars) > 0 {
		results = append(results, recheckFunctional(dir, functionalVars, latest, digests, commit)...)
	}
	if len(buildVars) > 0 {
		results = append(results, recheckBuild(dir, buildVars, latest, digests, commit)...)
	}

	report := buildDurabilityReport(results)

	if writeLog && len(results) > 0 {
		now := time.Now()
		today := now.Format("2006-01-02")
		for _, r := range results {
			if r.drift == driftUnresolvable {
				continue
			}
			mlog.Cycles = append(mlog.Cycles, Cycle{
				Timestamp:      now,
				Hypothesis:     r.hypothesis,
				Prediction:     "durability recheck: " + r.oldPredictionType + " verdict holds against current corpus",
				VulnType:       "trip_promotion",
				PredictionType: r.oldPredictionType + "_recheck",
				ResolvedAt:     today,
				Resolution:     r.newVerdict,
				Evidence:       r.newEvidence,
				OracleCommit:   r.newOracleCommit,
				OracleDigest:   r.newOracleDigest,
				CorpusDigest:   r.newCorpusDigest,
			})
		}
		if err := saveLog(logPath, mlog); err != nil {
			fmt.Fprintf(os.Stderr, "[durability] save log: %v\n", err)
			os.Exit(1)
		}
	}

	if jsonOut {
		emitDurabilityJSON(report, results)
	} else {
		emitDurabilityText(report, results, writeLog)
	}
}

type verdictKey struct {
	hypothesis     string
	predictionType string
}

type verdictRecord struct {
	resolution   string
	evidence     string
	oracleCommit string
	oracleDigest string
	corpusDigest string
	timestamp    time.Time
}

// latestVerdicts returns the most recent KB-internal verdict per
// (hypothesis, prediction_type). Ignores sensor prediction types and
// any _recheck entries (we only re-run originals).
func latestVerdicts(cycles []Cycle) map[verdictKey]verdictRecord {
	out := map[verdictKey]verdictRecord{}
	for _, c := range cycles {
		if !isRecheckable(c.PredictionType) {
			continue
		}
		key := verdictKey{hypothesis: c.Hypothesis, predictionType: c.PredictionType}
		if prev, ok := out[key]; ok && !c.Timestamp.After(prev.timestamp) {
			continue
		}
		out[key] = verdictRecord{
			resolution:   c.Resolution,
			evidence:     c.Evidence,
			oracleCommit: c.OracleCommit,
			oracleDigest: c.OracleDigest,
			corpusDigest: c.CorpusDigest,
			timestamp:    c.Timestamp,
		}
	}
	return out
}

func isRecheckable(pt string) bool {
	switch pt {
	case "trip_lint_durability", "trip_functional_durability", "trip_promotion_attempt":
		return true
	}
	return false
}

// latestRecheckState returns the most recent recorded state per BASE
// (hypothesis, predictionType), INCLUDING _recheck entries (mapped back to
// their base type via the "_recheck" suffix). The corpus-digest gate reads
// THIS, not latestVerdicts: it must know the inputs and verdict at the last
// time the resolver actually ran (or was carried forward), which lives on
// the newest entry regardless of the _recheck suffix. latestVerdicts
// deliberately ignores _recheck to preserve drift-vs-original semantics;
// the gate instead needs "has anything changed since the last run", and the
// CorpusDigest it keys on is only ever written on _recheck entries — so
// filtering them out (as latestVerdicts does) would make the gate dead.
func latestRecheckState(cycles []Cycle) map[verdictKey]verdictRecord {
	out := map[verdictKey]verdictRecord{}
	for _, c := range cycles {
		base := strings.TrimSuffix(c.PredictionType, "_recheck")
		if !isRecheckable(base) {
			continue
		}
		key := verdictKey{hypothesis: c.Hypothesis, predictionType: base}
		if prev, ok := out[key]; ok && !c.Timestamp.After(prev.timestamp) {
			continue
		}
		out[key] = verdictRecord{
			resolution:   c.Resolution,
			evidence:     c.Evidence,
			oracleCommit: c.OracleCommit,
			oracleDigest: c.OracleDigest,
			corpusDigest: c.CorpusDigest,
			timestamp:    c.Timestamp,
		}
	}
	return out
}

type driftCategory string

const (
	driftStable           driftCategory = "stable"
	driftFlippedConfirmed driftCategory = "flipped_to_confirmed"
	driftFlippedRefuted   driftCategory = "flipped_to_refuted"
	driftNowAmbiguous     driftCategory = "now_ambiguous"
	driftUnresolvable     driftCategory = "unresolvable"
	driftResolverChanged  driftCategory = "resolver_changed"
)

type durabilityResult struct {
	hypothesis        string
	oldPredictionType string

	oldVerdict      string
	oldEvidence     string
	oldOracleCommit string
	oldOracleDigest string

	newVerdict      string
	newEvidence     string
	newOracleCommit string
	newOracleDigest string
	newCorpusDigest string

	drift driftCategory
}

// corpusUnchanged reports whether every claimVar's last recheck observed
// the same corpus content AND the same oracle code as now. When true, the
// resolver is a pure function of unchanged inputs, so its verdicts cannot
// have moved and the (expensive) resolver invocation can be skipped. A var
// whose prior entry has no stored CorpusDigest (legacy, pre-gate) forces a
// full run — the conservative direction. The oracle half is mandatory:
// editing cmd/lint can change output on a byte-identical corpus.
func corpusUnchanged(claimVars []string, latest map[verdictKey]verdictRecord, predictionType, curCorpus, curOracle string) bool {
	if curCorpus == "" {
		return false
	}
	for _, v := range claimVars {
		old := latest[verdictKey{hypothesis: v, predictionType: predictionType}]
		if old.corpusDigest == "" || old.corpusDigest != curCorpus || old.oracleDigest != curOracle {
			return false
		}
	}
	return true
}

// stableCarryForward emits recheck results that reuse the last run's verdict
// without re-running the resolver. Used only when corpusUnchanged proved the
// resolver's inputs (corpus + oracle) are byte-identical to the last run, so
// a fresh run would reproduce that run's verdict exactly.
//
// The carried verdict comes from `last` (latestRecheckState — the newest
// entry, including _recheck), NOT the original: if a prior recheck already
// flipped the verdict, the flip persists under unchanged inputs, and
// carrying the original would mask it as stable. Drift is still classified
// against the ORIGINAL verdict (`orig`, from latestVerdicts), matching the
// non-skip path's drift-vs-original semantics.
func stableCarryForward(claimVars []string, latest, digests map[verdictKey]verdictRecord, predictionType, commit, corpusDig string) []durabilityResult {
	out := make([]durabilityResult, 0, len(claimVars))
	for _, v := range claimVars {
		key := verdictKey{hypothesis: v, predictionType: predictionType}
		orig := latest[key]   // original promotion verdict — drift baseline
		last := digests[key]  // newest run's verdict — what a fresh run would reproduce
		out = append(out, durabilityResult{
			hypothesis:        v,
			oldPredictionType: predictionType,
			oldVerdict:        orig.resolution,
			oldEvidence:       orig.evidence,
			oldOracleCommit:   orig.oracleCommit,
			oldOracleDigest:   orig.oracleDigest,
			newVerdict:        last.resolution,
			newEvidence:       "skipped: corpus+oracle digest unchanged since last run (" + corpusDig + ")",
			newOracleCommit:   commit,
			newOracleDigest:   last.oracleDigest, // == current oracle by the gate's construction
			newCorpusDigest:   corpusDig,
			drift:             classifyDrift(orig.resolution, last.resolution, orig.oracleDigest, last.oracleDigest),
		})
	}
	return out
}

// classifyDrift decides the drift category from old vs new verdicts and
// oracle digests. resolverChanged is flagged whenever the digest moved
// (and oldDigest isn't empty), even if the verdict happens to agree:
// an unchanged verdict under a changed oracle is still an attribution
// risk callers should see. driftUnresolvable takes precedence (verdict
// comparison is meaningless if the var no longer compiles).
func classifyDrift(oldVerdict, newVerdict, oldDigest, newDigest string) driftCategory {
	if newVerdict == "unresolvable" {
		return driftUnresolvable
	}
	resolverChanged := oldDigest != "" && newDigest != "" && oldDigest != newDigest
	switch {
	case newVerdict == oldVerdict:
		if resolverChanged {
			return driftResolverChanged
		}
		return driftStable
	case newVerdict == "confirmed":
		return driftFlippedConfirmed
	case newVerdict == "refuted":
		return driftFlippedRefuted
	default:
		return driftNowAmbiguous
	}
}

// ---- lint recheck ----

func recheckLint(dir string, claimVars []string, latest, digests map[verdictKey]verdictRecord, commit string) []durabilityResult {
	digest := oracleDigest(dir, "trip_lint_durability")
	curCorpus := corpusDigest(dir)
	if corpusUnchanged(claimVars, digests, "trip_lint_durability", curCorpus, digest) {
		fmt.Fprintln(os.Stderr, "[durability] lint recheck skipped: corpus+oracle digest unchanged")
		return stableCarryForward(claimVars, latest, digests, "trip_lint_durability", commit, curCorpus)
	}

	cmd := exec.Command("go", "run", "./cmd/lint", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	lintText := string(out)

	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		}
	}

	existing := existingVars(dir)

	var results []durabilityResult
	for _, v := range claimVars {
		key := verdictKey{hypothesis: v, predictionType: "trip_lint_durability"}
		old := latest[key]

		r := durabilityResult{
			hypothesis:        v,
			oldPredictionType: "trip_lint_durability",
			oldVerdict:        old.resolution,
			oldEvidence:       old.evidence,
			oldOracleCommit:   old.oracleCommit,
			oldOracleDigest:   old.oracleDigest,
			newOracleCommit:   commit,
			newOracleDigest:   digest,
			newCorpusDigest:   curCorpus,
		}
		if !existing[v] {
			r.newVerdict = "unresolvable"
			r.newEvidence = "claim var no longer present in corpus"
		} else if exitCode == 0 {
			r.newVerdict = "confirmed"
			r.newEvidence = "cmd/lint exited 0 — no findings on any rule"
		} else if line := findLintEvidence(lintText, v); line != "" {
			r.newVerdict = "refuted"
			r.newEvidence = line
		} else {
			r.newVerdict = "confirmed"
			r.newEvidence = fmt.Sprintf("cmd/lint exited %d but %s not mentioned in any finding", exitCode, v)
		}
		r.drift = classifyDrift(r.oldVerdict, r.newVerdict, r.oldOracleDigest, r.newOracleDigest)
		results = append(results, r)
	}
	return results
}

// ---- functional recheck ----

func recheckFunctional(dir string, claimVars []string, latest, digests map[verdictKey]verdictRecord, commit string) []durabilityResult {
	digest := oracleDigest(dir, "trip_functional_durability")
	curCorpus := corpusDigest(dir)
	if corpusUnchanged(claimVars, digests, "trip_functional_durability", curCorpus, digest) {
		// Bonus: this path needs defn, which can be down. When the corpus
		// is unchanged we carry verdicts forward without touching defn at
		// all, so an idle recheck stays "stable" instead of degrading to
		// now_ambiguous on a defn outage.
		fmt.Fprintln(os.Stderr, "[durability] functional recheck skipped: corpus+oracle digest unchanged")
		return stableCarryForward(claimVars, latest, digests, "trip_functional_durability", commit, curCorpus)
	}

	dbClient, err := defndb.New(dir)
	if err != nil {
		// defn unreachable — mark all as now_ambiguous
		var results []durabilityResult
		for _, v := range claimVars {
			key := verdictKey{hypothesis: v, predictionType: "trip_functional_durability"}
			old := latest[key]
			results = append(results, durabilityResult{
				hypothesis:        v,
				oldPredictionType: "trip_functional_durability",
				oldVerdict:        old.resolution,
				oldEvidence:       old.evidence,
				oldOracleCommit:   old.oracleCommit,
				oldOracleDigest:   old.oracleDigest,
				newVerdict:        "",
				newEvidence:       fmt.Sprintf("defndb unreachable: %v", err),
				newOracleCommit:   commit,
				newOracleDigest:   digest,
				newCorpusDigest:   curCorpus,
				drift:             driftNowAmbiguous,
			})
		}
		return results
	}
	defer dbClient.Close()

	functional, err := functionalPredicates(dbClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[durability] collect functional pragmas: %v\n", err)
		return nil
	}

	type claimInfo struct {
		predicate, subject, object string
	}
	byVar := map[string]*claimInfo{}
	allClaims, err := dbClient.ClaimFields()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[durability] collect claims: %v\n", err)
		return nil
	}
	for _, f := range allClaims {
		parts := strings.Split(f.TypeName, ".")
		tn := parts[len(parts)-1]
		p, ok := byVar[f.DefName]
		if !ok {
			p = &claimInfo{predicate: tn}
			byVar[f.DefName] = p
		}
		v := strings.Trim(f.FieldValue, "\"")
		switch f.FieldName {
		case "Subject":
			p.subject = v
		case "Object":
			p.object = v
		}
	}

	// (Subject, Predicate) -> []Object (from other claims) for collision lookup
	existingBySP := map[string][]string{}
	for varName, p := range byVar {
		if p.subject == "" || p.object == "" {
			continue
		}
		key := p.subject + "|" + p.predicate
		existingBySP[key] = append(existingBySP[key], fmt.Sprintf("%s=%s", varName, p.object))
	}

	var results []durabilityResult
	for _, v := range claimVars {
		key := verdictKey{hypothesis: v, predictionType: "trip_functional_durability"}
		old := latest[key]

		r := durabilityResult{
			hypothesis:        v,
			oldPredictionType: "trip_functional_durability",
			oldVerdict:        old.resolution,
			oldEvidence:       old.evidence,
			oldOracleCommit:   old.oracleCommit,
			oldOracleDigest:   old.oracleDigest,
			newOracleCommit:   commit,
			newOracleDigest:   digest,
			newCorpusDigest:   curCorpus,
		}

		info, present := byVar[v]
		if !present || info.subject == "" || info.object == "" {
			r.newVerdict = "unresolvable"
			r.newEvidence = "claim var no longer present in corpus"
			r.drift = classifyDrift(r.oldVerdict, r.newVerdict, r.oldOracleDigest, r.newOracleDigest)
			results = append(results, r)
			continue
		}

		if !functional[info.predicate] {
			r.newVerdict = "confirmed"
			r.newEvidence = fmt.Sprintf("%s is not //winze:functional — rule does not apply", info.predicate)
			r.drift = classifyDrift(r.oldVerdict, r.newVerdict, r.oldOracleDigest, r.newOracleDigest)
			results = append(results, r)
			continue
		}

		lookupKey := info.subject + "|" + info.predicate
		collision := ""
		for _, entry := range existingBySP[lookupKey] {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) != 2 {
				continue
			}
			otherVar, otherObject := parts[0], parts[1]
			if otherVar == v {
				continue
			}
			if otherObject != info.object {
				collision = fmt.Sprintf("%s has Object=%s", otherVar, otherObject)
				break
			}
		}
		if collision != "" {
			r.newVerdict = "refuted"
			r.newEvidence = fmt.Sprintf("functional collision: %s; this claim asserts Object=%s", collision, info.object)
		} else {
			r.newVerdict = "confirmed"
			r.newEvidence = fmt.Sprintf("functional and no collision: %s has at most one Object via %s", info.subject, info.predicate)
		}
		r.drift = classifyDrift(r.oldVerdict, r.newVerdict, r.oldOracleDigest, r.newOracleDigest)
		results = append(results, r)
	}
	return results
}

// ---- build-gate recheck ----

func recheckBuild(dir string, claimVars []string, latest, digests map[verdictKey]verdictRecord, commit string) []durabilityResult {
	digest := oracleDigest(dir, "trip_promotion_attempt")
	curCorpus := corpusDigest(dir)
	if corpusUnchanged(claimVars, digests, "trip_promotion_attempt", curCorpus, digest) {
		fmt.Fprintln(os.Stderr, "[durability] build recheck skipped: corpus+toolchain digest unchanged")
		return stableCarryForward(claimVars, latest, digests, "trip_promotion_attempt", commit, curCorpus)
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	buildOK := err == nil
	buildText := string(out)

	existing := existingVars(dir)

	var results []durabilityResult
	for _, v := range claimVars {
		key := verdictKey{hypothesis: v, predictionType: "trip_promotion_attempt"}
		old := latest[key]

		r := durabilityResult{
			hypothesis:        v,
			oldPredictionType: "trip_promotion_attempt",
			oldVerdict:        old.resolution,
			oldEvidence:       old.evidence,
			oldOracleCommit:   old.oracleCommit,
			oldOracleDigest:   old.oracleDigest,
			newOracleCommit:   commit,
			newOracleDigest:   digest,
			newCorpusDigest:   curCorpus,
		}
		if !existing[v] {
			r.newVerdict = "unresolvable"
			r.newEvidence = "claim var no longer present in corpus"
		} else if buildOK {
			r.newVerdict = "confirmed"
			r.newEvidence = "go build ./... succeeded"
		} else if strings.Contains(buildText, v) {
			r.newVerdict = "refuted"
			r.newEvidence = firstLineMentioning(buildText, v)
		} else {
			r.newVerdict = "confirmed"
			r.newEvidence = fmt.Sprintf("go build failed but %s not mentioned in errors", v)
		}
		r.drift = classifyDrift(r.oldVerdict, r.newVerdict, r.oldOracleDigest, r.newOracleDigest)
		results = append(results, r)
	}
	return results
}

func firstLineMentioning(text, needle string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, needle) {
			trim := strings.TrimSpace(line)
			if len(trim) > 240 {
				trim = trim[:240] + "..."
			}
			return trim
		}
	}
	return ""
}

// ---- existence check ----

// existingVars returns the set of top-level var names in the corpus root,
// via defndb. Used by recheck* to decide "unresolvable" when a claim var
// has been deleted between promotion and now.
func existingVars(dir string) map[string]bool {
	out := map[string]bool{}
	dbClient, err := defndb.New(dir)
	if err != nil {
		return out
	}
	defer dbClient.Close()
	fields, err := dbClient.ClaimFields()
	if err != nil {
		return out
	}
	for _, f := range fields {
		out[f.DefName] = true
	}
	return out
}

// ---- report types ----

type durabilityReport struct {
	Stable                 int            `json:"stable"`
	StableHeldAcrossCommit int            `json:"stable_held_across_commit"`
	FlippedToConfirmed     int            `json:"flipped_to_confirmed"`
	FlippedToRefuted       int            `json:"flipped_to_refuted"`
	NowAmbiguous           int            `json:"now_ambiguous"`
	Unresolvable           int            `json:"unresolvable"`
	ResolverChanged        int            `json:"resolver_changed"`
	ByResolver             map[string]int `json:"by_resolver"`
	Total                  int            `json:"total"`
}

func buildDurabilityReport(results []durabilityResult) durabilityReport {
	r := durabilityReport{ByResolver: map[string]int{}}
	for _, x := range results {
		r.Total++
		r.ByResolver[x.oldPredictionType]++
		switch x.drift {
		case driftStable:
			r.Stable++
			if x.oldOracleCommit != "" && x.newOracleCommit != "" && x.oldOracleCommit != x.newOracleCommit {
				r.StableHeldAcrossCommit++
			}
		case driftFlippedConfirmed:
			r.FlippedToConfirmed++
		case driftFlippedRefuted:
			r.FlippedToRefuted++
		case driftNowAmbiguous:
			r.NowAmbiguous++
		case driftUnresolvable:
			r.Unresolvable++
		case driftResolverChanged:
			r.ResolverChanged++
		}
	}
	return r
}

// ---- emitters ----

func emitDurabilityText(report durabilityReport, results []durabilityResult, wrote bool) {
	// Sort: flips first, then resolver_changed, then unresolvable, then stable
	order := map[driftCategory]int{
		driftFlippedRefuted:   0,
		driftFlippedConfirmed: 1,
		driftResolverChanged:  2,
		driftNowAmbiguous:     3,
		driftUnresolvable:     4,
		driftStable:           5,
	}
	sort.SliceStable(results, func(i, j int) bool {
		if order[results[i].drift] != order[results[j].drift] {
			return order[results[i].drift] < order[results[j].drift]
		}
		return results[i].hypothesis < results[j].hypothesis
	})

	fmt.Printf("[durability] rechecked %d historical verdicts across %d resolver(s)\n", report.Total, len(report.ByResolver))
	for _, pt := range sortedKeys(report.ByResolver) {
		fmt.Printf("  %s: %d\n", pt, report.ByResolver[pt])
	}
	fmt.Println()

	emitted := false
	for _, cat := range []driftCategory{driftFlippedRefuted, driftFlippedConfirmed, driftResolverChanged, driftNowAmbiguous, driftUnresolvable} {
		group := filterByDrift(results, cat)
		if len(group) == 0 {
			continue
		}
		emitted = true
		header := strings.ToUpper(string(cat))
		fmt.Printf("%s (%d):\n", header, len(group))
		for _, r := range group {
			fmt.Printf("  [%s] %s  %s → %s\n", shortResolver(r.oldPredictionType), r.hypothesis,
				valueOr(r.oldVerdict, "<unset>"), valueOr(r.newVerdict, "<ambiguous>"))
			if r.drift != driftUnresolvable {
				fmt.Printf("    old: %s\n", truncateEvidence(r.oldEvidence, 180))
				fmt.Printf("    new: %s\n", truncateEvidence(r.newEvidence, 180))
			} else {
				fmt.Printf("    note: %s\n", truncateEvidence(r.newEvidence, 180))
			}
			if r.drift == driftResolverChanged {
				fmt.Printf("    oracle digest: %s → %s (resolver code changed)\n",
					valueOr(r.oldOracleDigest, "<unset>"), valueOr(r.newOracleDigest, "<unset>"))
			} else if r.oldOracleCommit != "" && r.newOracleCommit != "" && r.oldOracleCommit != r.newOracleCommit {
				fmt.Printf("    corpus commit: %s → %s\n", r.oldOracleCommit, r.newOracleCommit)
			}
		}
		fmt.Println()
	}
	if !emitted {
		fmt.Println("(no flips or unresolvables — all historical verdicts held)")
		fmt.Println()
	}

	// Stable-despite-churn: positive durability signal that would otherwise
	// be hidden since the per-entry detail loop skips stable entries.
	if report.StableHeldAcrossCommit > 0 {
		fmt.Printf("DURABILITY  %d of %d stable verdicts held across a corpus commit change\n",
			report.StableHeldAcrossCommit, report.Stable)
	}

	fmt.Printf("SUMMARY  stable=%d  flipped→confirmed=%d  flipped→refuted=%d  resolver_changed=%d  ambiguous=%d  unresolvable=%d  (total=%d)\n",
		report.Stable, report.FlippedToConfirmed, report.FlippedToRefuted, report.ResolverChanged, report.NowAmbiguous, report.Unresolvable, report.Total)
	if !wrote {
		fmt.Println("(read-only; pass --write to append recheck entries to .metabolism-log.json)")
	} else {
		fmt.Println("(appended recheck entries with PredictionType suffix _recheck)")
	}
}

func emitDurabilityJSON(report durabilityReport, results []durabilityResult) {
	type flip struct {
		Hypothesis    string `json:"hypothesis"`
		Resolver      string `json:"resolver"`
		Drift         string `json:"drift"`
		OldVerdict    string `json:"old_verdict"`
		NewVerdict    string `json:"new_verdict"`
		OldEvidence   string `json:"old_evidence,omitempty"`
		NewEvidence   string `json:"new_evidence,omitempty"`
		OldCommit     string `json:"old_oracle_commit,omitempty"`
		NewCommit     string `json:"new_oracle_commit,omitempty"`
		OldDigest     string `json:"old_oracle_digest,omitempty"`
		NewDigest     string `json:"new_oracle_digest,omitempty"`
	}
	items := []flip{}
	for _, r := range results {
		if r.drift == driftStable {
			continue
		}
		items = append(items, flip{
			Hypothesis:  r.hypothesis,
			Resolver:    r.oldPredictionType,
			Drift:       string(r.drift),
			OldVerdict:  r.oldVerdict,
			NewVerdict:  r.newVerdict,
			OldEvidence: r.oldEvidence,
			NewEvidence: r.newEvidence,
			OldCommit:   r.oldOracleCommit,
			NewCommit:   r.newOracleCommit,
			OldDigest:   r.oldOracleDigest,
			NewDigest:   r.newOracleDigest,
		})
	}
	payload := struct {
		Report durabilityReport `json:"report"`
		Drift  []flip           `json:"drift"`
	}{Report: report, Drift: items}
	data, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Println(string(data))
}

func filterByDrift(results []durabilityResult, cat driftCategory) []durabilityResult {
	var out []durabilityResult
	for _, r := range results {
		if r.drift == cat {
			out = append(out, r)
		}
	}
	return out
}

func shortResolver(pt string) string {
	switch pt {
	case "trip_lint_durability":
		return "lint"
	case "trip_functional_durability":
		return "functional"
	case "trip_promotion_attempt":
		return "build"
	case "trip_llm_durability":
		return "llm"
	}
	return pt
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func valueOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func truncateEvidence(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ---- oracle identity ----

// oracleCommit returns the short git HEAD SHA of dir, or "" if git is
// unavailable. Used to tag when a verdict was computed; comparing this
// between old and new verdicts identifies corpus-change drift.
func oracleCommit(dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	sha := strings.TrimSpace(string(out))
	if len(sha) > 12 {
		sha = sha[:12]
	}
	return sha
}

// goVersionDigest returns a short hash of `go version` output. Used as
// the oracle digest for build-gate rechecks so Go toolchain changes
// surface as resolver_changed rather than looking like corpus drift.
func goVersionDigest() string {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return ""
	}
	h := sha256.Sum256(out)
	sum := fmt.Sprintf("%x", h)
	if len(sum) > 12 {
		sum = sum[:12]
	}
	return sum
}

// corpusDigest returns a short sha256 over the corpus content — every
// non-test *.go in the module plus go.mod/go.sum — sorted by module-
// relative path for determinism. It is the corpus-content counterpart to
// oracleDigest: together they let --durability skip a resolver invocation
// when neither the corpus nor the resolver code changed since the last
// recheck. The set is a deliberate SUPERSET of what any single resolver
// reads (it includes machinery .go a lint rule may not touch): a superset
// can only trigger an unnecessary re-run, never an unsound skip. Test
// files are excluded to match `go build ./...` semantics (it does not
// compile them) and to keep the digest stable under test-only edits.
func corpusDigest(dir string) string {
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".defn", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		switch {
		case name == "go.mod" || name == "go.sum":
			files = append(files, path)
		case strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go"):
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	h := sha256.New()
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		rel, relErr := filepath.Rel(dir, f)
		if relErr != nil {
			rel = f
		}
		h.Write([]byte(rel + "\x00"))
		h.Write(data)
		h.Write([]byte("\n"))
	}
	sum := fmt.Sprintf("%x", h.Sum(nil))
	if len(sum) > 12 {
		sum = sum[:12]
	}
	return sum
}

// oracleDigest returns a short sha256 over the source files that define
// a resolver's behavior at call time. A change between old and new
// verdicts for the same (hypothesis, prediction_type) means we edited
// the oracle, not the corpus — lets --durability filter out spurious
// drift attributable to resolver-code churn.
func oracleDigest(dir, predictionType string) string {
	var files []string
	switch predictionType {
	case "trip_lint_durability":
		matches, _ := filepath.Glob(filepath.Join(dir, "cmd", "lint", "*.go"))
		// Exclude test files to keep digest stable under test-only edits
		for _, m := range matches {
			if !strings.HasSuffix(m, "_test.go") {
				files = append(files, m)
			}
		}
	case "trip_functional_durability":
		files = []string{filepath.Join(dir, "cmd", "metabolism", "trip_functional_resolve.go")}
	case "trip_llm_durability":
		files = []string{filepath.Join(dir, "cmd", "metabolism", "trip_llm_resolve.go")}
	case "trip_promotion_attempt":
		// build-gate's oracle is the whole corpus + Go toolchain. We can't
		// digest the corpus (it IS the thing under test), but we can digest
		// the toolchain version so flips attributable to a Go upgrade show
		// up as resolver_changed rather than unattributed corpus drift.
		return goVersionDigest()
	default:
		return ""
	}
	sort.Strings(files)
	h := sha256.New()
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		h.Write([]byte(filepath.Base(f) + "\x00"))
		h.Write(data)
		h.Write([]byte("\n"))
	}
	sum := fmt.Sprintf("%x", h.Sum(nil))
	if len(sum) > 12 {
		sum = sum[:12]
	}
	return sum
}
