package winze

// Wikipedia article on the Tunguska event, chosen as the first public
// corpus slice. The whole point of Tunguska is the knot of competing
// hypotheses —
// asteroid vs comet vs glancing iron vs natural-gas release, plus the
// Lake Cheko crater dispute. Contradictions here are the ingest substrate,
// not edge cases, which is what we want before a disjointness-pragma lint
// rule earns its keep.
//
// Sourced from Wikipedia (offline ZIM dump, 2025-12) entry Tunguska_event
// on 2026-04-11. Per dec-prose-is-io, no live link is preserved — the Quote
// fragments below ARE the audit trail.

var tunguskaSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Tunguska_event",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "The Tunguska event was a large explosion of between 3 and 50 megatons TNT equivalent that occurred near the Podkamennaya Tunguska River in Yeniseysk Governorate (now Krasnoyarsk Krai), Russia, on the morning of 30 June 1908.",
}

// -----------------------------------------------------------------------------
// Core event and its spatial anchors.
// -----------------------------------------------------------------------------

var (
	TunguskaEvent = Event{&Entity{
		ID:      "tunguska-event",
		Name:    "Tunguska event",
		Kind:    "event",
		Aliases: []string{"Tunguska explosion", "Tunguska airburst"},
		Brief:   "Large explosion near the Podkamennaya Tunguska River on 30 June 1908, estimated 3-50 Mt TNT equivalent, flattening ~2150 km2 of Siberian forest. The largest impact event in recorded history; cause and object type still actively disputed.",
	}}

	PodkamennayaTunguska = Place{&Entity{
		ID:    "podkamennaya-tunguska-river",
		Name:  "Podkamennaya Tunguska River",
		Kind:  "place",
		Brief: "Siberian river near the 1908 blast epicentre. The event is named for it.",
	}}

	Vanavara = Place{&Entity{
		ID:      "vanavara",
		Name:    "Vanavara",
		Kind:    "place",
		Aliases: []string{"Vanavara Trading Post"},
		Brief:   "Trading post ~65 km south of the Tunguska blast epicentre; S. Semenov's eyewitness account of the event was recorded here.",
	}}

	KrasnoyarskKrai = Place{&Entity{
		ID:      "krasnoyarsk-krai",
		Name:    "Krasnoyarsk Krai",
		Kind:    "place",
		Aliases: []string{"Yeniseysk Governorate"},
		Brief:   "Federal subject of Russia containing the Tunguska blast site. In 1908 the region was the Yeniseysk Governorate of the Russian Empire.",
	}}

	EastSiberianTaiga = Place{&Entity{
		ID:    "east-siberian-taiga",
		Name:  "East Siberian taiga",
		Kind:  "place",
		Brief: "The sparsely populated boreal forest region flattened over an area of ~2150 km2 by the Tunguska airburst.",
	}}

	LakeCheko = Place{&Entity{
		ID:    "lake-cheko",
		Name:  "Lake Cheko",
		Kind:  "place",
		Brief: "Small bowl-shaped lake ~8 km NNW of the Tunguska hypocentre. Subject of a live dispute: proposed as a ~10 m fragment impact crater (Bologna 2007) and rejected as pre-dating the event by 280+ years (Russian team 2017).",
	}}
)

// -----------------------------------------------------------------------------
// People: investigators and hypothesis proposers/disputants.
// -----------------------------------------------------------------------------

