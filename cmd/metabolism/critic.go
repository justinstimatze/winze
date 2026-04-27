package main

// Adversarial post-extraction critic — Layer B / C of the 2026-04-27
// quality-bar overhaul.
//
// Layer A (prompt tightening) catches obvious slot-filler failures at
// generation time, but it cannot catch every drift mode (CommentaryOn
// semantic abuse, TheoryOf-as-analogy hedge, slot-filler entities the
// LLM rationalises away). The critic runs AFTER structural gates and
// BEFORE corpus emission. It rejects candidates that fail rubric checks
// against a sample of existing high-quality claims.
//
// Cost: one Haiku call per candidate that passes structural gates
// (typically 1-3 per cycle). ~$0.005 per call; ~$0.01-0.02 added per
// full --evolve cycle. The reduction in bad-quality commits more than
// pays back in human review time saved.

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/justinstimatze/winze/internal/astutil"
)

// criticVerdict is the parsed result of a single critic call.
type criticVerdict struct {
	Accept bool
	Reason string // populated only on reject
}

// claimExemplar is one high-quality reference claim sampled from the
// existing corpus, used as a quality-bar anchor for critic prompts.
type claimExemplar struct {
	Subject   string
	Predicate string
	Object    string
	Quote     string
}

// sampleHighQualityClaims walks the corpus and returns up to n random
// claim+provenance pairs where the Provenance.Quote is at least
// minQuoteChars long (filters out thin claims that wouldn't make a
// useful exemplar). Sampled deterministically per-process for cycle
// reproducibility while still varying across cycles.
func sampleHighQualityClaims(dir string, n int, minQuoteChars int) []claimExemplar {
	var pool []claimExemplar

	claimTypes := map[string]bool{
		"Proposes":           true,
		"Disputes":           true,
		"ProposesOrg":        true,
		"DisputesOrg":        true,
		"Accepts":            true,
		"AcceptsOrg":         true,
		"TheoryOf":           true,
		"HypothesisExplains": true,
		"BelongsTo":          true,
		"DerivedFrom":        true,
		"Authored":           true,
		"CommentaryOn":       true,
		"InfluencedBy":       true,
	}

	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	// First pass: collect all Provenance{...} vars with their Quote text.
	provQuotes := map[string]string{}
	// Second pass: collect claim vars and their Subject/Object/Prov fields.
	type rawClaim struct {
		predicate string
		subject   string
		object    string
		prov      string
	}
	var rawClaims []rawClaim

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		// Skip metabolism_cycle*.go — these are the very files we just
		// emitted. Including them would conflate "what's in the corpus"
		// with "what the polecat just produced" and weaken the exemplar
		// quality bar.
		if strings.HasPrefix(e.Name(), "metabolism_cycle") {
			continue
		}
		filePath := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, filePath, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, nameIdent := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					cl, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					typeName := compositeTypeName(cl)

					if typeName == "Provenance" {
						quote := astutil.ExtractStringField(cl, "Quote")
						if quote != "" {
							provQuotes[nameIdent.Name] = quote
						}
						continue
					}
					if claimTypes[typeName] {
						subj := extractClaimIdentField(cl, "Subject")
						obj := extractClaimIdentField(cl, "Object")
						prov := extractClaimIdentField(cl, "Prov")
						rawClaims = append(rawClaims, rawClaim{
							predicate: typeName,
							subject:   subj,
							object:    obj,
							prov:      prov,
						})
					}
				}
			}
		}
	}

	// Resolve claim vars to exemplars where the Quote is long enough.
	for _, rc := range rawClaims {
		quote, ok := provQuotes[rc.prov]
		if !ok || len(quote) < minQuoteChars {
			continue
		}
		// Skip speculative trip-cycle provenance — the whole point of
		// exemplars is to anchor on grounded claims, not on prior
		// speculative output.
		if strings.Contains(quote, "speculative cross-cluster connection") {
			continue
		}
		pool = append(pool, claimExemplar{
			Subject:   rc.subject,
			Predicate: rc.predicate,
			Object:    rc.object,
			Quote:     quote,
		})
	}

	if len(pool) == 0 {
		return nil
	}

	// Shuffle + take first n. New rand source per call so successive
	// cycles get different exemplars; the rubric is robust to which
	// exemplars happen to land, but variety helps the critic avoid
	// pattern-matching on a fixed set.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}

