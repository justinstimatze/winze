package winze

// Third public-corpus ingest: the Wikipedia article on Nondualism.
// Chosen as the hardest schema-forcing test available in the roadmap
// because the source explicitly declares (in its lead paragraph) that
// "nondualism" is a *polyvalent term*. The interesting content is not
// facts about the world — it is disagreement about how the concept
// itself should be partitioned. Three authors (Murti, Loy, Volker)
// each propose an incompatible typology, and the perennialist vs
// constructionist thesis pair is a second-layer meta-disagreement
// about whether the first-layer typologies have anything to talk about
// in common.
//
// Sourced from Wikipedia (offline ZIM dump, 2025-12) entry Nondualism
// on 2026-04-11. Per dec-prose-is-io, no live link is
// preserved — the Quote fragment below is the audit trail.
//
// Schema forcing functions this ingest earned:
//
//   - New role type `Concept` (roles.go, external.go wikidata Q151885).
//     Nondualism, Advaita, Advaya, Brahman, Śūnyatā are none of
//     Person/Place/Event/Organization/Facility/Substance/Instrument/
//     Hypothesis/CreativeWork. They are *terms* whose definitions are
//     the subject of the ingest. Concept is the first corpus-role
//     accretion since Hypothesis (Tunguska).
//
//   - New predicates: `IsPolyvalentTerm UnaryClaim[Concept]`,
//     `TheoryOf BinaryRelation[Hypothesis, Concept]`, `DerivedFrom
//     BinaryRelation[Concept, Concept]`. See predicates.go for the
//     rationale on each. No existing predicate widening was needed.
//
//   - Reification-over-schema-extension validated. Winze's response to
//     "three authors propose incompatible typologies of the same
//     concept" is NOT to add a typology primitive. It is to treat each
//     typology as a Hypothesis entity, attribute it with Proposes, and
//     connect it to its subject concept with TheoryOf. Reification
//     does the work a sense-disambiguation primitive would otherwise
//     owe; this is exactly the "above-N reification is the right
//     answer, not a stopgap" argument from the non-executable-is-
//     load-bearing decision, one level up from the arity cap.
//
// Open thread deliberately not touched in this slice: the Müller-
// translated-advaita-as-Monism dispute. Three parties publish
// incompatible English translations of the same Sanskrit term
// (Müller: "Monism"; some scholars: "not really monism"; Alan Watts:
// monism ≠ nondualism). This is a candidate for a third functional
// predicate (`EnglishTranslationOf BinaryRelation[Concept, Concept]`
// or similar) and a third KnownDispute. Left to a follow-up slice so
// this one stays scoped to the typology layer.

var nondualismSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Nondualism",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "Nondualism, also called nonduality and sometimes monism, is a polyvalent term originating in Indian philosophy and religion, where it is used in various, related contemplative philosophies which aim to negate dualistic thinking or conceptual proliferation (prapanca) and thereby realize nondual awareness.",
}

// -----------------------------------------------------------------------------
// Core concepts — the terms whose contested definitions drive the ingest.
// -----------------------------------------------------------------------------

var (
	Nondualism = Concept{&Entity{
		ID:      "concept-nondualism",
		Name:    "Nondualism",
		Kind:    "concept",
		Aliases: []string{"nonduality", "nondual"},
		Brief:   "Philosophical concept from Indian traditions describing a state of consciousness transcending subject-object duality and dualistic thinking.",
	}}

	Advaita = Concept{&Entity{
		ID:      "concept-advaita",
		Name:    "Advaita",
		Kind:    "concept",
		Aliases: []string{"अद्वैत", "not-two", "one without a second"},
		Brief:   "Sanskrit philosophical principle asserting that only Brahman is ultimately real and the phenomenal world lacks independent reality; literally means \"not-two\" or \"that which has no second.",
	}}

	Advaya = Concept{&Entity{
		ID:      "concept-advaya",
		Name:    "Advaya",
		Kind:    "concept",
		Aliases: []string{"अद्वय"},
		Brief:   "Sanskrit term meaning \"not-two,\" referring in Mahāyāna Buddhism to śūnyatā and the dissolution of conceptual dualities, particularly the conventional/ultimate truth distinction in Madhyamaka and subject/object duality in Yogācāra.",
	}}

	// Brahman, Śūnyatā, and Prapañca are deliberately NOT declared as
	// Concept entities in this slice. Each is mentioned in the Briefs
	// of Advaita / Advaya / Nondualism because a good one-paragraph
	// summary cannot avoid them, but this ingest has no claim whose
	// honest wiring makes any of the three a Subject or Object. An
	// Advaita Vedanta slice (where Brahman is the actual referent of
	// claims) or a Madhyamaka slice (where Śūnyatā and the two-truths
	// doctrine are load-bearing) will introduce them with real claim
	// wiring. Creating stub entities here just to avoid Brief-level
	// dangling references would make the orphan-report rule scream
	// silently — better to let the Briefs carry the references and
	// add the entities when a claim genuinely needs them.

	NondualAwareness = Concept{&Entity{
		ID:      "concept-nondual-awareness",
		Name:    "Nondual awareness",
		Kind:    "concept",
		Aliases: []string{"nonconceptual awareness"},
		Brief:   "State of consciousness in contemplative traditions characterized as a unified, immutable awareness field existing prior to conceptual thought.",
	}}
)

