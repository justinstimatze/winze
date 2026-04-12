package winze

// Thirteenth public-corpus ingest, fourth session-5 slice: the
// Universal Declaration of Human Rights, adopted by UN General
// Assembly resolution 217 A (III) in Paris on 10 December 1948 by
// a vote of 48 in favour, 0 against, 8 abstentions, and 2 not
// voting out of 58 member states. The slice ingests the UDHR
// directly (un.org/en/about-us/universal-declaration-of-human-rights)
// rather than via a Wikipedia article, because the document is its
// own canonical source and winze's mirror-source-commitments
// discipline prefers primary material when it is available and
// short enough to handle.
//
// Motivation: **genuinely distant corpus shape stress test.** Every
// prior winze ingest has been encyclopedic (Wikipedia articles),
// scientific (peer-reviewed papers, one review, one commentary),
// or taxonomic (lists of biases, misconceptions, human universals).
// The UDHR is distinct from all three on two axes at once:
//
//   1. It is a **normative legal document** — every substantive
//      article has the form "everyone has the right to X" or "no
//      one shall be subjected to Y", which is a *speech act* of
//      declaration rather than a claim *about* the world. Every
//      prior winze Hypothesis has been of the form "X is true of
//      the world"; UDHR articles are of the form "X should be the
//      case for all humans." The schema question: does winze's
//      predicate vocabulary survive the descriptive-vs-normative
//      distinction, or does one of the article-as-Hypothesis
//      claims force a new primitive to carry the normative weight?
//
//   2. It is **institutionally authored** — the Commission on Human
//      Rights drafted it as a committee, the General Assembly
//      adopted it by vote, and attributing the document to any
//      single drafter (Eleanor Roosevelt chaired; John Humphrey
//      produced the first draft; René Cassin gave it its structure;
//      P.C. Chang did the Confucian-compromise philosophical work;
//      Charles Malik was the rapporteur; Hansa Mehta changed "men"
//      to "human beings" in Article 1) erases the institutional
//      character of the act. This is the schema-forcing case the
//      Authored docstring has anticipated since session 4: the
//      first document in winze whose honest authorship claim is
//      per-organization rather than per-person.
//
// Schema forcing functions earned by this slice:
//
//   - **AuthoredOrg BinaryRelation[Organization, Concept]**. Second
//     schema-forcing slice in session 5 and the first institutional-
//     authorship case in winze. Parallel shape to Proposes/ProposesOrg
//     and MonitoredBy/MonitoredByOrg — the same Go-lacks-sum-types
//     work-around that earned ProposesOrg for the "2020 Russian team"
//     claim in tunguska.go and MonitoredByOrg for institutional
//     environmental sampling in bootstrap.go. Third time that exact
//     split-for-agent-shape pattern pays off, which is enough
//     occurrences to pattern-match on: **when a claim's Subject
//     slot wants to span Person and Organization, split the
//     predicate rather than widening the role**. The Agent-
//     interface promotion the Authored docstring mentioned as a
//     future refactor is still available, but at three occurrences
//     the Go-split discipline is working well enough that the
//     refactor is not yet forced.
//
//   - **Zero other new primitives.** Declined to earn
//     `IsNormativeClaim UnaryClaim[Hypothesis]` as a tag
//     distinguishing declarative/normative claims from
//     descriptive/factual hypotheses. Tempting because every
//     Hypothesis in winze pre-this-slice is descriptive and the
//     three UDHR articles wired here are normative, but no claim
//     in the slice structurally requires the distinction — the
//     same ProposesOrg predicate carries both "a Russian team
//     proposed the Tunguska comet hypothesis" (descriptive) and
//     "the UN General Assembly proposed Article 3" (normative),
//     and the difference lives in the Brief text rather than in
//     the claim graph. Earmarked as available for a future slice
//     that lands a query actually distinguishing the two types.
//     Same deferral reasoning as `IsScientificPaper` in Mattson
//     and `ProposedIn` in the commentary slice: a tag that would
//     be convenient but is not strictly forced stays deferred.
//
//   - **Zero other schema work.** No new roles, no new pragmas,
//     no new value structs. The slice reuses AuthoredOrg (new),
//     Authored (not used — institutional instead of personal),
//     ProposesOrg, TheoryOf, and AffiliatedWith — all existing
//     predicates — to wire 11 claims across 13 new entities.
//
// Schema-convergence status after this slice:
//
//   - Session 5 has now earned two new predicates on two different
//     corpus-shape boundaries: CommentaryOn (peer-commentary shape,
//     white_shergill_commentary.go) and AuthoredOrg (institutional-
//     authorship shape, this slice). Neither came from Wikipedia.
//     Both are cleanly earned forcing functions that the previous
//     vocabulary could not represent without concept-conflation.
//
//   - The three intervening vocabulary-fit slices (Mattson review,
//     human universals list, and whatever comes after this if it
//     does not earn new schema) bracket the accretion in both
//     directions. The convergence hypothesis has been pressure-
//     tested on five distinct source shapes — Wikipedia
//     encyclopedic, peer commentary, peer review article, course-
//     handout list, normative legal document — and the schema has
//     accreted twice and stayed stable three times. Accretion
//     lands at real forcing boundaries; stability holds when
//     content fits. The convergence claim is robust enough to
//     drop from "open question" to "calibrated finding with a
//     known accretion-rate of roughly 1-in-3 slices, lower as
//     source neighbourhood narrows."
//
// Cross-ingest bridges wired or deferred by this slice:
//
//   - **Brief-level only bridge: Brown's human universal "law
//     (rights and obligations)".** Verbatim on Brown's list
//     (confirmed during the human_universals.go slice recon),
//     directly adjacent to UDHR's subject matter, and obviously
//     relevant. Would wire cleanly as BelongsTo(UDHR1948,
//     LawAndRightsUniversal) if LawAndRightsUniversal were
//     already in winze — but it isn't. Two obstacles:
//
//     1. LawAndRightsUniversal was not part of the six universals
//        selected by human_universals.go (Language, Music,
//        Marriage, FearOfDeath, Mythology, ToolMaking); a
//        backfill is possible but is conceptually better handled
//        by the future slice that extends the human-universals
//        coverage rather than shoe-horned in here.
//
//     2. The semantic relationship is abstraction-level-mismatched.
//        UDHR is a specific document in a specific historical
//        context; Brown's "law (rights and obligations)" is a
//        universal-category claim about human societies. The
//        honest shape is something like
//        `Instantiates[Concept, Concept]` or
//        `Exemplifies[Concept, Concept]`, which no slice
//        currently forces. Under PrefersOrganicSchemaGrowth the
//        discipline is to leave the bridge Brief-level until a
//        second instance-of-universal case lands that would force
//        the primitive.
//
//     The deferred bridge is the second session-5 slice to surface
//     a honest-but-unforced bridge opportunity (joining
//     human_universals.go's "future, attempts to predict" /
//     "classification" / "figurative speech" adjacency cluster).
//     The pattern is becoming load-bearing: session 5 slices keep
//     finding structural adjacencies that cannot be wired without
//     premature schema, and the discipline keeps correctly
//     deferring them. The discipline's value is being validated
//     by its refusal to act, not by its contributions.
//
//   - **No direct cross-file entity bridges.** UDHR is topically
//     distant from every prior winze ingest. The nearest adjacencies
//     are Brown's "law" item (Brief-level, above), Sagan's
//     DemonHauntedWorld's skeptical framing of supernatural belief
//     (Brief-level, because UDHR Article 18's freedom-of-religion
//     claim is a legal protection for belief rather than a
//     philosophical stance about it), and the nondualism.go
//     slice's polyvalent-term treatment (Brief-level, because
//     "right" in UDHR is univocal in its legal sense rather than
//     polyvalent across traditions). Three Brief-level adjacencies,
//     zero claim-level bridges, zero fabrications — the session-5
//     discipline continues to hold.
//
//   - **HumanRights is introduced as a contested-target-ready
//     Concept.** Parallel shape to HumanCognition in
//     mattson_pattern_processing.go: UDHR1948 is wired as a
//     TheoryOf(HumanRights) claim, and HumanRights will attract
//     future rivals — a natural-law theory of human rights, a
//     social-contract theory, a capability-theory framing (Sen,
//     Nussbaum), the libertarian-negative-rights framing, and so
//     on. The contested-concept rule does not fire on HumanRights
//     from this slice alone (UDHR1948's is the only TheoryOf
//     claim), but the entity exists to let future slices land
//     rivals zero-touch, same pattern as HumanCognition.
//
// Slice scope and deliberate exclusions:
//
//   - UDHR1948 as document-Concept, UN General Assembly and UN
//     Commission on Human Rights as Organizations, the six
//     canonically-recognised drafters as Persons, HumanRights as
//     the meta-Concept target of the document's TheoryOf claim,
//     and three representative articles (1, 3, 18) as Hypotheses
//     proposed by the General Assembly. Articles 1, 3, and 18
//     were selected as a cross-section: Article 1 for its
//     foundational "born free and equal in dignity and rights"
//     formulation (and its historical load-bearing via Hansa
//     Mehta's "men" → "human beings" edit), Article 3 for the
//     most recognisable single-sentence rights statement, and
//     Article 18 for its explicitly cognitive / belief-adjacent
//     content, which preserves a future-bridge surface to
//     demon_haunted.go's Scientific Skepticism and nondualism.go's
//     religious-traditions content if later ingests warrant it.
//
//   - **Not wired: the per-article TheoryOf(HumanRights) false-
//     positive pattern.** The three articles COULD be wired as
//     TheoryOf(HumanRights), which would fire the contested-
//     concept rule on HumanRights as the sixth contested target.
//     This slice deliberately does not do that, because the three
//     articles are not rival theories of human rights — they are
//     complementary components of a single framework, co-signed by
//     the same adopting body. Firing the contested-concept rule on
//     co-signatories would be a false positive and would walk back
//     the rule's informational value for all prior uses. The
//     distinction between "plurality of co-signed components" and
//     "plurality of rival theories" is not currently machine-
//     distinguishable from the claim graph alone (both shapes land
//     as multiple Hypothesis subjects pointing at the same
//     Concept), and the honest move for this slice is to represent
//     the article-level Hypotheses as ProposesOrg-only Hypotheses
//     that do not themselves TheoryOf anything, and to put the
//     document-level TheoryOf(HumanRights) claim on UDHR1948
//     (which IS the theory-of-human-rights artefact) rather than
//     on its component norms. A future lint-rule refinement that
//     distinguishes co-signed plurality from rival plurality
//     would be a //winze:co-signed variant of the contested
//     pragma, but no slice currently forces it — this slice just
//     names the edge case so the first slice that forces it has
//     prior art to cite.
//
//   - **Not reified: the preamble, or the individual abstaining
//     countries, or the 27 articles not selected.** The preamble
//     is narrative framing that does not advance claim-level
//     content beyond "what is about to follow is proclaimed as a
//     common standard of achievement"; a Brief-level summary on
//     UDHR1948 suffices. The 8 abstaining countries at the 1948
//     vote (Saudi Arabia, South Africa, and the six Soviet-bloc
//     members) are a load-bearing historical finding and would be
//     worth wiring if a second slice on the adoption politics
//     landed, but this slice is focused on the document itself
//     rather than the vote; reifying the abstainers without a
//     source that details their objections would fabricate
//     structure the un.org page refuses to provide. The 27
//     unselected articles can be accreted by future slices with
//     Concept/Hypothesis + ProposesOrg pairs, same incremental-
//     accretion pattern as human_universals.go.
//
//   - **Not reified: the "adopts" vs "proposes" speech-act
//     distinction.** ProposesOrg is used for the General Assembly's
//     adoption of the three articles with the understanding that
//     "propose" here means "declare and adopt as collectively
//     binding." A hypothetical `AdoptsOrg[Organization, Hypothesis]`
//     predicate would be the honest refinement, but adding it for
//     three claims in one slice is exactly the premature-schema
//     trap the discipline exists to prevent. The Brief on each
//     article-Hypothesis flags the linguistic stretch so a future
//     reader can find it; promotion is available when a second
//     document-adoption case lands that would force the split.

