package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const ollamaEmbedURL = "http://localhost:11434/api/embeddings"

// rrfK is reciprocal rank fusion's damping constant, same value as cmd/query's
// own rrfFuse. Not imported from there -- this package deliberately keeps its
// own copies of shared mechanics rather than depending on a shipped
// production tool (see the tier-2 transcript plan's note on the boundary).
const rrfK = 60

func dotProduct(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var sum float64
	for i := 0; i < n; i++ {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

// embedModel names the ollama model used for the semantic channel. Mirrors
// cmd/query's own default (all-minilm) for score-distribution consistency
// across winze's tools, but is a fresh local copy, not an import.
func embedModel() string {
	if v := os.Getenv("WINZE_EMBED_MODEL"); v != "" {
		return v
	}
	return "all-minilm"
}

// embedOllama POSTs text to a local ollama server, halving the input and
// retrying on a 500 (the over-length signal) the same way cmd/query's
// embedRetry does, so a denser-than-expected fact degrades to a shorter
// embedding rather than failing the whole semantic pass.
func embedOllama(text string, tries int) ([]float32, error) {
	body, _ := json.Marshal(map[string]string{"model": embedModel(), "prompt": text})
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(ollamaEmbedURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama embed (is `ollama serve` running?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusInternalServerError && tries > 1 && len(text) > 64 {
			return embedOllama(text[:len(text)/2], tries-1)
		}
		return nil, fmt.Errorf("ollama embed status %d (have you run `ollama pull %s`?)", resp.StatusCode, embedModel())
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
	return l2Normalize(out.Embedding), nil
}

// factSemText mirrors rankFacts'/scoreByOverlap's hay text exactly, so the
// term-overlap and semantic channels being fused judge the same content.
func factSemText(f Fact) string {
	return f.Attribute + " " + f.Value + " " + f.Quote
}

// fuseRankMaps combines two rank maps (fact index -> 1-based rank) via
// reciprocal rank fusion and returns fact indices ordered by descending
// fused score, highest first. Pure and deterministic -- the testable core of
// semanticFuseFacts, independent of term-overlap scoring and embedding calls.
func fuseRankMaps(a, b map[int]int) []int {
	rrfScore := func(idx int) float64 {
		var s float64
		if r, ok := a[idx]; ok {
			s += 1 / float64(rrfK+r)
		}
		if r, ok := b[idx]; ok {
			s += 1 / float64(rrfK+r)
		}
		return s
	}
	idxs := collectIndices(a, b)
	sort.SliceStable(idxs, func(i, j int) bool {
		si, sj := rrfScore(idxs[i]), rrfScore(idxs[j])
		if si != sj {
			return si > sj
		}
		return idxs[i] < idxs[j]
	})
	return idxs
}

// l2Normalize scales v to unit length so a dot product between two normalized
// vectors is the cosine similarity.
func l2Normalize(v []float32) []float32 {
	var sumSq float64
	for _, x := range v {
		sumSq += float64(x) * float64(x)
	}
	if sumSq == 0 {
		return v
	}
	norm := float32(math.Sqrt(sumSq))
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = x / norm
	}
	return out
}

// embedCached embeds text through an on-disk cache keyed by sha256(model +
// text), so re-runs and facts shared across questions cost nothing -- same
// content-hash-plus-atomic-rename shape as extractSession's cache, and for
// the same reason: a concurrent question loop can miss the same key at once,
// and per-key files make that race harmless instead of needing a shared
// in-memory map and a lock.
func (r *runner) embedCached(text string) ([]float32, error) {
	h := sha256.Sum256([]byte(embedModel() + "\x00" + text))
	key := hex.EncodeToString(h[:])
	cachePath := filepath.Join(r.embedCacheDir, key+".json")
	if v, ok := readEmbedCache(cachePath); ok {
		return v, nil
	}
	v, err := embedOllama(text, 3)
	if err != nil {
		return nil, err
	}
	writeEmbedCache(r.embedCacheDir, cachePath, v)
	return v, nil
}

// semanticFuseFacts retrieves by RRF-fusing scoreByOverlap's term-overlap
// ranking with embedding cosine similarity, targeting the class of miss term
// overlap structurally can't reach: real signal present with zero vocabulary
// overlap between the question and the fact (see ROADMAP.md's 35a27287
// finding -- 47 "french" hits, 37 "language" hits, still wrong, because the
// question's own words never appear in the stored preference). Fails open to
// rankFacts if the question itself can't be embedded (ollama unreachable).
func (r *runner) semanticFuseFacts(facts []Fact, question string, k int) []Fact {
	qv, err := r.embedCached(question)
	if err != nil {
		return rankFacts(facts, question, k)
	}
	termOrder := scoreByOverlap(facts, question)
	termRank := make(map[int]int, len(termOrder))
	for rank, idx := range termOrder {
		termRank[idx] = rank + 1
	}
	semRank := r.semanticRankFacts(facts, qv)
	order := fuseRankMaps(termRank, semRank)
	out := make([]Fact, 0, k)
	for i := 0; i < len(order) && i < k; i++ {
		out = append(out, facts[order[i]])
	}
	return out
}

// collectIndices unions the keys of any number of rank maps, each key once,
// in first-seen order. Shared by fuseRankMaps so the two-map union isn't two
// separate nested loops.
func collectIndices(maps ...map[int]int) []int {
	seen := map[int]bool{}
	var idxs []int
	for _, m := range maps {
		for idx := range m {
			if !seen[idx] {
				seen[idx] = true
				idxs = append(idxs, idx)
			}
		}
	}
	return idxs
}

// semanticRankFacts embeds every fact and scores it by cosine similarity
// against an already-embedded question, returning a rank map (fact index ->
// 1-based rank). A single fact's embedding failure only drops that fact from
// this channel, since term overlap still covers it.
func (r *runner) semanticRankFacts(facts []Fact, qv []float32) map[int]int {
	type scored struct {
		idx   int
		score float64
	}
	var ss []scored
	for i, f := range facts {
		fv, err := r.embedCached(factSemText(f))
		if err != nil {
			continue
		}
		ss = append(ss, scored{idx: i, score: dotProduct(qv, fv)})
	}
	sort.SliceStable(ss, func(a, b int) bool {
		if ss[a].score != ss[b].score {
			return ss[a].score > ss[b].score
		}
		return ss[a].idx < ss[b].idx
	})
	semRank := make(map[int]int, len(ss))
	for rank, s := range ss {
		semRank[s.idx] = rank + 1
	}
	return semRank
}

func readEmbedCache(path string) ([]float32, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var v []float32
	if json.Unmarshal(b, &v) != nil {
		return nil, false
	}
	return v, true
}

func writeEmbedCache(dir, path string, v []float32) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return
	}
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr == nil && cerr == nil {
		_ = os.Rename(tmp.Name(), path)
	} else {
		_ = os.Remove(tmp.Name())
	}
}