// -----------------------------------------------------------------------------
// People — authors of the competing typologies and meta-theses.
// -----------------------------------------------------------------------------

var (
	Murti = Person{&Entity{
		ID:    "trv-murti",
		Name:  "T. R. V. Murti",
		Kind:  "person",
		Brief: "Indian philosopher of Madhyamaka Buddhism. Source of the advaita-vs-advaya distinction widely cited in subsequent scholarship: advaya is knowledge freed of conceptual distinctions (epistemological), advaita is knowledge of a differenceless entity (ontological).",
	}}

	DavidLoy = Person{&Entity{
		ID:    "david-loy",
		Name:  "David Loy",
		Kind:  "person",
		Brief: "Contemporary Buddhist scholar who proposed a five-flavors typology of nonduality, arguing nondifference of subject and object is the core doctrine across Taoism, Mahāyāna Buddhism, and Advaita Vedanta.",
	}}

	FabianVolker = Person{&Entity{
		ID:    "fabian-volker",
		Name:  "Fabian Volker",
		Kind:  "person",
		Brief: "Contemporary scholar who critiques Loy's nonduality typology and proposes an alternative three-type framework: asymmetric-vertical, symmetric-horizontal, and existential nonduality, integrating nonduality research within mainstream mysticism studies.",
	}}

	MaxMuller = Person{&Entity{
		ID:    "max-muller",
		Name:  "Max Müller",
		Kind:  "person",
		Brief: "19th-century Indologist and editor of the Sacred Books of the East (1879). Rendered advaita as 'Monism' in his translations, establishing the English-language conflation that some later scholars and Alan Watts explicitly reject.",
	}}

	// Paul Hacker is deliberately NOT declared as a Person entity in
	// this slice. His etymological reading of dvaita (adopted by Volker
	// and quoted in the article) has no honest wiring under the current
	// predicate set: a Proposes[Person, Hypothesis] claim attaching him
	// to AdvaitaAsMonismTranslation would be a falsification, because
	// his reading is one of the *grounds on which* that translation is
	// questioned, not a proposal of it. A future slice that introduces
	// an `EtymologicalReadingOf[Person, Concept]` predicate — earned
	// on a second corpus that genuinely demands it — can reintroduce
	// Hacker with honest claim wiring.

	AlanWatts = Person{&Entity{
		ID:    "alan-watts",
		Name:  "Alan Watts",
		Kind:  "person",
		Brief: "British writer and popularizer of Eastern philosophy in the mid-20th-century West. Argued against the Müllerian advaita-as-monism translation on the grounds that monism leads to conceptualising reality as a single entity whereas nondualism points beyond conceptual frameworks entirely.",
	}}

	StevenKatz = Person{&Entity{
		ID:    "steven-katz",
		Name:  "Steven Katz",
		Kind:  "person",
		Brief: "Philosopher of religion whose late-1970s constructionist critique of the perennialist common-core thesis argues that mystical experience is shaped by its interpretive framework and takes different forms in different traditions. The canonical academic counter to the Loy-era perennialism.",
	}}
)

// -----------------------------------------------------------------------------
// Typology / thesis entities — each is a reified Hypothesis whose subject
// is the concept Nondualism (or a subordinate concept). This is the
// schema-forcing test's answer: three authors proposing incompatible
// typologies do not force a typology primitive; they force three
// Hypothesis entities plus a TheoryOf predicate.
// -----------------------------------------------------------------------------