var udhrSource = Provenance{
	Origin:     "UN / un.org/en/about-us/universal-declaration-of-human-rights — Universal Declaration of Human Rights, adopted by UN General Assembly resolution 217 A (III), Paris, 10 December 1948, 48-0-8 with 2 not voting (58 UN member states at the time). Drafting history via un.org and research.un.org/en/undhr/draftingcommittee; voting record verified against Wikipedia / Universal_Declaration_of_Human_Rights",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 5 (UDHR ingest, normative-legal-document corpus shape, AuthoredOrg schema-forcing slice)",
	Quote:      "Whereas recognition of the inherent dignity and of the equal and inalienable rights of all members of the human family is the foundation of freedom, justice and peace in the world, [...] Proclaims this Universal Declaration of Human Rights as a common standard of achievement for all peoples and all nations [...]. Article 1: All human beings are born free and equal in dignity and rights. They are endowed with reason and conscience and should act towards one another in a spirit of brotherhood. Article 3: Everyone has the right to life, liberty and security of person. Article 18: Everyone has the right to freedom of thought, conscience and religion; this right includes freedom to change his religion or belief, and freedom, either alone or in community with others and in public or private, to manifest his religion or belief in teaching, practice, worship and observance.",
}

// -----------------------------------------------------------------------------
// The document, the organizations, the drafters, and the meta-concept.
// -----------------------------------------------------------------------------

