// Package vault reads a directory of markdown notes — an Obsidian vault, a
// Zettelkasten, a folder of game-design references — into a resolved link
// graph.
//
// This is the front half of "point winze at a pile of markdown and get a typed
// graph". It deliberately knows nothing about winze's schema: it extracts what
// the notes themselves commit to (titles, aliases, tags, frontmatter, and the
// links between them) and resolves those links to note identities. Turning that
// into typed claims is the ingest's job, not the parser's.
//
// # What counts as a link
//
// Both vault dialects, because real piles mix them:
//
//   - `[[Target]]`, `[[Target|display]]`, `[[Target#Heading]]`, `[[dir/Target]]`
//   - `[display](./other.md)` and `[display](other.md)` — relative markdown
//     links to another note in the vault
//
// Links inside fenced code blocks and inline code spans are NOT links. A
// game-design vault quotes code constantly, and `[[0]]` in a Go snippet is an
// array index. Treating it as a graph edge would put fabricated structure into
// the corpus, which is the one thing winze must never do.
//
// Each link records the heading it appeared under. A link under `## Depends on`
// carries more information than the same link in a prose paragraph, and the
// ingest uses that section to propose a typed predicate rather than a bare
// reference.
//
// # Resolution
//
// A target resolves against, in order: exact relative path, filename slug,
// normalized title, and declared alias. Ambiguous targets (two notes claiming
// one name) resolve to neither and are reported — silently picking one would
// put a wrong edge in the graph, and a wrong edge is worse than a missing one.
// Unresolved targets are reported too: in a vault they are usually notes that
// have not been written yet, and that gap is a real signal about the vault.
package vault

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"

	"gopkg.in/yaml.v2"
)

// maxNoteBytes caps how much of a single note is read. Reference vaults
// occasionally contain a multi-megabyte dump; parsing all of it buys nothing
// and lets one pathological file dominate ingest time.
const maxNoteBytes = 1 << 20 // 1 MiB

// Note is one parsed markdown file.
type Note struct {
	RelPath string // vault-relative path, slash-separated: "systems/combat.md"
	Dir     string // vault-relative directory, "" at the vault root
	Title   string // frontmatter `title`, else first H1, else the filename
	Aliases []string
	Tags    []string
	Type    string // frontmatter `type` — the OKF-compatible role hint

	// Frontmatter holds the remaining scalar keys verbatim, so an ingest can
	// use a vault's own conventions without this package having to know them.
	Frontmatter map[string]string

	Links []Link
	Lead  string // first prose paragraph: the note's own one-line summary
	Body  string // full text with frontmatter stripped
}

// Link is one outbound reference found in a note.
type Link struct {
	Target  string // raw target as written, minus display text and heading anchor
	Section string // nearest preceding heading, "" when above the first one
	Display string // display text, when the link supplied one
	Line    int    // 1-based line number, for reporting
	Wiki    bool   // true for [[...]], false for a relative markdown link
}

// Load parses every markdown file under dir. Files are parsed in parallel and
// returned in stable RelPath order, so ingest output does not depend on
// filesystem walk order or scheduling.
func Load(dir string) ([]Note, error) {
	paths, err := collect(dir)
	if err != nil {
		return nil, err
	}
	notes := make([]Note, len(paths))
	ok := make([]bool, len(paths))

	workers := 8
	if len(paths) < workers {
		workers = len(paths)
	}
	var wg sync.WaitGroup
	idx := make(chan int)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idx {
				n, err := parseFile(filepath.Join(dir, filepath.FromSlash(paths[i])), paths[i])
				if err != nil {
					continue // an unreadable note is skipped, never fatal
				}
				notes[i], ok[i] = *n, true
			}
		}()
	}
	for i := range paths {
		idx <- i
	}
	close(idx)
	wg.Wait()

	out := notes[:0]
	for i, good := range ok {
		if good {
			out = append(out, notes[i])
		}
	}
	return out, nil
}

