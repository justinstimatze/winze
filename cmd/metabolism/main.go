// Command metabolism runs one cycle of the epistemic metabolism loop:
//
//  1. Topology analysis identifies structurally fragile hypotheses
//  2. Sensor queries for external signal on each target (arXiv and/or Wikipedia ZIM)
//  3. Results are logged to .metabolism-log.json for calibration
//
// The core testable claim: structural vulnerability (single-source,
// uncontested, thin provenance) predicts where curation gaps exist.
// Calibration measures whether topology-driven sensor queries actually
// find relevant signal more often than random queries would.
//
// Usage:
//
//	go run ./cmd/metabolism .                              # run one cycle (arXiv)
//	go run ./cmd/metabolism --backend zim --zim /opt/zim/wikipedia.zim .  # Wikipedia ZIM
//	go run ./cmd/metabolism --backend all --zim /opt/zim/wikipedia.zim .  # both backends
//	go run ./cmd/metabolism --dry-run .                    # show targets only
//	go run ./cmd/metabolism --calibrate .                  # analyze log
//	go run ./cmd/metabolism --json .                       # JSON output
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
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	zim "github.com/justinstimatze/gozim"
)

// --- topology output types (subset) ---

type TopologyReport struct {
	Entities      int            `json:"entities"`
	Claims        int            `json:"claims"`
	Edges         int            `json:"edges"`
	Clusters      int            `json:"clusters"`
	SensorTargets []SensorTarget `json:"sensor_targets"`
}

type SensorTarget struct {
	Hypothesis string `json:"hypothesis"`
	Query      string `json:"query"`
	Prediction string `json:"prediction"`
	VulnType   string `json:"vuln_type"`
	VulnCount  int    `json:"vuln_count"`
}

// --- arXiv types ---

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
}

type arxivAuthor struct {
	Name string `xml:"name"`
}

// --- metabolism log ---

type MetabolismLog struct {
	Cycles []Cycle `json:"cycles"`
}

type Cycle struct {
	Timestamp   time.Time      `json:"timestamp"`
	Hypothesis  string         `json:"hypothesis"`
	Prediction  string         `json:"prediction"`
	Query       string         `json:"query"`
	Backend     string         `json:"backend,omitempty"` // "arxiv" or "zim"; empty = arxiv (legacy)
	VulnType    string         `json:"vuln_type"`
	VulnCount   int            `json:"vuln_count"`
	PapersFound int            `json:"papers_found"`
	Papers      []PaperSummary `json:"papers,omitempty"`
	// Resolution is set after review. Values:
	//   "corroborated" — signal supports existing claim, curation gap confirmed
	//   "challenged"   — signal contradicts existing claim, revision needed
	//   "irrelevant"   — signal found but not relevant to the hypothesis
	//   ""             — not yet reviewed
	Resolution string `json:"resolution,omitempty"`
	ResolvedAt string `json:"resolved_at,omitempty"` // ISO 8601 date
}

type PaperSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Year  int    `json:"year"`
}

