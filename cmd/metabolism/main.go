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
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	zim "github.com/justinstimatze/gozim"
	winze "github.com/justinstimatze/winze"
)

// --- topology output types (subset) ---

type TopologyReport struct {
	Entities        int               `json:"entities"`
	Claims          int               `json:"claims"`
	Edges           int               `json:"edges"`
	Clusters        int               `json:"clusters"`
	Vulnerabilities []TopologyVuln    `json:"vulnerabilities"`
	SensorTargets   []SensorTarget    `json:"sensor_targets"`
}

type TopologyVuln struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Entity      string   `json:"entity"`
	Description string   `json:"description"`
	ClaimNames  []string `json:"claim_names,omitempty"`
}

type SensorTarget struct {
	Hypothesis string `json:"hypothesis"`
	Query      string `json:"query"`                // arXiv-optimized: author + keywords
	ZimQuery   string `json:"zim_query,omitempty"`   // encyclopedia-optimized: topic name
	RssQuery   string `json:"rss_query,omitempty"`   // feed-optimized: broader keywords
	Prediction string `json:"prediction"`
	VulnType   string `json:"vuln_type"`
	VulnCount  int    `json:"vuln_count"`
}

// queryFor returns the best query string for a given backend.
func (t SensorTarget) queryFor(backend string) string {
	switch backend {
	case "zim":
		if t.ZimQuery != "" {
			return t.ZimQuery
		}
	case "rss":
		if t.RssQuery != "" {
			return t.RssQuery
		}
	}
	return t.Query
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
	Ingested       bool   `json:"ingested,omitempty"`        // true after pipeline extracted claims from this cycle's articles
	LLMLintSkipped bool   `json:"llm_lint_skipped,omitempty"` // true when LLM contradiction check was unavailable during pipeline
	// PredictionType categorizes the prediction this cycle represents.
	// Empty (legacy) is treated as "structural_fragility" for calibrate
	// bucketing. Non-tautological types resolve by KB-internal signal
	// (e.g. "trip_lint_durability" resolves by running cmd/lint and
	// observing whether the promoted claim var was flagged).
	PredictionType string `json:"prediction_type,omitempty"`
	// Evidence holds a short, machine-readable reason when Resolution is
	// set by a non-sensor resolver (e.g. the lint line that refuted a
	// trip_lint_durability prediction). Sensor resolutions continue to
	// carry their evidence in Papers; this field is for resolvers that
	// produce short textual evidence instead of paper summaries.
	Evidence string `json:"evidence,omitempty"`
	// OracleCommit is the short git HEAD SHA at resolution time. Empty on
	// pre-versioning entries. Used by --durability to attribute drift
	// between re-runs: a changed OracleCommit identifies corpus churn.
	OracleCommit string `json:"oracle_commit,omitempty"`
	// OracleDigest is a short sha256 over the resolver's source files at
	// resolution time (e.g. cmd/lint/*.go for trip_lint_durability). A
	// change between re-runs identifies oracle-code drift, distinguishing
	// "we edited the rule" from "the corpus changed under a stable rule".
	OracleDigest string `json:"oracle_digest,omitempty"`
	// PipelineClaims records per-claim accept/reject decisions during ingest.
	// Added for pipeline observability.
	PipelineClaims []PipelineClaim `json:"pipeline_claims,omitempty"`
}

// PipelineClaim records a single claim-level decision during pipeline ingest.
type PipelineClaim struct {
	EntityName string `json:"entity_name"`
	Predicate  string `json:"predicate"`
	Target     string `json:"target"`
	Accepted   bool   `json:"accepted"`
	Reason     string `json:"reason,omitempty"` // "quote_mismatch", "subject_slot_mismatch", "object_slot_mismatch", "unknown_predicate", "duplicate", "accepted", "off_topic", "read_error", "llm_error", "no_claims"
}

type PaperSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Year    int    `json:"year"`
	Snippet string `json:"snippet,omitempty"` // abstract (arXiv) or article intro (ZIM), up to 500 chars
}

// sensorConfig is loaded from .metabolism.json (if it exists) to avoid
// repeating --zim and --rss-feeds flags on every invocation.
type sensorConfig struct {
	ZimPath  string   `json:"zim_path"`
	ZimIndex string   `json:"zim_index,omitempty"`
	RSSFeeds []string `json:"rss_feeds,omitempty"`
	Backend  string   `json:"backend,omitempty"` // default: "all" when zim_path set
}

func loadSensorConfig(dir string) sensorConfig {
	var cfg sensorConfig
	path := filepath.Join(dir, ".metabolism.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "metabolism: warning: malformed %s: %v (using defaults)\n", path, err)
		return sensorConfig{}
	}
	return cfg
}

