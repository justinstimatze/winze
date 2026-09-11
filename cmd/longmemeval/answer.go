package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// answerSystem is the answerer's standing instruction — identical every call,
// so it's cache_control'd. The model answers only from the retrieved facts.
//
// Extended 2026-09-08 after reading all 53 failures on the v11 full-500 run:
// 19 of them had the right facts retrieved and still answered wrong, split
// into four repeatable shapes, none of them an extraction problem. `a96c20ee_abs`
// answered "Harvard University" for a poster presentation no retrieved fact
// mentions — a direct violation of the existing "do not invent" rule, just not
// one this prompt had named for the premise-mismatch case specifically.
// `982b5123` conflated "booked three months in advance" with "how many months
// ago" — two different relationships between the same two dates. `7024f17c`
// refused to total jogging+yoga hours because yoga was never logged, when
// "never logged" means zero, not unknown. `54026fce` drew on one stored
// preference and ignored the rest of what was available. Four new rules below,
// each answering one of those shapes directly.
//
// Measured on the 16 named failures this extension targeted: 7 flipped to
// correct, including all 3 pure date-arithmetic cases. One new failure shape
// surfaced in the same run: `gpt4_93159ced_abs` correctly named the premise
// mismatch ("your employer is NovaTech, not Google") and then kept going and
// answered a hypothetical anyway, contradicting the abstention it had just
// made. The premise-mismatch rule said to name the mismatch; it never said to
// stop there. Fixed by adding that explicitly. Separately, `a96c20ee_abs`
// (Harvard) still hallucinates despite the rule naming this exact shape —
// left as open, not papered over: some instances of this pattern may not be
// closable by prompt instruction alone.
//
// Full-500 confirmed this batch at 450/500 vs. raw control's 455/500, gap -5.
// Per-type: knowledge-update 74/78, multi-session 114/133, single-session-
// assistant 50/56, single-session-preference 28/30, single-session-user
// 66/70, temporal-reasoning 118/133. This is the reference point every later
// attempt in this file gets measured against.
//
// Two more rules were tried and reverted the same night (2026-09-08), never
// shipped past this file. First: scoping "the most recent value wins" to
// require the two facts describe the same specific context, aimed at
// `a4996e51` (45 vs gold 50 — the rule had fired across an unrelated "some
// weeks go up to 45" aside and a specific "peak season +10 hours" fact as if
// one updated the other). Second: an explicit rule against substituting
// general/outside knowledge for a missing specific number, aimed at
// `09ba9854`/`09ba9854_abs` (fabricating plausible Narita airport transit
// fares instead of saying I don't know).
//
// Isolated from every other change in flight that night (v11's lens, warm
// cache, byte-identical facts — see lensVersion's changelog for the full
// story of the OTHER thing tried and reverted alongside this), the two new
// rules alone took the full 500 from 450 to 441. Every type went flat or
// down — knowledge-update 74->71, multi-session 114->113, preference 28->25,
// temporal 118->116, assistant and single-user flat — and none improved.
// 19 questions that were correct under the rules above flipped wrong, 10 that
// were wrong flipped correct, for a net -9. A sample of the 19 regressions
// does not show one clean mechanism the way the k=120 dilution story does for
// the lens side: some look like straightforward judge/sampling variance on
// answers that read as equally correct in substance (`38146c39`'s reworded-
// but-equivalent turbinado-sugar suggestion flipped to wrong for no visible
// content reason), and at least one (`1c0ddc50`) got WORSE in a way that
// traces to the answer restating retrieved true-crime/self-improvement
// preference facts as generic commute options — exactly what the gold answer
// says the user does NOT want — despite no rule change touching that
// question's logic. Two added rules, both individually reasonable, netted a
// real loss with no clean per-rule attribution; not chasing which of the two
// carried more of the loss, since neither survives on its own merits.
//
// Reverted both; the rule body below is byte-identical to the 450/500
// version. The lesson: a rule that reads as an obvious, narrow improvement
// (and even flips its own named failures under a targeted `--only` check)
// is not evidence it helps — the full 500 is the only number that has ever
// told the truth in this file, and twice in one night a plausible-sounding
// fix looked good narrow and lost on the full set for reasons that only
// showed up once every other question got a chance to be affected too.
//
// 2026-09-09: read all 49 remaining full-500 failures directly against real
// question/gold/answer content (v14 baseline, 451/500). One mechanism
// dominates by frequency, dwarfing everything else found: undercounting on
// "how many X" questions. At least 8 clean instances across multi-session
// and knowledge-update -- `gpt4_ab202e7f` found 3 kitchen items against a
// gold of 5, `gpt4_7fce9456` 3 properties against 4, `gpt4_15e38248` 3
// furniture pieces against 4, `a08a253f` 3 fitness days against 4,
// `0a995998` 2 clothing items against 3, `45dc21b6` 2 recipes against 3,
// `e3038f8c` summed to 100 against a gold of 99 by including an item whose
// own count was never stated, `681a1674` overcounted 4 Marvel rewatches
// against a gold of 2. Most of these have facts == retrieved -- nothing was
// cut by k=120 -- so this is not the extraction-dilution story chased
// elsewhere in this file; the answerer has the full retrieved list and
// still doesn't work through all of it. New rule below.
//
// Separately, three EXISTING rules each have current, live violations
// despite already saying the right thing in prose: the premise-mismatch
// rule's "stop there" is violated by `2133c1b5_abs` (names Harajuku-not-
// Shinjuku correctly, then answers the Shinjuku duration anyway) and by
// `a96c20ee_abs` (the same Harvard hallucination the 2026-09-08 changelog
// entry above already flagged as unresolved -- still unresolved). The
// most-recent-value rule is violated by `852ce960` (picked a stale $350k
// pre-approval over an explicitly updated $400k). Rewrote both as an
// explicit two-step shape (name every candidate, THEN choose) rather than a
// single sentence of prose, on the theory that already proved out for
// enumerations in `lensSystem`'s own history: v4 didn't land "never
// collapse an enumeration" by restating it more emphatically, it landed by
// turning the instruction into a literal output-shape requirement. Whether
// prose-to-shape carries over from extraction to answering is the open
// question this change is testing, not an assumption.
//
// 2026-09-10: read all 44 remaining failures from the semantic+rerank-fused
// full run (cmd/longmemeval's haystack SOTA-comparison track, a separate
// measurement axis from the oracle-set full-500 numbers elsewhere in this
// file -- see ROADMAP.md). Two clean, narrow answerer-side mechanisms:
// `gpt4_e072b769` computes "20 days apart... approximately 2 weeks (and 6
// days)" against a gold of "3 weeks ago" -- right arithmetic, wrong rounding
// convention (floor instead of nearest). `gpt4_e414231f` cites a fact dated
// "Wednesday, March 15th" as the answer to "the past weekend" -- prints its
// own contradiction and doesn't act on it.
//
// First draft appended the rounding sentence inline into the existing
// elapsed-time paragraph and added the day/timeframe check as its own new
// bullet. A narrow --only test (18 qids) flipped both named targets exactly
// as predicted, but flagged two apparent regressions. Isolated both before
// trusting either: `gpt4_59149c78` turned out wrong under the OLD rules too
// on two repeats -- never a real regression, just an unstable question (the
// one "correct" reading earlier was a fluke). `gpt4_2f56ae70` was real:
// reliably correct under the old rules (Disney+ "last month" beats Apple
// TV+ "a few months"), reliably wrong under the new ones (picks Apple TV+)
// across three repeats -- and it isn't even the shape either new rule's
// content addresses; there's no exact elapsed-time count to round and no
// weekday/timeframe anchor to verify. Moving the rounding sentence out of
// the crowded elapsed-time paragraph into its own separate bullet (content
// unchanged, only its position) fixed it: two repeats post-move both landed
// `gpt4_e072b769` correct, `gpt4_e414231f` correct, `gpt4_2f56ae70` correct,
// `gpt4_59149c78` still wrong (expected, unrelated).
//
// Full 500 confirmed the promoted stack (-semantic -rerank, both now
// default, plus both rules above) at 444/500 vs. the semantic-only 441/500
// baseline: 13 wins, 10 losses, net +3. Diffed every flip. Multi-session
// alone went -4 (2 wins, 6 losses) despite gaining nothing from either new
// rule -- of the 7 temporal-reasoning wins, only `71017277` and
// `gpt4_e414231f` looked attributable to the day/timeframe rule at first
// read; `gpt4_e072b769` traces to the rounding rule, `982b5123` to the
// most-recent-value two-step rewrite, `6e984302`/`gpt4_483dd43c` to
// retrieval finding facts semantic-only had missed outright. Isolated the
// day/timeframe rule directly: removed it alone (rounding, most-recent-
// value, and placement fix all left untouched) and reran the 5 multi-
// session/knowledge-update losses plus the 2 apparently-attributable wins,
// with the other 3 rules' own wins as controls (all 3 controls held).
// `9aaed6a3` (SaveMart cashback) and `0977f2af` (kitchen gadget) both
// looked fixed on the first pass, and `gpt4_e414231f` flipped back to
// wrong as expected if it really depends on the rule -- but `71017277`
// stayed correct even with the rule gone, meaning it was never actually
// rule-dependent. Two repeats on the three qids that moved settled it:
// `9aaed6a3` held fixed 3/3 and `gpt4_e414231f` held broken 3/3 (both
// clean, reproducible, and reverting the rule is an even trade between
// them), but `0977f2af` split 2 correct / 1 wrong regardless of which
// rule text it saw -- an unstable question, not a rule effect, the same
// shape `gpt4_59149c78` turned out to be earlier in this same investigation.
// Net: reverting the rule is a wash on its own clean evidence, not a win,
// so it stays. The other 3 multi-session losses (`7024f17c`, `88432d0a`,
// `6e984301`) stayed wrong with the rule removed too -- they were never
// this rule's fault. Two of them show a fact appearing in the -rerank
// candidate pool that wasn't visible under semantic-only retrieval and
// wasn't there to help (`6e984301` cites a fabricated-sounding "6 weeks"
// intermediate that no BEFORE answer used), which points at -rerank's
// LLM-judged reordering surfacing a different, sometimes-wrong fact for
// multi-session questions specifically -- not an answerSystem problem, and
// not yet isolated. That, plus the other 3 non-rule multi-session losses
// from the same diff (`a56e767c`, `28dc39ac`, `92a0aa75`, none carrying
// day/weekday language at all), is the next thread on this track: whether
// -rerank's candidate reordering is systematically worse for multi-session's
// longer, higher-fact-count sessions than the semantic-only order it
// replaces.
//
// 2026-09-10 (later same night): read all 56 failures still wrong under the
// full default stack -- not just the 23 that flipped against the semantic-
// only baseline, the full remaining population. Counting/enumeration is by
// far the largest mechanism, 20 of 56 (36%), both directions: undercounting
// (`gpt4_ab202e7f` 3 kitchen items vs gold 5, `a9f6b44c` 1 bike vs 2,
// `gpt4_731e37d7` $220 vs $720) and overcounting (`gpt4_2f8be40d` 6
// weddings vs 3, `d851d5ba` $8,750 vs $3,750, `681a1674` 4 Marvel rewatches
// vs 2). The existing counting rule already tells the model to scan every
// fact and name uncertainty rather than silently resolve it -- and several
// answers show it doing exactly that in prose and then landing on the wrong
// number anyway: `6d550036` writes "if included, the total would be 4" and
// answers 3; `gpt4_a56e767c` writes "could bring the count to 4" and
// answers 3. The uncertainty is being named as a hedge AFTER the number is
// already picked, not resolved before it -- the same prose-vs-structural-
// shape gap this file's own 2026-09-09 entry already fixed once for
// premise-mismatch and most-recent-value. Rewrote the counting rule the
// same way: require a per-candidate INCLUDE/EXCLUDE line with a one-clause
// reason before the final number, so a borderline case gets decided in the
// list instead of leaking into a trailing hedge.
//
// Measured narrow (--only) against all 20 target qids above plus 6
// currently-correct multi-session counting questions as regression
// controls (gpt4_59c863d7, b5ef892d, e831120c, 3a704032, gpt4_d84a3211,
// aae3761f -- none touching any of the 20 target qids' content). Result:
// 4 of 20 targets flipped correct (`c4a1ceb8` citrus dedup, `gpt4_a56e767c`
// festivals, `gpt4_15e38248` furniture, `a9f6b44c` bikes), all 6 controls
// held, zero regressions observed. Real and clean, but the fix rate (20%)
// says the mechanism splits in two and this rule only reaches one half:
// the 4 that flipped all had the missing/extra item already sitting in the
// retrieved facts -- the failure was purely a resolution call the rule
// now forces correctly. Of the 16 that didn't move, several
// (`46a3abf7`, `28dc39ac`, `gpt4_ab202e7f`) don't even list the missing
// item as a candidate considered and excluded -- it never reached the
// list at all, which this rule structurally cannot fix, since it only
// governs how to decide among candidates the model already has. The other
// ~16/20 (80% of this population) is very likely a retrieval-completeness
// gap, not an answering gap -- worth returning to once the -rerank/multi-
// session retrieval thread below is further along, since a retrieval fix
// there could shrink this same residual for free. Shipped anyway: real,
// controls-clean, no reason to hold it while the retrieval half is
// investigated separately.
const answerSystem = `You answer a question about a user using ONLY the retrieved memory facts provided. Each fact carries the date it was stated.

Rules:
- Answer concisely and directly — a phrase or short sentence, not an essay. The one exception is the counting rule below, whose candidate list is required working, not prose to trim.
- For temporal questions, reason over the fact dates (which came first, most recent, etc.). Before computing an elapsed time, an interval, or which of two things came first, name the two dates or quantities involved and the operation that relates them. If a fact already states the relationship directly ("a week before Black Friday", "three months in advance of the trip"), use that relationship as given rather than re-deriving calendar dates independently — recomputing from an assumed date is how a stated relationship turns into a wrong number. Watch for which quantity the question actually asks for: "how many months in advance" and "how many months ago" are different questions even when both facts are true.
- When computing an elapsed amount of time that doesn't land on an exact number of the unit the question asks for, round to the nearest whole unit rather than truncating down — 20 days since something happened is 3 weeks ago, not 2 weeks (and some days).
- Before citing a fact as the answer to a question anchored to a specific day, weekday, or timeframe ("last Saturday", "the past weekend", "on Tuesday", "in March"), check that the fact's own date actually falls within that window. A fact from a different day or outside the stated range is not a match no matter how topically relevant it otherwise looks — set it aside for one that does fall in the window, or answer "I don't know" if none does.
- If more than one fact could answer the same question with a different value, first name every candidate value together with the date it was stated, then answer with the one carrying the latest date — never the one that happens to appear first, last, or largest in the retrieved list. A specific, dated update always overrides an earlier general or approximate mention, even when the general one reads as more prominent or is repeated more often.
- When the question asks for a "best"/"personal best"/"record" over measurements, reason about which direction is better before choosing: for race or completion times, LOWER is faster and therefore better; for scores or distances, higher is usually better. Pick the actual best by that direction, not the most recently mentioned value. Note that a value the user says they are "hoping to beat" is an EXISTING best, not a target they lack.
- When asked to count or total instances of something ("how many X", "list all the Y I did", "how much did I spend on X in total"), first list every candidate fact that could plausibly match, each on its own line marked INCLUDE or EXCLUDE with a one-clause reason, before stating a final number — scan every single retrieved fact for a candidate before starting this list, since missing one that sits later in the retrieved list is the most common way this goes wrong. Decide every borderline case explicitly in that list: a case you are unsure about still gets marked INCLUDE or EXCLUDE there, never left ambiguous for a hedge sentence after the number. The final answer is the count (or sum) of INCLUDE lines only — do not adjust it up or down afterward based on a case you didn't resolve in the list.
- Some questions ask you to ACT on what you remember rather than report it — "suggest a hotel for my trip", "recommend events this weekend", "what should I cook". There the retrieved facts are the user's preferences and constraints, and a good answer is a suggestion shaped by them. The specific item was never stored and never could be, so its absence is not a reason to refuse. Draw on every retrieved preference relevant to the situation, not just the first or most specific one, and explicitly avoid anything the user has stated they don't want. Lead the answer with what actually builds on the user's own specific stated facts — a generic tip that would apply to anyone is not what these questions are asking for, and belongs after the personalized content, if at all, not before it.
- If the question asks for something the user or you previously stated, and the facts do not contain it, say exactly: I don't know. Do not use that answer to sidestep a request for a suggestion.
- A missing piece is not automatically a blocker. If the question asks for a total or comparison across multiple things and one of them was never mentioned, that thing contributes zero or "none found" rather than making the whole answer unknown — answer with what the facts actually support instead of refusing outright.
- If the question's premise doesn't match anything in the retrieved facts — a wrong employer, wrong activity, wrong place, wrong person, an event that never happened — your answer is the mismatch statement and NOTHING ELSE. The sentence naming the mismatch is the entire response: no second sentence, no answering a hypothetical or nearby version of the question, no supplying the correct information as a courtesy. Any sentence after the one naming the mismatch is exactly the invention the next rule forbids, however accurate or plausible it sounds.
- Do not invent facts beyond what is retrieved. Applying a stated preference to a new situation is not inventing a fact; asserting a preference nobody stated is.`

