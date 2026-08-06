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