// isFlagSet returns true if the named flag was explicitly set on the command line.
func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	// Cap memory to avoid OOM during evolve cycles (Dolt caches default to ~544 MB).
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(512 << 20) // 512 MiB
	}

	limit := flag.Int("limit", 5, "max results per query")
	dryRun := flag.Bool("dry-run", false, "show targets without querying")
	calibrate := flag.Bool("calibrate", false, "analyze existing log instead of running a cycle")
	durability := flag.Bool("durability", false, "re-run KB-internal resolvers against current corpus and report drift vs historical verdicts")
	durabilityWrite := flag.Bool("write", false, "with --durability: append recheck entries to .metabolism-log.json (default: read-only)")
	irrelevanceAudit := flag.Bool("irrelevance-audit", false, "diagnostic: reclassify a sample of 'irrelevant' cycles; report flip rate (needs ANTHROPIC_API_KEY)")
	irrelevanceN := flag.Int("audit-n", 10, "with --irrelevance-audit: number of cycles to sample")
	auditHaiku := flag.Bool("audit-haiku", true, "with --irrelevance-audit: use Haiku (cheap) instead of Sonnet (ignored under --audit-mode=production, which uses llmResolve's configured model)")
	auditRequireSnippet := flag.Bool("audit-require-snippet", false, "with --irrelevance-audit: only sample cycles whose papers carry at least one snippet (filters out title-only cases)")
	auditMode := flag.String("audit-mode", "neutral", "with --irrelevance-audit: 'neutral' (audit's own default-neutral prompt) or 'production' (llmResolve path — tests current production prompt)")
	resolve := flag.String("resolve", "", "resolve a hypothesis: HYPOTHESIS=corroborated|challenged|irrelevant|no_signal")
	suggest := flag.Bool("suggest", false, "generate corpus template from corroborated cycles")
	ingest := flag.Bool("ingest", false, "LLM-assisted ingest from corroborated ZIM cycles (needs --zim and ANTHROPIC_API_KEY)")
	reify := flag.Bool("reify", false, "generate predictions.go from metabolism log (first-class Predicts/ResolvedAs claims)")
	entityCap := flag.Int("entity-cap", winze.DefaultEntityCap, "max entities allowed in KB; refuse ingest/pipeline above this")
	pipeline := flag.Bool("pipeline", false, "full quality pipeline: ingest → build → vet → lint → llm-contradiction → commit/reject")
	llmBudget := flag.Int("llm-budget", 3, "max LLM calls for contradiction check in pipeline mode")
	narrative := flag.Bool("narrative", false, "with --calibrate: tell the prediction story for each hypothesis")
	dream := flag.Bool("dream", false, "consolidation cycle: analyze KB health via topology+lint+adit, report maintenance opportunities (no new ingest)")
	fix := flag.Bool("fix", false, "auto-fix dream findings (with --dream; requires ANTHROPIC_API_KEY for --tighten)")
	tighten := flag.Bool("tighten", false, "also tighten overlong Briefs via LLM (with --dream --fix)")
	bias := flag.Bool("bias", false, "run cognitive bias self-audit on KB structure (standalone or with --dream)")
	cycle := flag.Bool("cycle", false, "full sleep cycle: sense → evaluate → ingest → dream → trip → calibrate (alias: --evolve)")
	evolve := flag.Bool("evolve", false, "full evolution cycle — same as --cycle")
	trip := flag.Bool("trip", false, "speculative cross-cluster connection generation (needs ANTHROPIC_API_KEY)")
	tripTemp := flag.Float64("temperature", 1.0, "LLM temperature for --trip (0.0-1.5; higher = wilder connections)")
	tripPrompt := flag.String("prompt-type", "analogy", "connection type for --trip: analogy, contradiction, genealogy, prediction")
	tripPairs := flag.Int("pairs", 5, "number of cross-cluster entity pairs to evaluate in --trip")
	tripPromote := flag.Bool("promote", false, "with --trip: also promote score-4+ connections to the corpus (off by default; --evolve always promotes)")
	tripMinScore := flag.Int("min-score", 4, "with --trip --promote: minimum score to promote (default 4; lower for resolver stress-tests)")
	pkm := flag.String("pkm", "", "path to PKM vault directory (markdown notes → typed Go corpus files)")
	jsonOut := flag.Bool("json", false, "output as JSON")
	backend := flag.String("backend", "arxiv", "sensor backend: arxiv, zim, rss, or all")
	zimPath := flag.String("zim", "", "path to .zim file (required for zim backend)")
	zimIndex := flag.String("zim-index", "", "path for Bleve index (default: <zimfile>.bleve/)")
	rssFeeds := flag.String("rss-feeds", "", "comma-separated RSS/Atom feed URLs (overrides defaults)")
	flag.Parse()

	// WINZE_ENTITY_CAP env var overrides flag default (but explicit --entity-cap wins)
	if envCap := os.Getenv("WINZE_ENTITY_CAP"); envCap != "" && !isFlagSet("entity-cap") {
		if n, err := strconv.Atoi(envCap); err == nil && n > 0 {
			*entityCap = n
		}
	}

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	// Load sensor config from .metabolism.json; CLI flags override
	cfg := loadSensorConfig(dir)
	if *zimPath == "" && cfg.ZimPath != "" {
		*zimPath = cfg.ZimPath
	}
	if *zimIndex == "" && cfg.ZimIndex != "" {
		*zimIndex = cfg.ZimIndex
	}
	if *rssFeeds == "" && len(cfg.RSSFeeds) > 0 {
		*rssFeeds = strings.Join(cfg.RSSFeeds, ",")
	}
	if *backend == "arxiv" && cfg.Backend != "" {
		// Only override if user didn't explicitly set --backend
		*backend = cfg.Backend
	}

	validBackends := map[string]bool{"arxiv": true, "zim": true, "rss": true, "all": true}
	if !validBackends[*backend] {
		fmt.Fprintf(os.Stderr, "metabolism: --backend must be arxiv, zim, rss, or all (got %q)\n", *backend)
		os.Exit(1)
	}
	useArxiv := *backend == "arxiv" || *backend == "all"
	useZim := *backend == "zim" || *backend == "all"
	useRSS := *backend == "rss" || *backend == "all"
	if useZim && *zimPath == "" {
		fmt.Fprintf(os.Stderr, "metabolism: --zim path required when backend includes zim\n")
		os.Exit(1)
	}

	defer func() {
		if zimArchive != nil {
			zimArchive.Close()
		}
	}()

	if *cycle || *evolve {
		runCycle(dir, *zimPath, *zimIndex, *llmBudget, *entityCap, *dryRun, *jsonOut)
		return
	}

	if *resolve != "" {
		runResolve(dir, *resolve)
		return
	}

	if *reify {
		runReify(dir)
		return
	}

	if *suggest {
		runSuggest(dir)
		return
	}

	// Entity cap: refuse ingest/pipeline when KB exceeds threshold
	if (*pipeline || *ingest) && *entityCap > 0 {
		count, err := countEntities(dir)
		if err == nil && count >= *entityCap {
			fmt.Fprintf(os.Stderr, "metabolism: entity cap reached (%d/%d) — refusing ingest\n", count, *entityCap)
			fmt.Fprintf(os.Stderr, "metabolism: deepen existing neighborhoods or raise --entity-cap\n")
			os.Exit(2)
		}
	}

	if *pkm != "" {
		runPKMIngest(dir, *pkm, entityCap, *dryRun, *jsonOut)
		return
	}

	if *pipeline {
		if *zimPath == "" {
			fmt.Fprintf(os.Stderr, "metabolism: --pipeline requires --zim\n")
			os.Exit(1)
		}
		pipelineStandalone = true
		runPipeline(dir, *zimPath, *zimIndex, *llmBudget)
		return
	}

	if *ingest {
		if *zimPath == "" {
			fmt.Fprintf(os.Stderr, "metabolism: --ingest requires --zim\n")
			os.Exit(1)
		}
		runIngest(dir, *zimPath, *zimIndex)
		return
	}

	if *trip {
		report := runTrip(dir, *tripTemp, *tripPrompt, *tripPairs, *dryRun, *jsonOut)
		if *tripPromote && !*dryRun && len(report.Connections) > 0 {
			if err := promoteConnections(dir, report.Connections, *tripMinScore); err != nil {
				fmt.Fprintf(os.Stderr, "metabolism: trip promotion: %v\n", err)
				os.Exit(1)
			}
		}
		return
	}

	if *bias && !*dream {
		runDreamBias(dir, *jsonOut)
		return
	}

	if *dream {
		if *fix {
			runDreamFix(dir, *tighten, *dryRun, *jsonOut)
		} else {
			runDream(dir, *bias, *jsonOut)
		}
		return
	}

	if *calibrate {
		if *narrative {
			runCalibrateNarrative(dir)
		} else {
			runCalibrate(dir, *jsonOut)
		}
		return
	}

	if *durability {
		runDurability(dir, *jsonOut, *durabilityWrite)
		return
	}

	if *irrelevanceAudit {
		loadDotEnv(dir)
		runIrrelevanceAudit(dir, *irrelevanceN, *jsonOut, *auditHaiku, *auditRequireSnippet, *auditMode)
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
			arxivQ := t.queryFor("arxiv")
			papers, err := searchArxiv(arxivQ, *limit)
			if err != nil {
				if strings.Contains(err.Error(), "429") {
					fmt.Fprintf(os.Stderr, "metabolism: rate limited, waiting 30s...\n")
					time.Sleep(30 * time.Second)
					papers, err = searchArxiv(arxivQ, *limit)
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "metabolism: arxiv %q: %v\n", arxivQ, err)
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
					Query:       arxivQ,
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
			zimQ := t.queryFor("zim")
			articles, err := searchZim(*zimPath, *zimIndex, zimQ, *limit)
			if err != nil {
				fmt.Fprintf(os.Stderr, "metabolism: zim %q: %v\n", zimQ, err)
			} else {
				cycles = append(cycles, Cycle{
					Timestamp:   time.Now(),
					Hypothesis:  t.Hypothesis,
					Prediction:  t.Prediction,
					Query:       zimQ,
					Backend:     "zim",
					VulnType:    t.VulnType,
					VulnCount:   t.VulnCount,
					PapersFound: len(articles),
					Papers:      articles,
				})
			}
		}

		// RSS backend — poll each feed, filter by query terms
		if useRSS {
			rssQ := t.queryFor("rss")
			feeds := defaultFeeds
			if *rssFeeds != "" {
				feeds = strings.Split(*rssFeeds, ",")
				for i := range feeds {
					feeds[i] = strings.TrimSpace(feeds[i])
				}
			}
			var allRSS []PaperSummary
			for _, feedURL := range feeds {
				results, err := searchRSS(feedURL, rssQ, *limit)
				if err != nil {
					fmt.Fprintf(os.Stderr, "metabolism: rss %q from %s: %v\n", rssQ, feedURL, err)
					continue
				}
				allRSS = append(allRSS, results...)
			}
			// Deduplicate by title (feeds may overlap)
			seen := make(map[string]bool)
			var deduped []PaperSummary
			for _, p := range allRSS {
				key := p.ID
				if key == "" {
					key = strings.ToLower(p.Title)
				}
				if seen[key] {
					continue
				}
				seen[key] = true
				deduped = append(deduped, p)
				if len(deduped) >= *limit {
					break
				}
			}
			cycles = append(cycles, Cycle{
				Timestamp:   time.Now(),
				Hypothesis:  t.Hypothesis,
				Prediction:  t.Prediction,
				Query:       rssQ,
				Backend:     "rss",
				VulnType:    t.VulnType,
				VulnCount:   t.VulnCount,
				PapersFound: len(deduped),
				Papers:      deduped,
			})
		}
	}

	// Auto-resolve zero-paper cycles as no_signal
	today := time.Now().Format("2006-01-02")
	for i := range cycles {
		if cycles[i].PapersFound == 0 && cycles[i].Resolution == "" {
			cycles[i].Resolution = "no_signal"
			cycles[i].ResolvedAt = today
		}
	}

	// 3. Append to log
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	// Backfill: resolve any existing unresolved zero-paper entries
	for i := range mlog.Cycles {
		if mlog.Cycles[i].PapersFound == 0 && mlog.Cycles[i].Resolution == "" {
			mlog.Cycles[i].Resolution = "no_signal"
			mlog.Cycles[i].ResolvedAt = today
		}
	}

	mlog.Cycles = append(mlog.Cycles, cycles...)
	if err := saveLog(logPath, mlog); err != nil {
		fmt.Fprintf(os.Stderr, "metabolism: save log: %v\n", err)
	}

	// 4. Output
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(cycles); err != nil {
			fmt.Fprintf(os.Stderr, "metabolism: JSON encode: %v\n", err)
		}
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

// topologyCache caches the result of runTopology to avoid redundant
// subprocess calls within a single cycle. The topology subprocess is
// expensive (~2-3s per call) and the KB state doesn't change between
// cache uses within a single evolve cycle.
var topologyCache struct {
	dir     string
	targets []SensorTarget
	report  TopologyReport
	valid   bool
}

func invalidateTopologyCache() {
	topologyCache.valid = false
}

// countEntities returns the current entity count, using the topology cache
// if available.
func countEntities(dir string) (int, error) {
	if topologyCache.valid && topologyCache.dir == dir {
		return topologyCache.report.Entities, nil
	}
	_, report, err := runTopology(dir)
	if err != nil {
		return 0, err
	}
	return report.Entities, nil
}

