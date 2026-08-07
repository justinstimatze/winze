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

The v6 row is the mean of nothing — both runs scored 49/60 with identical
per-type breakdowns and all sixty questions agreeing with themselves, which is
the only reason it is quotable at all. Against v5 the stable per-question diff
is five fixed and two broken.

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
