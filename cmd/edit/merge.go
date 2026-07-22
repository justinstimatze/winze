package main

// merge folds entity A into entity B. Every reference to A becomes B, A's
// declaration is removed, and A's claims retarget to B automatically because
// they reference the var — rewriting the var rewrites the claim. A's Brief /
// ID / Name are dropped; B is the canonical survivor (you merge the duplicate
// INTO the canonical one). Claim-level provenance is preserved for free: each
// claim keeps its own Prov and only its Subject/Object identifiers move.
//
// The build gate is the semantic check, not a type analysis in this tool. If
// A and B have incompatible types (A a Person, B a Concept), a claim that read
// `Subject: A` becomes `Subject: B` and fails to type-check — the gate rejects
// it and every touched file reverts. That is the winze discipline: the
// compiler validates the merge, the tool only performs the byte surgery.
//
// This is the compaction primitive for the log-structured multi-session KB
// (docs/multi-session-write-shape.md): rot-probe finds duplicate entities
// coined independently across session files; merge folds them into the
// canonical topic file so the write-ahead log compacts.
//
// NOT recorded as a typed claim yet. A MergedFrom / AlternateOf predicate
// (PROV-O alternateOf) would make the fold auditable and stop a re-ingest of
// A's source from recreating it — but adding a predicate is a schema-accretion
// decision reserved for a human. Until then the git commit is the record.

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/justinstimatze/winze/internal/astutil"
)

// edit is a single byte-range replacement. A deletion is repl == "".
type edit struct {
	offset int
	length int
	repl   string
}

// applyEdits performs every edit on src, back-to-front so earlier offsets do
// not shift under later splices. Overlapping edits are the caller's bug; the
// merge path never produces them because rename sites inside the deleted decl
// range are filtered out before edits are built.
func applyEdits(src []byte, edits []edit) []byte {
	ordered := append([]edit(nil), edits...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].offset > ordered[j].offset })
	out := src
	for _, e := range ordered {
		if e.offset < 0 || e.offset+e.length > len(out) {
			continue
		}
		next := make([]byte, 0, len(out)-e.length+len(e.repl))
		next = append(next, out[:e.offset]...)
		next = append(next, e.repl...)
		next = append(next, out[e.offset+e.length:]...)
		out = next
	}
	return out
}

func cmdMerge(args []string) int {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	from := fs.String("from", "", "entity var to fold away (consumed)")
	into := fs.String("into", "", "canonical entity var that survives")
	root := fs.String("root", ".", "winze repo root (the directory containing predicates.go)")
	dryRun := fs.Bool("dry-run", false, "report what would change; do not modify files")
	noRecord := fs.Bool("no-record", false, "skip appending the AbsorbedAlternate audit claim")
	fs.Parse(args)

	if *from == "" || *into == "" {
		fmt.Fprintln(os.Stderr, "error: --from and --into are required")
		fs.Usage()
		return 2
	}
	if *from == *into {
		fmt.Fprintln(os.Stderr, "error: --from and --into are identical")
		return 2
	}

	plan, err := planMerge(*root, *from, *into)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}
	if !plan.fromDeclared {
		fmt.Fprintf(os.Stderr, "error: no top-level var named %q to merge\n", *from)
		return 1
	}
	if !plan.intoDeclared {
		fmt.Fprintf(os.Stderr, "error: merge target %q is not declared; merge folds into an EXISTING entity\n", *into)
		return 1
	}

	// The survivor's file receives the audit claim even if it holds no
	// references to `from`, so it joins the touched set.
	fileSet := map[string]bool{}
	for f := range plan.edits {
		fileSet[f] = true
	}
	record := !*noRecord
	if record {
		fileSet[plan.intoFile] = true
	}
	files := make([]string, 0, len(fileSet))
	for f := range fileSet {
		files = append(files, f)
	}
	sort.Strings(files)

	refs := 0
	for _, es := range plan.edits {
		for _, e := range es {
			if e.repl == *into {
				refs++
			}
		}
	}

	fmt.Printf("merge %s -> %s: remove declaration in %s, retarget %d references across %d files\n",
		*from, *into, filepath.Base(plan.declFile), refs, len(files))
	for _, f := range files {
		note := ""
		if record && f == plan.intoFile {
			note = " +audit claim"
		}
		fmt.Printf("  %s (%d edits%s)\n", filepath.Base(f), len(plan.edits[f]), note)
	}
	if *dryRun {
		fmt.Println("(dry run — nothing written)")
		return 0
	}

	backups := map[string][]byte{}
	revert := func() {
		for path, content := range backups {
			_ = os.WriteFile(path, content, 0o644)
		}
	}

	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			revert()
			fmt.Fprintf(os.Stderr, "read %s: %v (reverted)\n", path, err)
			return 1
		}
		backups[path] = src
		out := applyEdits(src, plan.edits[path])
		if record && path == plan.intoFile {
			out = append(out, renderAuditClaim(*from, *into, plan.fromID, plan.fromName)...)
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			revert()
			fmt.Fprintf(os.Stderr, "write %s: %v (reverted)\n", path, err)
			return 1
		}
	}

	gofmtArgs := append([]string{"-w"}, files...)
	steps := [][]string{
		append([]string{"gofmt"}, gofmtArgs...),
		{"go", "build", "."},
		{"go", "vet", "."},
	}
	for _, step := range steps {
		if out, err := runCmd(*root, step[0], step[1:]...); err != nil {
			revert()
			fmt.Fprintf(os.Stderr, "%s failed (all files reverted):\n%s\n", step[0], out)
			return 1
		}
	}

	fmt.Fprintf(os.Stderr, "merged %s into %s across %d files (build gate passed)\n", *from, *into, len(files))
	return 0
}

