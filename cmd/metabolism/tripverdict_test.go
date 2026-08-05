package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendTripVerdicts_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	rows := []tripVerdictRow{
		{
			Subject: "HilbertsProgram", Object: "BaloneyDetectionKitThesis",
			SubjectRole: "Hypothesis", ObjectRole: "Hypothesis",
			ClusterA: 3, ClusterB: 7,
			Predicate:  "StructurallyAnalogousTo",
			Connection: "both are finite decision procedures for unbounded claim spaces",
			Rationale:  "each proposes a mechanical test that terminates on any input",
			Score:      4, PromptType: "analogy", Temperature: 0.9,
			Accept: false, Reason: "shallow_pattern_matching",
			RawReason:   "shallow pattern matching disguised as isomorphism",
			CriticModel: "claude-haiku-4-5",
		},
		{
			Subject: "KahnemanDualProcessFraming", Object: "ConradApopheniaClinicalFraming",
			ClusterA: 1, ClusterB: 4,
			Predicate:  "StructurallyAnalogousTo",
			Connection: "shared two-system failure mode",
			Score:      4, PromptType: "analogy", Temperature: 0.9,
			Accept: true, CriticModel: "claude-haiku-4-5",
		},
	}

	if n := appendTripVerdicts(dir, rows); n != 2 {
		t.Fatalf("appended %d rows, want 2", n)
	}
	// A second call must append, not truncate — a cycle's rulings
	// accumulate across runs into one labelling sample.
	if n := appendTripVerdicts(dir, rows[:1]); n != 1 {
		t.Fatalf("second append wrote %d rows, want 1", n)
	}

	got, err := readTripVerdicts(dir)
	if err != nil {
		t.Fatalf("readTripVerdicts: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("read %d rows, want 3 (2 appended + 1)", len(got))
	}

	first := got[0]
	if first.Timestamp == "" {
		t.Error("timestamp not filled on write")
	}
	if first.Accept {
		t.Error("first row should be a reject")
	}
	// The verbatim reason is the whole point: the sanitized Reason is a
	// slug, and a slug cannot be audited.
	if first.RawReason != "shallow pattern matching disguised as isomorphism" {
		t.Errorf("raw_reason not round-tripped: %q", first.RawReason)
	}
	if first.Rationale == "" || first.Connection == "" {
		t.Error("candidate argument dropped; the row cannot be hand-labelled without it")
	}
	if first.SubjectRole != "Hypothesis" || first.ObjectRole != "Hypothesis" {
		t.Errorf("roles not round-tripped: %q/%q", first.SubjectRole, first.ObjectRole)
	}
	if !got[1].Accept {
		t.Error("second row should be an accept — accepts must be logged too")
	}

	// The label slot must be present-and-empty in the JSON so the file
	// reads as a worksheet with a blank column, not one with a missing key.
	data, err := os.ReadFile(filepath.Join(dir, tripVerdictsFile))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), `"label":""`) {
		t.Error(`expected an empty "label" field on every row`)
	}
}

func TestAppendTripVerdicts_EmptyIsNoop(t *testing.T) {
	dir := t.TempDir()
	if n := appendTripVerdicts(dir, nil); n != 0 {
		t.Errorf("appended %d rows for nil input, want 0", n)
	}
	if _, err := os.Stat(filepath.Join(dir, tripVerdictsFile)); !os.IsNotExist(err) {
		t.Error("empty append created the file; it should not")
	}
}

func TestParseCriticVerdict_KeepsRawReasonUntruncated(t *testing.T) {
	// A real multi-clause rejection: the sanitized Reason is cut mid-word
	// at 60 chars, which is what made the existing promotion log unusable
	// for judging whether the rejection was correct.
	long := "shallow pattern matching plus predicate misuse StructurallyAnalogousTo lacks shared mechanism"
	v := parseCriticVerdict("VERDICT: REJECT\nREASON: " + long)

	if v.Accept {
		t.Fatal("expected reject")
	}
	if len(v.Reason) != 60 {
		t.Errorf("Reason len = %d, want 60 (truncation still applies)", len(v.Reason))
	}
	if strings.Contains(v.Reason, " ") {
		t.Errorf("Reason not sanitized: %q", v.Reason)
	}
	if v.RawReason != long {
		t.Errorf("RawReason = %q, want the verbatim line", v.RawReason)
	}
}

func TestParseCriticVerdict_AcceptHasNoRawReason(t *testing.T) {
	v := parseCriticVerdict("VERDICT: ACCEPT")
	if !v.Accept {
		t.Fatal("expected accept")
	}
	if v.RawReason != "" {
		t.Errorf("RawReason = %q on accept, want empty", v.RawReason)
	}
}

func TestSlotExpr_AnyRoleSlotTakesEmbeddedEntity(t *testing.T) {
	// StructurallyAnalogousTo was widened to *Entity slots so analogies can
	// cross role boundaries. The generator must follow the declaration or
	// every promotion of it fails the build gate and gets reverted.
	if got := slotExpr("StructurallyAnalogousTo", 0, "FiniteOntologyIncompleteness"); got != "FiniteOntologyIncompleteness.Entity" {
		t.Errorf("subject slot = %q, want the embedded .Entity", got)
	}
	if got := slotExpr("StructurallyAnalogousTo", 1, "HumanRights"); got != "HumanRights.Entity" {
		t.Errorf("object slot = %q, want the embedded .Entity", got)
	}
}

func TestSlotExpr_NamedRoleSlotTakesBareVar(t *testing.T) {
	if got := slotExpr("InfluencedBy", 0, "Kahneman"); got != "Kahneman" {
		t.Errorf("Person slot = %q, want the bare var", got)
	}
	if got := slotExpr("InfluencedBy", 1, "Tversky"); got != "Tversky" {
		t.Errorf("Person slot = %q, want the bare var", got)
	}
}

func TestSlotExpr_UnknownPredicateFallsThrough(t *testing.T) {
	if got := slotExpr("NotAPredicate", 0, "X"); got != "X" {
		t.Errorf("unknown predicate = %q, want the bare var", got)
	}
}

// TestSlotExpr_CoversEveryAnyRoleSlot guards the generator against the next
// widening: any predicate declared over *Entity must render .Entity on that
// side. Without this, widening a second predicate reintroduces the exact
// build-gate revert that slotExpr was written to fix.
func TestSlotExpr_CoversEveryAnyRoleSlot(t *testing.T) {
	for pred, slots := range predicateSlots {
		if slots.Subject == anyRole {
			if got := slotExpr(pred, 0, "V"); got != "V.Entity" {
				t.Errorf("%s subject slot is %s but rendered %q", pred, anyRole, got)
			}
		}
		if slots.Object == anyRole {
			if got := slotExpr(pred, 1, "V"); got != "V.Entity" {
				t.Errorf("%s object slot is %s but rendered %q", pred, anyRole, got)
			}
		}
	}
}
