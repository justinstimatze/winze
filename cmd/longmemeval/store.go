package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justinstimatze/winze/internal/defndb"
)

// Fact is the memory record, mirrored here as the type the generated memstore
// module declares. Retrieval reconstructs these from defn's LiteralFields.
type Fact struct {
	Attribute string
	Value     string
	Kind      string
	Date      string
	Session   string
	Quote     string
}

// memstoreSchema is the entire type system of a memory store: one struct. The
// point of the spike is that winze's machinery — the build gate as consistency
// check, defn as the index — works over any typed corpus, not just the
// epistemology schema. A personal-memory store needs exactly this much schema.
const memstoreSchema = `package memstore

// Fact is one durable thing the user stated about themselves, with the session
// date it was stated on so temporal questions can reason over ordering.
type Fact struct {
	Attribute string
	Value     string
	Kind      string
	Date      string
	Session   string
	Quote     string
}
`

// buildStore writes the facts as a standalone Go module, runs winze's build
// gate (gofmt + go build) as the consistency check, and returns the module dir.
// A store that doesn't compile is a store with a malformed fact — the same
// discipline the winze corpus runs on every write.
func (r *runner) buildStore(qid string, facts []Fact) (string, error) {
	dir := filepath.Join(r.workDir, "store-"+qid)
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module memstore\n\ngo 1.23\n"), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "schema.go"), []byte(memstoreSchema), 0o644); err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("package memstore\n\n")
	for i, f := range facts {
		fmt.Fprintf(&b, "var F%04d = Fact{\n", i)
		fmt.Fprintf(&b, "\tAttribute: %q,\n", f.Attribute)
		fmt.Fprintf(&b, "\tValue:     %q,\n", f.Value)
		fmt.Fprintf(&b, "\tKind:      %q,\n", f.Kind)
		fmt.Fprintf(&b, "\tDate:      %q,\n", f.Date)
		fmt.Fprintf(&b, "\tSession:   %q,\n", f.Session)
		fmt.Fprintf(&b, "\tQuote:     %q,\n", f.Quote)
		b.WriteString("}\n\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.go"), []byte(b.String()), 0o644); err != nil {
		return "", err
	}

	// Build gate: gofmt then build. Revert-on-failure isn't needed here (the
	// store is disposable), but a build failure is a hard error — it means the
	// lens emitted a value that broke Go syntax, which we want surfaced.
	if out, err := runIn(dir, "gofmt", "-w", "."); err != nil {
		return "", fmt.Errorf("gofmt: %v\n%s", err, out)
	}
	if out, err := runIn(dir, "go", "build", "./..."); err != nil {
		return "", fmt.Errorf("go build: %v\n%s", err, out)
	}
	return dir, nil
}

// syncAndRetrieve opens the store through defn (triggering the full type-checked
// ingest), pulls every Fact via LiteralFields, and returns the top-k by term
// overlap with the question. This is the winze-via-defn read path the whole
// perf story hangs on — sync time and retrieve time are timed separately by the
// caller.
func (r *runner) syncAndRetrieve(dir, question string, k int) (facts []Fact, syncNS, retrieveNS int64, err error) {
	tSync := nowNS()
	client, err := defndb.New(dir) // New syncs if the store is stale (always, first open)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("defndb.New: %w", err)
	}
	defer client.Close()
	syncNS = nowNS() - tSync

	tRet := nowNS()
	all, err := readFacts(client)
	if err != nil {
		return nil, syncNS, 0, err
	}
	ranked := rankFacts(all, question, k)
	retrieveNS = nowNS() - tRet
	return ranked, syncNS, retrieveNS, nil
}

// readFacts reconstructs Fact records from defn's per-field literals, grouping
// by the enclosing var (DefName).
func readFacts(client *defndb.Client) ([]Fact, error) {
	byVar := map[string]*Fact{}
	var order []string
	err := client.EachFieldOfType("Fact", func(f *defndb.LiteralField) bool {
		rec := byVar[f.DefName]
		if rec == nil {
			rec = &Fact{}
			byVar[f.DefName] = rec
			order = append(order, f.DefName)
		}
		switch f.FieldName {
		case "Attribute":
			rec.Attribute = f.FieldValue
		case "Value":
			rec.Value = f.FieldValue
		case "Kind":
			rec.Kind = f.FieldValue
		case "Date":
			rec.Date = f.FieldValue
		case "Session":
			rec.Session = f.FieldValue
		case "Quote":
			rec.Quote = f.FieldValue
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	out := make([]Fact, 0, len(order))
	for _, name := range order {
		out = append(out, *byVar[name])
	}
	return out, nil
}

// rankFacts scores each fact by how many distinct question terms appear in its
// attribute/value/quote, and returns the top k. Deterministic and dependency-
// free — the gate role, keeping the answerer's context small.
func rankFacts(facts []Fact, question string, k int) []Fact {
	qterms := terms(question)
	type scored struct {
		f     Fact
		score int
		idx   int
	}
	var ss []scored
	for i, f := range facts {
		hay := terms(f.Attribute + " " + f.Value + " " + f.Quote)
		score := 0
		for t := range qterms {
			if hay[t] {
				score++
			}
		}
		ss = append(ss, scored{f: f, score: score, idx: i})
	}
	sort.SliceStable(ss, func(a, b int) bool {
		if ss[a].score != ss[b].score {
			return ss[a].score > ss[b].score
		}
		return ss[a].idx < ss[b].idx // stable tie-break by original order
	})
	var out []Fact
	for i := 0; i < len(ss) && i < k; i++ {
		out = append(out, ss[i].f)
	}
	return out
}

// terms lowercases and splits text into a set of alphanumeric tokens ≥3 chars,
// dropping a few stopwords that would otherwise dominate overlap scores.
func terms(s string) map[string]bool {
	stop := map[string]bool{"the": true, "and": true, "did": true, "what": true, "when": true, "was": true, "for": true, "you": true, "your": true, "with": true, "have": true}
	out := map[string]bool{}
	var cur strings.Builder
	flush := func() {
		if cur.Len() >= 3 {
			w := cur.String()
			if !stop[w] {
				out[w] = true
			}
		}
		cur.Reset()
	}
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func runIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
