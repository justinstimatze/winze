package main

import (
	"math"
"sort"
	"strings"
)

// BM25 retriever over var-block text. Pure Go implementation of the
// ranking algorithm used by SQLite FTS5. No external dependencies.

const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

type bm25Index struct {
	docs     []varBlock
	avgDL    float64
	df       map[string]int
	docCount int
}

func buildBM25Index(blocks []varBlock) *bm25Index {
	idx := &bm25Index{
		docs: blocks,
		df:   make(map[string]int),
	}
	idx.docCount = len(blocks)

	var totalLen int
	for _, b := range blocks {
		tokens := tokenize(b.text)
		totalLen += len(tokens)
		seen := map[string]bool{}
		for _, t := range tokens {
			if !seen[t] {
				idx.df[t]++
				seen[t] = true
			}
		}
	}
	if idx.docCount > 0 {
		idx.avgDL = float64(totalLen) / float64(idx.docCount)
	}
	return idx
}

func tokenize(text string) []string {
	raw := tokenRe.FindAllString(strings.ToLower(text), -1)
	var out []string
	for _, t := range raw {
		if len(t) > 1 {
			out = append(out, t)
		}
	}
	return out
}

func (idx *bm25Index) score(queryTokens []string, docIdx int) float64 {
	docTokens := tokenize(idx.docs[docIdx].text)
	dl := float64(len(docTokens))

	tf := map[string]int{}
	for _, t := range docTokens {
		tf[t]++
	}

	var score float64
	for _, qt := range queryTokens {
		dfVal := idx.df[qt]
		if dfVal == 0 {
			continue
		}
		idf := math.Log((float64(idx.docCount)-float64(dfVal)+0.5) / (float64(dfVal) + 0.5))
		if idf < 0 {
			idf = 0
		}
		tfVal := float64(tf[qt])
		num := tfVal * (bm25K1 + 1)
		denom := tfVal + bm25K1*(1-bm25B+bm25B*(dl/idx.avgDL))
		score += idf * num / denom
	}
	return score
}

type bm25Result struct {
	name  string
	score float64
}

func fts5Retrieve(dir string, query string, k int) []string {
	blocks, err := collectVarBlocks(dir)
	if err != nil {
		return nil
	}

	idx := buildBM25Index(blocks)
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	var results []bm25Result
	for i, b := range blocks {
		s := idx.score(queryTokens, i)
		if s > 0 {
			results = append(results, bm25Result{name: b.name, score: s})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	var out []string
	for i, r := range results {
		if i >= k {
			break
		}
		out = append(out, r.name)
	}
	return out
}