// runTopology shells out to cmd/topology and parses the JSON output.
// Results are cached; call invalidateTopologyCache() after KB mutations.
func runTopology(dir string) ([]SensorTarget, TopologyReport, error) {
	if topologyCache.valid && topologyCache.dir == dir {
		return topologyCache.targets, topologyCache.report, nil
	}

	// Build topology binary once to avoid go-run compilation overhead (~3 GB).
	binPath := filepath.Join(os.TempDir(), "winze-topology")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/topology")
	buildCmd.Dir = dir
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return nil, TopologyReport{}, fmt.Errorf("build topology: %w", err)
	}

	cmd := exec.Command(binPath, "--json", dir)
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

	topologyCache.dir = dir
	topologyCache.targets = report.SensorTargets
	topologyCache.report = report
	topologyCache.valid = true

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

	resp, err := httpClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("arXiv API returned %d: %s", resp.StatusCode, string(body))
	}

	var feed arxivFeed
	if err := xml.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&feed); err != nil {
		return nil, err
	}

	var papers []PaperSummary
	for _, e := range feed.Entries {
		year := 0
		if t, err := time.Parse(time.RFC3339, e.Published); err == nil {
			year = t.Year()
		}
		snippet := strings.Join(strings.Fields(e.Summary), " ")
		if len(snippet) > 500 {
			snippet = snippet[:500]
		}
		papers = append(papers, PaperSummary{
			ID:      e.ID,
			Title:   strings.Join(strings.Fields(e.Title), " "),
			Year:    year,
			Snippet: snippet,
		})
	}
	return papers, nil
}

// zimArchive is lazily opened on first use and reused across queries.
// httpClient is used for all external HTTP requests (arXiv, RSS).
var httpClient = &http.Client{Timeout: 15 * time.Second}

var zimArchive *zim.Archive

var htmlStyleRe = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
var htmlScriptRe = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
var htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
// [^>] already matches newlines in Go regexp (unlike PCRE), so multi-line
// tags like <div\nclass="x"> are handled without the (?s) flag.
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)
var wsCollapseRe = regexp.MustCompile(`\s+`)

func stripHTML(html []byte) string {
	// Remove style/script blocks and comments before stripping tags
	text := htmlStyleRe.ReplaceAll(html, nil)
	text = htmlScriptRe.ReplaceAll(text, nil)
	text = htmlCommentRe.ReplaceAll(text, nil)
	text = htmlTagRe.ReplaceAll(text, []byte(" "))
	return strings.TrimSpace(wsCollapseRe.ReplaceAllString(string(text), " "))
}

// searchZim uses gozim for fulltext search against a ZIM file.
// On first call, opens the archive and builds/opens a Bleve index.
// Returns PaperSummary (Title is article title, ID is the ZIM path).
func searchZim(zimPath, indexPath, query string, limit int) ([]PaperSummary, error) {
	if zimArchive == nil {
		a, err := zim.Open(zimPath, zim.WithMmap())
		if err != nil {
			return nil, fmt.Errorf("open zim: %w", err)
		}
		zimArchive = a
	}

	var opts []zim.SearchOption
	if indexPath != "" {
		opts = append(opts, zim.WithIndexPath(indexPath))
	}
	results, err := zimArchive.Search(query, limit, opts...)
	if err != nil {
		return nil, fmt.Errorf("search zim: %w", err)
	}

	var papers []PaperSummary
	for _, r := range results {
		snippet := ""
		text, err := readZimSnippet(r.Entry)
		if err == nil {
			snippet = text
		}
		papers = append(papers, PaperSummary{
			ID:      "zim:" + r.Entry.Path(),
			Title:   r.Entry.Title(),
			Year:    0, // encyclopedia, not dated
			Snippet: snippet,
		})
	}
	return papers, nil
}

// readZimSnippet extracts the first ~500 chars of plaintext from a ZIM entry.
// Used by the sensor to give the auto-resolver actual content, not just titles.
func readZimSnippet(entry zim.Entry) (string, error) {
	// Follow redirects
	var err error
	for i := 0; i < 5 && entry.IsRedirect(); i++ {
		entry, err = entry.RedirectTarget()
		if err != nil {
			return "", err
		}
	}
	content, err := entry.Content()
	if err != nil {
		return "", err
	}
	text := stripHTML(content)
	if len(text) > 500 {
		text = text[:500]
	}
	return text, nil
}

// --- log persistence ---

func loadLog(path string) MetabolismLog {
	var mlog MetabolismLog
	data, err := os.ReadFile(path)
	if err != nil {
		return mlog
	}
	if err := json.Unmarshal(data, &mlog); err != nil {
		fmt.Fprintf(os.Stderr, "metabolism: warning: corrupt log %s: %v (starting fresh)\n", path, err)
		return MetabolismLog{}
	}
	return mlog
}

func saveLog(path string, mlog MetabolismLog) error {
	data, err := json.MarshalIndent(mlog, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// --- resolution ---

func runResolve(dir, spec string) {
	parts := strings.SplitN(spec, "=", 2)
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "metabolism: --resolve expects HYPOTHESIS=outcome (corroborated|challenged|irrelevant)\n")
		os.Exit(1)
	}
	hypothesis, outcome := parts[0], parts[1]

	valid := map[string]bool{"corroborated": true, "challenged": true, "irrelevant": true, "no_signal": true}
	if !valid[outcome] {
		fmt.Fprintf(os.Stderr, "metabolism: outcome must be corroborated, challenged, irrelevant, or no_signal (got %q)\n", outcome)
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

// --- auto-resolution ---

// autoResolve evaluates pending hypotheses with sufficient sensor signal and
// resolves them autonomously via LLM judgment. A hypothesis is a candidate
// when it has 3+ cycles with signal but no resolution.
//
// The LLM receives the hypothesis Brief (from the KB) and the paper titles
// found by the sensor, then classifies: corroborated, challenged, or irrelevant.
type resolveResult struct {
	Hypothesis string
	Outcome    string // corroborated, challenged, irrelevant
	Papers     []PaperSummary
}

func autoResolve(dir string) []resolveResult {
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	// Find candidates: hypotheses with 3+ signal cycles and at least one unresolved
	type candidate struct {
		hypothesis string
		brief      string
		papers     []PaperSummary // papers from signal cycles (with snippets)
		indices    []int          // unresolved cycle indices in mlog.Cycles
	}

	briefs := collectKBBriefs(dir)

	// Group by hypothesis
	type hypInfo struct {
		signalCycles    int
		resolved        int
		contentResolved int // resolutions made when snippets were available
		papers          []PaperSummary
		hasSnippets     bool
		unresolved      []int
	}
	byHyp := map[string]*hypInfo{}
	for i, c := range mlog.Cycles {
		h, ok := byHyp[c.Hypothesis]
		if !ok {
			h = &hypInfo{}
			byHyp[c.Hypothesis] = h
		}
		if c.PapersFound > 0 {
			h.signalCycles++
			h.papers = append(h.papers, c.Papers...)
			for _, p := range c.Papers {
				if p.Snippet != "" {
					h.hasSnippets = true
				}
			}
		}
		if c.Resolution != "" {
			h.resolved++
			// Check if this resolution had snippet data
			for _, p := range c.Papers {
				if p.Snippet != "" {
					h.contentResolved++
					break
				}
			}
		} else {
			h.unresolved = append(h.unresolved, i)
		}
	}

	// Count existing resolutions per hypothesis for majority check
	resolutionCounts := map[string]map[string]int{} // hyp → resolution → count
	for _, c := range mlog.Cycles {
		if c.Resolution != "" {
			if resolutionCounts[c.Hypothesis] == nil {
				resolutionCounts[c.Hypothesis] = map[string]int{}
			}
			resolutionCounts[c.Hypothesis][c.Resolution]++
		}
	}

	var candidates []candidate
	for hyp, info := range byHyp {
		if info.signalCycles < 3 || len(info.unresolved) == 0 {
			continue
		}

		// Skip if any existing resolution is "corroborated" or "challenged"
		// — these are high-value verdicts (evidence found). New cycles with
		// weaker signal shouldn't override them. Only re-evaluate hypotheses
		// where all existing resolutions are "irrelevant" or "no_signal".
		if counts, ok := resolutionCounts[hyp]; ok {
			if counts["corroborated"] > 0 || counts["challenged"] > 0 {
				continue
			}
		}

		{
			// Deduplicate papers by ID (or title fallback)
			seen := map[string]bool{}
			var unique []PaperSummary
			for _, p := range info.papers {
				key := p.ID
				if key == "" {
					key = p.Title
				}
				if !seen[key] {
					seen[key] = true
					unique = append(unique, p)
				}
			}
			candidates = append(candidates, candidate{
				hypothesis: hyp,
				brief:      briefs[hyp],
				papers:     unique,
				indices:    info.unresolved,
			})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "[auto-resolve] no ANTHROPIC_API_KEY — skipping\n")
		return nil
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	var results []resolveResult
	for _, cand := range candidates {
		outcome, err := llmResolve(client, cand.hypothesis, cand.brief, cand.papers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[auto-resolve] %s: LLM error: %v\n", cand.hypothesis, err)
			continue
		}

		// Apply resolution to all unresolved cycles for this hypothesis
		today := time.Now().Format("2006-01-02")
		for _, idx := range cand.indices {
			mlog.Cycles[idx].Resolution = outcome
			mlog.Cycles[idx].ResolvedAt = today
		}
		fmt.Printf("  %s → %s (%d cycles resolved)\n", cand.hypothesis, outcome, len(cand.indices))
		results = append(results, resolveResult{
			Hypothesis: cand.hypothesis,
			Outcome:    outcome,
			Papers:     cand.papers,
		})
	}

	if len(results) > 0 {
		if err := saveLog(logPath, mlog); err != nil {
			fmt.Fprintf(os.Stderr, "[auto-resolve] save log: %v\n", err)
		}
	}
	return results
}

func llmResolve(client anthropic.Client, hypothesis, brief string, papers []PaperSummary) (string, error) {
	// Build source descriptions with snippets when available
	sanitize := func(s string, maxLen int) string {
		cleaned := strings.Map(func(r rune) rune {
			if r < 32 || r == 127 {
				return ' '
			}
			return r
		}, s)
		if len(cleaned) > maxLen {
			cleaned = cleaned[:maxLen]
		}
		return cleaned
	}

	var sourceDescriptions []string
	for _, p := range papers {
		desc := sanitize(p.Title, 200)
		if p.Snippet != "" {
			desc += "\n  Content: " + sanitize(p.Snippet, 500)
		}
		sourceDescriptions = append(sourceDescriptions, desc)
	}

	// The prompt evaluates whether the sources provide substantive evidence
	// about the hypothesis. Keyword overlap alone is not evidence — the
	// sources must contain specific facts, data, or arguments that bear on
	// the hypothesis's truth.
	//
	// Recalibrated 2026-04-18 after the --irrelevance-audit diagnostic
	// found the prior version over-strict: of 10 snippet-bearing "irrelevant"
	// cycles sampled, 3 cleanly flipped to "corroborated" under a neutral
	// prompt (30% flip rate). The prior had four separate reinforcements of
	// "default to irrelevant" plus asymmetric criteria (1 condition for
	// corroboration, 4 for challenge). The current version keeps the core
	// "keywords aren't evidence" guard but drops the DEFAULT framing and
	// symmetrizes corroboration vs challenge. Expected side effect: the
	// survivorship-bias auditor should see the irrelevant:challenged ratio
	// fall from 197:1 toward a range where real challenges surface.
	prompt := fmt.Sprintf(`You are a careful evaluator. Classify whether the sources below provide substantive evidence about the hypothesis. Weigh evidence for and against with equal rigor.

Hypothesis: %s
Brief: %s

Sources found by automated sensor queries (titles and content excerpts):
- %s

Labels:
- "corroborated" — sources contain specific evidence, data, or arguments that support the hypothesis's central claim. Evidence must go beyond merely discussing the same topic — look for specific facts like dates, numbers, attributions, or experimental results that make the claim more likely true.
- "challenged" — sources contain specific evidence, data, or arguments that contradict the hypothesis, present a competing account, or undermine its evidentiary basis. Look for the same kinds of specifics — different dates, different attributions, contradictory results, alternative explanations — that make the claim less likely true.
- "irrelevant" — sources discuss the same topic area but do not provide evidence bearing on whether this hypothesis is true or false. Keyword overlap alone is not evidence.

Think step by step:
1. What specific claim does the hypothesis make?
2. What concrete facts in the sources bear on that claim — for or against?
3. If the sources provide substantive support, classify as corroborated. If they provide substantive contradiction, classify as challenged. If the sources only share keywords without touching the specific claim, classify as irrelevant.

State your final classification: irrelevant, corroborated, or challenged.`,
		hypothesis, brief, strings.Join(sourceDescriptions, "\n- "))

	resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_5,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", err
	}

	raw := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			raw = strings.TrimSpace(strings.ToLower(block.Text))
		}
	}

	return extractClassification(raw)
}

