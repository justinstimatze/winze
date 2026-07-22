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
)

const (
	embedModel    = "all-minilm"
	ollamaEmbed   = "http://localhost:11434/api/embeddings"
	embedCacheDir = ".winze-embed"
)

func embed(text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]string{"model": embedModel, "prompt": text})
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(ollamaEmbed, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama embed (is `ollama serve` running?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
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

func runSemantic(kb *kbIndex, query, dir string, jsonOut bool) {
	cache := loadVecCache(dir)

	type ev struct {
		idx int
		vec []float32
	}
	var vecs []ev
	built, hit := 0, 0
	for i, e := range kb.Entities {
		text := embedText(e)
		if text == "" || text == "." {
			continue
		}
		if v, ok := cache.m[embedKey(text)]; ok {
			vecs = append(vecs, ev{i, v})
			hit++
			continue
		}
		v, err := embed(text)
		if err != nil {
			fmt.Fprintf(os.Stderr, "semantic: %v\n", err)
			os.Exit(1)
		}
		cache.m[embedKey(text)] = v
		cache.dirty = true
		vecs = append(vecs, ev{i, v})
		built++
	}
	cache.save()
	if built > 0 {
		fmt.Fprintf(os.Stderr, "embedded %d new entities, %d from cache\n", built, hit)
	}

	qv, err := embed(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "semantic: %v\n", err)
		os.Exit(1)
	}

	hits := make([]semHit, 0, len(vecs))
	for _, e := range vecs {
		hits = append(hits, semHit{e.idx, dot(qv, e.vec)})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
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
			fmt.Printf("        %s\n", truncate(e.Brief, 200))
		}
	}
}
