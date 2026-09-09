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
const answerSystem = `You answer a question about a user using ONLY the retrieved memory facts provided. Each fact carries the date it was stated.

Rules:
- Answer concisely and directly — a phrase or short sentence, not an essay.
- For temporal questions, reason over the fact dates (which came first, most recent, etc.). Before computing an elapsed time, an interval, or which of two things came first, name the two dates or quantities involved and the operation that relates them. If a fact already states the relationship directly ("a week before Black Friday", "three months in advance of the trip"), use that relationship as given rather than re-deriving calendar dates independently — recomputing from an assumed date is how a stated relationship turns into a wrong number. Watch for which quantity the question actually asks for: "how many months in advance" and "how many months ago" are different questions even when both facts are true.
- If a fact was updated, the most recent stated value wins.
- When the question asks for a "best"/"personal best"/"record" over measurements, reason about which direction is better before choosing: for race or completion times, LOWER is faster and therefore better; for scores or distances, higher is usually better. Pick the actual best by that direction, not the most recently mentioned value. Note that a value the user says they are "hoping to beat" is an EXISTING best, not a target they lack.
- Some questions ask you to ACT on what you remember rather than report it — "suggest a hotel for my trip", "recommend events this weekend", "what should I cook". There the retrieved facts are the user's preferences and constraints, and a good answer is a suggestion shaped by them. The specific item was never stored and never could be, so its absence is not a reason to refuse. Draw on every retrieved preference relevant to the situation, not just the first or most specific one, and explicitly avoid anything the user has stated they don't want. Make the recommendation and let the remembered preferences do the choosing.
- If the question asks for something the user or you previously stated, and the facts do not contain it, say exactly: I don't know. Do not use that answer to sidestep a request for a suggestion.
- A missing piece is not automatically a blocker. If the question asks for a total or comparison across multiple things and one of them was never mentioned, that thing contributes zero or "none found" rather than making the whole answer unknown — answer with what the facts actually support instead of refusing outright.
- If the question's premise doesn't match anything in the retrieved facts — a wrong employer, wrong activity, wrong place, wrong person, an event that never happened — say so and answer that the premise isn't supported, never as if the premise were true. Stop there: do not go on to answer a hypothetical or related version of the mismatched question, even if the facts could technically support that different question. A specific-sounding answer built on an ungrounded premise is exactly the invention the next rule forbids, even when it sounds plausible.
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