type mergePlan struct {
	fromDeclared bool
	intoDeclared bool
	declFile     string // file holding `from`'s declaration
	intoFile     string // file holding the survivor's declaration (audit claim lands here)
	fromID       string // absorbed entity's ID field, for the audit record
	fromName     string // absorbed entity's Name field, for the audit record
	edits        map[string][]edit
}

// planMerge builds the per-file edit list: delete `from`'s declaration (its
// whole GenDecl when it is the group's only spec, else just its ValueSpec
// lines) and rewrite every other reference to `from` into `into`.
func planMerge(root, from, into string) (*mergePlan, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, root, astutil.GoFileFilter, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	plan := &mergePlan{edits: map[string][]edit{}}
	var declStart, declEnd int // byte range of the declaration to delete, in declFile

	// First pass: locate the declaration to delete and confirm `into` exists.
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			src, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, n := range vs.Names {
						switch n.Name {
						case into:
							plan.intoDeclared = true
							plan.intoFile = path
						case from:
							plan.fromDeclared = true
							plan.declFile = path
							// Capture the absorbed entity's identity for the
							// audit record before its declaration is deleted.
							if len(vs.Values) == 1 {
								if cl, ok := vs.Values[0].(*ast.CompositeLit); ok {
									plan.fromID = astutil.ExtractStringField(cl, "ID")
									plan.fromName = astutil.ExtractStringField(cl, "Name")
								}
							}
							// Delete the whole GenDecl when `from` is its only
							// spec (covers standalone `var X = ...` and a group
							// of one); otherwise delete just this spec's lines.
							var startPos, endPos token.Pos
							if len(gd.Specs) == 1 {
								startPos, endPos = declBounds(gd, gd.Doc)
							} else {
								startPos, endPos = declBounds(vs, vs.Doc)
							}
							declStart = lineStart(src, fset.Position(startPos).Offset)
							declEnd = lineEndAfterNewline(src, fset.Position(endPos).Offset)
						}
					}
				}
			}
		}
	}

	if !plan.fromDeclared || !plan.intoDeclared {
		return plan, nil // caller reports which is missing
	}

	// Second pass: every identifier referring to `from` becomes `into`, except
	// the ones inside the deleted declaration range (the defining ident and any
	// idents in the removed RHS).
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			var es []edit
			if path == plan.declFile {
				es = append(es, edit{offset: declStart, length: declEnd - declStart, repl: ""})
			}
			ast.Inspect(file, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if !ok || id.Name != from {
					return true
				}
				off := fset.Position(id.Pos()).Offset
				if path == plan.declFile && off >= declStart && off < declEnd {
					return true // inside the removed declaration
				}
				es = append(es, edit{offset: off, length: len(from), repl: into})
				return true
			})
			if len(es) > 0 {
				plan.edits[path] = es
			}
		}
	}
	return plan, nil
}

// renderAuditClaim builds the AbsorbedAlternate claim recording that `from`
// was folded into `into`. It is a unary claim on the survivor (reached via
// `.Entity` because the survivor's role type is not known here); the absorbed
// entity's identity lives in Provenance.Quote because its var is now gone.
func renderAuditClaim(from, into, fromID, fromName string) string {
	claimName := into + "Absorbed" + from
	quote := fmt.Sprintf("winze-edit merge: absorbed %s (id: %q, name: %q) into %s", from, fromID, fromName, into)
	today := time.Now().UTC().Format("2006-01-02")

	var b strings.Builder
	fmt.Fprintf(&b, "\nvar %s = AbsorbedAlternate{\n", claimName)
	fmt.Fprintf(&b, "\tSubject: %s.Entity,\n", into)
	fmt.Fprintf(&b, "\tProv: Provenance{\n")
	fmt.Fprintf(&b, "\t\tOrigin:     %q,\n", "winze-edit merge")
	fmt.Fprintf(&b, "\t\tIngestedAt: %s,\n", strconv.Quote(today))
	fmt.Fprintf(&b, "\t\tIngestedBy: %q,\n", "winze-edit-merge")
	fmt.Fprintf(&b, "\t\tQuote:      %s,\n", strconv.Quote(quote))
	fmt.Fprintf(&b, "\t},\n")
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

// declBounds returns the start position (doc comment if present, else the node)
// and end position of a declaration, so line-based removal takes the doc
// comment with the spec.
func declBounds(node ast.Node, doc *ast.CommentGroup) (token.Pos, token.Pos) {
	start := node.Pos()
	if doc != nil {
		start = doc.Pos()
	}
	return start, node.End()
}

// lineStart returns the offset of the first byte of the line containing off.
func lineStart(src []byte, off int) int {
	if off > len(src) {
		off = len(src)
	}
	for i := off - 1; i >= 0; i-- {
		if src[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

// lineEndAfterNewline returns the offset just past the newline that ends the
// line containing off (or len(src) at EOF).
func lineEndAfterNewline(src []byte, off int) int {
	for i := off; i < len(src); i++ {
		if src[i] == '\n' {
			return i + 1
		}
	}
	return len(src)
}
