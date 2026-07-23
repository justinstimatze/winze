package main

// Entity mode — append a new typed entity (the creation primitive the claim
// path lacked). A knowledge base you can only add claims to is one you cannot
// grow: every claim references entity vars that must already exist, so without
// a way to coin entities the corpus is frozen. This closes that gap, and it is
// what makes capture-into-memory work — most working-memory facts are new
// entities (a note, a decision, a fact), not relationships between existing ones.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/justinstimatze/winze/internal/corpusparse"
)

type entityOpts struct {
	role, name, brief, kind, aliases string
	target, repoRoot                 string
	dryRun                           bool
}

func runEntity(o entityOpts) int {
	if o.role == "" || o.brief == "" {
		fmt.Fprintln(os.Stderr, "add --entity: --role and --brief are required")
		return 2
	}
	if o.target == "" {
		fmt.Fprintln(os.Stderr, "add --entity: --to <file> is required")
		return 2
	}

	// Var name: explicit --name, else auto-generated from the brief and made
	// unique against existing vars (the build gate would reject a collision).
	used := map[string]bool{}
	if ents, claims, err := corpusparse.ParseCorpus(o.repoRoot); err == nil {
		for _, e := range ents {
			used[e.VarName] = true
		}
		for _, c := range claims {
			used[c.VarName] = true
		}
	}
	var name, display string
	if strings.TrimSpace(o.name) != "" {
		name, display = explicitName(o.name)
	} else {
		name, display = deriveNames(o.brief)
	}
	if name == "" {
		name, display = deriveNames(o.brief)
	}
	name = uniqueName(name, proposal{}, used)

	kind := o.kind
	if kind == "" {
		kind = strings.ToLower(o.role)
	}
	var aliases []string
	for _, a := range strings.Split(o.aliases, ",") {
		if a = strings.TrimSpace(a); a != "" {
			aliases = append(aliases, a)
		}
	}
	decl := renderEntity(o.role, name, display, kind, o.brief, aliases)

	if o.dryRun {
		fmt.Printf("--- would append to %s ---\n%s\n", o.target, decl)
		return 0
	}
	if err := commitDecl(o.repoRoot, o.target, decl); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "created entity %s (%s) in %s (build gate passed)\n", name, o.role, o.target)
	return 0
}

// renderEntity emits `var Name = Role{&Entity{...}}`, the role-wrapper form the
// corpus uses. display is the human-facing Name; ID is a slug of the var name.
func renderEntity(role, varName, display, kind, brief string, aliases []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nvar %s = %s{&Entity{\n", varName, role)
	fmt.Fprintf(&b, "\tID:    %q,\n", kind+"-"+slug(varName))
	fmt.Fprintf(&b, "\tName:  %q,\n", display)
	fmt.Fprintf(&b, "\tKind:  %q,\n", kind)
	if len(aliases) > 0 {
		quoted := make([]string, len(aliases))
		for i, a := range aliases {
			quoted[i] = strconv.Quote(a)
		}
		fmt.Fprintf(&b, "\tAliases: []string{%s},\n", strings.Join(quoted, ", "))
	}
	fmt.Fprintf(&b, "\tBrief: %s,\n", quoteLiteral(brief))
	b.WriteString("}}\n")
	return b.String()
}

// explicitName handles a user-supplied --name that may be either a Go
// identifier ("RecallHookGate") or a free-text title ("recall hook gate").
// A multi-word title is CamelCased for the varName but kept verbatim for the
// display Name — the words were chosen deliberately, so no stopword pruning.
func explicitName(raw string) (varName, display string) {
	raw = strings.TrimSpace(raw)
	if !strings.ContainsAny(raw, " \t-") {
		id := sanitizeIdent(raw)
		return id, humanize(id)
	}
	var camel []string
	for _, w := range strings.Fields(raw) {
		if clean := lettersDigits(w); clean != "" {
			camel = append(camel, strings.ToUpper(clean[:1])+clean[1:])
		}
	}
	return sanitizeIdent(strings.Join(camel, "")), capitalizeFirst(raw)
}

// deriveNames turns a free-text brief into a Go identifier (varName) and a
// human-facing display Name. A brief is usually a full sentence, not a title,
// so the naive "first N words" gives clumsy results ("The ... Winzemem"):
// leading stopwords survive, and the window runs past the natural title into
// a mid-phrase fragment. Instead: take the leading title segment (up to the
// first colon/dash/paren/sentence break), drop stopwords, keep the first few
// content words. varName CamelCases them; display keeps the original words so
// "winze-memory agentic interface" stays readable. Falls back to a timestamp
// note when nothing usable survives. An explicit --name bypasses all of this.
func deriveNames(brief string) (varName, display string) {
	seg := titleSegment(brief)
	var camel, words []string
	for _, w := range strings.Fields(seg) {
		clean := lettersDigits(w)
		if len(clean) < 3 || isStopword(strings.ToLower(clean)) {
			continue // skip filler and function words
		}
		camel = append(camel, strings.ToUpper(clean[:1])+clean[1:])
		words = append(words, strings.Trim(w, `.,;:—-()"'`))
		if len(camel) >= 4 {
			break
		}
	}
	varName = sanitizeIdent(strings.Join(camel, ""))
	if varName == "" {
		ts := "Note" + time.Now().UTC().Format("20060102150405")
		return ts, "Note " + time.Now().UTC().Format("2006-01-02")
	}
	return varName, capitalizeFirst(strings.Join(words, " "))
}

// titleSegment returns the leading clause of a brief — the part before the
// first strong boundary (colon, sentence break, dash clause, or parenthetical).
// That clause is where a title-like phrase lives; the rest is elaboration.
func titleSegment(brief string) string {
	best := len(brief)
	for _, bnd := range []string{": ", ":", ". ", "; ", " — ", " - ", " ("} {
		if i := strings.Index(brief, bnd); i >= 0 && i < best {
			best = i
		}
	}
	return strings.TrimSpace(brief[:best])
}

func lettersDigits(w string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, w)
}

// stopwords are function words that make poor identifier/title leads.
var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
	"of": true, "to": true, "in": true, "on": true, "at": true, "for": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "as": true,
	"by": true, "from": true, "into": true, "via": true, "that": true,
	"this": true, "it": true, "its": true, "with": true, "we": true, "our": true,
}

func isStopword(w string) bool { return stopwords[w] }

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func humanize(varName string) string {
	var b strings.Builder
	for i, r := range varName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func slug(varName string) string {
	return strings.ToLower(strings.ReplaceAll(humanize(varName), " ", "-"))
}
