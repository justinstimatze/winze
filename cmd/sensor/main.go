package main

import (
	"encoding/json"
	"encoding/xml"
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
	PaperID       string   `json:"paperId"`
	Title         string   `json:"title"`
	Year          int      `json:"year"`
	Abstract      string   `json:"abstract"`
	Authors       []Author `json:"authors"`
	CitationCount int      `json:"citationCount,omitempty"`
	URL           string   `json:"url"`
	Source        string   `json:"source"`
}

type Author struct {
	AuthorID string `json:"authorId,omitempty"`
	Name     string `json:"name"`
}

// Semantic Scholar types

type ScholarResult struct {
	Total int     `json:"total"`
	Data  []Paper `json:"data"`
}

// arXiv Atom types

type arxivFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	ID        string        `xml:"id"`
	Title     string        `xml:"title"`
	Summary   string        `xml:"summary"`
	Published string        `xml:"published"`
	Authors   []arxivAuthor `xml:"author"`
	Links     []arxivLink   `xml:"link"`
}

type arxivAuthor struct {
	Name string `xml:"name"`
}

type arxivLink struct {
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr"`
}

// State tracking

type SensorState struct {
	LastPoll time.Time       `json:"lastPoll"`
	Seen     map[string]bool `json:"seen"`
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

// arXiv backend

func searchArxiv(query string, limit int) ([]Paper, error) {
	u := fmt.Sprintf("http://export.arxiv.org/api/query?search_query=all:%s&start=0&max_results=%d&sortBy=submittedDate&sortOrder=descending",
		url.QueryEscape(query), limit)

	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("arXiv API returned %d: %s", resp.StatusCode, string(body))
	}

	var feed arxivFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	var papers []Paper
	for _, e := range feed.Entries {
		year := 0
		if t, err := time.Parse(time.RFC3339, e.Published); err == nil {
			year = t.Year()
		}

		authors := make([]Author, len(e.Authors))
		for i, a := range e.Authors {
			authors[i] = Author{Name: strings.TrimSpace(a.Name)}
		}

		link := e.ID
		for _, l := range e.Links {
			if l.Type == "text/html" {
				link = l.Href
				break
			}
		}

		papers = append(papers, Paper{
			PaperID:  e.ID,
			Title:    strings.Join(strings.Fields(e.Title), " "),
			Year:     year,
			Abstract: strings.Join(strings.Fields(e.Summary), " "),
			Authors:  authors,
			URL:      link,
			Source:   "arxiv",
		})
	}
	return papers, nil
}

// Semantic Scholar backend

func searchScholar(query string, limit int, apiKey string) ([]Paper, error) {
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

	var result ScholarResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	for i := range result.Data {
		result.Data[i].Source = "scholar"
	}
	return result.Data, nil
}

func main() {
	dir := flag.String("dir", ".", "winze project directory")
	limit := flag.Int("limit", 10, "results per query")
	backend := flag.String("backend", "arxiv", "backend: arxiv, scholar, all")
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

	useArxiv := *backend == "arxiv" || *backend == "all"
	useScholar := *backend == "scholar" || *backend == "all"

	var newPapers []Paper
	for _, q := range queries {
		if useArxiv {
			papers, err := searchArxiv(q, *limit)
			if err != nil {
				fmt.Fprintf(os.Stderr, "arxiv %q: %v\n", q, err)
			} else {
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
			}
			time.Sleep(3 * time.Second)
		}

		if useScholar {
			papers, err := searchScholar(q, *limit, apiKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "scholar %q: %v\n", q, err)
			} else {
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
			}
			time.Sleep(3 * time.Second)
		}
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
		fmt.Printf("  [%s] %s (%d) — %s\n", p.Source, p.Title, p.Year, strings.Join(authors, ", "))
		if p.Abstract != "" {
			abs := p.Abstract
			if len(abs) > 200 {
				abs = abs[:200] + "..."
			}
			fmt.Printf("    %s\n", abs)
		}
	}
}
