package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/justinstimatze/winze/internal/defndb"
)

// logTripFunctionalDurability resolves each promoted claim against the
// //winze:functional pragma. For any promoted claim whose predicate is
// declared functional, check whether the Subject already has a *different*
// Object via that predicate elsewhere in the corpus. A collision is a
// refuted; no collision (or non-functional predicate) is confirmed.
//
// Deterministic, no LLM, no API cost — a harder floor than either
// trip_lint_durability or trip_llm_durability. Lint catches what its rules
// look for; the LLM is only as good as its prompt; this resolver is a pure
// type-system property check pinned to a pragma the corpus controls.
//
// Confirmed when:
//   - Predicate is not //winze:functional (rule does not apply, recorded as
//     vacuously confirmed for legibility — calibrate's funnel stays even).
//   - Predicate is functional and no other claim has the same Subject.
//
// Refuted when:
//   - Predicate is functional and some other existing claim with the same
//     Subject has a different Object.
func logTripFunctionalDurability(dir string, claims []promotedClaim) error {
	if len(claims) == 0 {
		return nil
	}

	dbClient, err := defndb.New(dir)
	if err != nil {
		// defn not reachable — skip without logging. The other resolvers
		// already cover this case; failing here would silently flatter
		// calibrate with bogus pendings.
		fmt.Printf("[trip-promote] functional-durability: skipped (%v)\n", err)
		return nil
	}
	defer dbClient.Close()

	functional, err := functionalPredicates(dbClient)
	if err != nil {
		return fmt.Errorf("collect functional pragmas: %w", err)
	}

	// Build per-(Subject, Predicate) → (varName, Object) map of existing
	// claims so we can look up collisions in O(1) per promoted claim.
	type existing struct {
		varName string
		object  string
	}
	existingBySP := map[string][]existing{}
	allClaims, err := dbClient.ClaimFields()
	if err != nil {
		return fmt.Errorf("collect claims: %w", err)
	}
	type partial struct{ predicate, subject, object string }
	byVar := map[string]*partial{}
	for _, f := range allClaims {
		parts := strings.Split(f.TypeName, ".")
		tn := parts[len(parts)-1]
		p, ok := byVar[f.DefName]
		if !ok {
			p = &partial{predicate: tn}
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
	for varName, p := range byVar {
		if p.subject == "" || p.object == "" {
			continue
		}
		key := p.subject + "|" + p.predicate
		existingBySP[key] = append(existingBySP[key], existing{varName: varName, object: p.object})
	}

	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)
	today := time.Now().Format("2006-01-02")
	now := time.Now()

	confirmed, refuted, vacuous := 0, 0, 0
	for _, pc := range claims {
		c := Cycle{
			Timestamp:      now,
			Hypothesis:     pc.VarName,
			Prediction:     "trip-promoted claim respects functional-predicate uniqueness",
			VulnType:       "trip_promotion",
			PredictionType: "trip_functional_durability",
			ResolvedAt:     today,
		}
		if !functional[pc.Predicate] {
			c.Resolution = "confirmed"
			c.Evidence = fmt.Sprintf("%s is not //winze:functional — rule does not apply", pc.Predicate)
			vacuous++
			confirmed++
			mlog.Cycles = append(mlog.Cycles, c)
			continue
		}
		key := pc.Subject + "|" + pc.Predicate
		var conflict *existing
		for i := range existingBySP[key] {
			ex := existingBySP[key][i]
			if ex.varName == pc.VarName {
				continue // skip self
			}
			if ex.object != pc.Object {
				conflict = &ex
				break
			}
		}
		if conflict != nil {
			c.Resolution = "refuted"
			c.Evidence = fmt.Sprintf("functional collision: %s already has %s(%s) → %s; new claim asserts → %s",
				pc.Subject, pc.Predicate, pc.Subject, conflict.object, pc.Object)
			refuted++
		} else {
			c.Resolution = "confirmed"
			c.Evidence = fmt.Sprintf("functional and no collision: %s has at most one Object via %s after this claim", pc.Subject, pc.Predicate)
			confirmed++
		}
		mlog.Cycles = append(mlog.Cycles, c)
	}

	if err := saveLog(logPath, mlog); err != nil {
		return fmt.Errorf("save log: %w", err)
	}

	fmt.Printf("[trip-promote] functional-durability: %d confirmed (%d vacuous), %d refuted (logged as trip_functional_durability)\n",
		confirmed, vacuous, refuted)
	return nil
}

// functionalPredicates returns the set of predicate type names annotated
// with //winze:functional. The pragma's DefName carries the type name
// because the pragma sits as a line comment on the type declaration.
func functionalPredicates(c *defndb.Client) (map[string]bool, error) {
	pragmas, err := c.Pragmas("winze:functional")
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, p := range pragmas {
		if p.Key != "winze:functional" {
			continue
		}
		out[p.DefName] = true
	}
	return out, nil
}