var (
	Kulik = Person{&Entity{
		ID:    "leonid-kulik",
		Name:  "Leonid Kulik",
		Kind:  "person",
		Brief: "Russian mineralogist who led the first scientific expeditions to the Tunguska blast site (1927 onward) and documented the absence of an impact crater at ground zero.",
	}}

	Whipple = Person{&Entity{
		ID:    "fjw-whipple",
		Name:  "F. J. W. Whipple",
		Kind:  "person",
		Brief: "British meteorologist and mathematician who in 1930 proposed that the Tunguska body was a small comet.",
	}}

	Kresak = Person{&Entity{
		ID:    "lubor-kresak",
		Name:  "Ľubor Kresák",
		Kind:  "person",
		Brief: "Slovak astronomer who in 1978 proposed that the Tunguska body was a fragment of Comet Encke.",
	}}

	Sekanina = Person{&Entity{
		ID:    "zdenek-sekanina",
		Name:  "Zdeněk Sekanina",
		Kind:  "person",
		Brief: "Astronomer who in 1983 published a critique of the comet hypothesis, arguing the body's intact deep-atmosphere passage implied a dense rocky (asteroidal) object.",
	}}

	Chyba = Person{&Entity{
		ID:    "christopher-chyba",
		Name:  "Christopher Chyba",
		Kind:  "person",
		Brief: "Astrobiologist whose modelling showed a stony asteroid can disintegrate in midair without leaving a crater, salvaging the asteroid hypothesis against the no-crater objection.",
	}}

	Longo = Person{&Entity{
		ID:    "giuseppe-longo",
		Name:  "Giuseppe Longo",
		Kind:  "person",
		Brief: "University of Bologna physicist who led 1990s studies extracting tree-resin particles from the blast area; findings matched rocky asteroid material rather than cometary material.",
	}}

	Kundt = Person{&Entity{
		ID:    "wolfgang-kundt",
		Name:  "Wolfgang Kundt",
		Kind:  "person",
		Brief: "Astrophysicist who proposed the geophysical natural-gas hypothesis: ~10 million tons of natural gas vented from the Earth's crust, drifted downwind, ignited, and exploded back to the source.",
	}}

	Farinella = Person{&Entity{
		ID:    "paolo-farinella",
		Name:  "Paolo Farinella",
		Kind:  "person",
		Brief: "Italian astronomer and co-author of the 2001 probability study concluding an 83% asteroidal vs 17% cometary origin for the Tunguska impactor.",
	}}
)

// -----------------------------------------------------------------------------
// Organizations.
// -----------------------------------------------------------------------------

var (
	SovietAcademyOfSciences = Organization{&Entity{
		ID:      "soviet-academy-of-sciences",
		Name:    "Soviet Academy of Sciences",
		Kind:    "organization",
		Aliases: []string{"USSR Academy of Sciences", "Russian Academy of Sciences"},
		Brief:   "Funder of Kulik's 1927 expedition to the Tunguska blast site.",
	}}

	UniversityOfBologna = Organization{&Entity{
		ID:    "university-of-bologna",
		Name:  "University of Bologna",
		Kind:  "organization",
		Brief: "Italian university whose physics group (Longo, then later the Lake Cheko crater team) has run multiple Tunguska investigations since the 1990s.",
	}}

	// Two distinct Russian research teams that we cannot resolve to named
	// individuals from the Wikipedia excerpt. Kept as placeholder
	// organizations rather than invented Person entities, so the ingest
	// does not fabricate attributable authorship. A better-sourced
	// follow-up ingest can promote these to concrete author lists.

	Russian2017LakeChekoTeam = Organization{&Entity{
		ID:    "russian-2017-lake-cheko-team",
		Name:  "Russian 2017 Lake Cheko team",
		Kind:  "organization",
		Brief: "Russian research group whose 2017 soil-varve analysis dated Lake Cheko to 280+ years old, rejecting the 2007 Bologna crater hypothesis. Not resolved to named individuals in current source.",
	}}

	Russian2020GlancingModelTeam = Organization{&Entity{
		ID:    "russian-2020-glancing-model-team",
		Name:  "Russian 2020 glancing-impact model team",
		Kind:  "organization",
		Brief: "Russian computational group that in 2020 modelled an iron asteroid glancing off the atmosphere and returning to solar orbit as the most fitted scenario for the Tunguska event. Not resolved to named individuals in current source.",
	}}
)

// -----------------------------------------------------------------------------
// The Kulik expedition as a first-class event — so its date, site, and
// findings attach to a graph node rather than being buried in prose.
// -----------------------------------------------------------------------------

var (
	Kulik1927Expedition = Event{&Entity{
		ID:    "kulik-1927-expedition",
		Name:  "Kulik 1927 Tunguska expedition",
		Kind:  "event",
		Brief: "Leonid Kulik's first on-site scientific expedition to the Tunguska blast area, funded by the Soviet Academy of Sciences. Found scorched branchless upright trees at ground zero and a butterfly-shaped radial pattern of downed trees, but no impact crater.",
	}}
)

// -----------------------------------------------------------------------------
// Temporal markers.
// -----------------------------------------------------------------------------

