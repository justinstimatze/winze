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
	name := sanitizeIdent(o.name)
	if name == "" {
		name = nameFromBrief(o.brief)
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
	decl := renderEntity(o.role, name, kind, o.brief, aliases)

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
// corpus uses. Name (display) is derived from the var name; ID is a slug.
func renderEntity(role, varName, kind, brief string, aliases []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nvar %s = %s{&Entity{\n", varName, role)
	fmt.Fprintf(&b, "\tID:    %q,\n", kind+"-"+slug(varName))
	fmt.Fprintf(&b, "\tName:  %q,\n", humanize(varName))
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

// nameFromBrief builds a CamelCase Go identifier from the first few significant
// words of the brief, falling back to a timestamp when nothing usable survives.
func nameFromBrief(brief string) string {
	var parts []string
	for _, w := range strings.Fields(brief) {
		clean := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return r
			}
			return -1
		}, w)
		if len(clean) < 3 {
			continue // skip short filler words
		}
		parts = append(parts, strings.ToUpper(clean[:1])+clean[1:])
		if len(parts) >= 5 {
			break
		}
	}
	name := sanitizeIdent(strings.Join(parts, ""))
	if name == "" {
		return "Note" + time.Now().UTC().Format("20060102150405")
	}
	return name
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
