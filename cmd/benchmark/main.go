// Command benchmark runs winze's retrieval benchmark v0.1.
//
// Four retrieval modes compete on the same 24-question corpus:
//   - grep: keyword match over var-block text
//   - bm25: BM25 ranking over var-block text (proper unstructured baseline)
//   - defn: SQL queries against defn's Dolt database (structured, realistic)
//   - ast:  hand-written go/ast queries (structured, ceiling)
//
// Usage:
//
//	go run ./cmd/benchmark .
//	go run ./cmd/benchmark /path/to/winze
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const topK = 10

type Result struct {
	QuestionID string
	Category   string
	Mode       string
	Retrieved  []string
	Gold       []string
	Recall     float64
	Precision  float64
	Latency    time.Duration
}



func computeSetRecall(retrieved, gold []string) float64 {
	if len(gold) == 0 {
		return 1.0
	}
	goldSet := map[string]bool{}
	for _, g := range gold {
		goldSet[g] = true
	}
	hits := 0
	for _, r := range retrieved {
		if goldSet[r] {
			hits++
		}
	}
	return float64(hits) / float64(len(gold))
}

func computePrecision(retrieved, gold []string) float64 {
	if len(retrieved) == 0 {
		return 0.0
	}
	goldSet := map[string]bool{}
	for _, g := range gold {
		goldSet[g] = true
	}
	hits := 0
	for _, r := range retrieved {
		if goldSet[r] {
			hits++
		}
	}
	return float64(hits) / float64(len(retrieved))
}

func normalizeResult(retrieved []string) []string {
	var out []string
	for _, r := range retrieved {
		r = strings.TrimSpace(r)
		if r != "" {
			out = append(out, r)
		}
	}
	return out
}

func matchGold(retrieved, gold []string) float64 {
	if len(gold) == 1 {
		g := gold[0]
		if isNumericGold(g) {
			for _, r := range retrieved {
				if r == g {
					return 1.0
				}
			}
			return 0.0
		}
	}
	return computeSetRecall(retrieved, gold)
}

func isNumericGold(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func runMode(dir string, mode string, q Question) Result {
	start := time.Now()
	var retrieved []string

	switch mode {
	case "grep":
		retrieved = grepRetrieve(dir, q.Query, topK)
	case "bm25":
		retrieved = fts5Retrieve(dir, q.Query, topK)
	case "defn":
		retrieved = defnRetrieve(q)
	case "ast":
		retrieved = astRetrieve(dir, q)
	}

	retrieved = normalizeResult(retrieved)
	elapsed := time.Since(start)

	recall := matchGold(retrieved, q.Gold)
	precision := computePrecision(retrieved, q.Gold)

	return Result{
		QuestionID: q.ID,
		Category:   q.Category,
		Mode:       mode,
		Retrieved:  retrieved,
		Gold:       q.Gold,
		Recall:     recall,
		Precision:  precision,
		Latency:    elapsed,
	}
}

type categoryStats struct {
	category string
	count    int
	recall   map[string]float64
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	verbose := false
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		}
	}

	modes := []string{"grep", "bm25", "defn", "ast"}
	categories := []string{"lexical", "aggregation", "multi-hop", "contested"}

	stats := map[string]*categoryStats{}
	for _, cat := range categories {
		stats[cat] = &categoryStats{
			category: cat,
			recall:   map[string]float64{},
		}
	}

	var allResults []Result

	for _, q := range questions {
		for _, mode := range modes {
			r := runMode(dir, mode, q)
			allResults = append(allResults, r)
			s := stats[q.Category]
			s.count = countQuestionsInCategory(q.Category)
			s.recall[mode] += r.Recall
		}
	}

	fmt.Println()
	fmt.Printf("winze benchmark v0.1  (%d questions, %d modes)\n", len(questions), len(modes))
	fmt.Println(strings.Repeat("─", 72))
	fmt.Printf("%-20s", "")
	for _, m := range modes {
		fmt.Printf("  %s R@1", m)
	}
	fmt.Println()

	var overallRecall = map[string]float64{}

	for _, cat := range categories {
		s := stats[cat]
		n := float64(s.count)
		label := strings.ToUpper(cat)
		if len(label) > 12 {
			label = label[:12]
		}
		fmt.Printf("%-16s(%d)", label, s.count)
		for _, m := range modes {
			avg := s.recall[m] / n
			fmt.Printf("    %5.3f  ", avg)
			overallRecall[m] += s.recall[m]
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("─", 72))
	total := float64(len(questions))
	fmt.Printf("%-16s(%d)", "OVERALL", len(questions))
	for _, m := range modes {
		fmt.Printf("    %5.3f  ", overallRecall[m]/total)
	}
	fmt.Println()
	fmt.Println()

	if verbose {
		fmt.Println("=== Per-question detail ===")
		fmt.Println()
		sort.Slice(allResults, func(i, j int) bool {
			if allResults[i].QuestionID != allResults[j].QuestionID {
				return allResults[i].QuestionID < allResults[j].QuestionID
			}
			return allResults[i].Mode < allResults[j].Mode
		})
		currentQ := ""
		for _, r := range allResults {
			if r.QuestionID != currentQ {
				currentQ = r.QuestionID
				q := findQuestion(r.QuestionID)
				fmt.Printf("[%s] %s\n", r.QuestionID, q.Query)
				fmt.Printf("  gold: %v\n", q.Gold)
			}
			status := "✗"
			if r.Recall >= 1.0 {
				status = "✓"
			} else if r.Recall > 0 {
				status = "◐"
			}
			retrieved := r.Retrieved
			if len(retrieved) > 5 {
				retrieved = retrieved[:5]
			}
			fmt.Printf("  %-6s %s  recall=%.3f  %v\n", r.Mode, status, r.Recall, retrieved)
		}
	}
}

func countQuestionsInCategory(cat string) int {
	n := 0
	for _, q := range questions {
		if q.Category == cat {
			n++
		}
	}
	return n
}

func findQuestion(id string) Question {
	for _, q := range questions {
		if q.ID == id {
			return q
		}
	}
	return Question{}
}