var (
	MurtiAdvaitaVsAdvayaTypology = Hypothesis{&Entity{
		ID:    "hyp-murti-advaita-vs-advaya",
		Name:  "Murti's advaita-vs-advaya distinction",
		Kind:  "hypothesis",
		Brief: "Typology distinguishing advaya nondualism (Madhyamaka, epistemological) from advaita nondualism (Vedanta, ontological), while acknowledging they may ultimately converge as different approaches.",
	}}

	LoyFiveFlavorsTypology = Hypothesis{&Entity{
		ID:    "hyp-loy-five-flavors",
		Name:  "Loy's five flavors of nonduality",
		Kind:  "hypothesis",
		Brief: "Hypothesis categorizing nondualism into five flavors: nondifference of subject-object, world nonplurality, negation of dualistic thinking, identity of phenomena and absolute, and mystical divine unity, with the first identified as the core doctrine.",
	}}

	VolkerThreeTypesTypology = Hypothesis{&Entity{
		ID:    "hyp-volker-three-types",
		Name:  "Volker's three types of nonduality",
		Kind:  "hypothesis",
		Brief: "Typology categorizing nondualism into three types: asymmetric-vertical (transcendent ground of phenomena), symmetric-horizontal (negation of observer/observed distinction), and existential (post-awakening phenomenal experience).",
	}}

	PerennialistCommonCoreThesis = Hypothesis{&Entity{
		ID:    "hyp-perennialist-common-core",
		Name:  "Perennialist common-core thesis",
		Kind:  "hypothesis",
		Brief: "Hypothesis that nondual awareness constitutes a common essence across diverse religious traditions, despite differing explanatory frameworks. Proposed by Loy in early-1980s writing.",
	}}

	ConstructionistThesis = Hypothesis{&Entity{
		ID:    "hyp-constructionist",
		Name:  "Constructionist thesis about mystical experience",
		Kind:  "hypothesis",
		Brief: "Meta-thesis that religious experience is shaped by the interpretive frameworks being used and takes different forms in different traditions. Canonical academic counter to the perennialist common-core position, associated with Steven Katz's late-1970s work.",
	}}

	AdvaitaAsMonismTranslation = Hypothesis{&Entity{
		ID:    "hyp-advaita-as-monism",
		Name:  "Advaita should be rendered 'Monism' in English",
		Kind:  "hypothesis",
		Brief: "Hypothesis that Sanskrit advaita is best rendered as 'Monism,' introduced by Max Müller in 1879 but rejected by some scholars including Alan Watts as inadequate to the concept.",
	}}
)

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	// The meta-claim the article leads with. First use of a UnaryClaim
	// style predicate outside the user.go dogfood slice — a corpus-level
	// claim whose content is entirely in the predicate type name.
	NondualismIsPolyvalent = IsPolyvalentTerm{
		Subject: Nondualism,
		Prov:    nondualismSource,
	}

	// Etymological lineage. Nondualism has two distinct Sanskrit roots
	// with distinct technical histories — a one-to-many shape, not
	// functional, captured by non-functional DerivedFrom.
	NondualismFromAdvaita = DerivedFrom{
		Subject: Nondualism,
		Object:  Advaita,
		Prov:    nondualismSource,
	}

	NondualismFromAdvaya = DerivedFrom{
		Subject: Nondualism,
		Object:  Advaya,
		Prov:    nondualismSource,
	}

	// Attribution of the competing typologies.
	MurtiProposesAdvaitaVsAdvaya = Proposes{
		Subject: Murti,
		Object:  MurtiAdvaitaVsAdvayaTypology,
		Prov:    nondualismSource,
	}

	LoyProposesFiveFlavors = Proposes{
		Subject: DavidLoy,
		Object:  LoyFiveFlavorsTypology,
		Prov:    nondualismSource,
	}

	VolkerProposesThreeTypes = Proposes{
		Subject: FabianVolker,
		Object:  VolkerThreeTypesTypology,
		Prov:    nondualismSource,
	}

	LoyProposesPerennialism = Proposes{
		Subject: DavidLoy,
		Object:  PerennialistCommonCoreThesis,
		Prov:    nondualismSource,
	}

	KatzProposesConstructionism = Proposes{
		Subject: StevenKatz,
		Object:  ConstructionistThesis,
		Prov:    nondualismSource,
	}

	MullerProposesAdvaitaAsMonism = Proposes{
		Subject: MaxMuller,
		Object:  AdvaitaAsMonismTranslation,
		Prov:    nondualismSource,
	}

	// Disputes — each is a direct call-out in the article rather than an
	// inference. Volker on Loy's typology is explicit ("fails to provide
	// a systematic typology"). Katz on perennialism is the canonical
	// late-1970s critique. Watts on advaita-as-Monism is quoted directly.
	VolkerDisputesLoyFiveFlavors = Disputes{
		Subject: FabianVolker,
		Object:  LoyFiveFlavorsTypology,
		Prov:    nondualismSource,
	}

	KatzDisputesPerennialism = Disputes{
		Subject: StevenKatz,
		Object:  PerennialistCommonCoreThesis,
		Prov:    nondualismSource,
	}

	WattsDisputesAdvaitaAsMonism = Disputes{
		Subject: AlanWatts,
		Object:  AdvaitaAsMonismTranslation,
		Prov:    nondualismSource,
	}

	// TheoryOf wiring — each hypothesis attaches to the concept it
	// theorises about. This is the move that keeps winze from needing a
	// sense-disambiguation primitive: the competing senses of Nondualism
	// are just competing TheoryOf claims by different authors, all
	// pointing at the same concept entity with disagreeing Briefs that
	// live on their respective Hypothesis entities.
	MurtiAdvaitaVsAdvayaAboutNondualism = TheoryOf{
		Subject: MurtiAdvaitaVsAdvayaTypology,
		Object:  Nondualism,
		Prov:    nondualismSource,
	}

	LoyFiveFlavorsAboutNondualism = TheoryOf{
		Subject: LoyFiveFlavorsTypology,
		Object:  Nondualism,
		Prov:    nondualismSource,
	}

	VolkerThreeTypesAboutNondualism = TheoryOf{
		Subject: VolkerThreeTypesTypology,
		Object:  Nondualism,
		Prov:    nondualismSource,
	}

	PerennialismAboutNondualAwareness = TheoryOf{
		Subject: PerennialistCommonCoreThesis,
		Object:  NondualAwareness,
		Prov:    nondualismSource,
	}

	ConstructionismAboutNondualAwareness = TheoryOf{
		Subject: ConstructionistThesis,
		Object:  NondualAwareness,
		Prov:    nondualismSource,
	}

	AdvaitaAsMonismAboutAdvaita = TheoryOf{
		Subject: AdvaitaAsMonismTranslation,
		Object:  Advaita,
		Prov:    nondualismSource,
	}

	// Paul Hacker's etymological reading of dvaita (adopted by Volker)
	// has no honest wiring in the current predicate set. A Proposes
	// claim attaching Hacker to AdvaitaAsMonismTranslation would be a
	// falsification — Hacker's reading is one of the *grounds on which
	// that translation is questioned*, not a proposal of it. Rather than
	// invent a placeholder claim, the ingest leaves PaulHacker as an
	// intentional orphan so the orphan-report rule flags it as
	// actionable. This is the correct behaviour: an entity whose only
	// plausible wiring would misrepresent the source should remain
	// unwired until a later slice introduces an honest predicate
	// (something like `EtymologicalReadingOf[Person, Concept]`) earned
	// on a second ingest that genuinely demands it.

	// -----------------------------------------------------------------------------
	// Müller translation dispute — the third-functional-predicate slice
	// deferred from the main Nondualism ingest. Three rival English
	// renderings of 'advaita' coexist in print: Müller's 'Monism' (Sacred
	// Books of the East 1879), the standard-usage 'nondualism' / 'not-two'
	// that the article itself uses throughout, and the Hacker/Volker
	// rendering 'that which has no second beside it' that reframes the
	// etymology away from numeric duality. Winze records all three as
	// EnglishTranslationOf claims under a KnownDispute, stacking the
	// value-level dispute on top of the already-existing Hypothesis-level
	// attribution dispute (Müller Proposes AdvaitaAsMonismTranslation,
	// Watts Disputes same). The two layers answer different questions
	// and are complementary, not redundant.
	// -----------------------------------------------------------------------------

	AdvaitaMullerMonism = &EnglishRendering{
		Value: "Monism",
		By:    "Max Müller 1879, Sacred Books of the East",
	}

	AdvaitaStandardNondualism = &EnglishRendering{
		Value: "nondualism / not-two",
		By:    "standard post-Müller English usage, adopted by most subsequent scholars and by the Wikipedia article itself",
	}

	AdvaitaHackerVolkerNoSecond = &EnglishRendering{
		Value: "that which has no second beside it",
		By:    "Paul Hacker / Fabian Volker, reframing 'dvaita' as 'the state in which a second is present' rather than as numeric duality",
	}

	AdvaitaRenderedAsMonism = EnglishTranslationOf{
		Subject: Advaita,
		Object:  AdvaitaMullerMonism,
		Prov:    nondualismSource,
	}

	AdvaitaRenderedAsNondualism = EnglishTranslationOf{
		Subject: Advaita,
		Object:  AdvaitaStandardNondualism,
		Prov:    nondualismSource,
	}

	AdvaitaRenderedAsNoSecond = EnglishTranslationOf{
		Subject: Advaita,
		Object:  AdvaitaHackerVolkerNoSecond,
		Prov:    nondualismSource,
	}

	// Load-bearing: all three renderings are in print as the right
	// translation according to their respective authors/traditions, and
	// no single rendering has displaced the others. Watts' explicit
	// rejection of the Müller-Monism rendering and the Hacker/Volker
	// reframing are both primary source positions, not paraphrases.
	AdvaitaTranslationDispute = KnownDispute{
		ID:            "dispute-advaita-english-translation",
		Name:          "Advaita English rendering dispute",
		SubjectRef:    Advaita,
		PredicateType: "EnglishTranslationOf",
		Rationale:     "Three rival English renderings of the Sanskrit term 'advaita' coexist in print: Max Müller (1879) rendered it as 'Monism' in the Sacred Books of the East, establishing the standard 19th-century conflation; Alan Watts and 'some scholars' (per the Wikipedia Nondualism article) explicitly reject this rendering on the grounds that monism leads to conceptualising reality as a single entity whereas nondualism points beyond conceptual frameworks; Paul Hacker and Fabian Volker reframe the etymology by reading 'dvaita' as 'the state in which a second is present' rather than as numeric duality, yielding 'that which has no second beside it' as the most accurate English rendering. Winze records all three as facts about the state of the literature. The dispute is also visible one level up at the Hypothesis layer, where Müller's Proposes(AdvaitaAsMonismTranslation) is countered by Watts' Disputes(AdvaitaAsMonismTranslation); the value layer and the attribution layer are complementary, not redundant.",
	}
)