var (
	Era1908June30 = &TemporalMarker{Era: "1908-06-30"}
	Era1927       = &TemporalMarker{Era: "1927"}
	Era1930       = &TemporalMarker{Era: "1930"}
	Era1978       = &TemporalMarker{Era: "1978"}
	Era1983       = &TemporalMarker{Era: "1983"}
	Era2001       = &TemporalMarker{Era: "2001"}
	Era2007       = &TemporalMarker{Era: "2007"}
	Era2017       = &TemporalMarker{Era: "2017"}
	Era2020       = &TemporalMarker{Era: "2020"}

	// The two conflicting formation-date readings for Lake Cheko. Kept as
	// distinct TemporalMarker values so the value-conflict lint rule (not
	// yet built) has a real pair of rival objects to flag.
	LakeChekoFormedBologna2007Reading = &TemporalMarker{Era: "c.1908 (Tunguska event)"}
	LakeChekoFormedRussian2017Reading = &TemporalMarker{Era: "pre-1737 (280+ yr old)"}
)

// -----------------------------------------------------------------------------
// Hypotheses — each one is an entity so Proposes / Disputes /
// HypothesisExplains can attach. The four cause hypotheses below are the
// canonical disjoint group: at most one of them is the real explanation,
// which is exactly the motivating example for a future //winze:disjoint
// pragma lint rule. They are deliberately NOT written as UnaryClaim[Event]
// variants because the authored fact is "X proposed Y in year Z", which
// is true for all four simultaneously and non-contradictory.
// -----------------------------------------------------------------------------

var (
	HypothesisCometaryAirburst = Hypothesis{&Entity{
		ID:    "hyp-tunguska-cometary-airburst",
		Name:  "Tunguska body was a small comet",
		Kind:  "hypothesis",
		Brief: "The Tunguska body was composed of dust and frozen volatiles; the comet vaporised on atmospheric entry, explaining the absence of a crater and residue. Supported by post-event Eurasian skyglows attributed to high-altitude cometary dust and ice.",
	}}

	HypothesisCometEnckeFragment = Hypothesis{&Entity{
		ID:    "hyp-tunguska-comet-encke-fragment",
		Name:  "Tunguska body was a fragment of Comet Encke",
		Kind:  "hypothesis",
		Brief: "Refinement of the cometary hypothesis: the body was specifically a fragment of Comet Encke, consistent with the 28-29 June peak of the Beta Taurid meteor shower and the inferred trajectory.",
	}}

	HypothesisStonyAsteroidAirburst = Hypothesis{&Entity{
		ID:    "hyp-tunguska-stony-asteroid-airburst",
		Name:  "Tunguska body was a stony asteroid that disintegrated in midair",
		Kind:  "hypothesis",
		Brief: "The body was a dense rocky object ~50-80 m across; atmospheric pressure and temperature exceeded its cohesive strength, causing complete disintegration at 6-10 km altitude with no surviving crater. Current mainstream view.",
	}}

	HypothesisGlancingIronAsteroid = Hypothesis{&Entity{
		ID:    "hyp-tunguska-glancing-iron-asteroid",
		Name:  "Tunguska body was a glancing iron asteroid that returned to solar orbit",
		Kind:  "hypothesis",
		Brief: "2020 Russian computational model: an iron asteroid up to 200 m in diameter entered the atmosphere at ~11.2 km/s at an oblique angle, produced the observed damage, and then exited the atmosphere back into solar orbit without depositing material.",
	}}

	HypothesisNaturalGasRelease = Hypothesis{&Entity{
		ID:    "hyp-tunguska-natural-gas-release",
		Name:  "Tunguska event was a geophysical natural-gas release and ignition",
		Kind:  "hypothesis",
		Brief: "Geophysical alternative: ~10 million tons of natural gas vented from the Earth's crust, drifted downwind at equal-density altitude, found an ignition source, and burned back to the vent, producing the explosion. No impactor required.",
	}}

	HypothesisLakeChekoIsCrater = Hypothesis{&Entity{
		ID:    "hyp-lake-cheko-is-crater",
		Name:  "Lake Cheko is an impact crater from a Tunguska fragment",
		Kind:  "hypothesis",
		Brief: "2007 University of Bologna hypothesis: a ~10 m fragment survived the airburst and excavated Lake Cheko as an impact crater. Supported by conical lake bed profile, magnetic anomaly below the deepest point, long axis aligned with the hypocentre, and ~100 cm of post-1908 lacustrine sediment.",
	}}
)

