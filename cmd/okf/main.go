// Command winze-okf projects the typed corpus into a conformant Open Knowledge
// Format bundle — Google Cloud's OKF v0.2, a directory of markdown files with
// YAML frontmatter (github.com/GoogleCloudPlatform/knowledge-catalog).
//
//	winze-okf --out DIR [CORPUS]     # emit a bundle from the corpus
//	winze-okf --validate DIR         # check a bundle for OKF v0.2 conformance
//
// # Why export at all
//
// OKF and winze made opposite bets from the same premise. Both hold that
// agent-facing knowledge needs curation, provenance, and a generated-vs-sourced
// distinction; OKF v0.2 arrived at `generated` / `verified` trust tiers
// independently of winze's Attribution sum. Where they part is enforcement.
// OKF's conformance floor is deliberately near-nil — one required field, and
// consumers MUST NOT reject unknown types, unknown keys, or broken links —
// because it is buying adoption. Winze's floor is `go build`, because it is
// buying integrity.
//
// Those bets are compatible in one direction only: a corpus that satisfies the
// compiler can always be projected down to a bundle that satisfies the floor.
// The reverse is not true. So winze exports, and the export is not a
// concession — it is the argument, in a form other tools can read.
//
// # What survives the projection, and what the projection adds
//
// OKF concepts link with bare markdown links; the spec is explicit that the
// KIND of a relationship is carried by surrounding prose, not by the link. That
// is the one thing winze cannot afford to drop, so claims are emitted under
// `## <Predicate>` headings inside a `# Claims` section. Unknown headings are
// conformant, so the bundle stays legal for any OKF consumer while a consumer
// that looks recovers the full typed edge.
//
// The attribution split survives structurally, not by convention:
//
//   - A claim backed by Provenance contributes its source to the document's
//     `sources:` list and marks the claim with that source's footnote id. The
//     exact Quote lands in the footnote, so the audit record travels with the
//     bundle instead of depending on a `resource` URL that may already be dead.
//   - A claim backed by Conjecture contributes NOTHING to `sources:`. It is
//     emitted in a separate `# Conjectures` section carrying winze's own
//     rationale, generator, cycle, and score. corpusparse.Conjecture has no
//     Quote field for the same reason schema.go's does not, so there is no
//     value the exporter could put in a source entry even by mistake.
//
// Trust tiers follow from that split rather than being asserted alongside it.
// `generated` records who produced the document (always a winze process today).
// `verified` is emitted only for documents with at least one Quote-bearing
// sourced claim, which puts them in OKF's machine-confirmed tier; a document
// backed only by conjecture carries no `verified` entry and `status: draft`,
// landing in the unverified tier where it belongs. See docs/okf.md.
//
// # Determinism
//
// Emission is fully deterministic — entities, sections, sources and footnotes
// are all sorted, and nothing carries a wall-clock stamp — so re-exporting an
// unchanged corpus produces a byte-identical bundle. A committed bundle diffs
// cleanly against the corpus change that caused it.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/justinstimatze/winze/internal/usagelog"
)

func main() {
	start := time.Now()

	out := flag.String("out", "", "emit an OKF bundle into this directory")
	validate := flag.String("validate", "", "check an existing bundle for OKF v0.2 conformance and exit")
	force := flag.Bool("force", false, "with --out: overwrite a directory that is not a previously generated bundle")
	quiet := flag.Bool("quiet", false, "suppress the per-section summary")
	flag.Parse()

	dir := "."
	if args := flag.Args(); len(args) > 0 {
		dir = args[0]
	}

	if *validate != "" {
		report, err := validateBundle(*validate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "okf: %v\n", err)
			os.Exit(1)
		}
		report.print(os.Stdout)
		if !report.ok() {
			os.Exit(1)
		}
		return
	}

	if *out == "" {
		fmt.Fprintf(os.Stderr, "usage: winze-okf --out DIR [CORPUS]\n       winze-okf --validate DIR\n")
		os.Exit(1)
	}

	defer usagelog.Log(dir, "okf", os.Args[1:], start)

	summary, err := export(dir, *out, *force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "okf: %v\n", err)
		os.Exit(1)
	}
	if !*quiet {
		summary.print(os.Stdout)
	}
}
