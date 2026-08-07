package main

// Semantic (embedding) search over entity prose via a local ollama model.
// Complements --fulltext: BM25 is lexical (matches tokens, instant, no
// word-sense disambiguation); this matches meaning, at the cost of one
// embedding call per query (~42ms on all-minilm). Entity vectors are
// content-addressed and cached to .winze-embed/ (gitignored), so the ~360-brief
// index build is paid once and incrementally — only a changed Brief re-embeds.
//
// No new build dependency: net/http to a local ollama daemon. ollama is a
// runtime requirement (`ollama serve` + `ollama pull all-minilm`), not a
// compile-time one — absence degrades to a clear error, it never breaks build.

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/justinstimatze/winze/internal/cliutil"
)

const (
	embedModel    = "all-minilm"
	ollamaEmbed   = "http://localhost:11434/api/embeddings"
	embedCacheDir = ".winze-embed"
	// maxEmbedChars caps embed input. all-minilm truncates at ~256 tokens and
	// ollama returns HTTP 500 (not a truncated vector) for input past that,
	// which would fail the whole semantic pass because of one long Brief.
	// Char count is only a proxy for token count — code-heavy prose (paths,
	// snake_case, backticks) tokenizes far denser than plain English — so this
	// is set conservatively and embed() also halves-and-retries on a 500. An
	// embedding captures a Brief's gist from its opening prose, so tail loss is
	// immaterial for recall. Truncated on a rune boundary (no split UTF-8).
	maxEmbedChars = 512
)

func embed(text string) ([]float32, error) {
	if len(text) > maxEmbedChars {
		text = truncateRunes(text, maxEmbedChars)
	}
	return embedRetry(text, 3)
}

// embedRetry POSTs text to ollama, halving the input and retrying on a 500 (the
// over-length signal) so a denser-than-expected Brief degrades to a shorter
// embedding rather than failing the whole semantic pass.
func embedRetry(text string, tries int) ([]float32, error) {
	body, _ := json.Marshal(map[string]string{"model": embedModel, "prompt": text})
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(ollamaEmbed, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama embed (is `ollama serve` running?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusInternalServerError && tries > 1 && len(text) > 64 {
			return embedRetry(truncateRunes(text, len(text)/2), tries-1)
		}
		return nil, fmt.Errorf("ollama embed status %d (have you run `ollama pull %s`?)", resp.StatusCode, embedModel)
	}
	var out struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("ollama returned an empty embedding")
	}
	return normalize(out.Embedding), nil
}

// truncateRunes returns the longest prefix of s that is at most n bytes and
// ends on a UTF-8 rune boundary (never emits an invalid partial rune).
func truncateRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// normalize to unit length so cosine similarity reduces to a dot product.
func normalize(v []float32) []float32 {
	var s float64
	for _, x := range v {
		s += float64(x) * float64(x)
	}
	n := float32(math.Sqrt(s))
	if n == 0 {
		return v
	}
	for i := range v {
		v[i] /= n
	}
	return v
}

func dot(a, b []float32) float64 {
	var d float64
	for i := range a {
		d += float64(a[i]) * float64(b[i])
	}
	return d
}

func embedKey(text string) string {
	h := sha256.Sum256([]byte(embedModel + "\x00" + text))
	return fmt.Sprintf("%x", h[:16])
}

type vecCache struct {
	path  string
	m     map[string][]float32
	dirty bool
}

func loadVecCache(dir string) *vecCache {
	c := &vecCache{path: filepath.Join(dir, embedCacheDir, embedModel+".gob"), m: map[string][]float32{}}
	if f, err := os.Open(c.path); err == nil {
		defer f.Close()
		_ = gob.NewDecoder(f).Decode(&c.m)
	}
	return c
}

func (c *vecCache) save() {
	if !c.dirty {
		return
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return
	}
	if f, err := os.Create(c.path); err == nil {
		defer f.Close()
		_ = gob.NewEncoder(f).Encode(c.m)
	}
}

func embedText(e entityRecord) string {
	return strings.TrimSpace(e.Name + ". " + e.Brief)
}

type semHit struct {
	idx   int
	score float64
}

