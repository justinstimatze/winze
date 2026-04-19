package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

// --- RSS/Atom feed backend ---

// atomFeed parses Atom feeds (most academic/news feeds use Atom or RSS 2.0).
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string `xml:"title"`
	Link      string `xml:"id"` // Atom uses <id> as permalink
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
	Summary   string `xml:"summary"`
}

// rssFeed parses RSS 2.0 feeds.
type rssFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
	Desc    string `xml:"description"`
	GUID    string `xml:"guid"`
}

// searchRSS fetches an RSS or Atom feed URL, optionally filtering entries
// by a query string (case-insensitive substring match on title + description).
// If query is empty, returns all entries up to limit.
func searchRSS(feedURL, query string, limit int) ([]PaperSummary, error) {
	resp, err := httpClient.Get(feedURL)
	if err != nil {
		return nil, fmt.Errorf("rss fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("rss %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("rss read: %w", err)
	}

	// Try Atom first, fall back to RSS 2.0
	papers, parsed := tryAtomParse(body, query, limit)
	if !parsed {
		papers, parsed = tryRSSParse(body, query, limit)
	}
	if !parsed {
		return nil, fmt.Errorf("rss: could not parse feed from %s", feedURL)
	}
	return papers, nil
}

func tryAtomParse(data []byte, query string, limit int) ([]PaperSummary, bool) {
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil || len(feed.Entries) == 0 {
		return nil, false
	}

	var papers []PaperSummary
	for _, e := range feed.Entries {
		if !matchesAnyTerm(e.Title+e.Summary, query) {
			continue
		}
		year := parseYear(e.Published)
		if year == 0 {
			year = parseYear(e.Updated)
		}
		papers = append(papers, PaperSummary{
			ID:    e.Link,
			Title: strings.Join(strings.Fields(e.Title), " "),
			Year:  year,
		})
		if len(papers) >= limit {
			break
		}
	}
	return papers, true
}

func tryRSSParse(data []byte, query string, limit int) ([]PaperSummary, bool) {
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil || len(feed.Channel.Items) == 0 {
		return nil, false
	}

	var papers []PaperSummary
	for _, item := range feed.Channel.Items {
		if !matchesAnyTerm(item.Title+item.Desc, query) {
			continue
		}
		id := item.GUID
		if id == "" {
			id = item.Link
		}
		papers = append(papers, PaperSummary{
			ID:    id,
			Title: strings.Join(strings.Fields(item.Title), " "),
			Year:  parseYear(item.PubDate),
		})
		if len(papers) >= limit {
			break
		}
	}
	return papers, true
}

// matchesAnyTerm returns true if at least one content word from query
// (length ≥ 4, case-insensitive) appears as a substring of text. Blank
// query matches everything.
//
// The prior matcher required the entire query string as one verbatim
// substring — fine for ZIM fulltext search, disastrous for RSS, where
// topic feeds virtually never contain a multi-word entity-derived
// phrase verbatim. Any-term matching hands llmResolve a broader
// candidate set; relevance filtering happens at classification time.
//
// This is still a weak matcher. The real unlock for RSS signal is
// feed curation — current default feeds are topic-broad streams
// (Nature Reviews Neuroscience, PhilPapers consciousness) unlikely to
// mention specific KB entities regardless of matcher quality. The
// natural upgrade for the "real-time external sensor" slot is a
// search API (kagi, google-scholar) that builds URLs per-query rather
// than polling static feeds.
func matchesAnyTerm(text, query string) bool {
	if query == "" {
		return true
	}
	lt := strings.ToLower(text)
	for _, w := range strings.Fields(strings.ToLower(query)) {
		w = strings.Trim(w, ".,;:\"'()-/")
		if len(w) < 4 {
			continue
		}
		if strings.Contains(lt, w) {
			return true
		}
	}
	return false
}

func parseYear(dateStr string) int {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"Mon, 02 Jan 2006 15:04:05 -0700", // RSS pubDate
		"Mon, 02 Jan 2006 15:04:05 MST",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, strings.TrimSpace(dateStr)); err == nil {
			return t.Year()
		}
	}
	return 0
}

// --- RSS feed configuration ---

// defaultFeeds are curated RSS/Atom feeds relevant to the KB's domain
// (epistemology of minds, cognitive science, AI). Users can add custom
// feeds via --rss-feeds flag (comma-separated URLs).
var defaultFeeds = []string{
	// Cognitive science / neuroscience
	"https://www.nature.com/nrn.rss",                   // Nature Reviews Neuroscience
	"https://rss.sciencedirect.com/publication/science/1364-6613", // Trends in Cognitive Sciences
	// AI / ML
	"https://arxiv.org/rss/cs.AI",   // arXiv cs.AI new submissions
	"https://arxiv.org/rss/cs.CL",   // arXiv cs.CL (computation + language)
	// Philosophy of mind
	"https://philpapers.org/asearch.pl?filterMode=filt&start=0&format=atom&sqc=&categorizerModule=default&onlineOnly=&newWindow=&publishedOnly=&langFilter=&catId=5892", // PhilPapers consciousness
}