// collect walks dir for .md files, skipping the dot-directories vault tools
// leave behind (.obsidian, .git, .trash) and anything else hidden.
func collect(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != dir && strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !strings.EqualFold(filepath.Ext(name), ".md") {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func parseFile(fullPath, relPath string) (*Note, error) {
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	n := &Note{RelPath: relPath, Frontmatter: map[string]string{}}
	if d := path.Dir(relPath); d != "." {
		n.Dir = d
	}

	r := bufio.NewReader(&limitReader{r: f, left: maxNoteBytes})
	lines, err := readLines(r)
	if err != nil {
		return nil, err
	}

	body := parseFrontmatter(n, lines)
	scanBody(n, body)

	if n.Title == "" {
		n.Title = titleFromFilename(relPath)
	}
	n.Body = strings.Join(body, "\n")
	return n, nil
}

// readLines reads the whole note. bufio.Scanner is avoided deliberately: its
// default 64 KiB token limit silently truncates a note containing one very long
// line (a minified blob, a base64 image, a single-line table), and the error it
// sets is easy to forget to check. A note that parses to half its content is
// worse than one that fails loudly.
func readLines(r *bufio.Reader) ([]string, error) {
	var lines []string
	var cur strings.Builder
	for {
		chunk, isPrefix, err := r.ReadLine()
		cur.Write(chunk)
		if isPrefix {
			continue
		}
		lines = append(lines, cur.String())
		cur.Reset()
		if err != nil {
			return lines, nil // io.EOF and truncation both end the note cleanly
		}
	}
}

type limitReader struct {
	r    *os.File
	left int
}

func (l *limitReader) Read(p []byte) (int, error) {
	if l.left <= 0 {
		return 0, os.ErrClosed
	}
	if len(p) > l.left {
		p = p[:l.left]
	}
	n, err := l.r.Read(p)
	l.left -= n
	return n, err
}

// parseFrontmatter consumes a leading `---` YAML block, populating the note's
// declared identity, and returns the remaining body lines. A block that does
// not parse is left in the body rather than discarded — losing content to a
// stray delimiter would be silent data loss.
func parseFrontmatter(n *Note, lines []string) []string {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return lines
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if t := strings.TrimSpace(lines[i]); t == "---" || t == "..." {
			end = i
			break
		}
	}
	if end < 0 {
		return lines
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &raw); err != nil {
		return lines
	}
	for k, v := range raw {
		switch strings.ToLower(k) {
		case "title":
			n.Title = scalar(v)
		case "type":
			n.Type = scalar(v)
		case "aliases", "alias":
			n.Aliases = stringList(v)
		case "tags", "tag":
			n.Tags = stringList(v)
		default:
			if s := scalar(v); s != "" {
				n.Frontmatter[k] = s
			}
		}
	}
	return lines[end+1:]
}

// scanBody walks the note once, tracking fence state and the current heading,
// collecting links and the lead paragraph.
func scanBody(n *Note, lines []string) {
	var fence string
	var lead strings.Builder
	section := ""
	seen := map[string]bool{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Fenced blocks: ``` or ~~~, closed by the same marker. Everything
		// inside is code, not prose and not links.
		if marker := fenceMarker(trimmed); marker != "" {
			switch {
			case fence == "":
				fence = marker
			case strings.HasPrefix(marker, fence[:1]):
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}

		if h, level, ok := heading(trimmed); ok {
			if level == 1 && n.Title == "" {
				// The note's own H1 is its title, not a section. Treating it as
				// one labels every link in a single-H1 note with the note's own
				// name, which is noise where a real section heading is signal.
				n.Title = h
				section = ""
				continue
			}
			section = h
			continue
		}

		stripped := stripInlineCode(line)
		for _, l := range extractLinks(stripped, i+1) {
			l.Section = section
			key := l.Target + "\x00" + section
			if seen[key] {
				continue
			}
			seen[key] = true
			n.Links = append(n.Links, l)
		}

		for _, tag := range inlineTags(stripped) {
			n.Tags = appendUnique(n.Tags, tag)
		}

		if lead.Len() == 0 && trimmed != "" && !isListOrMeta(trimmed) {
			lead.WriteString(plainText(trimmed))
		}
	}
	n.Lead = firstSentence(lead.String())
}

func fenceMarker(t string) string {
	for _, m := range []string{"```", "~~~"} {
		if strings.HasPrefix(t, m) {
			return m
		}
	}
	return ""
}

// heading returns a heading's text and level (1 for `#`, 2 for `##`, ...).
func heading(t string) (string, int, bool) {
	if !strings.HasPrefix(t, "#") {
		return "", 0, false
	}
	i := 0
	for i < len(t) && t[i] == '#' {
		i++
	}
	if i > 6 || i >= len(t) || t[i] != ' ' {
		return "", 0, false
	}
	return strings.TrimSpace(strings.Trim(t[i:], "#")), i, true
}

// stripInlineCode blanks `code span` regions so a bracket inside one cannot be
// read as a link. Content is replaced by spaces rather than removed so that
// nothing shifts position.
func stripInlineCode(line string) string {
	out := []byte(line)
	inCode := false
	start := 0
	for i := 0; i < len(out); i++ {
		if out[i] != '`' {
			continue
		}
		if !inCode {
			inCode, start = true, i
			continue
		}
		for j := start; j <= i; j++ {
			out[j] = ' '
		}
		inCode = false
	}
	return string(out)
}

// extractLinks pulls both dialects out of one line. Written as an explicit
// scan rather than a regex: it is the hot loop over every line of every note,
// and it has to handle nesting (`[[a]]` inside `[x](y)`) without backtracking.
func extractLinks(line string, lineNo int) []Link {
	var out []Link
	for i := 0; i < len(line); i++ {
		switch {
		case strings.HasPrefix(line[i:], "[["):
			end := strings.Index(line[i+2:], "]]")
			if end < 0 {
				return out
			}
			inner := line[i+2 : i+2+end]
			i += end + 3
			target, display := splitDisplay(inner)
			if target = strings.TrimSpace(dropAnchor(target)); target != "" {
				out = append(out, Link{Target: target, Display: display, Line: lineNo, Wiki: true})
			}
		case line[i] == '[':
			close := strings.IndexByte(line[i:], ']')
			if close < 0 || i+close+1 >= len(line) || line[i+close+1] != '(' {
				continue
			}
			paren := strings.IndexByte(line[i+close+1:], ')')
			if paren < 0 {
				continue
			}
			display := line[i+1 : i+close]
			target := strings.TrimSpace(line[i+close+2 : i+close+1+paren])
			i += close + paren + 1
			if !isLocalNote(target) {
				continue
			}
			out = append(out, Link{Target: dropAnchor(target), Display: display, Line: lineNo})
		}
	}
	return out
}

// isLocalNote rejects external URLs, anchors, and non-markdown targets. Only a
// relative path to another .md file is a graph edge.
func isLocalNote(target string) bool {
	if target == "" || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
		return false
	}
	if strings.Contains(target, "://") {
		return false
	}
	base := dropAnchor(target)
	return strings.EqualFold(path.Ext(base), ".md")
}

