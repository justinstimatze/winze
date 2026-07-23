package main

import (
	"go/ast"
	"strconv"

	"github.com/justinstimatze/winze/internal/astutil"
)

// Self-directed learning goals, sense side. The corpus declares LearningGoal
// entities and GoalSpec parameters in goals.go (package winze); this reads them
// and turns each active, in-domain, unsatisfied goal into sensor targets that
// flow through the same sense loop as topology's fragility targets. Goals are
// the exploration drive; topology is the exploitation drive.

// goalTarget is the sense-side view of a corpus GoalSpec — just what steering
// the sensor needs.
type goalTarget struct {
	goalVar  string // the LearningGoal var name; matches AdvancesGoal.Object
	seeds    []string
	inDomain bool
	coverAt  int
}

// parseGoalSpecs reads the corpus's GoalSpec literals from goals.go. Returns
// nil (never an error) if the corpus has no goals or cannot be parsed —
// self-directed learning is additive, so its absence must never break a cycle.
func parseGoalSpecs(dir string) []goalTarget {
	pkgs, _, err := astutil.ParseCorpus(dir)
	if err != nil {
		return nil
	}
	var goals []goalTarget
	astutil.WalkVarDecls(pkgs, func(v astutil.VarDecl) {
		if v.TypeName != "GoalSpec" {
			return
		}
		g := goalTarget{}
		for _, elt := range v.Lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			switch key.Name {
			case "Goal":
				if id, ok := kv.Value.(*ast.Ident); ok {
					g.goalVar = id.Name
				}
			case "Seeds":
				if cl, ok := kv.Value.(*ast.CompositeLit); ok {
					for _, e := range cl.Elts {
						if s := astutil.Unquote(e); s != "" {
							g.seeds = append(g.seeds, s)
						}
					}
				}
			case "InDomain":
				if id, ok := kv.Value.(*ast.Ident); ok {
					g.inDomain = id.Name == "true"
				}
			case "CoverAt":
				if lit, ok := kv.Value.(*ast.BasicLit); ok {
					g.coverAt, _ = strconv.Atoi(lit.Value)
				}
			}
		}
		if g.goalVar != "" {
			goals = append(goals, g)
		}
	})
	return goals
}

// countGoalTagged counts the AdvancesGoal claims whose Object is the given
// LearningGoal — the coverage measure that tells a goal when it is satisfied.
func countGoalTagged(dir, goalVar string) int {
	pkgs, _, err := astutil.ParseCorpus(dir)
	if err != nil {
		return 0
	}
	n := 0
	astutil.WalkVarDecls(pkgs, func(v astutil.VarDecl) {
		if v.TypeName != "AdvancesGoal" {
			return
		}
		if _, obj := astutil.ExtractSubjectObject(v.Lit); obj == goalVar {
			n++
		}
	})
	return n
}

// goalSensorTargets turns the corpus's active, in-domain, unsatisfied learning
// goals into sensor targets. Cross-domain goals are skipped — they seed a fork
// and must never ingest into main. A goal at or past its CoverAt threshold is
// satisfied and generates nothing (curiosity with a defined "full"). Returns
// empty when no goals are active, so a corpus with no goals.go behaves exactly
// as before.
func goalSensorTargets(dir string) []SensorTarget {
	var out []SensorTarget
	for _, g := range parseGoalSpecs(dir) {
		if !g.inDomain {
			continue
		}
		if g.coverAt > 0 && countGoalTagged(dir, g.goalVar) >= g.coverAt {
			continue
		}
		for _, seed := range g.seeds {
			out = append(out, SensorTarget{
				Hypothesis: "goal:" + g.goalVar,
				Query:      seed,
				ZimQuery:   seed,
				RssQuery:   seed,
				VulnType:   "learning_goal",
			})
		}
	}
	return out
}