var (
	UDHR1948 = Concept{&Entity{
		ID:      "concept-udhr-1948",
		Name:    "Universal Declaration of Human Rights (1948)",
		Kind:    "concept",
		Aliases: []string{"UDHR", "Universal Declaration of Human Rights", "UN General Assembly Resolution 217 A (III)"},
		Brief:   "The Universal Declaration of Human Rights, a 30-article document proclaimed by the UN General Assembly in Paris on 10 December 1948 as a 'common standard of achievement for all peoples and all nations'. Drafted between 1946 and 1948 by the UN Commission on Human Rights, with Eleanor Roosevelt as chair, John Humphrey producing the first draft, René Cassin restructuring it, P.C. Chang resolving philosophical deadlocks via Confucian-tradition compromises, Charles Malik serving as rapporteur, and Hansa Mehta successfully changing 'all men' to 'all human beings' in Article 1. Adopted 48-0-8 with 2 not voting out of the 58 UN member states at the time — the abstaining bloc was Saudi Arabia, South Africa, and the six Soviet-bloc members, whose objections centred on freedom of religion (Saudi Arabia), apartheid (South Africa), and the economic/social-rights vs civil/political-rights balance (Soviet bloc). Not a binding legal treaty itself but treated subsequently as the foundational text from which modern international human rights law derives, including the two 1966 covenants (ICCPR and ICESCR) that were negotiated specifically to give binding force to the UDHR's content. This slice reifies the document as a Concept with a TheoryOf(HumanRights) claim plus three representative articles as Hypotheses proposed by the General Assembly; the 27 unselected articles, the preamble, the adoption vote's individual country positions, and the two 1966 covenants are all available for future-slice accretion.",
	}}

	HumanRights = Concept{&Entity{
		ID:      "concept-human-rights",
		Name:    "Human rights",
		Kind:    "concept",
		Aliases: []string{"human-rights framework"},
		Brief:   "The meta-concept of universal, inalienable entitlements held by every human being by virtue of being human. Treated as a standalone Concept in winze so that multiple rival framings (natural-law tradition rooted in Aquinas and Locke; social-contract theories; Amartya Sen and Martha Nussbaum's capability approach; libertarian negative-rights framings; Islamic, Confucian, and Ubuntu-based non-Western accounts) can be wired as future TheoryOf(HumanRights) claims that fire the contested-concept rule on HumanRights as a future contested target. The pattern mirrors HumanCognition in mattson_pattern_processing.go, which was also seeded as a contested-target-ready Concept with a single initial TheoryOf claim waiting for rivals. UDHR1948 is the first TheoryOf claim; the contested-concept rule does not fire on HumanRights from this slice alone.",
	}}

	UNGeneralAssembly = Organization{&Entity{
		ID:      "org-un-general-assembly",
		Name:    "United Nations General Assembly",
		Kind:    "organization",
		Aliases: []string{"UN General Assembly", "UNGA"},
		Brief:   "The main deliberative assembly of the United Nations, composed of all UN member states with each state having one vote. Adopted the UDHR on 10 December 1948 via resolution 217 A (III) by a vote of 48-0-8 with 2 not voting. In winze the UNGeneralAssembly is the Subject of three ProposesOrg claims on Articles 1, 3, and 18 — the 'propose' here is a deliberate stretch covering 'declare and adopt as collectively binding', noted in each article's Brief. A hypothetical AdoptsOrg predicate would be the honest refinement but is deferred under the organic-schema-growth discipline until a second adoption-case forces it.",
	}}

	UNCommissionOnHumanRights = Organization{&Entity{
		ID:      "org-un-commission-on-human-rights",
		Name:    "UN Commission on Human Rights",
		Kind:    "organization",
		Aliases: []string{"Commission on Human Rights", "UNCHR (1946-2006)"},
		Brief:   "The UN body responsible for drafting the UDHR between 1946 and 1948, and later the core international human rights instruments through 2006, when it was replaced by the UN Human Rights Council. In winze the Commission is the Subject of winze's first AuthoredOrg claim — the institutional-authorship primitive earned by this slice. Six of the Commission's eight or nine drafting members are reified as Persons in this slice: Eleanor Roosevelt (chair, US), John Humphrey (first draft, Canada), René Cassin (structure, France), P.C. Chang (philosophical compromise, Republic of China), Charles Malik (rapporteur, Lebanon), and Hansa Mehta (India, successfully changed 'men' to 'human beings' in Article 1). Each is linked to the Commission via an AffiliatedWith claim.",
	}}
)

