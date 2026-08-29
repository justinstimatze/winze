package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// anyRoleSlots reports, for predicate as declared in <repoRoot>/predicates.go,
// whether its Subject and/or Object slot accepts any role — declared over
// *Entity directly rather than a named role type like Concept or Hypothesis
// (StructurallyAnalogousTo and Supersedes are the two such predicates today).
// ok is false when predicate isn't found there (a typo, or a predicate the
// build gate will reject for some other reason) — resolveEntityRef leaves
// the ref untouched in that case, same as before this existed.
func anyRoleSlots(repoRoot, predicate string) (subjectAnyRole, objectAnyRole, ok bool) {
	src, err := os.ReadFile(filepath.Join(repoRoot, "predicates.go"))
	if err != nil {
		return false, false, false
	}
	for _, m := range predDeclRe.FindAllStringSubmatch(string(src), -1) {
		if m[1] != predicate {
			continue
		}
		parts := strings.Split(m[3], ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		subjectAnyRole = len(parts) > 0 && parts[0] == "*Entity"
		objectAnyRole = len(parts) > 1 && parts[1] == "*Entity"
		return subjectAnyRole, objectAnyRole, true
	}
	return false, false, false
}

// resolveEntityRef appends .Entity to a role-typed var name when the
// predicate's slot wants any role rather than a specific named one. Every
// real corpus or store entity is declared through a role wrapper (Concept,
// Hypothesis, Person, ...) that embeds *Entity — never as a bare *Entity —
// so the slot type alone decides whether the field access is needed; no
// need to resolve the ref's own type. A ref that already names a field
// (contains ".") is left alone, so a caller that already knows to write
// "X.Entity" is never double-suffixed.
//
// Without this, winze_link(relation="Supersedes") — the tool's own
// advertised use — failed the build gate on every ordinary (role-typed)
// memory pair, because winze-add's --subject/--object flags reached
// renderClaim as bare var names with no type-directed resolution at all.
func resolveEntityRef(ref string, anyRole bool) string {
	if ref == "" || !anyRole || strings.Contains(ref, ".") {
		return ref
	}
	return ref + ".Entity"
}

// predDeclRe matches a predicate's generic instantiation in predicates.go,
// e.g. `type Supersedes BinaryRelation[*Entity, *Entity]`. Same pattern
// cmd/metabolism/sharedprefix.go's predDeclRe uses to extract the predicate
// vocabulary for the shared prompt prefix — this is the same declaration
// shape read for a different purpose, not a coincidence to dedup across
// packages that otherwise share nothing.
var predDeclRe = regexp.MustCompile(`type (\w+) (BinaryRelation|UnaryClaim)\[([^\]]+)\]`)