func splitDisplay(inner string) (target, display string) {
	if i := strings.IndexByte(inner, '|'); i >= 0 {
		return inner[:i], strings.TrimSpace(inner[i+1:])
	}
	return inner, ""
}

func dropAnchor(s string) string {
	if i := strings.IndexByte(s, '#'); i > 0 {
		return s[:i]
	}
	return s
}

func inlineTags(line string) []string {
	var out []string
	for i := 0; i < len(line); i++ {
		if line[i] != '#' || (i > 0 && !isSpace(line[i-1])) {
			continue
		}
		j := i + 1
		for j < len(line) && (isTagRune(rune(line[j]))) {
			j++
		}
		if j > i+1 {
			out = append(out, line[i+1:j])
		}
		i = j
	}
	return out
}

func isTagRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '/'
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' }

// isListOrMeta skips the lines that are structure rather than the note's own
// summary sentence: list items, quotes, tables, and bold key-value metadata.
func isListOrMeta(t string) bool {
	switch t[0] {
	case '-', '*', '+', '>', '|', '!':
		return true
	}
	if len(t) > 1 && t[0] >= '0' && t[0] <= '9' && (t[1] == '.' || t[1] == ')') {
		return true
	}
	return strings.HasPrefix(t, "**") && strings.Contains(t, ":")
}

// plainText removes the markup that would otherwise land verbatim in an entity
// Brief: link syntax, emphasis, and code ticks.
func plainText(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case strings.HasPrefix(s[i:], "[["):
			end := strings.Index(s[i+2:], "]]")
			if end < 0 {
				b.WriteByte(s[i])
				continue
			}
			target, display := splitDisplay(s[i+2 : i+2+end])
			if display == "" {
				display = dropAnchor(target)
			}
			b.WriteString(display)
			i += end + 3
		case s[i] == '[':
			close := strings.IndexByte(s[i:], ']')
			if close < 0 || i+close+1 >= len(s) || s[i+close+1] != '(' {
				b.WriteByte(s[i])
				continue
			}
			paren := strings.IndexByte(s[i+close+1:], ')')
			if paren < 0 {
				b.WriteByte(s[i])
				continue
			}
			b.WriteString(s[i+1 : i+close])
			i += close + paren + 1
		case s[i] == '`' || s[i] == '*' || s[i] == '_':
			// emphasis and code ticks carry no meaning once flattened
		default:
			b.WriteByte(s[i])
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func firstSentence(s string) string {
	for i := 0; i+1 < len(s); i++ {
		if (s[i] == '.' || s[i] == '!' || s[i] == '?') && s[i+1] == ' ' {
			return s[:i+1]
		}
	}
	return s
}

func titleFromFilename(relPath string) string {
	base := strings.TrimSuffix(path.Base(relPath), path.Ext(relPath))
	base = strings.NewReplacer("-", " ", "_", " ").Replace(base)
	fields := strings.Fields(base)
	for i, f := range fields {
		r := []rune(f)
		r[0] = unicode.ToUpper(r[0])
		fields[i] = string(r)
	}
	return strings.Join(fields, " ")
}

func scalar(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case int, int64, float64, bool:
		return strings.TrimSpace(sprint(t))
	}
	return ""
}

func stringList(v any) []string {
	switch t := v.(type) {
	case string:
		var out []string
		for _, part := range strings.Split(t, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, p)
			}
		}
		return out
	case []any:
		var out []string
		for _, item := range t {
			if s := scalar(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func appendUnique(list []string, v string) []string {
	for _, existing := range list {
		if existing == v {
			return list
		}
	}
	return append(list, v)
}

func sprint(v any) string { return fmt.Sprint(v) }