func main() {
	limit := flag.Int("limit", 5, "max results per query")
	dryRun := flag.Bool("dry-run", false, "show targets without querying")
	calibrate := flag.Bool("calibrate", false, "analyze existing log instead of running a cycle")
	resolve := flag.String("resolve", "", "resolve a hypothesis: HYPOTHESIS=corroborated|challenged|irrelevant")
	jsonOut := flag.Bool("json", false, "output as JSON")
	backend := flag.String("backend", "arxiv", "sensor backend: arxiv, zim, or all")
	zimPath := flag.String("zim", "", "path to .zim file (required for zim backend)")
	flag.Parse()

	validBackends := map[string]bool{"arxiv": true, "zim": true, "all": true}
	if !validBackends[*backend] {
		fmt.Fprintf(os.Stderr, "metabolism: --backend must be arxiv, zim, or all (got %q)\n", *backend)
		os.Exit(1)
	}
	useArxiv := *backend == "arxiv" || *backend == "all"
	useZim := *backend == "zim" || *backend == "all"
	if useZim && *zimPath == "" {
		fmt.Fprintf(os.Stderr, "metabolism: --zim path required when backend includes zim\n")
		os.Exit(1)
	}

	defer func() {
		if zimArchive != nil {
			zimArchive.Close()
		}
	}()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	if *resolve != "" {
		runResolve(dir, *resolve)
		return
	}

	if *calibrate {
		runCalibrate(dir, *jsonOut)
		return
	}

	// 1. Run topology analysis
	targets, report, err := runTopology(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "metabolism: topology: %v\n", err)
		os.Exit(1)
	}

	if len(targets) == 0 {
		fmt.Println("[metabolism] no sensor targets — graph is well-covered")
		return
	}

	if *dryRun {
		fmt.Printf("[metabolism] %d targets from topology (%d entities, %d claims):\n",
			len(targets), report.Entities, report.Claims)
		for _, t := range targets {
			fmt.Printf("  %s → %q\n    prediction: %s\n", t.Hypothesis, t.Query, t.Prediction)
		}
		return
	}

	// 2. Query sensors for each target
	var cycles []Cycle
	for i, t := range targets {
		// arXiv backend
		if useArxiv {
			if i > 0 {
				time.Sleep(5 * time.Second) // arXiv rate limit
			}

			papers, err := searchArxiv(t.Query, *limit)
			if err != nil {
				if strings.Contains(err.Error(), "429") {
					fmt.Fprintf(os.Stderr, "metabolism: rate limited, waiting 30s...\n")
					time.Sleep(30 * time.Second)
					papers, err = searchArxiv(t.Query, *limit)
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "metabolism: arxiv %q: %v\n", t.Query, err)
				}
			}
			if err == nil {
				var recent []PaperSummary
				for _, p := range papers {
					if p.Year >= 2024 {
						recent = append(recent, p)
					}
				}
				cycles = append(cycles, Cycle{
					Timestamp:   time.Now(),
					Hypothesis:  t.Hypothesis,
					Prediction:  t.Prediction,
					Query:       t.Query,
					Backend:     "arxiv",
					VulnType:    t.VulnType,
					VulnCount:   t.VulnCount,
					PapersFound: len(recent),
					Papers:      recent,
				})
			}
		}

		// ZIM backend
		if useZim {
			articles, err := searchZim(*zimPath, t.Query, *limit)
			if err != nil {
				fmt.Fprintf(os.Stderr, "metabolism: zim %q: %v\n", t.Query, err)
			} else {
				cycles = append(cycles, Cycle{
					Timestamp:   time.Now(),
					Hypothesis:  t.Hypothesis,
					Prediction:  t.Prediction,
					Query:       t.Query,
					Backend:     "zim",
					VulnType:    t.VulnType,
					VulnCount:   t.VulnCount,
					PapersFound: len(articles),
					Papers:      articles,
				})
			}
		}
	}

	// 3. Append to log
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)
	mlog.Cycles = append(mlog.Cycles, cycles...)
	if err := saveLog(logPath, mlog); err != nil {
		fmt.Fprintf(os.Stderr, "metabolism: save log: %v\n", err)
	}

	// 4. Output
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(cycles)
		return
	}

	total := 0
	for _, c := range cycles {
		total += c.PapersFound
	}
	fmt.Printf("[metabolism] cycle complete — %d targets, %d results found\n\n", len(cycles), total)
	for _, c := range cycles {
		signal := "no signal"
		if c.PapersFound > 0 {
			label := "papers"
			if c.Backend == "zim" {
				label = "articles"
			}
			signal = fmt.Sprintf("%d %s", c.PapersFound, label)
		}
		be := c.Backend
		if be == "" {
			be = "arxiv"
		}
		fmt.Printf("  %s [%s] (%s)\n", c.Hypothesis, signal, be)
		fmt.Printf("    query: %q\n", c.Query)
		fmt.Printf("    prediction: %s\n", c.Prediction)
		for _, p := range c.Papers {
			if p.Year > 0 {
				fmt.Printf("    → [%d] %s\n", p.Year, p.Title)
			} else {
				fmt.Printf("    → %s\n", p.Title)
			}
		}
	}

	// 5. Running calibration stats
	withSignal := 0
	for _, c := range mlog.Cycles {
		if c.PapersFound > 0 {
			withSignal++
		}
	}
	fmt.Printf("\n[calibration] %d total cycles logged, signal rate %.0f%% (%d/%d)\n",
		len(mlog.Cycles), pct(withSignal, len(mlog.Cycles)), withSignal, len(mlog.Cycles))
}

// runTopology shells out to cmd/topology and parses the JSON output.
func runTopology(dir string) ([]SensorTarget, TopologyReport, error) {
	cmd := exec.Command("go", "run", "./cmd/topology", "--json", dir)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, TopologyReport{}, fmt.Errorf("run topology: %w", err)
	}

	var report TopologyReport
	if err := json.Unmarshal(out, &report); err != nil {
		return nil, TopologyReport{}, fmt.Errorf("parse topology output: %w", err)
	}

	return report.SensorTargets, report, nil
}