// extractClassification parses an LLM response for a resolution classification.
// The production prompt ends with "state your final classification: X" so the
// label is expected to appear after that marker. Multi-label reasoning (e.g.
// "source A is irrelevant but source B corroborates → final: corroborated")
// used to be rejected as ambiguous by the whole-response scan; the marker-
// aware path preserves the LLM's final verdict.
//
// Ordering:
//  1. If "final classification" marker exists, pick the first label after the
//     LAST occurrence (LLMs sometimes echo the marker once mid-reasoning).
//  2. Otherwise fall back to the original strict whole-response scan — a
//     single label anywhere is accepted; more than one is rejected.
//
// Uses containsWordBoundary in the fallback path to avoid false positives
// from negations like "not challenged".
func extractClassification(raw string) (string, error) {
	valid := []string{"irrelevant", "corroborated", "challenged"}
	lower := strings.ToLower(raw)

	if idx := strings.LastIndex(lower, "final classification"); idx >= 0 {
		tail := lower[idx:]
		pos := len(tail)
		chosen := ""
		for _, v := range valid {
			if i := strings.Index(tail, v); i >= 0 && i < pos {
				pos = i
				chosen = v
			}
		}
		if chosen != "" {
			return chosen, nil
		}
	}

	var matches []string
	for _, v := range valid {
		if containsWordBoundary(raw, v) {
			matches = append(matches, v)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("unexpected LLM response: %q", raw)
	default:
		return "", fmt.Errorf("ambiguous LLM response (multiple classifications): %q", raw)
	}
}

// containsWordBoundary checks if text contains word as a standalone word
// (not preceded by "not " or "no "). This avoids false positives where
// "not challenged" would match "challenged".
func containsWordBoundary(text, word string) bool {
	idx := 0
	for {
		pos := strings.Index(text[idx:], word)
		if pos < 0 {
			return false
		}
		absPos := idx + pos
		// Check for negation prefix
		negated := false
		prefix := text[:absPos]
		trimmed := strings.TrimRight(prefix, " ")
		if strings.HasSuffix(trimmed, "not") || strings.HasSuffix(trimmed, "no") {
			negated = true
		}
		if !negated {
			return true
		}
		idx = absPos + len(word)
		if idx >= len(text) {
			return false
		}
	}
}

// --- calibration ---

// hypothesisScore aggregates all cycles for a single hypothesis into a verdict.
type hypothesisScore struct {
	Name           string  `json:"name"`
	VulnType       string  `json:"vuln_type"`
	PredictionType string  `json:"prediction_type,omitempty"`
	TotalCycles    int     `json:"total_cycles"`
	WithSignal     int     `json:"with_signal"`
	Corroborated   int     `json:"corroborated"`
	Challenged     int     `json:"challenged"`
	Irrelevant     int     `json:"irrelevant"`
	NoSignal       int     `json:"no_signal"`
	Confirmed      int     `json:"confirmed"` // KB-internal resolvers (e.g. trip_lint_durability)
	Refuted        int     `json:"refuted"`   // KB-internal resolvers (e.g. trip_lint_durability)
	Pending        int     `json:"pending"`
	Verdict        string  `json:"verdict"`          // corroborated|challenged|confirmed|irrelevant|no_signal|refuted|pending
	CyclesToVerdict int    `json:"cycles_to_verdict"` // cycles until first useful resolution; 0 if pending
	Precision      float64 `json:"precision"`         // useful cycles / signal cycles (0 if no signal)
}

// isHit maps a verdict to hit/miss/pending for aggregate hit-rate calculation.
// corroborated/challenged/confirmed are hits; irrelevant/no_signal/refuted are
// misses. This keeps sensor-based and KB-internal resolution axes on the
// same scoreboard.
func isHit(verdict string) bool {
	return verdict == "corroborated" || verdict == "challenged" || verdict == "confirmed"
}
func isMiss(verdict string) bool {
	return verdict == "irrelevant" || verdict == "no_signal" || verdict == "refuted"
}

// sensorVerdict reproduces the original sensor-resolution majority-vote logic
// for backward compatibility. Factored out so the extended scoreHypotheses
// can dispatch between sensor-based and KB-internal verdict rules.
func sensorVerdict(s hypothesisScore) string {
	resolved := s.Corroborated + s.Challenged + s.Irrelevant
	switch {
	case resolved == 0 && s.Pending > 0:
		return "pending"
	case resolved == 0:
		return "no_signal"
	case s.Challenged > s.Corroborated && s.Challenged >= s.Irrelevant:
		return "challenged"
	case s.Corroborated > s.Irrelevant:
		return "corroborated"
	case s.Irrelevant > 0:
		return "irrelevant"
	case s.Challenged > 0:
		return "challenged"
	default:
		return "pending"
	}
}

func scoreHypotheses(cycles []Cycle) []hypothesisScore {
	// Group cycles by (hypothesis, prediction_type), preserving order of
	// first appearance. Key-by-composite so that a single claim var with
	// trip_promotion_attempt + trip_lint_durability + trip_llm_durability
	// rows produces three separate scores rather than collapsing into
	// whichever row came first. Display name drops the prediction_type
	// suffix for the default (structural_fragility) to preserve legacy
	// output.
	type entry struct {
		cycles         []Cycle
		vulnType       string
		predictionType string
		displayName    string
	}
	byKey := map[string]*entry{}
	var order []string
	for _, c := range cycles {
		pt := c.PredictionType
		if pt == "" {
			pt = "structural_fragility"
		}
		key := c.Hypothesis + "|" + pt
		e, ok := byKey[key]
		if !ok {
			display := c.Hypothesis
			if pt != "structural_fragility" {
				display = fmt.Sprintf("%s [%s]", c.Hypothesis, pt)
			}
			e = &entry{vulnType: c.VulnType, predictionType: pt, displayName: display}
			byKey[key] = e
			order = append(order, key)
		}
		e.cycles = append(e.cycles, c)
	}

	var scores []hypothesisScore
	for _, key := range order {
		e := byKey[key]
		s := hypothesisScore{Name: e.displayName, VulnType: e.vulnType, PredictionType: e.predictionType}
		for _, c := range e.cycles {
			s.TotalCycles++
			if c.PapersFound > 0 {
				s.WithSignal++
			}
			switch c.Resolution {
			case "corroborated":
				s.Corroborated++
			case "challenged":
				s.Challenged++
			case "irrelevant":
				s.Irrelevant++
			case "no_signal":
				s.NoSignal++
			case "confirmed":
				s.Confirmed++
			case "refuted":
				s.Refuted++
			default:
				s.Pending++
			}
		}

		// Verdict selection branches on resolution vocabulary. Sensor-based
		// rows (corroborated/challenged/irrelevant/no_signal) use the
		// existing majority logic; KB-internal rows (confirmed/refuted)
		// use a simpler majority between the two.
		kbInternal := s.Confirmed + s.Refuted
		sensor := s.Corroborated + s.Challenged + s.Irrelevant + s.NoSignal

		switch {
		case kbInternal > 0 && sensor == 0:
			// Pure KB-internal prediction (e.g. trip_lint_durability).
			if s.Refuted > s.Confirmed {
				s.Verdict = "refuted"
			} else {
				s.Verdict = "confirmed"
			}
		case kbInternal > 0 && sensor > 0:
			// Mixed resolution paths on the same hypothesis name (rare).
			// Prefer whichever side has more signal; on a tie, KB-internal
			// wins because it's deterministic.
			if sensor > kbInternal {
				s.Verdict = sensorVerdict(s)
			} else {
				if s.Refuted > s.Confirmed {
					s.Verdict = "refuted"
				} else {
					s.Verdict = "confirmed"
				}
			}
		default:
			s.Verdict = sensorVerdict(s)
		}

		// Cycles to verdict: count cycles until first hit-shaped resolution.
		if isHit(s.Verdict) {
			for i, c := range e.cycles {
				if c.Resolution == "corroborated" || c.Resolution == "challenged" || c.Resolution == "confirmed" {
					s.CyclesToVerdict = i + 1
					break
				}
			}
		}

		// Precision: signal cycles with useful resolution / total signal cycles.
		// Only counts cycles where papers were actually found AND resolution was useful.
		if s.WithSignal > 0 {
			usefulSignal := 0
			for _, c := range e.cycles {
				if c.PapersFound > 0 && (c.Resolution == "corroborated" || c.Resolution == "challenged") {
					usefulSignal++
				}
			}
			s.Precision = float64(usefulSignal) / float64(s.WithSignal) * 100
		}

		scores = append(scores, s)
	}
	return scores
}

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

	// Score hypotheses
	scores := scoreHypotheses(mlog.Cycles)

	// Aggregate prediction accuracy across hypotheses.
	var resolved, hits, misses, pending int
	var totalCyclesToVerdict int
	var hitsWithCycles int
	for _, s := range scores {
		switch {
		case isHit(s.Verdict):
			resolved++
			hits++
			totalCyclesToVerdict += s.CyclesToVerdict
			hitsWithCycles++
		case isMiss(s.Verdict):
			resolved++
			misses++
		case s.Verdict == "pending":
			pending++
		}
	}

	// By vuln type accuracy (per hypothesis, not per cycle).
	type vulnAccuracy struct {
		hypotheses int
		hits       int
		misses     int
		pending    int
	}
	byVulnAcc := map[string]*vulnAccuracy{}
	for _, s := range scores {
		vt := s.VulnType
		if byVulnAcc[vt] == nil {
			byVulnAcc[vt] = &vulnAccuracy{}
		}
		a := byVulnAcc[vt]
		a.hypotheses++
		switch {
		case isHit(s.Verdict):
			a.hits++
		case isMiss(s.Verdict):
			a.misses++
		default:
			a.pending++
		}
	}

	// By prediction_type accuracy — the load-bearing bucketing. Keeps
	// the tautological structural_fragility rows in their own bucket
	// separate from KB-internal resolvers like trip_lint_durability,
	// so their hit rates are not averaged together.
	byPredTypeAcc := map[string]*vulnAccuracy{}
	for _, s := range scores {
		pt := s.PredictionType
		if pt == "" {
			pt = "structural_fragility"
		}
		if byPredTypeAcc[pt] == nil {
			byPredTypeAcc[pt] = &vulnAccuracy{}
		}
		a := byPredTypeAcc[pt]
		a.hypotheses++
		switch {
		case isHit(s.Verdict):
			a.hits++
		case isMiss(s.Verdict):
			a.misses++
		default:
			a.pending++
		}
	}

	// Resolution stats (per cycle)
	resolutions := map[string]int{}
	unresolved := 0
	for _, c := range mlog.Cycles {
		if c.Resolution != "" {
			resolutions[c.Resolution]++
		} else {
			unresolved++
		}
	}

	// Post-hoc tautology scan: distinguish corroborations from novel
	// sources vs corroborations from sources already in the corpus
	// (gap_confirmed vs no_gap). Recomputed at calibrate time because
	// novelty is a moving target as the corpus grows.
	provIdx, provErr := collectCorpusProvenance(dir)
	if provErr != nil {
		fmt.Fprintf(os.Stderr, "calibrate: provenance scan: %v\n", provErr)
	}
	gapCounts := map[string]int{}
	// corrobNovel counts corroborations whose cycles have at least one
	// source not in the corpus (gap_confirmed + mixed_overlap). corrobTaut
	// counts corroborations where every source was already ingested
	// (no_gap). Only the latter is purely tautological signal.
	corrobNovel, corrobTaut := 0, 0
	challNovel, challTaut := 0, 0
	for _, c := range mlog.Cycles {
		status := classifyGapStatus(c, provIdx)
		if status == "" {
			continue
		}
		gapCounts[status]++
		hasNovel := status == "gap_confirmed" || status == "mixed_overlap"
		switch c.Resolution {
		case "corroborated":
			if hasNovel {
				corrobNovel++
			} else if status == "no_gap" {
				corrobTaut++
			}
		case "challenged":
			if hasNovel {
				challNovel++
			} else if status == "no_gap" {
				challTaut++
			}
		}
	}

	if jsonOut {
		type VulnTypeScore struct {
			VulnType   string  `json:"vuln_type"`
			Hypotheses int     `json:"hypotheses"`
			Hits       int     `json:"hits"`
			Misses     int     `json:"misses"`
			Pending    int     `json:"pending"`
			HitRate    float64 `json:"hit_rate"`
		}
		type CalReport struct {
			TotalCycles int     `json:"total_cycles"`
			SignalRate  float64 `json:"signal_rate"`
			WithSignal  int     `json:"with_signal"`
			TotalPapers int     `json:"total_papers"`
			Earliest    string  `json:"earliest"`
			Latest      string  `json:"latest"`
			// Prediction accuracy (per hypothesis)
			Hypotheses    int              `json:"hypotheses"`
			HitRate       float64          `json:"hit_rate"`
			Hits          int              `json:"hits"`
			Misses        int              `json:"misses"`
			Pending       int              `json:"pending"`
			AvgCyclesToHit float64         `json:"avg_cycles_to_hit"`
			// Tautology scan (gap_confirmed, mixed_overlap, no_gap)
			GapConfirmed      int `json:"gap_confirmed"`
			MixedOverlap      int `json:"mixed_overlap"`
			NoGap             int `json:"no_gap"`
			CorroboratedNovel int `json:"corroborated_novel"`
			CorroboratedTaut  int `json:"corroborated_tautological"`
			ChallengedNovel   int `json:"challenged_novel"`
			ChallengedTaut    int `json:"challenged_tautological"`
			ByVulnType    []VulnTypeScore  `json:"by_vuln_type"`
			Scores        []hypothesisScore `json:"scores"`
		}
		r := CalReport{
			TotalCycles: overall.total,
			SignalRate:  pct(overall.withSignal, overall.total),
			WithSignal:  overall.withSignal,
			TotalPapers: overall.totalPaper,
			Earliest:    earliest.Format("2006-01-02"),
			Latest:      latest.Format("2006-01-02"),
			Hypotheses:  len(scores),
			HitRate:     pct(hits, hits+misses),
			Hits:        hits,
			Misses:      misses,
			Pending:     pending,
			AvgCyclesToHit: avg(totalCyclesToVerdict, hitsWithCycles),
			GapConfirmed:      gapCounts["gap_confirmed"],
			MixedOverlap:      gapCounts["mixed_overlap"],
			NoGap:             gapCounts["no_gap"],
			CorroboratedNovel: corrobNovel,
			CorroboratedTaut:  corrobTaut,
			ChallengedNovel:   challNovel,
			ChallengedTaut:    challTaut,
			Scores:      scores,
		}
		for vt, a := range byVulnAcc {
			r.ByVulnType = append(r.ByVulnType, VulnTypeScore{
				VulnType:   vt,
				Hypotheses: a.hypotheses,
				Hits:       a.hits,
				Misses:     a.misses,
				Pending:    a.pending,
				HitRate:    pct(a.hits, a.hits+a.misses),
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(r); err != nil {
			fmt.Fprintf(os.Stderr, "calibrate: JSON encode: %v\n", err)
		}
		return
	}

	// --- text output ---

	fmt.Printf("[calibrate] %d cycles logged (%s to %s)\n\n",
		len(mlog.Cycles), earliest.Format("2006-01-02"), latest.Format("2006-01-02"))

	// Signal quantity (existing)
	fmt.Printf("  signal quantity (per cycle):\n")
	fmt.Printf("    signal rate:       %.0f%% (%d/%d cycles found papers)\n",
		pct(overall.withSignal, overall.total), overall.withSignal, overall.total)
	fmt.Printf("    total papers:      %d (avg %.1f per cycle)\n\n",
		overall.totalPaper, avg(overall.totalPaper, overall.total))

	// Prediction accuracy (new — the core metric)
	fmt.Printf("  prediction accuracy (per hypothesis):\n")
	fmt.Printf("    hypotheses scored: %d\n", len(scores))
	if resolved > 0 {
		fmt.Printf("    hit rate:          %.0f%% (%d/%d) — topology flagged, evidence found\n",
			pct(hits, resolved), hits, resolved)
		fmt.Printf("    miss rate:         %.0f%% (%d/%d) — topology flagged, no useful evidence\n",
			pct(misses, resolved), misses, resolved)
	}
	if pending > 0 {
		fmt.Printf("    pending:           %d hypotheses with unresolved cycles\n", pending)
	}
	if hitsWithCycles > 0 {
		fmt.Printf("    avg cycles to hit: %.1f\n", avg(totalCyclesToVerdict, hitsWithCycles))
	}

	// Signal quality breakdown (per cycle)
	fmt.Printf("\n  signal quality (resolved cycles):\n")
	totalResolved := resolutions["corroborated"] + resolutions["challenged"] + resolutions["irrelevant"] + resolutions["no_signal"] + resolutions["confirmed"] + resolutions["refuted"]
	useful := resolutions["corroborated"] + resolutions["challenged"] + resolutions["confirmed"]
	fmt.Printf("    useful:    %.0f%% (%d/%d) — corroborated, challenged, or confirmed\n",
		pct(useful, totalResolved), useful, totalResolved)
	if resolutions["irrelevant"] > 0 {
		fmt.Printf("    noise:     %.0f%% (%d/%d) — papers found but irrelevant\n",
			pct(resolutions["irrelevant"], totalResolved), resolutions["irrelevant"], totalResolved)
	}
	if resolutions["no_signal"] > 0 {
		fmt.Printf("    silence:   %.0f%% (%d/%d) — no papers found\n",
			pct(resolutions["no_signal"], totalResolved), resolutions["no_signal"], totalResolved)
	}
	if resolutions["refuted"] > 0 {
		fmt.Printf("    refuted:   %.0f%% (%d/%d) — KB-internal check rejected the claim\n",
			pct(resolutions["refuted"], totalResolved), resolutions["refuted"], totalResolved)
	}
	if unresolved > 0 {
		fmt.Printf("    unresolved: %d cycles\n", unresolved)
	}

	// Provenance overlap (gap_confirmed vs no_gap). A corroborated cycle
	// whose sources are all already in the corpus provenance is
	// tautological — the KB "corroborates" what it was built from.
	// Scanned post-hoc so the stat tracks the current corpus, not
	// whatever state existed when the cycle was logged.
	totalScanned := gapCounts["gap_confirmed"] + gapCounts["mixed_overlap"] + gapCounts["no_gap"]
	if totalScanned > 0 {
		fmt.Printf("\n  provenance overlap (sensor cycles with papers):\n")
		fmt.Printf("    gap_confirmed:     %.0f%% (%d/%d) — every source new to corpus\n",
			pct(gapCounts["gap_confirmed"], totalScanned), gapCounts["gap_confirmed"], totalScanned)
		fmt.Printf("    mixed_overlap:     %.0f%% (%d/%d) — some sources already in corpus, some new\n",
			pct(gapCounts["mixed_overlap"], totalScanned), gapCounts["mixed_overlap"], totalScanned)
		fmt.Printf("    no_gap:            %.0f%% (%d/%d) — every source already in corpus (tautological)\n",
			pct(gapCounts["no_gap"], totalScanned), gapCounts["no_gap"], totalScanned)
		if corrobNovel+corrobTaut > 0 {
			fmt.Printf("    of corroborated:   %d with novel source, %d tautological (%.0f%% non-tautological)\n",
				corrobNovel, corrobTaut, pct(corrobNovel, corrobNovel+corrobTaut))
		}
		if challNovel+challTaut > 0 {
			fmt.Printf("    of challenged:     %d with novel source, %d tautological (%.0f%% non-tautological)\n",
				challNovel, challTaut, pct(challNovel, challNovel+challTaut))
		}
	}

	// By prediction type — primary bucketing. Separates tautological
	// sensor-based predictions from KB-internal resolvers so their hit
	// rates are not averaged together.
	hasNonDefault := false
	for k := range byPredTypeAcc {
		if k != "structural_fragility" {
			hasNonDefault = true
			break
		}
	}
	if hasNonDefault {
		fmt.Printf("\n  by prediction type:\n")
		var types []string
		hasDefault := false
		for k := range byPredTypeAcc {
			if k == "structural_fragility" {
				hasDefault = true
				continue
			}
			types = append(types, k)
		}
		sort.Strings(types)
		if hasDefault {
			types = append([]string{"structural_fragility"}, types...)
		}
		for _, pt := range types {
			a := byPredTypeAcc[pt]
			hitRate := pct(a.hits, a.hits+a.misses)
			fmt.Printf("    %-25s %.0f%% hit rate (%d/%d hypotheses, %d pending)\n",
				pt, hitRate, a.hits, a.hits+a.misses, a.pending)
		}
	}

	// By vulnerability type
	fmt.Printf("\n  by vulnerability type:\n")
	for vt, s := range byVuln {
		a := byVulnAcc[vt]
		hitRate := pct(a.hits, a.hits+a.misses)
		fmt.Printf("    %-25s %.0f%% signal (%d/%d cycles), %.0f%% hit rate (%d/%d hypotheses)\n",
			vt, pct(s.withSignal, s.total), s.withSignal, s.total, hitRate, a.hits, a.hits+a.misses)
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

	// Per-hypothesis scorecard (replaces old flat list)
	fmt.Println("\n  per hypothesis:")
	for _, s := range scores {
		verdictMarker := " "
		switch s.Verdict {
		case "corroborated":
			verdictMarker = "+"
		case "challenged":
			verdictMarker = "!"
		case "confirmed":
			verdictMarker = "✓"
		case "irrelevant":
			verdictMarker = "-"
		case "no_signal":
			verdictMarker = "."
		case "refuted":
			verdictMarker = "✗"
		case "pending":
			verdictMarker = "?"
		}
		efficiency := ""
		if s.CyclesToVerdict > 0 {
			efficiency = fmt.Sprintf(" (%d cycles to hit)", s.CyclesToVerdict)
		}
		precisionStr := ""
		if s.WithSignal > 0 {
			precisionStr = fmt.Sprintf(", %.0f%% precision", s.Precision)
		}
		fmt.Printf("  %s %-42s %-14s %d/%d signal%s%s\n",
			verdictMarker, s.Name, s.Verdict, s.WithSignal, s.TotalCycles, precisionStr, efficiency)
	}
	fmt.Println("\n  legend: + corroborated  ! challenged  ✓ confirmed  - irrelevant  . no_signal  ✗ refuted  ? pending")
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

// runCalibrateNarrative produces a temporal story for each hypothesis:
// when the prediction was made, what happened, and whether reality confirmed.
func runCalibrateNarrative(dir string) {
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	if len(mlog.Cycles) == 0 {
		fmt.Println("[calibrate] no cycles logged yet")
		return
	}

	scores := scoreHypotheses(mlog.Cycles)

	// Group cycles by hypothesis with temporal ordering
	byHyp := map[string][]Cycle{}
	for _, c := range mlog.Cycles {
		byHyp[c.Hypothesis] = append(byHyp[c.Hypothesis], c)
	}

	// Count resolved
	var resolved, hits int
	for _, s := range scores {
		switch s.Verdict {
		case "corroborated", "challenged":
			resolved++
			hits++
		case "irrelevant", "no_signal":
			resolved++
		}
	}

	fmt.Println("# Prediction Report")
	fmt.Println()

	// Honest sample size framing
	total := resolved
	if total > 0 {
		fmt.Printf("**%d of %d predictions confirmed** (%.0f%% hit rate, N=%d).\n",
			hits, total, pct(hits, total), total)
		fmt.Println()
		if total < 30 {
			fmt.Printf("*Note: N=%d is too small for statistical significance. ", total)
			fmt.Printf("Treat this as directional evidence, not proof.*\n")
		}
		fmt.Println()
	}

	// Per-hypothesis narratives
	for _, s := range scores {
		cycles := byHyp[s.Name]
		if len(cycles) == 0 {
			continue
		}

		verdictEmoji := "?"
		switch s.Verdict {
		case "corroborated":
			verdictEmoji = "+"
		case "challenged":
			verdictEmoji = "!"
		case "irrelevant":
			verdictEmoji = "-"
		case "no_signal":
			verdictEmoji = "."
		}

		fmt.Printf("## %s %s\n\n", verdictEmoji, s.Name)

		// First cycle = when prediction was made
		first := cycles[0]
		fmt.Printf("**Predicted** (%s): %s\n", first.Timestamp.Format("2006-01-02"), first.Prediction)
		fmt.Printf("**Query**: %q\n", first.Query)
		fmt.Printf("**Vulnerability**: %s\n\n", first.VulnType)

		// Timeline
		fmt.Println("**Timeline**:")
		for i, c := range cycles {
			status := "queried"
			if c.Resolution != "" {
				status = c.Resolution
			}
			backend := c.Backend
			if backend == "" {
				backend = "arxiv"
			}
			fmt.Printf("  %d. %s [%s] %d results → %s",
				i+1, c.Timestamp.Format("2006-01-02 15:04"), backend, c.PapersFound, status)

			// Show first paper title if available
			if len(c.Papers) > 0 {
				title := c.Papers[0].Title
				if len(title) > 60 {
					title = title[:57] + "..."
				}
				fmt.Printf(" — %q", title)
			}
			fmt.Println()
		}

		// Verdict
		fmt.Println()
		switch s.Verdict {
		case "corroborated":
			fmt.Printf("**Verdict**: Confirmed in %d cycles. External sources independently ", s.CyclesToVerdict)
			fmt.Printf("corroborate the structural vulnerability topology identified.\n")
		case "challenged":
			fmt.Printf("**Verdict**: Challenged. External sources contradict the KB's position.\n")
		case "irrelevant":
			fmt.Printf("**Verdict**: Irrelevant. Sources were found but didn't bear on the hypothesis.\n")
		case "pending":
			fmt.Printf("**Verdict**: Pending. %d cycles run, no conclusive resolution yet.\n", s.TotalCycles)
		case "no_signal":
			fmt.Printf("**Verdict**: No signal. Sensors found nothing relevant.\n")
		}
		fmt.Println()
	}
}

// --- cycle (full sleep cycle) ---

// runCycle executes the full epistemic evolution cycle:
//
//  1. Sense (wake): topology → sensor queries across all backends
//  2. Evaluate: auto-resolve pending hypotheses with content-aware LLM
//  3. Ingest: pipeline quality-gated ingest from corroborated ZIM cycles
//  4. Dream (NREM): consolidation + brief tightening + bias audit
//  5. Trip (REM): speculative cross-cluster connections
//  6. Calibrate: prediction accuracy + reify predictions.go
//
// Each phase runs independently. If one fails, the others still execute.
// This is the single entry point for autonomous KB evolution.
// The KB grows and self-corrects on every run.
func runCycle(dir, zimPath, zimIndex string, llmBudget, entityCap int, dryRun, jsonOut bool) {
	fmt.Println("[cycle] starting full sleep cycle")
	fmt.Println()

	phases := 0
	failures := 0

	// Phase 0: Bias audit gates downstream phases. README flagged
	// "triggered bias auditors don't gate the next metabolism phase"
	// as a known problem — wiring it here closes that gap. At minimum
	// the triggered set is surfaced; specific triggers can modify
	// phase behavior (currently: availability_heuristic → skip ZIM).
	biasGates := runBiasGates(dir)
	phases++
	fmt.Println()

	// Phase 1: Metabolism (sensor queries + optional ingest)
	{
		fmt.Println("=== Phase 1: Metabolism (wake) ===")
		fmt.Println()

		// Check entity cap
		count, err := countEntities(dir)
		if err == nil && count >= entityCap {
			fmt.Printf("[cycle] entity cap reached (%d/%d) — skipping ingest, deepening only\n", count, entityCap)
		} else {
			// Run one sensor cycle
			targets, report, err := runTopology(dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[cycle] topology: %v\n", err)
				failures++
			} else if len(targets) > 0 {
				fmt.Printf("[cycle] %d sensor targets from topology (%d entities, %d claims)\n",
					len(targets), report.Entities, report.Claims)

				// Query backends for first 3 targets
				limit := 3
				if len(targets) < limit {
					limit = len(targets)
				}
				for i := 0; i < limit; i++ {
					t := targets[i]
					if dryRun {
						fmt.Printf("  [dry-run] would query: %s → %q\n", t.Hypothesis, t.Query)
						continue
					}
					// ZIM backend (if available and not gated by bias audit).
					// Availability-heuristic trigger means provenance HHI > 0.25,
					// i.e. the KB is already over-concentrated on one source.
					// Querying ZIM (Wikipedia) while concentrated would deepen the
					// concentration; skip it this cycle to diversify.
					if zimPath != "" && !biasGates.skipZim {
						zimQ := t.queryFor("zim")
						fmt.Printf("  querying (zim): %s → %q\n", t.Hypothesis, zimQ)
						runSensorCycle(dir, zimPath, zimIndex, t, "zim", nil)
					} else if zimPath != "" && biasGates.skipZim {
						fmt.Printf("  skipping zim for %s (availability-heuristic bias gate)\n", t.Hypothesis)
					}
					// RSS backend (always available, no prereqs)
					rssQ := t.queryFor("rss")
					fmt.Printf("  querying (rss): %s → %q\n", t.Hypothesis, rssQ)
					runSensorCycle(dir, "", "", t, "rss", nil)
				}
				phases++
			} else {
				fmt.Println("[cycle] no sensor targets — graph is well-covered")
			}
		}
		fmt.Println()
	}

	// Load API key for LLM phases (auto-resolve, dream-fix, trip)
	loadDotEnv(dir)

	// Phase 1b: Auto-resolve pending hypotheses with sufficient signal
	if os.Getenv("ANTHROPIC_API_KEY") != "" && !dryRun {
		fmt.Println("=== Phase 1b: Evaluate (auto-resolve pending hypotheses) ===")
		fmt.Println()
		results := autoResolve(dir)
		if len(results) > 0 {
			fmt.Printf("[auto-resolve] resolved %d hypotheses\n", len(results))

			// Act on challenges: generate Disputes claims in corpus files
			kbVars := collectKBVars(dir)
			for _, r := range results {
				if r.Outcome != "challenged" {
					continue
				}
				info, ok := kbVars[r.Hypothesis]
				if !ok || info.File == "" {
					continue
				}
				// Build a summary of contradicting evidence from paper snippets
				var evidence []string
				for _, p := range r.Papers {
					if p.Snippet != "" {
						evidence = append(evidence, p.Title+": "+p.Snippet)
					} else {
						evidence = append(evidence, p.Title)
					}
				}
				evidenceText := strings.Join(evidence, "; ")
				if len(evidenceText) > 300 {
					evidenceText = evidenceText[:300]
				}

				// Generate and append a Disputes claim
				code := fmt.Sprintf("\n// ---------------------------------------------------------------------------\n"+
					"// Auto-resolved challenge: external sources contradict %s\n"+
					"// ---------------------------------------------------------------------------\n\n"+
					"var metabolismChallenge%sSource = Provenance{\n"+
					"\tOrigin:     \"metabolism auto-resolve (content-aware evaluation)\",\n"+
					"\tIngestedAt: %q,\n"+
					"\tIngestedBy: \"winze metabolism auto-resolve (Sonnet, skeptical prompt)\",\n"+
					"\tQuote:      %q,\n"+
					"}\n",
					r.Hypothesis, r.Hypothesis,
					time.Now().Format("2006-01-02"),
					cleanLLMString(evidenceText))

				existing, err := os.ReadFile(info.File)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[challenge-action] read %s: %v\n", info.File, err)
					continue
				}
				backup := make([]byte, len(existing))
				copy(backup, existing)

				appended := string(existing) + code
				formatted, fmtErr := format.Source([]byte(appended))
				if fmtErr != nil {
					formatted = []byte(appended)
				}
				if err := os.WriteFile(info.File, formatted, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "[challenge-action] write %s: %v\n", info.File, err)
					continue
				}

				// Verify build
				buildCmd := exec.Command("go", "build", "./...")
				buildCmd.Dir = dir
				if buildErr := buildCmd.Run(); buildErr != nil {
					if rbErr := os.WriteFile(info.File, backup, 0644); rbErr != nil {
						fmt.Fprintf(os.Stderr, "[challenge-action] CRITICAL: rollback failed for %s: %v\n", info.File, rbErr)
					}
					fmt.Fprintf(os.Stderr, "[challenge-action] build failed, reverted %s\n", filepath.Base(info.File))
				} else {
					fmt.Printf("[challenge-action] appended challenge provenance to %s\n", filepath.Base(info.File))
				}
			}
		} else {
			fmt.Println("[auto-resolve] no candidates (need 3+ cycles with signal)")
		}
		fmt.Println()
	}

	// Phase 2: Pipeline ingest (evolve the KB from corroborated findings)
	if os.Getenv("ANTHROPIC_API_KEY") == "" && !dryRun {
		fmt.Println("[cycle] skipping auto-resolve (no ANTHROPIC_API_KEY)")
	}

	if os.Getenv("ANTHROPIC_API_KEY") != "" && zimPath != "" && !dryRun {
		fmt.Println("=== Phase 2: Ingest (evolve KB from corroborated findings) ===")
		fmt.Println()

		// Check entity cap before ingesting
		count, err := countEntities(dir)
		if err == nil && count >= entityCap {
			fmt.Printf("[ingest] entity cap reached (%d/%d) — skipping ingest\n", count, entityCap)
		} else {
			runPipeline(dir, zimPath, "", llmBudget)
			invalidateTopologyCache() // KB may have changed
			phases++
		}
		fmt.Println()
	} else if !dryRun {
		fmt.Println("[cycle] skipping ingest (no ANTHROPIC_API_KEY or --zim)")
	}

	// Phase 3: Trip (speculative connections — runs BEFORE dream so dream can clean up)
	var tripReport TripReport
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		fmt.Println("=== Phase 3: Trip (REM speculation) ===")
		fmt.Println()
		if dryRun {
			fmt.Println("[cycle] [dry-run] would run trip cycle")
		} else {
			tripReport = runTrip(dir, 1.0, "analogy", 3, false, jsonOut) // temp=1.0, analogy, 3 pairs
			// Promote high-scoring connections to corpus claims
			if len(tripReport.Connections) > 0 {
				if err := promoteConnections(dir, tripReport.Connections, 4); err != nil {
					fmt.Fprintf(os.Stderr, "[cycle] trip promotion: %v\n", err)
				}
			}
		}
		phases++
		fmt.Println()
	} else {
		fmt.Println("=== Phase 3: Trip (skipped — no ANTHROPIC_API_KEY) ===")
		fmt.Println()
	}

	// Phase 4: Dream (consolidation + cleanup — runs AFTER ingest + trip to catch their noise)
	fmt.Println("=== Phase 4: Dream (NREM consolidation) ===")
	fmt.Println()
	dreamReport := runDream(dir, true, jsonOut) // includeBias=true
	phases++
	_ = dreamReport

	// Auto-fix overlong Briefs if API key is available
	if os.Getenv("ANTHROPIC_API_KEY") != "" && !dryRun {
		fmt.Println()
		fmt.Println("[cycle] running Brief tightening...")
		runDreamFix(dir, true, false, jsonOut) // tighten=true, dryRun=false
		// Future: consolidateProvenance from dreamReport.ProvenanceSplits
		// Future: fix framing effect from dreamReport.BiasAudit
	} else if !dryRun {
		fmt.Println("[cycle] skipping Brief tightening (no ANTHROPIC_API_KEY)")
	}
	fmt.Println()

	// Phase 5: Calibrate + reify
	fmt.Println("=== Phase 5: Calibrate ===")
	fmt.Println()
	runCalibrate(dir, false)
	phases++

	// Reify predictions
	fmt.Println()
	runReify(dir)
	fmt.Println()

	// Summary
	fmt.Printf("[cycle] complete: %d phases executed, %d failures\n", phases, failures)
	if failures > 0 {
		os.Exit(1)
	}
}

// runSensorCycle runs a single sensor query and logs the result.
func runSensorCycle(dir, zimPath, zimIndex string, target SensorTarget, backend string, rssURLs []string) {
	q := target.queryFor(backend)
	var results []PaperSummary
	var err error
	switch backend {
	case "zim":
		results, err = searchZim(zimPath, zimIndex, q, 5)
	case "rss":
		feeds := rssURLs
		if len(feeds) == 0 {
			feeds = defaultFeeds
		}
		for _, feedURL := range feeds {
			r, ferr := searchRSS(feedURL, q, 5)
			if ferr != nil {
				continue
			}
			results = append(results, r...)
		}
	case "arxiv":
		results, err = searchArxiv(q, 5)
	default:
		results, err = searchZim(zimPath, zimIndex, q, 5)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "  sensor: search: %v\n", err)
		return
	}

	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	cycle := Cycle{
		Timestamp:   time.Now(),
		Hypothesis:  target.Hypothesis,
		Prediction:  target.Prediction,
		Query:       q,
		Backend:     backend,
		VulnType:    target.VulnType,
		VulnCount:   target.VulnCount,
		PapersFound: len(results),
		Papers:      results,
	}

	mlog.Cycles = append(mlog.Cycles, cycle)
	if err := saveLog(logPath, mlog); err != nil {
		fmt.Fprintf(os.Stderr, "  sensor: save log: %v\n", err)
	}

	fmt.Printf("  → %d results for %s\n", len(results), target.Hypothesis)
}

// --- suggest ---

// runSuggest generates a template .go corpus file from corroborated metabolism
// cycles. The output follows the pattern established by metabolism_cycle2.go:
// provenance stubs, entity stubs, and claim stubs with TODOs for human review.
func runSuggest(dir string) {
	logPath := filepath.Join(dir, ".metabolism-log.json")
	mlog := loadLog(logPath)

	// Collect corroborated cycles with papers, grouped by hypothesis
	type suggestion struct {
		hypothesis string
		prediction string
		backend    string
		papers     []PaperSummary
	}
	seen := map[string]*suggestion{}
	var order []string // preserve first-seen order
	for _, c := range mlog.Cycles {
		if c.Resolution != "corroborated" || c.PapersFound == 0 {
			continue
		}
		if s, ok := seen[c.Hypothesis]; ok {
			// Merge papers from additional cycles
			s.papers = append(s.papers, c.Papers...)
		} else {
			seen[c.Hypothesis] = &suggestion{
				hypothesis: c.Hypothesis,
				prediction: c.Prediction,
				backend:    c.Backend,
				papers:     append([]PaperSummary{}, c.Papers...),
			}
			order = append(order, c.Hypothesis)
		}
	}

	if len(order) == 0 {
		fmt.Fprintln(os.Stderr, "metabolism: no corroborated cycles with papers found — nothing to suggest")
		return
	}

	// Deduplicate papers by ID within each suggestion
	for _, s := range seen {
		idSeen := map[string]bool{}
		var deduped []PaperSummary
		for _, p := range s.papers {
			if !idSeen[p.ID] {
				idSeen[p.ID] = true
				deduped = append(deduped, p)
			}
		}
		s.papers = deduped
	}

	// Determine cycle number from existing corpus files
	cycleNum := nextCycleNumber(dir)

	// Emit Go template
	fmt.Printf("package winze\n\n")
	fmt.Printf("// Metabolism cycle %d ingest: corroboration claims surfaced by\n", cycleNum)
	fmt.Printf("// topology-driven sensor queries.\n")
	fmt.Printf("//\n")
	fmt.Printf("// Generated by: go run ./cmd/metabolism --suggest .\n")
	fmt.Printf("// Review each TODO, fill in provenance quotes from the source,\n")
	fmt.Printf("// then: go build ./... && go run ./cmd/lint .\n")

	for _, hypName := range order {
		s := seen[hypName]
		fmt.Printf("\n// ---------------------------------------------------------------------------\n")
		fmt.Printf("// %s — %s\n", s.hypothesis, s.prediction)
		fmt.Printf("// ---------------------------------------------------------------------------\n\n")

		// Determine claim type from prediction
		claimType := "Proposes" // single-source needs a second proposer
		if strings.Contains(s.prediction, "uncontested") {
			claimType = "Disputes" // uncontested needs a disputant
		}

		for i, p := range s.papers {
			varPrefix := sanitizeVarName(s.hypothesis, i)
			backend := s.backend
			if backend == "" {
				backend = "arxiv"
			}

			fmt.Printf("// Source: %s\n", p.Title)
			fmt.Printf("// ID: %s\n", p.ID)
			if p.Year > 0 {
				fmt.Printf("// Year: %d\n", p.Year)
			}
			fmt.Printf("//\n")
			fmt.Printf("// TODO: Read the source. If it explicitly commits to a %s\n", strings.ToLower(claimType))
			fmt.Printf("// relationship with %s, fill in the template below.\n", s.hypothesis)
			fmt.Printf("// If not, delete this block.\n\n")

			fmt.Printf("// var %sSource = Provenance{\n", varPrefix)
			fmt.Printf("// \tOrigin:     %q,\n", p.ID)
			fmt.Printf("// \tIngestedAt: \"TODO\",\n")
			fmt.Printf("// \tIngestedBy: \"winze metabolism cycle %d (sensor: %s)\",\n", cycleNum, backend)
			fmt.Printf("// \tQuote:      \"TODO: exact text from source supporting the claim\",\n")
			fmt.Printf("// }\n\n")

			fmt.Printf("// TODO: create entity if needed (check if already exists in KB)\n")
			fmt.Printf("// var %sEntity = Person{&Entity{\n", varPrefix)
			fmt.Printf("// \tID:    \"TODO\",\n")
			fmt.Printf("// \tName:  \"TODO\",\n")
			fmt.Printf("// \tKind:  \"person\",\n")
			fmt.Printf("// \tBrief: \"TODO\",\n")
			fmt.Printf("// }}\n\n")

			fmt.Printf("// var %sClaim = %s{\n", varPrefix, claimType)
			fmt.Printf("// \tSubject: %sEntity,  // TODO: use existing entity if one exists\n", varPrefix)
			fmt.Printf("// \tObject:  %s,\n", s.hypothesis)
			fmt.Printf("// \tProv:    %sSource,\n", varPrefix)
			fmt.Printf("// }\n\n")
		}
	}
}

// nextCycleNumber finds the highest N in metabolism_cycleN.go and returns N+1.
func nextCycleNumber(dir string) int {
	matches, _ := filepath.Glob(filepath.Join(dir, "metabolism_cycle*.go"))
	max := 0
	for _, m := range matches {
		var n int
		if _, err := fmt.Sscanf(filepath.Base(m), "metabolism_cycle%d.go", &n); err == nil {
			if n > max {
				max = n
			}
		}
	}
	if max == 0 {
		return 1
	}
	return max + 1
}

// sanitizeVarName creates a Go variable name prefix from a hypothesis name
// and paper index.
func sanitizeVarName(hypothesis string, index int) string {
	// Use first ~30 chars of hypothesis + index suffix
	name := hypothesis
	if len(name) > 30 {
		name = name[:30]
	}
	if index > 0 {
		return fmt.Sprintf("%sPaper%d", name, index+1)
	}
	return name + "Paper"
}
