// Command add appends a typed claim declaration to a winze corpus file
// and verifies the result against the build gate. The tool does no
// slot-type checking of its own — that is delegated to `go build .`
// (the root corpus package), which is the load-bearing consistency
// check for typed claim authoring. On build failure the target file is
// restored to its prior contents and the tool exits non-zero with the
// build output for diagnosis.
//
// The gate is `go build .` (corpus only), not `go build ./...`, because
// cmd subtrees can pull transitive cgo deps (dolthub/go-icu-regex needs
// libicu headers) that fail in environments where the corpus itself
// builds cleanly. Validating the corpus is the load-bearing check; cmd
// validation belongs to the cmd's own tests.
//
// MVP scope (wi-wvvi Phase 1):
//   - inline Provenance literal at the call site; named-source reuse is
//     a 1b improvement (--provenance-var) and not in this cut.
//   - explicit --to <file>; auto-routing by subject location is 1b.
//   - explicit --name <ClaimVar>; auto-naming is 1b.
//
// Example:
//
//	go run ./cmd/add \
//	    --to apophenia.go \
//	    --name MyShermerBeliefClaim \
//	    --predicate Accepts \
//	    --subject MichaelShermer \
//	    --object ShermerPatternicityFraming \
//	    --quote "Shermer himself accepts the patternicity framing." \
//	    --origin "fictional test source"
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/justinstimatze/winze/internal/corpuslock"
	"github.com/justinstimatze/winze/internal/usagelog"
)

func main() {
	var (
		predicate  = flag.String("predicate", "", "predicate type name (e.g. Proposes, TheoryOf)")
		subject    = flag.String("subject", "", "subject entity var name (e.g. KlausConrad)")
		object     = flag.String("object", "", "object entity var name (omit for --unary predicates)")
		quote      = flag.String("quote", "", "exact source quote (required unless --provenance-var is set)")
		origin     = flag.String("origin", "", "free-form provenance origin hint (required unless --provenance-var is set)")
		ingestedBy = flag.String("ingested-by", "winze-add", "ingestor tag for inline Provenance.IngestedBy (ignored when --provenance-var is set)")
		provVar    = flag.String("provenance-var", "", "name of an existing Provenance var to reuse (e.g. apopheniaSource); mutually exclusive with --quote/--origin/--ingested-by")
		target     = flag.String("to", "", "target corpus file (relative to --root, e.g. apophenia.go)")
		claimName  = flag.String("name", "", "Go var name for the new claim")
		repoRoot   = flag.String("root", ".", "winze repo root (the directory containing predicates.go)")
		unary      = flag.Bool("unary", false, "set for UnaryClaim predicates (omit --object)")
		dryRun     = flag.Bool("dry-run", false, "print what would be written; do not modify files or build")
		batch      = flag.String("batch", "", "append many claims from a JSONL file (or '-' for stdin) under one build gate; ignores the single-claim flags")
		propose    = flag.String("propose", "", "rough natural-language note; an LLM proposes the typed claim (predicate/subject/object) from the existing vocabulary, then the normal gate validates it (needs ANTHROPIC_API_KEY)")
		commit     = flag.Bool("commit", false, "with --propose: actually write the proposed claim through the build gate (default: show the proposal only)")
		model      = flag.String("model", "", "with --propose: model override (default Claude Haiku 4.5)")
		entity     = flag.Bool("entity", false, "create a new entity instead of a claim (needs --role and --brief; the creation primitive claims lack)")
		role       = flag.String("role", "", "with --entity: role type (e.g. Concept, Person, Hypothesis)")
		brief      = flag.String("brief", "", "with --entity: the entity's Brief prose")
		aliases    = flag.String("aliases", "", "with --entity: comma-separated aliases")
		kind       = flag.String("kind", "", "with --entity: Kind field (defaults to lowercased role)")
	)
	flag.Parse()
	start := time.Now()

	// Entity mode: create a new typed entity (the primitive the claim path
	// lacks). --name is optional here — a var name is generated from the brief.
	if *entity {
		code := runEntity(entityOpts{
			role: *role, name: *claimName, brief: *brief, kind: *kind, aliases: *aliases,
			target: *target, repoRoot: *repoRoot, dryRun: *dryRun,
		})
		usagelog.Log(*repoRoot, "add-entity", os.Args[1:], start)
		os.Exit(code)
	}

	// Batch mode is the burst-write path: K claims, one ~91ms gate. It takes
	// its own JSONL input, so the single-claim required-flag validation below
	// does not apply.
	if *batch != "" {
		code := runBatch(*batch, *repoRoot, *dryRun)
		usagelog.Log(*repoRoot, "add-batch", os.Args[1:], start)
		os.Exit(code)
	}

	// Propose mode is the human-via-agent write path: a rough note becomes a
	// typed claim proposal (LLM maps it onto the existing predicate/entity
	// vocabulary); the same build gate validates it. Provenance is never
	// invented — --quote/--origin or --provenance-var still supply the source.
	if *propose != "" {
		code := runPropose(proposeOpts{
			note: *propose, quote: *quote, origin: *origin, ingestedBy: *ingestedBy,
			provVar: *provVar, target: *target, model: *model, repoRoot: *repoRoot,
			commit: *commit && !*dryRun,
		})
		usagelog.Log(*repoRoot, "add-propose", os.Args[1:], start)
		os.Exit(code)
	}
	defer usagelog.Log(*repoRoot, "add", os.Args[1:], start)

	if err := validateFlags(*predicate, *subject, *object, *quote, *origin, *provVar, *target, *claimName, *unary); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		flag.Usage()
		os.Exit(2)
	}

	decl := renderClaim(*predicate, *subject, *object, *quote, *origin, *ingestedBy, *provVar, *claimName, *unary)

	if *dryRun {
		fmt.Printf("--- would append to %s ---\n", *target)
		fmt.Println(decl)
		return
	}

	if err := commitDecl(*repoRoot, *target, decl); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "added %s to %s (build gate passed)\n", *claimName, *target)
}