// searchArxiv queries the arXiv API. Terms are AND-joined.
func searchArxiv(query string, limit int) ([]PaperSummary, error) {
	terms := strings.Fields(query)
	var parts []string
	for _, t := range terms {
		parts = append(parts, "all:"+url.QueryEscape(t))
	}
	searchQuery := strings.Join(parts, "+AND+")
	u := fmt.Sprintf("https://export.arxiv.org/api/query?search_query=%s&start=0&max_results=%d&sortBy=submittedDate&sortOrder=descending",
		searchQuery, limit)

	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("arXiv API returned %d: %s", resp.StatusCode, string(body))
	}

	var feed arxivFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	var papers []PaperSummary
	for _, e := range feed.Entries {
		year := 0
		if t, err := time.Parse(time.RFC3339, e.Published); err == nil {
			year = t.Year()
		}
		papers = append(papers, PaperSummary{
			ID:    e.ID,
			Title: strings.Join(strings.Fields(e.Title), " "),
			Year:  year,
		})
	}
	return papers, nil
}

// zimArchive is lazily opened on first use and reused across queries.
var zimArchive *zim.Archive

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)
var wsCollapseRe = regexp.MustCompile(`\s+`)

func stripHTML(html []byte) string {
	text := htmlTagRe.ReplaceAll(html, []byte(" "))
	return strings.TrimSpace(wsCollapseRe.ReplaceAllString(string(text), " "))
}

// searchZim uses gozim for fulltext search against a ZIM file.
// On first call, opens the archive and builds/opens a Bleve index.
// Returns PaperSummary (Title is article title, ID is the ZIM path).
func searchZim(zimPath, query string, limit int) ([]PaperSummary, error) {
	if zimArchive == nil {
		a, err := zim.Open(zimPath, zim.WithMmap())
		if err != nil {
			return nil, fmt.Errorf("open zim: %w", err)
		}
		zimArchive = a
	}

	results, err := zimArchive.Search(query, limit)
	if err != nil {
		return nil, fmt.Errorf("search zim: %w", err)
	}

	var papers []PaperSummary
	for _, r := range results {
		papers = append(papers, PaperSummary{
			ID:    "zim:" + r.Entry.Path(),
			Title: r.Entry.Title(),
			Year:  0, // encyclopedia, not dated
		})
	}
	return papers, nil
}

// --- log persistence ---

func loadLog(path string) MetabolismLog {
	var mlog MetabolismLog
	data, err := os.ReadFile(path)
	if err != nil {
		return mlog
	}
	_ = json.Unmarshal(data, &mlog)
	return mlog
}