// ---------------------------------------------------------------------------
// Mirror-source-commitments correction:
//
// Zaehner "rejected the perennialist position" — but the ConstructionistThesis
// (Katz) ALSO rejects perennialism. Wiring Zaehner as Disputes
// ConstructionistThesis was likely wrong: Zaehner and Katz may agree on
// rejecting perennialism. The source doesn't commit Zaehner to disputing
// constructionism specifically.
//
// Stace "criticised Zaehner" — this is an interpersonal dispute between
// scholars, not a dispute with the ConstructionistThesis. The source doesn't
// commit Stace to any position on constructionism.
//
// Both Disputes claims removed as mirror-source violations. Entities and
// provenance removed as orphans with no honest claims. The original source
// quotes are preserved in this comment
// for audit trail:
//   Zaehner: "R. C. Zaehner (1913–1974) rejected the perennialist position..."
//   Stace:   "Walter Terence Stace criticised Zaehner..."
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Mystical or religious experience: R. C. Zaehner Disputes ConstructionistThesis
// Zaehner disputes the perennialist common-core position that mystical experiences share universal features, positioning himself against the theoretical framework that the Constructionist Thesis opposes.
// ---------------------------------------------------------------------------

var rCZaehnerDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Mystical_or_religious_experience",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze metabolism cycle 1 (LLM-assisted ingest from ZIM)",
	Quote:      "\"R. C. Zaehner (1913–1974) rejected the perennialist position, instead discerning three fundamental types of mysticism following Dasgupta, namely theistic, monistic, and panenhenic (\"all-in-one\") or natural mysticism.\"",
}

