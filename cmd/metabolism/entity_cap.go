package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	"github.com/justinstimatze/winze/internal/astutil"
)

// reifyBookkeepingFile is the reify phase's generated output — evidence-search
// predictions reified as first-class Event/Hypothesis claims (see the header
// of the file itself). Its entities are derived calibration scaffolding, not
// ingested knowledge.
const reifyBookkeepingFile = "predictions.go"

// reifyEntityCount counts the entity declarations in the reify bookkeeping
// file. The entity cap exists to bound the *knowledge* base ("max entities
// allowed in the KB; refuse ingest above it"), but topology counts every
// package-winze entity, including reify's Events. Left unchecked, the loop's
// own calibration output grows to consume the entity budget and tips the count
// over the cap, at which point the ingest phase refuses to run — the loop
// calibrates itself into an inability to learn. Excluding these keeps the cap
// measuring knowledge, not scaffolding.
//
// Fails open (returns 0) if the file is absent (reify never ran) or
// unparseable — the cap is a guard, not a correctness invariant, and
// over-counting is the safe direction for a guard.
func reifyEntityCount(dir string) int {
	src, err := os.ReadFile(filepath.Join(dir, reifyBookkeepingFile))
	if err != nil {
		return 0
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, reifyBookkeepingFile, src, parser.SkipObjectResolution)
	if err != nil {
		return 0
	}
	n := 0
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
			if cl, ok := vs.Values[0].(*ast.CompositeLit); ok && isEntityLiteral(cl) {
				n++
			}
		}
	}
	return n
}

// isEntityLiteral reports whether a composite literal is an entity declaration
// — a role type wrapping an &Entity{...} pointer, the RoleType{&Entity{...}}
// shape every winze entity uses. Role-agnostic, so it counts Event, Hypothesis,
// or any future role reify emits, and ignores claim literals (Predicts,
// ResolvedAs), which carry Subject/Object entity *references*, not an embedded
// &Entity.
func isEntityLiteral(cl *ast.CompositeLit) bool {
	for _, elt := range cl.Elts {
		v := elt
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			v = kv.Value
		}
		u, ok := v.(*ast.UnaryExpr)
		if !ok || u.Op != token.AND {
			continue
		}
		if inner, ok := u.X.(*ast.CompositeLit); ok && astutil.CompositeTypeName(inner) == "Entity" {
			return true
		}
	}
	return false
}
