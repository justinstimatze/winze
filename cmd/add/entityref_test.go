package main

import (
	"os"
	"path/filepath"
	"testing"
)

const anyRoleFixture = `package corpus

type Supersedes BinaryRelation[*Entity, *Entity]
type TheoryOf BinaryRelation[Hypothesis, Concept]
type IsCognitiveBias UnaryClaim[Concept]
`

func TestAnyRoleSlots(t *testing.T) {
	dir := writePredicatesFixture(t, anyRoleFixture)

	subj, obj, ok := anyRoleSlots(dir, "Supersedes")
	if !ok || !subj || !obj {
		t.Errorf("Supersedes: got (subj=%v obj=%v ok=%v), want (true, true, true)", subj, obj, ok)
	}

	subj, obj, ok = anyRoleSlots(dir, "TheoryOf")
	if !ok || subj || obj {
		t.Errorf("TheoryOf (named roles, not *Entity): got (subj=%v obj=%v ok=%v), want (false, false, true)", subj, obj, ok)
	}

	subj, obj, ok = anyRoleSlots(dir, "IsCognitiveBias")
	if !ok || subj || obj {
		t.Errorf("IsCognitiveBias (UnaryClaim, single slot): got (subj=%v obj=%v ok=%v), want (false, false, true)", subj, obj, ok)
	}

	if _, _, ok := anyRoleSlots(dir, "NoSuchPredicate"); ok {
		t.Error("unknown predicate: want ok=false, not a false positive")
	}

	if _, _, ok := anyRoleSlots(filepath.Join(dir, "does-not-exist"), "Supersedes"); ok {
		t.Error("missing predicates.go: want ok=false (fail open, let the build gate be the real check)")
	}
}

func TestResolveEntityRef(t *testing.T) {
	cases := []struct {
		name    string
		ref     string
		anyRole bool
		want    string
	}{
		{"named-role slot leaves a bare var untouched", "NewDecision", false, "NewDecision"},
		{"any-role slot appends .Entity to a bare var", "NewDecision", true, "NewDecision.Entity"},
		{"any-role slot does not double-suffix an explicit field access", "NewDecision.Entity", true, "NewDecision.Entity"},
		{"empty ref is left empty regardless (--unary omits Object)", "", true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveEntityRef(tc.ref, tc.anyRole); got != tc.want {
				t.Errorf("resolveEntityRef(%q, %v) = %q, want %q", tc.ref, tc.anyRole, got, tc.want)
			}
		})
	}
}

func writePredicatesFixture(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "predicates.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