// -----------------------------------------------------------------------------
// The drafters.
// -----------------------------------------------------------------------------

var (
	EleanorRoosevelt = Person{&Entity{
		ID:    "eleanor-roosevelt",
		Name:  "Eleanor Roosevelt",
		Kind:  "person",
		Brief: "American diplomat, activist, and widow of US President Franklin D. Roosevelt. Chaired the UN Commission on Human Rights from 1946 to 1951 and steered the UDHR drafting process through its three sessions, using her political capital with both superpower blocs to hold the coalition together through the Cold War's early deadlock. Not the primary drafter of the text — Humphrey and Cassin did the initial drafting work — but the person whose diplomatic authority kept the project alive.",
	}}

	JohnHumphrey = Person{&Entity{
		ID:    "john-humphrey",
		Name:  "John Peters Humphrey",
		Kind:  "person",
		Brief: "Canadian jurist and first Director of the UN Secretariat's Division for Human Rights (1946-1966). Produced the first working draft of the UDHR, a 400-page document compiling rights from existing national constitutions and international instruments, which served as the raw material René Cassin subsequently restructured into the final 30-article form. Humphrey's role as institutional first-drafter means he carries load-bearing contribution status despite being less widely known than Roosevelt, Cassin, or Malik.",
	}}

	ReneCassin = Person{&Entity{
		ID:    "rene-cassin",
		Name:  "René Cassin",
		Kind:  "person",
		Brief: "French jurist and Nobel Peace Prize laureate (1968) for his role in drafting the UDHR. Restructured Humphrey's initial 400-page draft into the final 30-article declaration, introducing the portico-based architecture (preamble, general-principles articles, civil-and-political-rights articles, economic-and-social-rights articles, duties and limits articles) that is the declaration's structural signature. Frequently credited as the UDHR's 'principal architect' in popular retellings, though the honest picture is that the architecture was Cassin's while the rights content came from Humphrey's compilation and the deadlock-breaking philosophical work came from Chang and Malik.",
	}}

	PengChunChang = Person{&Entity{
		ID:      "peng-chun-chang",
		Name:    "P.C. Chang",
		Kind:    "person",
		Aliases: []string{"Peng Chun Chang", "Chang Peng-chun"},
		Brief:   "Chinese philosopher, playwright, and diplomat, Vice-Chairman of the UN Commission on Human Rights during the UDHR drafting. Used Confucian ethical concepts — particularly the notion of 'ren' (humaneness, conscience) — to broker philosophical compromises between the natural-law positions advocated by Charles Malik and the more pragmatic framings favoured by Humphrey and Cassin, resolving several drafting stalemates that might otherwise have ended the project. Chang's contributions are load-bearing for the specific philosophical non-Westernness of the final text: the declaration's explicit appeal to 'conscience' in Article 1 and its pluralistic framing of freedom of thought in Article 18 both carry his influence.",
	}}

	CharlesMalik = Person{&Entity{
		ID:    "charles-malik",
		Name:  "Charles Malik",
		Kind:  "person",
		Brief: "Lebanese philosopher, diplomat, and Rapporteur of the UN Commission on Human Rights drafting committee. Trained in natural-law tradition and strongly influenced by both Thomist and personalist philosophical frameworks, Malik's contributions centred on the philosophical basis for human rights as grounded in the inherent dignity of the human person — the framing that became the foundation of the preamble's 'inherent dignity' language. Debated P.C. Chang through multiple drafting sessions over the balance between natural-law universalism and cross-cultural pluralism; the text's final form reflects a negotiated synthesis of both positions.",
	}}

	HansaMehta = Person{&Entity{
		ID:    "hansa-mehta",
		Name:  "Hansa Mehta",
		Kind:  "person",
		Brief: "Indian activist, writer, and member of the UN Commission on Human Rights drafting committee. Successfully argued for changing the draft Article 1 language from 'All men are born free and equal' to 'All human beings are born free and equal' — a specific, attributable edit whose load-bearing status is recognised on the UN's own UDHR history page. The change mattered at the time because several delegations had read 'men' as gender-neutral in their own languages and the edit forced the English text into the more explicitly universal form that has anchored the document's subsequent interpretation. A rare case in winze of a Person entity whose load-bearing contribution is a single specific wording change.",
	}}
)

