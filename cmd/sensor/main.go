package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Paper struct {
	PaperID    string   `json:"paperId"`
	Title      string   `json:"title"`
	Year       int      `json:"year"`
	Abstract   string   `json:"abstract"`
	Authors    []Author `json:"authors"`
	CitationCount int   `json:"citationCount"`
	URL        string   `json:"url"`
}

type Author struct {
	AuthorID string `json:"authorId"`
	Name     string `json:"name"`
}

type SearchResult struct {
	Total int     `json:"total"`
	Data  []Paper `json:"data"`
}

type SensorState struct {
	LastPoll time.Time         `json:"lastPoll"`
	Seen     map[string]bool   `json:"seen"`
}

func loadState(path string) SensorState {
	var s SensorState
	s.Seen = map[string]bool{}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	json.Unmarshal(data, &s)
	if s.Seen == nil {
		s.Seen = map[string]bool{}
	}
	return s
}

func saveState(path string, s SensorState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func searchPapers(query string, limit int, apiKey string) ([]Paper, error) {
	u := fmt.Sprintf("https://api.semanticscholar.org/graph/v1/paper/search?query=%s&limit=%d&fields=paperId,title,year,abstract,authors,citationCount,url",
		url.QueryEscape(query), limit)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

func main() {
	dir := flag.String("dir", ".", "winze project directory")
	limit := flag.Int("limit", 10, "results per query")
	jsonOut := flag.Bool("json", false, "output as JSON")
	flag.Parse()

	apiKey := os.Getenv("SEMANTIC_SCHOLAR_API_KEY")

	statePath := filepath.Join(*dir, ".sensor-state.json")
	state := loadState(statePath)

	queries := []string{
		"predictive processing hierarchical prediction",
		"superior pattern processing cognition",
		"apophenia pattern recognition cognitive",
		"forecasting calibration superforecasting",
		"knowledge base consistency checking",
	}

	var newPapers []Paper
	for _, q := range queries {
		papers, err := searchPapers(q, *limit, apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "query %q: %v\n", q, err)
			continue
		}
		for _, p := range papers {
			if p.PaperID == "" || state.Seen[p.PaperID] {
				continue
			}
			if p.Year < 2024 {
				continue
			}
			newPapers = append(newPapers, p)
			state.Seen[p.PaperID] = true
		}
		time.Sleep(3 * time.Second)
	}

	state.LastPoll = time.Now()
	if err := saveState(statePath, state); err != nil {
		fmt.Fprintf(os.Stderr, "save state: %v\n", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(newPapers)
		return
	}

	fmt.Printf("[sensor] %d new papers (since %s)\n", len(newPapers), state.LastPoll.Format("2006-01-02"))
	for _, p := range newPapers {
		authors := make([]string, len(p.Authors))
		for i, a := range p.Authors {
			authors[i] = a.Name
		}
		fmt.Printf("  %s (%d) — %s\n", p.Title, p.Year, strings.Join(authors, ", "))
		if p.Abstract != "" {
			abs := p.Abstract
			if len(abs) > 200 {
				abs = abs[:200] + "..."
			}
			fmt.Printf("    %s\n", abs)
		}
	}
}
