package main

import (
	"strings"
	"testing"
)

func TestValidateFlags(t *testing.T) {
	cases := []struct {
		name      string
		predicate string
		subject   string
		object    string
		quote     string
		origin    string
		provVar   string
		target    string
		claim     string
		unary     bool
		wantErr   string // substring; "" means must succeed
	}{
		{
			name:      "binary inline-source happy path",
			predicate: "Proposes",
			subject:   "KlausConrad",
			object:    "ConradFraming",
			quote:     "q",
			origin:    "o",
			target:    "apophenia.go",
			claim:     "MyClaim",
		},
		{
			name:      "binary provenance-var happy path",
			predicate: "Proposes",
			subject:   "KlausConrad",
			object:    "ConradFraming",
			provVar:   "apopheniaSource",
			target:    "apophenia.go",
			claim:     "MyClaim",
		},
		{
			name:      "unary inline happy path",
			predicate: "IsCognitiveBias",
			subject:   "ClusteringIllusion",
			quote:     "q",
			origin:    "o",
			target:    "apophenia.go",
			claim:     "MyTag",
			unary:     true,
		},
		{
			name:      "provenance-var with inline quote should reject",
			predicate: "Proposes",
			subject:   "X",
			object:    "Y",
			quote:     "q",
			provVar:   "src",
			target:    "f.go",
			claim:     "C",
			wantErr:   "mutually exclusive",
		},
		{
			name:      "provenance-var with bad identifier",
			predicate: "Proposes",
			subject:   "X",
			object:    "Y",
			provVar:   "1bad",
			target:    "f.go",
			claim:     "C",
			wantErr:   "not a valid Go identifier",
		},
		{
			name:      "binary missing object",
			predicate: "Proposes",
			subject:   "X",
			quote:     "q",
			origin:    "o",
			target:    "f.go",
			claim:     "C",
			wantErr:   "--object required",
		},
		{
			name:      "unary with object",
			predicate: "IsCognitiveBias",
			subject:   "X",
			object:    "Y",
			quote:     "q",
			origin:    "o",
			target:    "f.go",
			claim:     "C",
			unary:     true,
			wantErr:   "pick one",
		},
		{
			name:    "missing all",
			wantErr: "missing required flags",
		},
		{
			name:      "inline mode missing quote",
			predicate: "Proposes",
			subject:   "X",
			object:    "Y",
			origin:    "o",
			target:    "f.go",
			claim:     "C",
			wantErr:   "--quote",
		},
		{
			name:      "bad claim name",
			predicate: "Proposes",
			subject:   "X",
			object:    "Y",
			quote:     "q",
			origin:    "o",
			target:    "f.go",
			claim:     "1Bad",
			wantErr:   "not a valid Go identifier",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFlags(tc.predicate, tc.subject, tc.object, tc.quote, tc.origin, tc.provVar, tc.target, tc.claim, tc.unary)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestIsValidGoIdent(t *testing.T) {
	good := []string{"X", "MyClaim", "claim123", "_under", "A1_b2"}
	for _, s := range good {
		if !isValidGoIdent(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	bad := []string{"", "1Bad", "has space", "has-dash", "has.dot"}
	for _, s := range bad {
		if isValidGoIdent(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}

func TestQuoteLiteral(t *testing.T) {
	// no backtick → raw string
	if got := quoteLiteral("plain text"); got != "`plain text`" {
		t.Errorf("plain text: got %q", got)
	}
	// quote with double-quotes inside is fine in raw string
	got := quoteLiteral(`he said "hi"`)
	if got != "`he said \"hi\"`" {
		t.Errorf("double-quote in raw: got %q", got)
	}
	// backtick in quote → fallback to strconv.Quote
	got = quoteLiteral("has ` backtick")
	if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) {
		t.Errorf("backtick text should fall back to escaped string, got %q", got)
	}
}

func TestRenderClaimBinary(t *testing.T) {
	out := renderClaim("Proposes", "KlausConrad", "ConradFraming",
		"the quote", "src/Apophenia", "winze-add", "", "MyClaim", false)
	want := []string{
		"var MyClaim = Proposes{",
		"Subject: KlausConrad,",
		"Object:  ConradFraming,",
		"Prov: Provenance{",
		`Origin:     "src/Apophenia",`,
		`IngestedBy: "winze-add",`,
		"Quote:      `the quote`,",
		"}",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("output missing %q:\n%s", w, out)
		}
	}
	if strings.Contains(out, "Object: ,") {
		t.Errorf("binary render emitted empty Object: %s", out)
	}
}

func TestRenderClaimBinaryWithProvVar(t *testing.T) {
	out := renderClaim("Proposes", "KlausConrad", "ConradFraming",
		"", "", "", "apopheniaSource", "MyClaim", false)
	want := []string{
		"var MyClaim = Proposes{",
		"Subject: KlausConrad,",
		"Object:  ConradFraming,",
		"Prov:    apopheniaSource,",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("output missing %q:\n%s", w, out)
		}
	}
	// Must NOT emit the inline Provenance block when reusing a named var.
	if strings.Contains(out, "Provenance{") {
		t.Errorf("provenance-var mode must not emit inline Provenance{}: %s", out)
	}
}

func TestRenderClaimUnary(t *testing.T) {
	out := renderClaim("IsCognitiveBias", "ClusteringIllusion", "",
		"the quote", "src", "winze-add", "", "MyTag", true)
	if !strings.Contains(out, "var MyTag = IsCognitiveBias{") {
		t.Errorf("missing var decl: %s", out)
	}
	if !strings.Contains(out, "Subject: ClusteringIllusion,") {
		t.Errorf("missing subject: %s", out)
	}
	if strings.Contains(out, "Object:") {
		t.Errorf("unary render must not emit Object: %s", out)
	}
}