// -----------------------------------------------------------------------------
// The three selected articles as Hypotheses.
// -----------------------------------------------------------------------------

var (
	UDHRArticle1 = Hypothesis{&Entity{
		ID:    "hyp-udhr-article-1",
		Name:  "All human beings are born free and equal in dignity and rights — endowed with reason and conscience, they should act towards one another in a spirit of brotherhood",
		Kind:  "hypothesis",
		Brief: "UDHR Article 1, verbatim: 'All human beings are born free and equal in dignity and rights. They are endowed with reason and conscience and should act towards one another in a spirit of brotherhood.' The foundational article — the one whose 'men' → 'human beings' edit by Hansa Mehta is UDHR-drafting historiography's most-cited specific contribution. Represented as a Hypothesis rather than a Concept because its structure is assertive (something is claimed to be the case) rather than definitional, even though the 'case' being claimed is normative (should be the case) rather than descriptive (is the case). The normative-vs-descriptive distinction is flagged in the slice header as available for a future IsNormativeClaim tag if a query forces it; no query does yet. The UN General Assembly ProposesOrg this Hypothesis as a collectively adopted norm — 'proposes' is a deliberate stretch covering 'declares and adopts as binding', flagged here for future-slice refinement.",
	}}

	UDHRArticle3 = Hypothesis{&Entity{
		ID:    "hyp-udhr-article-3",
		Name:  "Everyone has the right to life, liberty and security of person",
		Kind:  "hypothesis",
		Brief: "UDHR Article 3, verbatim: 'Everyone has the right to life, liberty and security of person.' The most widely-recognised single-sentence rights statement in the declaration, and the foundation of the civil-and-political-rights articles that follow it (Articles 4-21). Chosen for this slice because its brevity and universal recognition make it a clean representative of the UDHR's canonical form. Same normative-vs-descriptive caveat as Article 1.",
	}}

	UDHRArticle18 = Hypothesis{&Entity{
		ID:    "hyp-udhr-article-18",
		Name:  "Everyone has the right to freedom of thought, conscience and religion — including the freedom to change religion or belief, and to manifest it in teaching, practice, worship, and observance, alone or in community, in public or private",
		Kind:  "hypothesis",
		Brief: "UDHR Article 18, verbatim: 'Everyone has the right to freedom of thought, conscience and religion; this right includes freedom to change his religion or belief, and freedom, either alone or in community with others and in public or private, to manifest his religion or belief in teaching, practice, worship and observance.' Chosen for this slice because of its load-bearing cognitive content — it is the article most directly relevant to winze's existing intellectual neighbourhood around belief, skepticism, and supernatural claims (see demon_haunted.go's Scientific Skepticism material, apophenia.go's clinical-vs-secular framings of apophenia, and nondualism.go's religious-traditions content). The freedom-of-religion framing is the one Saudi Arabia cited as grounds for its 1948 abstention from the UDHR vote, so the article is also the load-bearing case for the adoption history. The Brief-level adjacency to winze's existing skeptical-rationalist neighbourhood is flagged in the slice header but NOT wired as a structural claim, because none of the UDHR sources commit to the philosophical content at the level that would honestly support one.",
	}}
)

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	// The schema-forcing claim: winze's first AuthoredOrg use.
	UNCommissionAuthoredUDHR = AuthoredOrg{
		Subject: UNCommissionOnHumanRights,
		Object:  UDHR1948,
		Prov:    udhrSource,
	}

	// Article-level proposals — the General Assembly declares each.
	// The document-level TheoryOf(HumanRights) claim is routed
	// through a separate Hypothesis entity below, because TheoryOf
	// is BinaryRelation[Hypothesis, Concept] and UDHR1948 is itself
	// a Concept — a direct TheoryOf claim from the Concept would
	// not compile and the slot-type discipline is the whole point
	// of the role wrappers.
	UNGAProposesArticle1 = ProposesOrg{
		Subject: UNGeneralAssembly,
		Object:  UDHRArticle1,
		Prov:    udhrSource,
	}
	UNGAProposesArticle3 = ProposesOrg{
		Subject: UNGeneralAssembly,
		Object:  UDHRArticle3,
		Prov:    udhrSource,
	}
	UNGAProposesArticle18 = ProposesOrg{
		Subject: UNGeneralAssembly,
		Object:  UDHRArticle18,
		Prov:    udhrSource,
	}

	// Drafter affiliations with the Commission. Six AffiliatedWith
	// claims, one per drafter — the institutional context that
	// makes AuthoredOrg semantically correct rather than any
	// per-person Authored attribution.
	RooseveltAffiliatedWithCommission = AffiliatedWith{
		Subject: EleanorRoosevelt,
		Object:  UNCommissionOnHumanRights,
		Prov:    udhrSource,
	}
	HumphreyAffiliatedWithCommission = AffiliatedWith{
		Subject: JohnHumphrey,
		Object:  UNCommissionOnHumanRights,
		Prov:    udhrSource,
	}
	CassinAffiliatedWithCommission = AffiliatedWith{
		Subject: ReneCassin,
		Object:  UNCommissionOnHumanRights,
		Prov:    udhrSource,
	}
	ChangAffiliatedWithCommission = AffiliatedWith{
		Subject: PengChunChang,
		Object:  UNCommissionOnHumanRights,
		Prov:    udhrSource,
	}
	MalikAffiliatedWithCommission = AffiliatedWith{
		Subject: CharlesMalik,
		Object:  UNCommissionOnHumanRights,
		Prov:    udhrSource,
	}
	MehtaAffiliatedWithCommission = AffiliatedWith{
		Subject: HansaMehta,
		Object:  UNCommissionOnHumanRights,
		Prov:    udhrSource,
	}
)