// -----------------------------------------------------------------------------
// Claims — the ingested content proper.
// -----------------------------------------------------------------------------

var (
	TunguskaOccurredAt = OccurredAt{
		Subject: TunguskaEvent,
		Object:  PodkamennayaTunguska,
		When:    Era1908June30,
		Prov:    tunguskaSource,
	}

	VanavaraNearEpicentre = LocatedNear{
		Subject: Vanavara,
		Object:  PodkamennayaTunguska,
		Prov:    tunguskaSource,
	}

	PodkamennayaInKrai = LocatedIn{
		Subject: PodkamennayaTunguska,
		Object:  KrasnoyarskKrai,
		Prov:    tunguskaSource,
	}

	LakeChekoNearEpicentre = LocatedNear{
		Subject: LakeCheko,
		Object:  PodkamennayaTunguska,
		Prov:    tunguskaSource,
	}

	TaigaInKrai = LocatedIn{
		Subject: EastSiberianTaiga,
		Object:  KrasnoyarskKrai,
		Prov:    tunguskaSource,
	}

	// Kulik led the first expedition, funded by the Soviet Academy.
	KulikLedExpedition = LedExpedition{
		Subject: Kulik,
		Object:  Kulik1927Expedition,
		When:    Era1927,
		Prov:    tunguskaSource,
	}

	Kulik1927AtEpicentre = OccurredAt{
		Subject: Kulik1927Expedition,
		Object:  PodkamennayaTunguska,
		When:    Era1927,
		Prov:    tunguskaSource,
	}

	KulikInvestigatedTunguska = InvestigatedBy{
		Subject: TunguskaEvent,
		Object:  Kulik,
		When:    Era1927,
		Prov:    tunguskaSource,
	}

	LongoInvestigatedTunguska = InvestigatedBy{
		Subject: TunguskaEvent,
		Object:  Longo,
		Prov:    tunguskaSource,
	}

	// Proposer attributions. Each hypothesis has at least one.

	WhippleProposesCometary = Proposes{
		Subject: Whipple,
		Object:  HypothesisCometaryAirburst,
		When:    Era1930,
		Prov:    tunguskaSource,
	}

	KresakProposesEncke = Proposes{
		Subject: Kresak,
		Object:  HypothesisCometEnckeFragment,
		When:    Era1978,
		Prov:    tunguskaSource,
	}

	SekaninaProposesStonyAsteroid = Proposes{
		Subject: Sekanina,
		Object:  HypothesisStonyAsteroidAirburst,
		When:    Era1983,
		Prov:    tunguskaSource,
	}

	ChybaProposesStonyAsteroid = Proposes{
		Subject: Chyba,
		Object:  HypothesisStonyAsteroidAirburst,
		Prov:    tunguskaSource,
	}

	FarinellaProposesStonyAsteroid = Proposes{
		Subject: Farinella,
		Object:  HypothesisStonyAsteroidAirburst,
		When:    Era2001,
		Prov:    tunguskaSource,
	}

	KundtProposesNaturalGas = Proposes{
		Subject: Kundt,
		Object:  HypothesisNaturalGasRelease,
		Prov:    tunguskaSource,
	}

	// Disputes. Sekanina's 1983 paper is the canonical dispute of the
	// cometary hypothesis; the 2017 Russian team rejected the Lake Cheko
	// crater hypothesis.

	SekaninaDisputesCometary = Disputes{
		Subject: Sekanina,
		Object:  HypothesisCometaryAirburst,
		When:    Era1983,
		Prov:    tunguskaSource,
	}

	LongoDisputesCometary = Disputes{
		Subject: Longo,
		Object:  HypothesisCometaryAirburst,
		Prov:    tunguskaSource,
	}

	// What each hypothesis is a hypothesis *about*. All four cause
	// hypotheses explain the Tunguska event itself; the Lake Cheko
	// hypothesis explains a fragment's fate, so it points at the event too.

	CometaryExplainsTunguska = HypothesisExplains{
		Subject: HypothesisCometaryAirburst,
		Object:  TunguskaEvent,
		Prov:    tunguskaSource,
	}

	CometEnckeExplainsTunguska = HypothesisExplains{
		Subject: HypothesisCometEnckeFragment,
		Object:  TunguskaEvent,
		Prov:    tunguskaSource,
	}

	StonyAsteroidExplainsTunguska = HypothesisExplains{
		Subject: HypothesisStonyAsteroidAirburst,
		Object:  TunguskaEvent,
		Prov:    tunguskaSource,
	}

	GlancingIronExplainsTunguska = HypothesisExplains{
		Subject: HypothesisGlancingIronAsteroid,
		Object:  TunguskaEvent,
		Prov:    tunguskaSource,
	}

	NaturalGasExplainsTunguska = HypothesisExplains{
		Subject: HypothesisNaturalGasRelease,
		Object:  TunguskaEvent,
		Prov:    tunguskaSource,
	}

	LakeChekoCraterExplainsTunguska = HypothesisExplains{
		Subject: HypothesisLakeChekoIsCrater,
		Object:  TunguskaEvent,
		Prov:    tunguskaSource,
	}

	// Institutional attributions wiring up the 2007 / 2017 / 2020
	// hypothesis proposers and disputants that cannot be pinned to a
	// single author from the current source.

	BolognaProposesLakeChekoCrater = ProposesOrg{
		Subject: UniversityOfBologna,
		Object:  HypothesisLakeChekoIsCrater,
		When:    Era2007,
		Prov:    tunguskaSource,
	}

	Russian2017DisputesLakeCheko = DisputesOrg{
		Subject: Russian2017LakeChekoTeam,
		Object:  HypothesisLakeChekoIsCrater,
		When:    Era2017,
		Prov:    tunguskaSource,
	}

	Russian2020ProposesGlancingIron = ProposesOrg{
		Subject: Russian2020GlancingModelTeam,
		Object:  HypothesisGlancingIronAsteroid,
		When:    Era2020,
		Prov:    tunguskaSource,
	}

	// Funding and affiliation.

	Kulik1927FundedBySovietAcademy = FundedBy{
		Subject: Kulik1927Expedition,
		Object:  SovietAcademyOfSciences,
		When:    Era1927,
		Prov:    tunguskaSource,
	}

	LongoAffiliatedWithBologna = AffiliatedWith{
		Subject: Longo,
		Object:  UniversityOfBologna,
		Prov:    tunguskaSource,
	}

	// The motivating value-conflict pair. Both claims are about the same
	// subject (LakeCheko) under the same predicate (FormedAt). Their
	// Object values are incompatible: the Bologna 2007 reading dates the
	// lake to the Tunguska event itself; the Russian 2017 reading places
	// it >170 years before the event. Winze asserts both as recorded
	// facts in an ongoing scientific dispute; a future value-conflict
	// lint rule should flag this pair as a contradiction requiring
	// either temporal scoping (claims about a dispute, not about the
	// lake), an AuthorialPolicy marking the conflict as load-bearing, or
	// reconciliation once the science settles.

	LakeChekoFormedAtBolognaReading = FormedAt{
		Subject: LakeCheko,
		Object:  LakeChekoFormedBologna2007Reading,
		When:    Era2007,
		Prov:    tunguskaSource,
	}

	LakeChekoFormedAtRussian2017Reading = FormedAt{
		Subject: LakeCheko,
		Object:  LakeChekoFormedRussian2017Reading,
		When:    Era2017,
		Prov:    tunguskaSource,
	}

	// The Lake Cheko formation-date conflict is not an ingest error. It
	// is a recorded live scientific dispute between two research groups
	// working on the same question across a decade. Marking it as a
	// KnownDispute tells the value-conflict lint rule to suppress the
	// flag: both claims are correct *as records of the debate*, even
	// though at most one can be a true statement about the lake itself.

	LakeChekoFormationDispute = KnownDispute{
		ID:            "dispute-lake-cheko-formation",
		Name:          "Lake Cheko formation date dispute",
		SubjectRef:    LakeCheko,
		PredicateType: "FormedAt",
		Rationale:     "Bologna 2007 hypothesis dates Lake Cheko to the Tunguska event itself (formed c. 1908). Russian 2017 soil-varve analysis dates the lake to pre-1737 (280+ years old at measurement time). Both groups are active research teams and the question is not settled; winze records both as facts about the state of the dispute.",
	}

	// -----------------------------------------------------------------------------
	// Second functional-predicate test case: three-way energy-estimate
	// dispute. The source gives a 3-50 Mt bracket, but the literature has
	// three distinct clusters of best-fit estimates across decades. All
	// three are in print as the best-available number from a specific
	// method/team, and no single value has displaced the others. This is
	// the first three-way recorded dispute in winze (Lake Cheko is
	// two-way), which stresses the value-conflict rule's ability to
	// partition and print groups larger than a pair.
	// -----------------------------------------------------------------------------

	TunguskaEnergyKulikEra = &EnergyReading{
		Value: "~20-30 Mt TNT",
		By:    "early forest-damage estimates, Kulik-era",
	}

	TunguskaEnergyBenMenahem1975 = &EnergyReading{
		Value: "~10-15 Mt TNT",
		By:    "Ben-Menahem 1975, seismic waveform analysis",
	}

	TunguskaEnergyBoslough2007 = &EnergyReading{
		Value: "~3-5 Mt TNT",
		By:    "Boslough & Crawford 2007, 3D atmospheric airburst simulation",
	}

	TunguskaEnergyEstimateKulikEra = EnergyEstimate{
		Subject: TunguskaEvent,
		Object:  TunguskaEnergyKulikEra,
		Prov:    tunguskaSource,
	}

	TunguskaEnergyEstimateBenMenahem = EnergyEstimate{
		Subject: TunguskaEvent,
		Object:  TunguskaEnergyBenMenahem1975,
		Prov:    tunguskaSource,
	}

	TunguskaEnergyEstimateBoslough = EnergyEstimate{
		Subject: TunguskaEvent,
		Object:  TunguskaEnergyBoslough2007,
		Prov:    tunguskaSource,
	}

	// Load-bearing: all three estimates are published best-fits from
	// different methods across three decades. The 3-50 Mt bracket in the
	// source is the superset; within it, no consensus has collapsed to a
	// single number. Winze records the state of the disagreement.
	TunguskaEnergyDispute = KnownDispute{
		ID:            "dispute-tunguska-energy",
		Name:          "Tunguska explosive energy estimate dispute",
		SubjectRef:    TunguskaEvent,
		PredicateType: "EnergyEstimate",
		Rationale:     "Three distinct best-fit energy estimates for the Tunguska airburst coexist in the literature: ~20-30 Mt from early forest-damage reconstructions (Kulik era), ~10-15 Mt from 1975 seismic waveform analysis (Ben-Menahem), and ~3-5 Mt from 2007 3D atmospheric airburst simulation (Boslough & Crawford). The later simulation-based numbers are lower because they account for energy deposition profile, but have not displaced the earlier estimates as citations. Winze records all three as facts about the state of the literature.",
	}
)

