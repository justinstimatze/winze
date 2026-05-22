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
)

func main() {
	var (
		predicate  = flag.String("predicate", "", "predicate type name (e.g. Proposes, TheoryOf)")
		subject    = flag.String("subject", "", "subject entity var name (e.g. KlausConrad)")
		object     = flag.String("object", "", "object entity var name (omit for --unary predicates)")
		quote      = flag.String("quote", "", "exact source quote (mandatory; post-2026-04-13 corpus discipline)")
		origin     = flag.String("origin", "", "free-form provenance origin hint (mandatory)")
		ingestedBy = flag.String("ingested-by", "winze-add", "ingestor tag for Provenance.IngestedBy")
		target     = flag.String("to", "", "target corpus file (relative to --root, e.g. apophenia.go)")
		claimName  = flag.String("name", "", "Go var name for the new claim")
		repoRoot   = flag.String("root", ".", "winze repo root (the directory containing predicates.go)")
		unary      = flag.Bool("unary", false, "set for UnaryClaim predicates (omit --object)")
		dryRun     = flag.Bool("dry-run", false, "print what would be written; do not modify files or build")
	)
	flag.Parse()

	if err := validateFlags(*predicate, *subject, *object, *quote, *origin, *target, *claimName, *unary); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		flag.Usage()
		os.Exit(2)
	}

	decl := renderClaim(*predicate, *subject, *object, *quote, *origin, *ingestedBy, *claimName, *unary)

	if *dryRun {
		fmt.Printf("--- would append to %s ---\n", *target)
		fmt.Println(decl)
		return
	}

	targetPath := filepath.Join(*repoRoot, *target)
	backup, err := os.ReadFile(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", targetPath, err)
		os.Exit(1)
	}

	if err := appendDecl(targetPath, decl); err != nil {
		_ = os.WriteFile(targetPath, backup, 0o644)
		fmt.Fprintf(os.Stderr, "append failed (reverted): %v\n", err)
		os.Exit(1)
	}

	if out, err := runCmd(*repoRoot, "gofmt", "-w", targetPath); err != nil {
		_ = os.WriteFile(targetPath, backup, 0o644)
		fmt.Fprintf(os.Stderr, "gofmt failed (reverted):\n%s\n", out)
		os.Exit(1)
	}

	if out, err := runCmd(*repoRoot, "go", "build", "."); err != nil {
		_ = os.WriteFile(targetPath, backup, 0o644)
		fmt.Fprintf(os.Stderr, "go build failed (reverted %s):\n%s\n", targetPath, out)
		os.Exit(1)
	}
	if out, err := runCmd(*repoRoot, "go", "vet", "."); err != nil {
		_ = os.WriteFile(targetPath, backup, 0o644)
		fmt.Fprintf(os.Stderr, "go vet failed (reverted %s):\n%s\n", targetPath, out)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "added %s to %s (build gate passed)\n", *claimName, *target)
}

func validateFlags(predicate, subject, object, quote, origin, target, claimName string, unary bool) error {
	missing := []string{}
	if predicate == "" {
		missing = append(missing, "--predicate")
	}
	if subject == "" {
		missing = append(missing, "--subject")
	}
	if quote == "" {
		missing = append(missing, "--quote")
	}
	if origin == "" {
		missing = append(missing, "--origin")
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

func renderClaim(predicate, subject, object, quote, origin, ingestedBy, claimName string, unary bool) string {
	today := time.Now().UTC().Format("2006-01-02")
	var b strings.Builder
	fmt.Fprintf(&b, "\nvar %s = %s{\n", claimName, predicate)
	fmt.Fprintf(&b, "\tSubject: %s,\n", subject)
	if !unary {
		fmt.Fprintf(&b, "\tObject:  %s,\n", object)
	}
	fmt.Fprintf(&b, "\tProv: Provenance{\n")
	fmt.Fprintf(&b, "\t\tOrigin:     %s,\n", strconv.Quote(origin))
	fmt.Fprintf(&b, "\t\tIngestedAt: %s,\n", strconv.Quote(today))
	fmt.Fprintf(&b, "\t\tIngestedBy: %s,\n", strconv.Quote(ingestedBy))
	fmt.Fprintf(&b, "\t\tQuote:      %s,\n", quoteLiteral(quote))
	fmt.Fprintf(&b, "\t},\n")
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
