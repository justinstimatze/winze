package main

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slug renders a name as a filename-safe path segment. Non-ASCII letters are
// dropped rather than transliterated — the corpus carries Sanskrit and Russian
// terms whose transliteration is itself a contested claim, and a lossy guess in
// a filename is not the place to take a position. Name and title in the
// document keep the original.
func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonSlug.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// firstSentence extracts a single-sentence summary for OKF's `description`,
// which the spec asks to keep to one sentence. Briefs in the corpus are often
// several; the full text still lands in the document body, so nothing is lost.
func firstSentence(s string) string {
	s = collapse(s)
	if s == "" {
		return ""
	}
	for i := 0; i < len(s)-1; i++ {
		if s[i] != '.' && s[i] != '?' && s[i] != '!' {
			continue
		}
		if s[i+1] == ' ' && !endsWithInitialism(s[:i]) {
			return s[:i+1]
		}
	}
	return s
}

// endsWithInitialism guards the common abbreviations that would otherwise cut a
// sentence in half ("e.g. ", "Fig. ", "vs. "). A one- or two-letter token before
// the period is treated as an abbreviation rather than a sentence end.
func endsWithInitialism(s string) bool {
	i := strings.LastIndexAny(s, " (")
	last := s[i+1:]
	return len(last) <= 2
}

var wsRE = regexp.MustCompile(`\s+`)

// collapse folds a value onto one line. Frontmatter and markdown list items are
// line-oriented; a Quote containing a newline would otherwise break the
// footnote it lives in.
func collapse(s string) string {
	return strings.TrimSpace(wsRE.ReplaceAllString(s, " "))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}

// spaceCamel renders a predicate type name as prose: IsCognitiveBias ->
// "Is Cognitive Bias". Used for unary claims, where the predicate is the whole
// statement and has no object to link to.
func spaceCamel(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func titleize(section string) string {
	parts := strings.Split(section, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// tagsFor derives OKF `tags` from what the corpus already commits to: the
// entity's aliases and the corpus slice it lives in. The file stem is a real
// grouping signal — a winze file is a coherent ingest neighborhood — and it is
// the only topical label available without inventing one.
func tagsFor(e corpusparse.Entity) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if s = slug(s); s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if e.File != "" {
		add(strings.TrimSuffix(e.File, ".go"))
	}
	for _, a := range e.Aliases {
		add(a)
	}
	return out
}

func conjectureLabel(c *corpusparse.Conjecture) string {
	parts := []string{}
	if c.GeneratedBy != "" {
		parts = append(parts, c.GeneratedBy)
	}
	if c.PromptType != "" {
		parts = append(parts, c.PromptType)
	}
	if c.CycleN > 0 {
		parts = append(parts, "cycle "+itoa(c.CycleN))
	}
	if c.Score > 0 {
		parts = append(parts, "score "+itoa(c.Score))
	}
	if len(parts) == 0 {
		return "winze conjecture"
	}
	return "*" + strings.Join(parts, ", ") + "*"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}

// predicateGroup is a run of claims sharing an edge type — the unit the
// `## <Predicate>` headings are built from.
type predicateGroup struct {
	name   string
	claims []corpusparse.Claim
}

func groupByPredicate(claims []corpusparse.Claim) []predicateGroup {
	byPred := map[string][]corpusparse.Claim{}
	for _, cl := range claims {
		byPred[cl.PredicateType] = append(byPred[cl.PredicateType], cl)
	}
	names := make([]string, 0, len(byPred))
	for n := range byPred {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]predicateGroup, 0, len(names))
	for _, n := range names {
		g := byPred[n]
		sort.Slice(g, func(i, j int) bool { return g[i].VarName < g[j].VarName })
		out = append(out, predicateGroup{name: spaceCamel(n), claims: g})
	}
	return out
}

func sortedKeys(m map[string]corpusparse.Provenance) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedBoolKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedIntKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedDocKeys(m map[string][]*doc) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