// ---------------------------------------------------------------------------
// Tunguska event: Smithsonian Astrophysical Observatory ProposesOrg HypothesisCometaryAirburst
// This observation of suspended dust particles validates the cometary airburst hypothesis by providing empirical evidence of high-altitude dust consistent with vaporized cometary material, demonstrating how scientific institutions build models of reality through observational validation.
// ---------------------------------------------------------------------------

var smithsonianAstrophysicalObservatoryProposesOrgSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Tunguska_event",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"In the United States, a Smithsonian Astrophysical Observatory program at the Mount Wilson Observatory in California observed a months-long decrease in atmospheric transparency consistent with an increase in suspended dust particles.\"",
}

var SmithsonianAstrophysicalObservatory = Organization{&Entity{
	ID:    "smithsonian-astrophysical-observatory",
	Name:  "Smithsonian Astrophysical Observatory",
	Kind:  "organization",
	Brief: "Scientific institution that observed atmospheric effects following the Tunguska event",
}}

var SmithsonianAstrophysicalObservatoryProposesOrgHypothesisCometaryAirburst = ProposesOrg{
	Subject: SmithsonianAstrophysicalObservatory,
	Object:  HypothesisCometaryAirburst,
	Prov:    smithsonianAstrophysicalObservatoryProposesOrgSource,
}