// semanticRank embeds every entity's prose (cached, incremental) plus the
// query, then returns entities ranked by cosine similarity, highest first.
// Shared by runSemantic and the hybrid fusion (runHybrid).
func semanticRank(kb *kbIndex, query, dir string) ([]semHit, error) {
	cache := loadVecCache(dir)

	type ev struct {
		idx  int
		segs [][]float32
	}
	var vecs []ev
	built, hit := 0, 0
	for i, e := range kb.Entities {
		text := embedText(e)
		if text == "" || text == "." {
			continue
		}
		segs, n, err := embedSegments(cache, text)
		if err != nil {
			return nil, err
		}
		built += n
		hit += len(segs) - n
		vecs = append(vecs, ev{i, segs})
	}
	cache.save()
	if built > 0 {
		fmt.Fprintf(os.Stderr, "embedded %d new segments, %d from cache\n", built, hit)
	}

	qv, err := embed(query)
	if err != nil {
		return nil, err
	}

	hits := make([]semHit, 0, len(vecs))
	for _, e := range vecs {
		hits = append(hits, semHit{e.idx, bestCosine(qv, e.segs)})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	return hits, nil
}

func runSemantic(kb *kbIndex, query, dir string, jsonOut bool) {
	hits, err := semanticRank(kb, query, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "semantic: %v\n", err)
		os.Exit(1)
	}
	if len(hits) > 15 {
		hits = hits[:15]
	}

	if jsonOut {
		out := make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			e := kb.Entities[h.idx]
			out = append(out, map[string]any{
				"var_name": e.VarName, "name": e.Name, "score": h.score, "brief": e.Brief, "file": e.File,
			})
		}
		printJSON(map[string]any{"query": query, "model": embedModel, "count": len(hits), "hits": out})
		return
	}

	fmt.Printf("Semantic matches for %q (%s):\n\n", query, embedModel)
	for _, h := range hits {
		e := kb.Entities[h.idx]
		fmt.Printf("  [%.3f] %s (%s)  %s\n", h.score, e.Name, e.VarName, e.File)
		if e.Brief != "" {
			fmt.Printf("        %s\n", cliutil.Truncate(e.Brief, 200))
		}
	}
}

// segmentForEmbed splits text into pieces that each fit inside the embedder's
// window, so content past maxEmbedChars is reachable instead of discarded.
//
// The obvious alternative — a longer-window model embedding the whole entry —
// loses on its own terms. A vector over 1,500 characters is an average over
// everything in them: measured against this shape, whole-document embedding
// scored WORSE than head-only on head queries (MRR 0.810 against 0.908) even
// while it won on tail queries. Segmenting and scoring by the best segment
// (bestCosine) gets both, which is why this exists rather than a bigger cap or
// a different model.
//
// Text at or under the cap comes back as a single unchanged segment. That is
// load-bearing, not tidiness: its embedKey is then exactly what it was before
// segmenting existed, so every short entry's cached vector survives this
// change and only the entries that were being truncated re-embed.
func segmentForEmbed(text string) []string {
	if len(text) <= maxEmbedChars {
		return []string{text}
	}
	var segs []string
	for len(text) > maxEmbedChars {
		cut := embedCut(text)
		if s := strings.TrimSpace(text[:cut]); s != "" {
			segs = append(segs, s)
		}
		text = strings.TrimLeftFunc(text[cut:], unicode.IsSpace)
	}
	if tail := strings.TrimSpace(text); tail != "" {
		segs = append(segs, tail)
	}
	return segs
}

// embedCut picks where to break, preferring the last whitespace in the closing
// sixth of the window so a segment does not begin mid-word.
//
// It always returns a positive, rune-aligned offset. Unbroken text — a long
// path, a base64 blob, a wall of CJK — offers no whitespace to find, and a cut
// of 0 would leave the input unchanged and loop forever.
func embedCut(text string) int {
	cut := maxEmbedChars
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	if cut == 0 {
		return maxEmbedChars // pathological input; guarantee forward progress
	}
	if i := strings.LastIndexFunc(text[:cut], unicode.IsSpace); i > 0 && i > cut-maxEmbedChars/6 {
		return i
	}
	return cut
}

// embedSegments returns one vector per segment of text, filling and reusing the
// cache. The second return is how many were newly embedded, for the progress
// line. Each segment is keyed on its own text, so segments never collide with
// the whole-text keys written before this existed.
func embedSegments(cache *vecCache, text string) ([][]float32, int, error) {
	segs := segmentForEmbed(text)
	out := make([][]float32, 0, len(segs))
	built := 0
	for _, s := range segs {
		if v, ok := cache.m[embedKey(s)]; ok {
			out = append(out, v)
			continue
		}
		v, err := embed(s) // already normalized by embedRetry
		if err != nil {
			return nil, built, err
		}
		cache.m[embedKey(s)] = v
		cache.dirty = true
		out = append(out, v)
		built++
	}
	return out, built, nil
}

// bestCosine scores a document by its single best-matching segment.
//
// Max, not mean. A long entry is usually about several things and a query is
// about one of them, so averaging the segments reintroduces exactly the
// dilution segmenting exists to remove. Vectors are unit length, so the dot
// product is the cosine.
func bestCosine(qv []float32, segs [][]float32) float64 {
	if len(segs) == 0 {
		return 0
	}
	best := math.Inf(-1)
	for _, v := range segs {
		if d := dot(qv, v); d > best {
			best = d
		}
	}
	return best
}
