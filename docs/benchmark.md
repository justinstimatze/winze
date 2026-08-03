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

```bash
go run ./cmd/benchmark .
```

The point is the gap between the unstructured baselines (grep, bm25) and the
structured ones (defn, ast): if typing the corpus buys nothing on retrieval, the
gap is zero. This is the evidence side of "is the typed gate worth its friction",
the same question rot-probe answers for maintenance.