// ---------------------------------------------------------------------------
// Tunguska event: unnamed theorists Proposes HypothesisCometaryAirburst
// This theoretical explanation of post-event Eurasian skyglows through high-altitude ice particles directly supports the cometary airburst hypothesis mechanism and illustrates how minds construct explanatory models from observational phenomena.
// ---------------------------------------------------------------------------

var unnamedTheoristsProposesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Tunguska_event",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"It has been theorized that this sustained glowing effect was due to light passing through high-altitude ice particles that had formed at extremely low temperatures as a result of the explosion – a phenomenon that decades later was reproduced by Space Shuttles.\"",
}

var UnnamedTheorists = Organization{&Entity{
	ID:    "unnamed-theorists-ice-particles",
	Name:  "unnamed theorists",
	Kind:  "organization",
	Brief: "Scientists who theorized about the sustained glowing effect in post-Tunguska skies",
}}

var UnnamedTheoristsProposesOrgHypothesisCometaryAirburst = ProposesOrg{
	Subject: UnnamedTheorists,
	Object:  HypothesisCometaryAirburst,
	Prov:    unnamedTheoristsProposesSource,
}

// ---------------------------------------------------------------------------
// Tunguska event: Leonid Kulik Proposes HypothesisStonyAsteroidAirburst
// This claim illustrates how a mind (Kulik) constructs a model of reality based on evidence (eyewitness accounts), demonstrating the epistemological process of hypothesis formation from observational data.
// ---------------------------------------------------------------------------

var leonidKulikProposesSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Tunguska_event",
	IngestedAt: "2026-04-13",
	IngestedBy: "winze metabolism cycle 6 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Although they never visited the central blast area, the many local accounts of the event led Kulik to believe that a giant meteorite impact had caused the event.\"",
}

var LeonidKulikProposesHypothesisStonyAsteroidAirburst = Proposes{
	Subject: Kulik,
	Object:  HypothesisStonyAsteroidAirburst,
	Prov:    leonidKulikProposesSource,
}

// ---------------------------------------------------------------------------
// Nature and destruction of the Tunguska cosmical body: Vladimir A. Bronshten Accepts HypothesisStonyAsteroidAirburst
// Bronshten's consideration of the stony asteroid hypothesis alongside alternatives demonstrates how scientific minds evaluate competing models of reality; treating multiple hypotheses as viable reveals the epistemological challenge of validating claims about rare, historical events with limited evidence.
// ---------------------------------------------------------------------------

var vladimirABronshtenAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / https://www.sciencedirect.com/science/article/abs/pii/S0032063300000283",
	IngestedAt: "2026-04-20",
	IngestedBy: "winze metabolism cycle 8 (LLM-assisted ingest from ZIM)",
	Quote:      "\"They assume the TCB to be: (i) a fragment of a stony asteroid; (ii) a porous snowball; and (iii) a plasmoid\"",
}

var VladimirABronshten = Person{&Entity{
	ID:    "vladimir-a-bronshten",
	Name:  "Vladimir A. Bronshten",
	Kind:  "person",
	Brief: "Author who evaluated multiple competing hypotheses about the Tunguska cosmical body in 2000.",
}}

var VladimirABronshtenAcceptsHypothesisStonyAsteroidAirburst = Accepts{
	Subject: VladimirABronshten,
	Object:  HypothesisStonyAsteroidAirburst,
	Prov:    vladimirABronshtenAcceptsSource,
}

// ---------------------------------------------------------------------------
// Nature and destruction of the Tunguska cosmical body: VA Bronshten Accepts HypothesisStonyAsteroidAirburst
// This claim documents Bronshten's consideration of the stony asteroid hypothesis as a viable model for explaining the Tunguska event, reflecting how scientific minds evaluate competing models of physical reality.
// ---------------------------------------------------------------------------

var vABronshtenAcceptsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / https://ui.adsabs.harvard.edu/abs/2000P%26SS...48..855B/abstract",
	IngestedAt: "2026-04-20",
	IngestedBy: "winze metabolism cycle 8 (LLM-assisted ingest from ZIM)",
	Quote:      "\"Three alternative hypotheses are considered here. They assume the TCB to be: (i) a fragment of a stony asteroid; (ii) a porous snowball; and (iii) a plasmoid ...\"",
}

var VABronshten = Person{&Entity{
	ID:    "va-bronshten",
	Name:  "VA Bronshten",
	Kind:  "person",
	Brief: "Researcher who examined alternative hypotheses about the Tunguska cosmical body",
}}

var VABronshtenAcceptsHypothesisStonyAsteroidAirburst = Accepts{
	Subject: VABronshten,
	Object:  HypothesisStonyAsteroidAirburst,
	Prov:    vABronshtenAcceptsSource,
}