// commitDecl is the shared write path for `add` and `--propose --commit`: it
// takes the corpus-wide lock, appends decl to target, runs the
// gofmt+build+vet gate, and reverts the touched file on any failure. The build
// gate is the load-bearing semantic check — do not bypass it.
func commitDecl(repoRoot, target, decl string) error {
	// Serialize with any concurrent winze writer: the read-append-gate-commit
	// section is not safe against a parallel mutator sharing this corpus.
	unlock, err := corpuslock.Acquire(repoRoot)
	if err != nil {
		return fmt.Errorf("corpus lock: %w", err)
	}
	defer unlock()

	targetPath := filepath.Join(repoRoot, target)
	backup, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", targetPath, err)
	}
	revert := func() { _ = os.WriteFile(targetPath, backup, 0o644) }

	if err := appendDecl(targetPath, decl); err != nil {
		revert()
		return fmt.Errorf("append failed (reverted): %w", err)
	}
	if out, err := runCmd(repoRoot, "gofmt", "-w", targetPath); err != nil {
		revert()
		return fmt.Errorf("gofmt failed (reverted):\n%s", out)
	}
	if out, err := runCmd(repoRoot, "go", "build", "."); err != nil {
		revert()
		return fmt.Errorf("go build failed (reverted %s):\n%s", targetPath, out)
	}
	if out, err := runCmd(repoRoot, "go", "vet", "."); err != nil {
		revert()
		return fmt.Errorf("go vet failed (reverted %s):\n%s", targetPath, out)
	}
	return nil
}

func validateFlags(predicate, subject, object, quote, origin, provVar, target, claimName string, unary bool) error {
	if provVar != "" {
		// Reusing a named Provenance var — inline-source flags must NOT be set.
		if quote != "" || origin != "" {
			return fmt.Errorf("--provenance-var is mutually exclusive with --quote / --origin (pick one source mode)")
		}
		if !isValidGoIdent(provVar) {
			return fmt.Errorf("--provenance-var %q is not a valid Go identifier", provVar)
		}
	}
	missing := []string{}
	if predicate == "" {
		missing = append(missing, "--predicate")
	}
	if subject == "" {
		missing = append(missing, "--subject")
	}
	if provVar == "" {
		// Inline-source mode — quote and origin are mandatory.
		if quote == "" {
			missing = append(missing, "--quote")
		}
		if origin == "" {
			missing = append(missing, "--origin")
		}
	}
	if target == "" {
		missing = append(missing, "--to")
	}
	if claimName == "" {
		missing = append(missing, "--name")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required flags: %s", strings.Join(missing, ", "))
	}
	if !unary && object == "" {
		return fmt.Errorf("--object required for binary predicates (or set --unary)")
	}
	if unary && object != "" {
		return fmt.Errorf("--object set but --unary specified; pick one")
	}
	if !isValidGoIdent(claimName) {
		return fmt.Errorf("--name %q is not a valid Go identifier", claimName)
	}
	return nil
}

func isValidGoIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		ok := r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
		if i > 0 {
			ok = ok || (r >= '0' && r <= '9')
		}
		if !ok {
			return false
		}
	}
	return true
}

func renderClaim(predicate, subject, object, quote, origin, ingestedBy, provVar, claimName string, unary bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nvar %s = %s{\n", claimName, predicate)
	fmt.Fprintf(&b, "\tSubject: %s,\n", subject)
	if !unary {
		fmt.Fprintf(&b, "\tObject:  %s,\n", object)
	}
	if provVar != "" {
		// Reuse-mode: the named Provenance var is referenced directly. If
		// it doesn't exist in scope, the build gate will catch it and the
		// file will be reverted — no special validation here.
		fmt.Fprintf(&b, "\tProv:    %s,\n", provVar)
	} else {
		today := time.Now().UTC().Format("2006-01-02")
		fmt.Fprintf(&b, "\tProv: Provenance{\n")
		fmt.Fprintf(&b, "\t\tOrigin:     %s,\n", strconv.Quote(origin))
		fmt.Fprintf(&b, "\t\tIngestedAt: %s,\n", strconv.Quote(today))
		fmt.Fprintf(&b, "\t\tIngestedBy: %s,\n", strconv.Quote(ingestedBy))
		fmt.Fprintf(&b, "\t\tQuote:      %s,\n", quoteLiteral(quote))
		fmt.Fprintf(&b, "\t},\n")
	}
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

// quoteLiteral picks a Go string literal that preserves the quote text
// readably. Raw-string when possible (preserves newlines and avoids
// escaping); falls back to strconv.Quote when the text contains a
// backtick (which raw strings cannot escape).
func quoteLiteral(q string) string {
	if strings.Contains(q, "`") {
		return strconv.Quote(q)
	}
	return "`" + q + "`"
}

func appendDecl(path, decl string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(decl); err != nil {
		return err
	}
	return nil
}

func runCmd(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
