# winze-benchmark (retrieval benchmark)

`winze-benchmark` runs winze's retrieval benchmark: four retrieval modes compete
on the same fixed question corpus, so the value of the structured substrate over
a plain-text baseline is measured rather than asserted.

The modes:

- **grep** — keyword match over var-block text (the naive baseline)
- **bm25** — BM25 ranking over var-block text (a proper unstructured baseline)
- **defn** — SQL queries against defn's SQLite database (structured, realistic)
- **ast** — hand-written `go/ast` queries (structured, the ceiling)

A separate, external-corpus benchmark lives in `cmd/longmemeval`: it runs winze's
hybrid extract→substrate→gate→reason loop against the LongMemEval-S long-term
memory dataset, measuring end-to-end answer accuracy (not just retrieval) with a
per-hop timing breakdown that separates the winze-via-defn machinery from the LLM
hops. Where `winze-benchmark` measures retrieval on winze's own corpus,
`cmd/longmemeval` measures the whole memory pipeline on a third-party dataset.
See [Running longmemeval](#running-longmemeval) below.

```bash
go run ./cmd/benchmark .
```

The point is the gap between the unstructured baselines (grep, bm25) and the
structured ones (defn, ast): if typing the corpus buys nothing on retrieval, the
gap is zero. This is the evidence side of "is the typed gate worth its friction",
the same question rot-probe answers for maintenance.

## Running longmemeval

The dataset is not in git (oracle ~15 MB, full `_s` ~265 MB). Re-acquire it with
`cmd/longmemeval/fetch-data.sh`, which pulls the cleaned HuggingFace release
(`xiaowu0162/longmemeval-cleaned`) into `cmd/longmemeval/data/`.

```bash
go build -o /tmp/lme ./cmd/longmemeval
WINZE_BIN=$PWD/bin /tmp/lme \
  --dataset cmd/longmemeval/data/longmemeval_oracle.json \
  --work <workdir> --cache <cachedir> \
  --temporal 10 --knowledge 10 --single 10 \
  --multi 10 --assistant 10 --preference 10
```

Per-type counts default to a five-question smoke subset that runs **none** of
`--multi`, `--assistant`, or `--preference`. Those three are the hard types and
the ones the `--probe` work flagged as the lens-scoping risk, so a run that
leaves them at zero is a pipeline check, not a result. Available in the oracle
set: 133 multi-session, 133 temporal-reasoning, 78 knowledge-update, 70
single-session-user, 56 single-session-assistant, 30 single-session-preference.

`--cache` is worth passing explicitly and reusing. Extraction dominates
wall-clock and caches per session to disk, so a warm re-run pays only
sync + retrieve + answer + judge.

`--dry-run` prints the selected subset and its session counts without spending
anything; `--probe` reports whether gold-answer turns survive `renderSession`
truncation, also without API calls.

### Reading the two accuracy lines

A question that errors is skipped, so `rows` holds survivors. The report prints
`scored / errored / attempted` and, when they differ, accuracy both over scored
questions and over attempted ones (counting errors as wrong). Neither is
automatically the right number — whether an error is a harness bug or a genuine
failure to answer is a judgement for whoever reads the stderr lines. Quote the
one you can defend, and say which.

### Committing a baseline

`--baseline <path>` writes the per-question outcomes as JSONL: qid, type,
question, gold, answer, verdict, and the fact counts. Pass it on any run whose
configuration you might later want to compare against, and commit the file to
`cmd/longmemeval/baselines/`.

This exists because of the noise floor below. A two-question move in the
aggregate is unreadable; *which* questions flipped is not, and that comparison
needs the previous run's rows. Answers and judgements do not cache, so
regenerating a baseline costs real spend — cheaper to keep it than to re-earn
it. The run log stays out of git: its lessons belong in this file, and its
timing columns are contaminated (see below). The diff recipe is in
`cmd/longmemeval/baselines/README.md`.

### What these numbers are not

**Every score on this page is on the oracle set, which is the benchmark with
the distractors removed.** `cmd/longmemeval/data/` holds
`longmemeval_oracle.json` and nothing else: only the answer-bearing sessions,
a mean of 1.9 per question, min 1 and max 6, 948 sessions across all 500
questions. The full `longmemeval_s` haystack is 265 MB against the oracle's
15, and finding the needle in it is the thing LongMemEval was built to measure.

So these runs measure extraction and answering. They do not measure retrieval
among distractors, and a gain here is not evidence of one there. The single
clearest example sits in the failure list below: `0edc2aef` retrieved a
Seattle trip for a question about Miami, which is exactly the class of error
the oracle set under-represents by construction.

Fixing that means running the real haystack. See the cost table at the end —
it is affordable and, before concurrency landed, it was not feasible.

### Results, 2026-08-06

First scoring runs the harness has ever had. Balanced 60-question subset, ten
of each type, oracle set. Accuracy over scored questions; every run was 60
scored of 60 attempted, no errors.

| lens | k | total | temporal | knowledge | single-user | multi | preference | assistant |
|------|---|-------|----------|-----------|-------------|-------|------------|-----------|
| v2   | 15 | 36/60 60% | 9/10 | 9/10 | 7/10 | 6/10 | 4/10 | 1/10 |
| v3   | 15 | 42/60 70% | 9/10 | 9/10 | 9/10 | 6/10 | 5/10 | 4/10 |
| v4   | 15 | 44/60 73% | 8/10 | 8/10 | 10/10 | 7/10 | 6/10 | 5/10 |
| v4   | 30 | 45/60 75% | 9/10 | 8/10 | 10/10 | 8/10 | 5/10 | 5/10 |
| v4   | 60 | 47/60 78% | 9/10 | 8/10 | 10/10 | 9/10 | 6/10 | 5/10 |
| v5   | 60 | 46/60 77% | 9/10 | 8/10 | 9/10 | 9/10 | 6/10 | 5/10 |
| v6   | 60 | 49/60 82% | 10/10 | 9/10 | 9/10 | 8/10 | 7/10 | 6/10 |
| v7   | 60 | 49/60 82% | 10/10 | 9/10 | 9/10 | 8/10 | 6/10 | 7/10 |
| v9   | 60 | 52/60 87% | 10/10 | 9/10 | 10/10 | 8/10 | 6/10 | 9/10 |
| v9 + answerer | 60 | 54/60 90% | 10/10 | 9/10 | 10/10 | 8/10 | 8/10 | 9/10 |

The v6 row is the mean of nothing — both runs scored 49/60 with identical
per-type breakdowns and all sixty questions agreeing with themselves, which is
the only reason it is quotable at all. Against v5 the stable per-question diff
is five fixed and two broken.

The last three rows are one evening's work and each was found by reading model
output rather than by reasoning about prompts:

- **v7 raised `MaxTokens` from 1024 to 4096.** Rule 3a had told the lens since
  v4 that thirty meaningful cells is thirty lines and that compressing an
  enumeration is wrong; the cap made that instruction unfollowable. Total facts
  across the sixty questions went 1402 → 1784, so a quarter of every extraction
  was being discarded. The score did not move — `e9327a54` went 21 facts → 54
  and wrong → right, and one preference question went the other way on noise.
  Correctness bought without a score is still worth having: the same cap sat on
  `cmd/metabolism/ingest.go`, where a mid-quote cut would have written half a
  sentence into the corpus presented as verbatim.
- **v8 added a second extraction pass** for sessions the first pass returns
  nothing on, and **v9 gave that pass the temporal clause** the primary prompt
  has carried since v3. Starvation went 7 of 60 to 2. The assistant row, flat
  at 5/10 across every `k` and every lens version through v6, went to 9/10.
- **The answerer change scoped its "I don't know" rule** to questions about
  stored facts. Preference 6/10 → 8/10 with knowledge-update and temporal
  unmoved, on a warm cache, so the facts were held fixed and only the answerer
  varied.

A later cold run at concurrency 8 scored **55/60 with preference at 9/10 and
all five retried questions right**, which is one question above the row above
it and therefore inside the noise. The per-type shape is what repeats.

**The noise floor is ±2 questions, and pinning the sampler does not close it.**
Two runs of byte-identical input — same lens version, same `k`, all 102
sessions served from the extraction cache — scored 43/60 and 44/60 while all
three hops sampled at the API default. `Temperature: 0` is now set on all
three (`lens.go`, `answer.go`, `judge.go`) and the same experiment repeated
scored **45/60 and 47/60**: 16 of 60 answers came back not byte-identical, and
two of those differed enough to flip the judge. The residual is API-side, not
sampler-side, so no client setting reaches it.

The practical consequence: a one- or two-question difference between
configurations is not a result, and a per-question diff is not automatically
one either. To claim a specific question flipped because of a change, run the
new configuration twice and trust only the questions that agree with
themselves. Do not read the per-type columns above as trends unless a row
moves by three or more.

What survives that bar:

- **v2 → v3 on assistant, 1/10 → 4/10.** The lens was told to skip assistant
  content and did. See `lensVersion` for the whole diagnosis.
- **k=15 → k=60, 44 → 47, driven by multi-session 7 → 8 → 9.** Monotone, and
  concentrated in the type with the most sessions per question and therefore
  the most facts competing for the window. The v3 run measured 27 of 60
  questions producing more facts than `k=15` admits, so the mechanism was
  visible in the fact counts before it was visible in the score.

Two rows are flat across every `k`, which is its own finding:

- **knowledge-update, 8/10 at all three.** More window does not reach it, so
  those failures are extraction-side. "How many Korean restaurants have I
  tried" answered correctly under v2 and v3 and wrongly under v4 — the
  granularity rule emits the superseded and current counts as sibling rows
  rather than one updated fact, and the answerer picks whichever ranks.
- **assistant, 5/10 at all three.** The remaining failures answer "I don't
  know", and no window recovers a fact that was never extracted. v4's
  enumeration rule is not firing on those cases.

  *That last sentence was wrong, and the way it was wrong is worth keeping.*
  A `WINZE_LENS_DEBUG=1` dump over the ten assistant sessions on a cold cache
  found the enumeration rule firing hard enough to overrun the budget: three
  sessions stopped at `max_tokens`, and `e9327a54` scored wrong because the
  dessert shop it asked about sat in the discarded tail. The four failures
  decompose as three refusals and one truncation, needing two different fixes.
  Neither is "the rule is not firing". The claim was inferred from a fact count
  and never checked against a completion.

**Preference questions are graded against the wrong shape.** Their gold is not
an answer, it is a rubric — "the user would prefer relaxing activities before
9:30 pm" — while the model produces a response that *satisfies* the rubric.
The judge asks whether the answer conveys the same information as the gold, so
it correctly reports that a list of activities is not a description of what a
good list contains. A hand audit of all 15 failures at k=30 found exactly one
clear mis-score of this kind; the other three preference failures answered "I
don't know" and are genuine retrieval misses, not grading artifacts.

**The last clause was wrong, and this one was expensive.** Those "I don't
know" answers were not retrieval misses. `answerSystem` had a single escape
hatch — *if the facts do not contain the answer, say exactly: I don't know* —
which is right for a question about a stored fact and fires by construction on
a preference question, because "suggest a hotel for my Miami trip" asks the
model to act on what it remembers and the specific hotel is never stored and
never could be. On `0edc2aef` the answerer held the preferences (great views,
rooftop pool, balcony hot tub) and replied that the trip was actually to
Seattle. With the rule scoped to stored-fact questions it recommends a
specific Miami hotel on exactly those preferences and flags the Seattle
discrepancy rather than hiding behind it.

Measured on a warm cache so the facts were identical and only the answerer
varied: preference 6/10 → 8/10 and 7/10 over two runs, knowledge-update 9/10
and temporal 10/10 unchanged in both. The stable diff is exactly two
questions, both preference, both wrong → right. The feared trade — buying
preference points by fabricating where refusal is correct — did not appear.

The grading artifact above is real and the rubric mismatch stands. What was
wrong was attributing the *other* failures to retrieval on the strength of
their surface form. "I don't know" is what a refusal and a retrieval miss both
look like from the outside, which is the same mistake as reading a fact count
and calling it an extraction failure.

### Two ways an extraction fails, and only one of them is visible

The report prints two blocks after the score. They look similar and diagnose
opposite things.

**ZERO-FACT EXTRACTIONS** counts questions where the lens produced nothing and
the answerer was handed an empty context. Seven of sixty went that way in the
v4 baseline and all seven scored wrong while the report showed only 47/60.
Four of them starve under every lens version tried — `75832dbd`, `89527b6b`,
`1903aded`, `ceb54acb` — and the debug dump shows why: the lens returns the
literal string `NO_FACTS` in seven output tokens. Three are assistant-recall
questions where the user genuinely said nothing about themselves ("Brainstorm
ideas for work from home jobs for seniors", then later "what was the 7th job in
the list you provided?"). The answer lives entirely in the assistant's output,
and a user-fact lens declines. Rule 3a names `wfh_job_7` and
`plesiosaur_body_colour` as its worked examples — the exact two failures — so
this is not an instruction the prompt is missing.

**TRUNCATED EXTRACTIONS** counts questions where the lens was still talking
when the token budget ran out. This one hides. A cut session reports a
normal-looking fact count, because what was dropped was never counted, and it
surfaces only as a wrong answer on a question whose evidence sat past the cut —
which reads as a retrieval miss. `MaxTokens` was 1024 through v6 while rule 3a
instructed the lens that thirty meaningful cells is thirty lines; v7 raises it
to 4096 and widens the extraction cache so a warm entry carries the flag.

Do not confuse either with render-side truncation. `--probe` reports whether
gold-answer turns survive `renderSession`, costs nothing, and returned `cut=0`
on all sixty — so the lens sees the whole session in every case above.

The general lesson is cheaper than the runs that taught it: **a fact count
cannot distinguish a model that declined, a model that errored, and a model
that ran out of room.** Those need three different fixes and look identical
from outside. The dump that separates them is ten API calls. Reach for it
before spending a cold re-extraction on a hypothesis.

### A third block: what the retry recovered

**RETRIED EXTRACTIONS** names sessions the first pass returned nothing on and
`lensRetrySystem` rescued. It is the entire case for the second call existing:
empty means the retry is buying nothing and should come out.

The retry is itself a sampled decision, not a deterministic rescue. On a
seven-question probe it recovered 7 of 7; on the full sixty it recovered 5 of
7, with `75832dbd` and `ceb54acb` returning `NO_FACTS` from *both* passes. So
the honest claim is that it fixes most starvation most of the time, and a
third pass would presumably shave the remainder with diminishing returns.

Rows here that still score wrong are informative in the other direction: the
facts arrived and the answer did not, which moves the failure downstream. That
is how the answerer bug above was found — `58ef2f1c` came back with 42 facts
and still scored wrong, and reading its cache entry showed an event with no
date on a question asking when.

### What a run costs

Measured off the v9 cold run's own token counters — 109 Haiku calls, 120
Sonnet, nothing served from cache — at list pricing. Extraction scales with
sessions; answer and judge scale with questions.

| run | cost | wall (serial) | wall (`--concurrency 8`) |
|-----|------|---------------|--------------------------|
| 60-question subset, cold | $1.34 | 25 min | **2.3 min** |
| 60-question subset, warm | $0.51 | 4.9 min | 1.1 min |
| full oracle, 500 q / 948 sessions | ~$12 | 3.9 h | ~21 min |
| full `longmemeval_s` haystack | ~$138 | 67 h | ~5.6 h |

The cold subset numbers are measured, not projected: 1514 s serial against
138 s at concurrency 8, an 11× speedup on a run that is ~99% blocked on the
API. Per question the split is extract 12.5 s, answer 2.5 s, judge 1.1 s
against 0.9 s for the whole winze machinery — build 128 ms, defn sync 795 ms,
retrieve 2.2 ms. The rest of the table scales those two measurements.

The money was never the obstacle. 67 hours was, and it is why every result
above is on sixty questions with the distractors removed. That constraint is
gone.

### Timings from a busy machine are ceilings, not measurements

Accuracy is load-independent here: all three LLM calls use `context.Background()`
with no deadline (`lens.go`, `answer.go`, `judge.go`), so contention cannot trip a
timeout and silently drop a question. A score taken on a loaded box is a real
score.

The timing breakdown is not. It exists to separate the winze-via-defn machinery
(build + sync + retrieve) from the LLM hops, which is precisely the column a perf
claim would quote — and precisely the one that inflates under contention. This
development box runs other agent sessions and an hourly metabolism timer, so
numbers taken on it are upper bounds. **Publish a perf claim only from a quiet
dedicated host.** Check `uptime` first either way; see the
`Check-load-average-before-trusting-cpu-bound-timings` memory for the incident
that convention came from.
