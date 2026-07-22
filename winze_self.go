package winze

// winze_self.go — the doc→code half of winze's typed-citation thesis,
// dogfooded on winze's own internals.
//
// A knowledge entity here cites a *live code symbol* by holding the symbol
// itself (CodeRef.Symbol), not a string naming it. Because the symbol is a
// real Go value, `go build .` type-checks the citation: rename or remove the
// symbol and this file stops compiling. That is the whole point — a
// documentation reference that CANNOT go stale silently, because staleness is
// a build error instead of a thing a reviewer might notice.
//
// Contrast every prose KB (Confluence, Notion, Obsidian, a README): a
// reference to `handleAuth()` survives long after handleAuth is gone. Here it
// can't. This is the same enforcement winze already applies to concept→concept
// claim links, pointed at the codebase instead of at other entities.
//
// Cleanly separable: `rm winze_self.go && go build .` removes the self-doc and
// its two internal imports with no trace, exactly like the pkm_* convention.

import (
	"github.com/justinstimatze/winze/internal/corpuslock"
	"github.com/justinstimatze/winze/internal/corpusparse"
)

// CodeRef is a typed citation from a knowledge entity to a live code symbol.
// Symbol holds the actual Go symbol (here, a function value) so its existence
// is checked at build time. Path is the human-readable label for rendering and
// query; it is derivable from Symbol via runtime.FuncForPC (a later refinement
// so Path itself can't drift from Symbol), kept explicit for now.
type CodeRef struct {
	Symbol any    // the real code symbol — compile-checked existence
	Path   string // human label, e.g. "internal/corpuslock.Acquire"
	Note   string // what the citing entity asserts about this symbol
}

// SourceDoc is a knowledge entity that documents part of the winze codebase.
// It is an ordinary *Entity (identity + prose Brief) plus typed code citations
// — prose for meaning, typed references for the links, the same split the rest
// of the corpus uses.
type SourceDoc struct {
	*Entity
	Refs []CodeRef
}

var (
	CorpusLockDoc = SourceDoc{
		Entity: &Entity{
			ID:    "doc-corpuslock",
			Name:  "Corpus write lock",
			Kind:  "sourcedoc",
			Brief: "Every winze write path takes a corpus-wide advisory flock on .winze.lock before its read-gate-commit section, so concurrent sessions sharing one worktree serialize instead of racing the shared build gate.",
		},
		Refs: []CodeRef{{
			Symbol: corpuslock.Acquire,
			Path:   "internal/corpuslock.Acquire",
			Note:   "acquires the .winze.lock flock and returns a release closure; per-fd, so a crashed holder is released by the kernel",
		}},
	}

	CorpusParseDoc = SourceDoc{
		Entity: &Entity{
			ID:    "doc-corpusparse",
			Name:  "Corpus read side",
			Kind:  "sourcedoc",
			Brief: "The read side parses corpus .go files with go/ast into an in-memory index of typed entities and claims. Because the KB is Go, the parser knows which textual occurrences of a name are the symbol and which are prose.",
		},
		Refs: []CodeRef{
			{
				Symbol: corpusparse.ParseCorpus,
				Path:   "internal/corpusparse.ParseCorpus",
				Note:   "walks a corpus directory and returns its entities and claims",
			},
			{
				Symbol: corpusparse.IsTripGenerated,
				Path:   "internal/corpusparse.IsTripGenerated",
				Note:   "var-name heuristic separating source-grounded claims from trip-cycle conjecture",
			},
		},
	}
)