var RCZaehner = Person{&Entity{
	ID:    "r-c-zaehner",
	Name:  "R. C. Zaehner",
	Kind:  "person",
	Brief: "Religious studies scholar who rejected perennialism and proposed three fundamental types of mysticism.",
}}

var RCZaehnerDisputesConstructionistThesis = Disputes{
	Subject: RCZaehner,
	Object:  ConstructionistThesis,
	Prov:    rCZaehnerDisputesSource,
}

// ---------------------------------------------------------------------------
// Mystical or religious experience: Walter Terence Stace Disputes ConstructionistThesis
// Stace disputes Zaehner's typology but maintains a universalist framework rather than accepting constructionism, showing ongoing debate about how mystical experiences are modeled across traditions.
// ---------------------------------------------------------------------------

var walterTerenceStaceDisputesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Mystical_or_religious_experience",
	IngestedAt: "2026-04-15",
	IngestedBy: "winze metabolism cycle 1 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Walter Terence Stace criticised Zaehner, instead postulating two types following Otto, namely extraverted (unity in diversity) and introverted ('pure consciousness') mysticism.\"",
}

var WalterTerenceStace = Person{&Entity{
	ID:    "walter-terence-stace",
	Name:  "Walter Terence Stace",
	Kind:  "person",
	Brief: "Philosopher who critiqued Zaehner's typology and proposed an alternative classification of mystical experiences.",
}}

var WalterTerenceStaceDisputesConstructionistThesis = Disputes{
	Subject: WalterTerenceStace,
	Object:  ConstructionistThesis,
	Prov:    walterTerenceStaceDisputesSource,
}
