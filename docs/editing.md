# Editing helper (referentially-safe mutation)

```bash
winze-edit rename --from Apophenia --to Pareidolia .            # rewrite every reference
winze-edit rename --from A --to B --dry-run .                    # report sites, write nothing
winze-edit merge  --from DupEntity --into CanonEntity .          # fold A into B
winze-edit merge  --from A --into B --dry-run .                  # report the fold, write nothing
```

`cmd/add` appends; `cmd/edit` mutates. A KB you can only append to is one you
cannot maintain — when rot-probe reports two entities that are probably the
same thing, or a framing gets refined and its claims should retarget, there
has to be a tool to act on the finding.

## Rename

Rename works on byte offsets the parser identifies, not text substitution.
On this corpus `Apophenia` has **119 textual occurrences but 7 real
identifier references** — the rest are Briefs, Quotes, comments, and the
longer identifier `ApopheniaClinicalFraming`. A `sed` would corrupt all 112.
That gap is the whole argument for Go-shaped knowledge: the parser knows
which occurrences are the symbol.

Every mutation runs the same gate as `cmd/add` (gofmt, `go build`, `go vet`)
and reverts **all** touched files if any step fails. gofmt is applied only to
files the mutation touched — a mutation tool must not have a blast radius
wider than its mutation.

## Concurrent-write safety

Every write path — `cmd/add`, `cmd/add --batch`, `cmd/edit rename`/`merge` —
takes a corpus-wide advisory `flock(2)` on `.winze.lock` (`internal/corpuslock`)
around its whole read→gate→commit section, so multiple sessions sharing one
worktree serialize instead of racing. Without it the shared `go build .` gate
lets concurrent writers lose updates, clobber each other's reverts, and — worst
for an agent author — false-revert a valid change after tripping over another
session's half-written file. The lock is per-fd, so a crashed holder is released
by the kernel (no stale-lock reaping); uncontended cost is one syscall, so the
single-writer path is unaffected. See `docs/multi-session-write-shape.md`.

## Merge

`merge` folds entity A into entity B: every reference to A is retargeted to
B, A's declaration is removed (its whole `var (…)` group when A is the only
member, else just A's spec), and A's claims retarget automatically because
they reference the var. B is the canonical survivor — A's Brief/ID/Name are
dropped; claim-level provenance is preserved for free (each claim keeps its
own `Prov`, only its Subject/Object identifiers move). The **build gate is
the semantic check**: fold two entities of incompatible type and the
retargeted claims fail to type-check, so the merge reverts. This is the
compaction primitive for the log-structured multi-session KB
(`docs/multi-session-write-shape.md`): rot-probe finds duplicates coined
across session files, merge folds them into the canonical topic file.

Merge records itself as a typed `AbsorbedAlternate` claim appended to the
survivor's file (maps to PROV-O `alternateOf`) — so the fold is auditable
and queryable, not just a git diff. It is a UnaryClaim on the survivor
(`Subject: B.Entity`), not a binary `MergedFrom`, because merge deletes A's
declaration — there is no var left to reference. A's consumed identity (old
var name, ID, Name) is captured in the claim's `Provenance.Quote`. Suppress
with `--no-record`. The claim is visible via `query --provenance "winze-edit
merge"` and `query --claims <survivor>`.

Not yet implemented: retarget (bulk Object rewrite), safe delete.