// answer asks the reasoner to answer the question from the retrieved facts.
func (r *runner) answer(question, questionDate string, facts []Fact) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Question (asked on %s): %s\n\n", questionDate, question)
	b.WriteString("Retrieved memory facts:\n")
	if len(facts) == 0 {
		b.WriteString("(none)\n")
	}
	for _, f := range facts {
		fmt.Fprintf(&b, "- [%s] %s = %s (%s) — %q\n", f.Date, f.Attribute, f.Value, f.Kind, f.Quote)
	}

	resp, err := r.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:       anthropic.ModelClaudeSonnet4_6,
		MaxTokens:   512,
		Temperature: anthropic.Float(0),
		System: []anthropic.TextBlockParam{{
			Text:         answerSystem,
			CacheControl: anthropic.CacheControlEphemeralParam{TTL: anthropic.CacheControlEphemeralTTLTTL1h},
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(b.String())),
		},
	})
	if err != nil {
		return "", fmt.Errorf("answer API error: %w", err)
	}
	r.stats.record(string(anthropic.ModelClaudeSonnet4_6), resp.Usage.InputTokens, resp.Usage.CacheReadInputTokens, resp.Usage.OutputTokens)

	for _, block := range resp.Content {
		if block.Type == "text" {
			return strings.TrimSpace(block.Text), nil
		}
	}
	return "", fmt.Errorf("no text in answer response")
}
