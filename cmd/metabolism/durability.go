package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justinstimatze/winze/internal/defndb"
)

// runDurability re-runs KB-internal resolvers against the current corpus
// and reports drift vs historical verdicts. This is the A3 mechanism:
// turning calibrate into a moving time-series statistic by detecting
// which historical promotions have flipped verdicts under the current
// corpus + current oracle code.
//
// Three resolvers are rechecked in this first cut:
//   - trip_lint_durability: re-runs cmd/lint once, substring-matches per var
//   - trip_functional_durability: re-runs //winze:functional collision check
//   - trip_promotion_attempt: re-runs `go build ./...`
//
// Skipped: trip_llm_durability (API cost + stochasticity). Sensor cycles
// are a separate concern — their oracle is external (arXiv/ZIM).
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
		results = append(results, recheckLint(dir, lintVars, latest, commit)...)
	}
	if len(functionalVars) > 0 {
		results = append(results, recheckFunctional(dir, functionalVars, latest, commit)...)
	}
	if len(buildVars) > 0 {
		results = append(results, recheckBuild(dir, buildVars, latest, commit)...)
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

	drift driftCategory
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

func recheckLint(dir string, claimVars []string, latest map[verdictKey]verdictRecord, commit string) []durabilityResult {
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

	digest := oracleDigest(dir, "trip_lint_durability")
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

func recheckFunctional(dir string, claimVars []string, latest map[verdictKey]verdictRecord, commit string) []durabilityResult {
	digest := oracleDigest(dir, "trip_functional_durability")

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

func recheckBuild(dir string, claimVars []string, latest map[verdictKey]verdictRecord, commit string) []durabilityResult {
	digest := oracleDigest(dir, "trip_promotion_attempt")

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
	Stable              int            `json:"stable"`
	FlippedToConfirmed  int            `json:"flipped_to_confirmed"`
	FlippedToRefuted    int            `json:"flipped_to_refuted"`
	NowAmbiguous        int            `json:"now_ambiguous"`
	Unresolvable        int            `json:"unresolvable"`
	ResolverChanged     int            `json:"resolver_changed"`
	ByResolver          map[string]int `json:"by_resolver"`
	Total               int            `json:"total"`
}

func buildDurabilityReport(results []durabilityResult) durabilityReport {
	r := durabilityReport{ByResolver: map[string]int{}}
	for _, x := range results {
		r.Total++
		r.ByResolver[x.oldPredictionType]++
		switch x.drift {
		case driftStable:
			r.Stable++
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
	var items []flip
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
		// build-gate depends on the whole corpus + Go toolchain; digesting
		// a single file would be misleading. Leave empty.
		return ""
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