// extractClaimIdentField pulls an identifier-valued field from a
// composite literal by name. Returns "" if not present or not an Ident.
// (astutil.ExtractStringField covers the string case; this mirror-helper
// covers the Ident case used for Subject/Object/Prov fields. Named
// distinct from the package's other helpers to avoid the redeclaration
// collision with dream.go's extractStringField shim.)
func extractClaimIdentField(cl *ast.CompositeLit, fieldName string) string {
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != fieldName {
			continue
		}
		if id, ok := kv.Value.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// formatExemplars renders the sampled exemplars as a prompt-friendly block.
func formatExemplars(exemplars []claimExemplar) string {
	if len(exemplars) == 0 {
		return "(no exemplars sampled)\n"
	}
	var b strings.Builder
	for i, ex := range exemplars {
		b.WriteString(fmt.Sprintf("Exemplar %d:\n", i+1))
		b.WriteString(fmt.Sprintf("  Subject: %s\n", ex.Subject))
		b.WriteString(fmt.Sprintf("  Predicate: %s\n", ex.Predicate))
		if ex.Object != "" {
			b.WriteString(fmt.Sprintf("  Object: %s\n", ex.Object))
		}
		b.WriteString(fmt.Sprintf("  Quote: %s\n\n", truncateQuote(ex.Quote, 600)))
	}
	return b.String()
}

func truncateQuote(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + " […]"
}

// critiqueIngestClaim runs the post-extraction critic on a single
// extracted claim. Returns ACCEPT (true, "") or REJECT (false, reason).
// Errors fall back to ACCEPT — the critic is best-effort quality, not
// a hard correctness gate, so an LLM hiccup must not block legitimate
// extractions.
func critiqueIngestClaim(client anthropic.Client, candidate *ingestResult, predicate, target string, exemplars []claimExemplar) criticVerdict {
	prompt := buildIngestCriticPrompt(candidate, predicate, target, exemplars)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 256,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[critic] ingest critic error: %v — defaulting to ACCEPT\n", err)
		return criticVerdict{Accept: true}
	}
	recordActualUsage(string(anthropic.ModelClaudeHaiku4_5), resp.Usage.InputTokens, resp.Usage.CacheReadInputTokens, resp.Usage.OutputTokens)

	for _, block := range resp.Content {
		if block.Type == "text" {
			return parseCriticVerdict(block.Text)
		}
	}
	return criticVerdict{Accept: true}
}

// critiqueTripConnection runs the critic on a single trip-promoted
// speculative connection. Same accept-on-error semantics as ingest.
func critiqueTripConnection(client anthropic.Client, conn TripConnection, exemplars []claimExemplar) criticVerdict {
	prompt := buildTripCriticPrompt(conn, exemplars)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 256,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[critic] trip critic error: %v — defaulting to ACCEPT\n", err)
		return criticVerdict{Accept: true}
	}
	recordActualUsage(string(anthropic.ModelClaudeHaiku4_5), resp.Usage.InputTokens, resp.Usage.CacheReadInputTokens, resp.Usage.OutputTokens)

	for _, block := range resp.Content {
		if block.Type == "text" {
			return parseCriticVerdict(block.Text)
		}
	}
	return criticVerdict{Accept: true}
}

// parseCriticVerdict reads "VERDICT: ACCEPT" or "VERDICT: REJECT\nREASON: ..."
// from the critic response. Tolerates extra whitespace and casing.
func parseCriticVerdict(text string) criticVerdict {
	text = strings.TrimSpace(text)
	upper := strings.ToUpper(text)
	if strings.Contains(upper, "VERDICT: ACCEPT") || strings.Contains(upper, "VERDICT:ACCEPT") {
		return criticVerdict{Accept: true}
	}
	if strings.Contains(upper, "VERDICT: REJECT") || strings.Contains(upper, "VERDICT:REJECT") {
		// Pull the REASON line if present.
		reason := ""
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToUpper(line), "REASON:") {
				reason = strings.TrimSpace(line[len("REASON:"):])
				break
			}
		}
		if reason == "" {
			reason = "unspecified"
		}
		// Sanitize for use in pipeline reason field (no spaces, no
		// special chars — match the existing Reason convention).
		reason = strings.ReplaceAll(reason, " ", "_")
		reason = strings.ToLower(reason)
		if len(reason) > 60 {
			reason = reason[:60]
		}
		return criticVerdict{Accept: false, Reason: reason}
	}
	// Ambiguous response — default to ACCEPT to avoid false negatives
	// from a malformed critic call.
	return criticVerdict{Accept: true}
}