// -----------------------------------------------------------------------------
// Document-level theory-of-human-rights claim, routed through a
// Hypothesis entity because TheoryOf is BinaryRelation[Hypothesis,
// Concept] and UDHR1948 is itself a Concept. Added during draft
// review (see the flagged comment above). The Hypothesis entity
// represents the document-level normative claim that "what the UDHR
// articulates IS what human rights are" — an interpretive claim
// about the document's scope, distinct from any single article's
// content.
// -----------------------------------------------------------------------------

var (
	UDHRAsTheoryOfHumanRights = Hypothesis{&Entity{
		ID:    "hyp-udhr-as-theory-of-human-rights",
		Name:  "The Universal Declaration of Human Rights articulates a substantive and enumerable common standard of achievement for what human rights are — a 30-article list of inalienable rights held universally by every member of the human family",
		Kind:  "hypothesis",
		Brief: "The interpretive meta-claim that the UDHR is itself a theory of human rights — i.e., that the 30-article enumeration is not merely a political declaration but a substantive account of what human rights consist of. This claim is the bridge between UDHR1948 as a document-Concept and HumanRights as the meta-Concept the document is about. Attributed via ProposesOrg to the UN General Assembly (the body whose act of declaring turned the drafting output into a normative claim) rather than to any individual drafter. The contested-concept rule does not fire on HumanRights from this slice alone because this is the only TheoryOf(HumanRights) claim in winze; the entity exists to make HumanRights a contested-target-ready Concept for future slices.",
	}}

	UNGAProposesUDHRAsTheory = ProposesOrg{
		Subject: UNGeneralAssembly,
		Object:  UDHRAsTheoryOfHumanRights,
		Prov:    udhrSource,
	}

	UDHRTheoryOfHumanRights = TheoryOf{
		Subject: UDHRAsTheoryOfHumanRights,
		Object:  HumanRights,
		Prov:    udhrSource,
	}
)