func saveLog(path string, mlog MetabolismLog) error {
	data, err := json.MarshalIndent(mlog, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// --- resolution ---

func runResolve(dir, spec string) {
	parts := strings.SplitN(spec, "=", 2)
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "metabolism: --resolve expects HYPOTHESIS=outcome (corroborated|challenged|irrelevant)\n")
		os.Exit(1)
	}
	hypothesis, outcome := parts[0], parts[1]

	valid := map[string]bool{"corroborated": true, "challenged": true, "irrelevant": true}
	if !valid[outcome] {
		fmt.Fprintf(os.Stderr, "metabolism: outcome must be corroborated, challenged, or irrelevant (got %q)\n", outcome)
		os.Exit(1)
	}

	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	found := false
	for i := len(mlog.Cycles) - 1; i >= 0; i-- {
		if mlog.Cycles[i].Hypothesis == hypothesis && mlog.Cycles[i].Resolution == "" {
			mlog.Cycles[i].Resolution = outcome
			mlog.Cycles[i].ResolvedAt = time.Now().Format("2006-01-02")
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "metabolism: no unresolved cycle found for %q\n", hypothesis)
		os.Exit(1)
	}

	if err := saveLog(logPath, mlog); err != nil {
		fmt.Fprintf(os.Stderr, "metabolism: save log: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[metabolism] resolved %s → %s\n", hypothesis, outcome)
}

// --- calibration ---

func runCalibrate(dir string, jsonOut bool) {
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	if len(mlog.Cycles) == 0 {
		fmt.Println("[calibrate] no cycles logged yet — run a metabolism cycle first")
		return
	}

	type vulnStats struct {
		total      int
		withSignal int
		totalPaper int
	}

	byVuln := map[string]*vulnStats{}
	overall := &vulnStats{}

	for _, c := range mlog.Cycles {
		overall.total++
		overall.totalPaper += c.PapersFound
		if c.PapersFound > 0 {
			overall.withSignal++
		}

		key := c.VulnType
		if byVuln[key] == nil {
			byVuln[key] = &vulnStats{}
		}
		s := byVuln[key]
		s.total++
		s.totalPaper += c.PapersFound
		if c.PapersFound > 0 {
			s.withSignal++
		}
	}

	// Find date range
	earliest := mlog.Cycles[0].Timestamp
	latest := mlog.Cycles[0].Timestamp
	for _, c := range mlog.Cycles[1:] {
		if c.Timestamp.Before(earliest) {
			earliest = c.Timestamp
		}
		if c.Timestamp.After(latest) {
			latest = c.Timestamp
		}
	}

	if jsonOut {
		type CalReport struct {
			TotalCycles int     `json:"total_cycles"`
			SignalRate   float64 `json:"signal_rate"`
			WithSignal  int     `json:"with_signal"`
			TotalPapers int     `json:"total_papers"`
			Earliest    string  `json:"earliest"`
			Latest      string  `json:"latest"`
		}
		r := CalReport{
			TotalCycles: overall.total,
			SignalRate:   pct(overall.withSignal, overall.total),
			WithSignal:  overall.withSignal,
			TotalPapers: overall.totalPaper,
			Earliest:    earliest.Format("2006-01-02"),
			Latest:      latest.Format("2006-01-02"),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(r)
		return
	}

	fmt.Printf("[calibrate] %d cycles logged (%s to %s)\n\n",
		len(mlog.Cycles), earliest.Format("2006-01-02"), latest.Format("2006-01-02"))
	fmt.Printf("  overall signal rate: %.0f%% (%d/%d targets had papers)\n",
		pct(overall.withSignal, overall.total), overall.withSignal, overall.total)
	fmt.Printf("  total papers found:  %d (avg %.1f per target)\n\n",
		overall.totalPaper, avg(overall.totalPaper, overall.total))

	fmt.Println("  by vulnerability type:")
	for vt, s := range byVuln {
		fmt.Printf("    %-25s %.0f%% signal (%d/%d), avg %.1f papers\n",
			vt, pct(s.withSignal, s.total), s.withSignal, s.total, avg(s.totalPaper, s.total))
	}

	// Resolution stats
	resolutions := map[string]int{}
	unresolved := 0
	for _, c := range mlog.Cycles {
		if c.Resolution != "" {
			resolutions[c.Resolution]++
		} else {
			unresolved++
		}
	}
	if len(resolutions) > 0 || unresolved > 0 {
		fmt.Println("  resolutions:")
		for _, r := range []string{"corroborated", "challenged", "irrelevant"} {
			if n := resolutions[r]; n > 0 {
				fmt.Printf("    %-15s %d\n", r, n)
			}
		}
		if unresolved > 0 {
			fmt.Printf("    %-15s %d\n", "pending", unresolved)
		}
	}

	// Per-backend stats
	byBackend := map[string]*vulnStats{}
	for _, c := range mlog.Cycles {
		be := c.Backend
		if be == "" {
			be = "arxiv"
		}
		if byBackend[be] == nil {
			byBackend[be] = &vulnStats{}
		}
		s := byBackend[be]
		s.total++
		s.totalPaper += c.PapersFound
		if c.PapersFound > 0 {
			s.withSignal++
		}
	}
	if len(byBackend) > 1 {
		fmt.Println("\n  by backend:")
		for be, s := range byBackend {
			fmt.Printf("    %-25s %.0f%% signal (%d/%d), avg %.1f results\n",
				be, pct(s.withSignal, s.total), s.withSignal, s.total, avg(s.totalPaper, s.total))
		}
	}

	// Hypothesis-level detail
	fmt.Println("\n  per hypothesis:")
	for _, c := range mlog.Cycles {
		signal := "no signal"
		if c.PapersFound > 0 {
			label := "papers"
			if c.Backend == "zim" {
				label = "articles"
			}
			signal = fmt.Sprintf("%d %s", c.PapersFound, label)
		}
		res := ""
		if c.Resolution != "" {
			res = " → " + c.Resolution
		}
		be := c.Backend
		if be == "" {
			be = "arxiv"
		}
		fmt.Printf("    %-40s [%s] %-5s%s  %s\n", c.Hypothesis, signal, be, res, c.Timestamp.Format("2006-01-02"))
	}
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d) * 100
}

func avg(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}