func buildIngestCriticPrompt(candidate *ingestResult, predicate, target string, exemplars []claimExemplar) string {
	var b strings.Builder
	b.WriteString(`You are a quality-bar enforcer for a typed knowledge base about the epistemology of minds — how minds (human and artificial) build, validate, and fail at modeling reality. Reject candidate claims that don't meet the corpus quality bar.

# Existing high-quality claims (the bar to match)

`)
	b.WriteString(formatExemplars(exemplars))
	b.WriteString(`# Candidate claim under review

Subject: `)
	b.WriteString(candidate.entityName)
	b.WriteString(" (")
	b.WriteString(candidate.entityKind)
	b.WriteString(")\nSubject brief: ")
	b.WriteString(candidate.entityBrief)
	b.WriteString("\nPredicate: ")
	b.WriteString(predicate)
	b.WriteString("\nObject: ")
	b.WriteString(target)
	b.WriteString("\nQuote: ")
	b.WriteString(truncateQuote(candidate.quote, 800))
	b.WriteString(`

# Rubric (REJECT if ANY of these fails)

1. NAMED-IN-QUOTE-AS-AGENT. The Quote must contain the Subject's name AND must explicitly attribute the predicate to them. Quote citing the Subject (e.g. "X is well-documented (Smith 2020)") is NOT attribution; that's a citation, not an agency claim.

2. PREDICATE-FIT. The predicate's semantic preconditions match the Quote's content:
   - Proposes/Disputes/Accepts: Subject is the named agent of the position
   - AcceptsOrg / ProposesOrg / DisputesOrg / AuthoredOrg: Subject is an institutional body that takes positions (university, scientific society, research lab, government body) — NOT a publishing platform / journal venue (Cambridge Core, Springer, BBS-as-platform, arXiv, Wikipedia)
   - CommentaryOn: Subject is a creative work that EXPLICITLY references Object work
   - TheoryOf: Subject is a structural account whose explanans is Object Concept
   - IsCognitiveBias: Subject is a documented bias in the Tversky-Kahneman tradition (not a logical fallacy / philosophical category-error / discursive practice)

3. NOT-A-DUPLICATE-RENAMING. The Subject is not just a renamed version of an existing canonical entity. ("ChybaEtAl" when "Chyba" exists is a duplicate.)

4. QUALITY-PARITY-WITH-EXEMPLARS. Compared to the exemplars above, this claim is of comparable substantive quality — not thin, not paraphrased, not extracted from a navigation page or editorial blurb.

# Response

Respond in EXACTLY this format:

VERDICT: ACCEPT

or

VERDICT: REJECT
REASON: <short phrase, e.g. "subject_not_named_in_quote", "predicate_misuse_acceptsorg_for_venue", "duplicate_of_canonical_chyba", "quality_below_exemplar_bar">
`)
	return b.String()
}

func buildTripCriticPrompt(conn TripConnection, exemplars []claimExemplar) string {
	var b strings.Builder
	b.WriteString(`You are a quality-bar enforcer for speculative cross-cluster trip connections in a typed knowledge base about the epistemology of minds. Trip connections are deliberately speculative — but they must still identify a SUBSTANTIVE structural isomorphism, not surface analogy or pattern-matching on shared vocabulary.

# Existing high-quality claims (substance bar to match — note: these are non-speculative; speculative claims must still feel as solid as these)

`)
	b.WriteString(formatExemplars(exemplars))
	b.WriteString(`# Candidate trip connection

Subject: `)
	b.WriteString(conn.EntityA)
	b.WriteString("\nObject: ")
	b.WriteString(conn.EntityB)
	b.WriteString("\nPredicate: ")
	b.WriteString(conn.Predicate)
	b.WriteString("\nLLM rationale (Quote): ")
	b.WriteString(truncateQuote(conn.Rationale, 800))
	b.WriteString(`

# Rubric (REJECT if ANY of these fails)

1. SUBSTANTIVE-ISOMORPHISM. The connection identifies a SPECIFIC shared mechanism, failure mode, or epistemic structure — not a generic "both X" framing. "Both extract patterns" / "both involve prediction" / "both are about how minds work" are SHALLOW and should be rejected.

2. PREDICATE-PRECONDITIONS:
   - CommentaryOn requires Subject to be a creative work that EXPLICITLY references Object — NOT just a structural analogy.
   - TheoryOf requires Subject Hypothesis to be a structural account whose explanans is Object Concept — NOT analogy or "operates similarly."
   - HypothesisExplains requires actual mechanistic explanation — NOT speculative resemblance.
   - BelongsTo / DerivedFrom require sub-classing or derivation — NOT analogy.

3. CATEGORY-FIT. The two entities aren't from category-incompatible domains where the connection is a category error. (E.g. "Searle's Chinese Room CommentaryOn Advaita nondualism" conflates syntax/semantics dualism with subject/object metaphysical dualism.)

4. QUALITY-PARITY-WITH-EXEMPLARS. The connection's rationale is at least as substantive as the exemplars' Quote text — concrete mechanism, falsifiable claim, real domain insight.

# Response

Respond in EXACTLY this format:

VERDICT: ACCEPT

or

VERDICT: REJECT
REASON: <short phrase, e.g. "shallow_pattern_matching", "predicate_misuse_commentaryon_no_paper", "category_mismatch_dualism", "below_exemplar_substance_bar">
`)
	return b.String()
}
