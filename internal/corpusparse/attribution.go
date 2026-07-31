package corpusparse

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// Provenance is a sourced attribution record: `var fooSource = Provenance{...}`.
// It mirrors winze.Provenance minus the type-system guarantees — this package
// AST-scrapes rather than type-checks, so a field the walker cannot resolve to
// a literal (early corpus vars built by string concatenation) comes back empty
// rather than wrong.
type Provenance struct {
	VarName    string
	Origin     string
	IngestedAt string
	IngestedBy string
	Quote      string
	File       string
}

// Conjecture is winze's OWN generation backing a claim — the uncitable half of
// the Attribution sum. It has no Quote field here for the same reason it has
// none in schema.go: a generated claim must not be able to wear a source.
// Callers that project the corpus outward (cmd/okf) rely on that absence to
// keep generated content out of source lists structurally, not by convention.
type Conjecture struct {
	GeneratedBy      string
	From             []string // entity var names the conjecture was generated from
	CycleN           int
	Temperature      float64
	PromptType       string
	Score            int
	Rationale        string
	GeneratedAt      string
	GeneratedByAgent string
}

// Corpus is the full parse of a corpus directory: entities, claims, and the
// provenance vars claims attribute to. ParseCorpus returns the first two for
// the callers that only need shape; tools that need the epistemic backing of
// each claim (export, audit) want this.
type Corpus struct {
	Entities   []Entity
	Claims     []Claim
	Provenance []Provenance
}

// ProvenanceByVar indexes the corpus's provenance records by var name, the key
// Claim.ProvVar holds.
func (c *Corpus) ProvenanceByVar() map[string]Provenance {
	m := make(map[string]Provenance, len(c.Provenance))
	for _, p := range c.Provenance {
		m[p.VarName] = p
	}
	return m
}

// EntityByVar indexes the corpus's entities by var name, the key
// Claim.SubjectVar and Claim.ObjectVar hold.
func (c *Corpus) EntityByVar() map[string]Entity {
	m := make(map[string]Entity, len(c.Entities))
	for _, e := range c.Entities {
		m[e.VarName] = e
	}
	return m
}

// tryParseProvenance recognises a top-level provenance var: `Provenance{...}`.
// Checked before the claim shape because a Provenance literal has no Subject
// and would be rejected there anyway — this just names the rejection.
func tryParseProvenance(varName string, cl *ast.CompositeLit, file string) (Provenance, bool) {
	if typeIdent(cl.Type) != "Provenance" {
		return Provenance{}, false
	}
	return provenanceFields(varName, cl, file), true
}

func provenanceFields(varName string, cl *ast.CompositeLit, file string) Provenance {
	p := Provenance{VarName: varName, File: file}
	for _, elt := range cl.Elts {
		key, val, ok := keyValue(elt)
		if !ok {
			continue
		}
		switch key {
		case "Origin":
			p.Origin = stringLit(val)
		case "IngestedAt":
			p.IngestedAt = stringLit(val)
		case "IngestedBy":
			p.IngestedBy = stringLit(val)
		case "Quote":
			p.Quote = stringLit(val)
		}
	}
	return p
}

func conjectureFields(cl *ast.CompositeLit) *Conjecture {
	c := &Conjecture{}
	for _, elt := range cl.Elts {
		key, val, ok := keyValue(elt)
		if !ok {
			continue
		}
		switch key {
		case "GeneratedBy":
			c.GeneratedBy = stringLit(val)
		case "From":
			c.From = identSliceLit(val)
		case "CycleN":
			c.CycleN = intLit(val)
		case "Temperature":
			c.Temperature = floatLit(val)
		case "PromptType":
			c.PromptType = stringLit(val)
		case "Score":
			c.Score = intLit(val)
		case "Rationale":
			c.Rationale = stringLit(val)
		case "GeneratedAt":
			c.GeneratedAt = stringLit(val)
		case "GeneratedByAgent":
			c.GeneratedByAgent = stringLit(val)
		}
	}
	return c
}

// codeRefs reads a carrier's `Refs: []CodeRef{{Path: ..., Note: ...}}` field.
// Returns nil for the ordinary role wrappers, which have no such field.
func codeRefs(cl *ast.CompositeLit) []CodeRef {
	for _, elt := range cl.Elts {
		key, val, ok := keyValue(elt)
		if !ok || key != "Refs" {
			continue
		}
		lit, ok := val.(*ast.CompositeLit)
		if !ok {
			return nil
		}
		var out []CodeRef
		for _, item := range lit.Elts {
			icl, ok := item.(*ast.CompositeLit)
			if !ok {
				continue
			}
			var ref CodeRef
			for _, f := range icl.Elts {
				k, v, ok := keyValue(f)
				if !ok {
					continue
				}
				switch k {
				case "Path":
					ref.Path = stringLit(v)
				case "Note":
					ref.Note = stringLit(v)
				}
			}
			if ref.Path != "" {
				out = append(out, ref)
			}
		}
		return out
	}
	return nil
}

func keyValue(elt ast.Expr) (string, ast.Expr, bool) {
	kv, ok := elt.(*ast.KeyValueExpr)
	if !ok {
		return "", nil, false
	}
	key, ok := kv.Key.(*ast.Ident)
	if !ok {
		return "", nil, false
	}
	return key.Name, kv.Value, true
}

// identSliceLit reads `[]*Entity{Foo.Entity, Bar.Entity}` as the var names in
// each element's head — the same head-of-selector rule query's baseVar uses.
func identSliceLit(e ast.Expr) []string {
	cl, ok := e.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(cl.Elts))
	for _, elt := range cl.Elts {
		expr := elt
		if ue, ok := expr.(*ast.UnaryExpr); ok {
			expr = ue.X
		}
		if n := headIdent(expr); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// headIdent resolves Foo and Foo.Entity alike to "Foo". identName takes the
// selector's tail (Sel), which is right for a claim slot naming a field but
// wrong here, where the entity is the head.
func headIdent(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return headIdent(t.X)
	}
	return ""
}

func intLit(e ast.Expr) int {
	bl, ok := e.(*ast.BasicLit)
	if !ok || bl.Kind != token.INT {
		return 0
	}
	n, err := strconv.Atoi(bl.Value)
	if err != nil {
		return 0
	}
	return n
}

func floatLit(e ast.Expr) float64 {
	bl, ok := e.(*ast.BasicLit)
	if !ok || (bl.Kind != token.FLOAT && bl.Kind != token.INT) {
		return 0
	}
	f, err := strconv.ParseFloat(bl.Value, 64)
	if err != nil {
		return 0
	}
	return f
}

// Actor renders a winze identity in OKF's actor convention: `human:<id>` for
// people, `process:<id>` for automated processes. Every IngestedBy in the
// corpus today is a winze process (the corpus is machine-ingested through the
// build gate), so the default is `process:` and a name is promoted to `human:`
// only when it carries no process marker. Getting this backwards would inflate
// the trust tier of every claim in an exported bundle, which is why the rule
// lives here next to the data rather than in the exporter's formatting code.
func Actor(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "process:winze"
	}
	low := strings.ToLower(id)
	for _, marker := range []string{"winze", "metabolism", "agent", "bot", "process", "sensor", "ingest"} {
		if strings.Contains(low, marker) {
			return "process:" + id
		}
	}
	return "human:" + id
}
