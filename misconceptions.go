package winze

// Fourth public-corpus ingest: Wikipedia's List of common misconceptions
// about science, technology, and mathematics. Chosen specifically because
// every entry in the source is shaped as a correction of a widely-held
// false belief — a genuinely new claim shape for winze, distinct from
// both Tunguska's contested-causes pattern and Nondualism's
// competing-typologies pattern.
//
// Source discipline: the Wikipedia article opens with the explicit
// commitment "Each entry on this list of common misconceptions is
// worded as a correction; the misconceptions themselves are implied
// rather than stated." Winze mirrors that commitment. For each entry
// the slice creates exactly one Hypothesis entity, whose Name is the
// corrected fact, and tags it with the `CorrectsCommonMisconception`
// unary predicate. No separate entity represents the false belief —
// inventing one would be fabrication of content the source does not
// provide. The existence of a widespread misbelief is recorded by
// the tag alone; a future ingest that genuinely needs structured
// misbelief-content can introduce a separate false-belief shape.
//
// Schema forcing functions earned by this slice:
//
//   - New UnaryClaim predicate CorrectsCommonMisconception[Hypothesis]
//     (predicates.go). First non-Person UnaryClaim target in winze;
//     previously all UnaryClaim predicates were style-claims about a
//     Person subject (user.go) or polyvalence about a Concept
//     (nondualism.go). Expanding the pattern to Hypothesis subjects
//     is the first use of UnaryClaim as a meta-annotation on a
//     reified intellectual-position entity.
//
//   - No new role types, no new binary predicates, no new functional
//     predicates. Existing Concept role and existing TheoryOf
//     relation handle the subject-linking. Organic growth discipline:
//     if a shape fits, do not invent.
//
// Slice scope: three entries from the Astronomy and spaceflight
// section, all with Concept-shaped subjects so TheoryOf handles them
// cleanly. A follow-up slice can tackle Place/Person/Facility-shaped
// subjects (Napoleon's height, Great Wall as a Place entity, Sun's
// actual colour, etc.) once a role-split decision has been made for
// the HypothesisAbout predicate family — the current TheoryOf only
// accepts Concept objects, and widening that should be driven by
// a second real need, not by this ingest's convenience.

var misconceptionsSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / List_of_common_misconceptions_about_science,_technology,_and_mathematics",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 3 (common misconceptions first slice)",
	Quote:      "Each entry on this list of common misconceptions is worded as a correction; the misconceptions themselves are implied rather than stated. [...] Seasons are not caused by the Earth being closer to the Sun in the summer than in the winter, but by the effects of Earth's 23.4-degree axial tilt. [...] When a meteor or spacecraft enters the atmosphere, the heat of entry is not primarily caused by friction, but by adiabatic compression of air in front of the object. [...] The Great Wall of China is not the only human-made object visible from space or from the Moon.",
}

// -----------------------------------------------------------------------------
// Subject concepts — each is the topic about which a misconception exists.
// Deliberately scoped to Concept so the existing TheoryOf predicate handles
// the linking without a role-widening decision.
// -----------------------------------------------------------------------------

var (
	EarthSeasons = Concept{&Entity{
		ID:      "concept-earth-seasons",
		Name:    "Earth's seasonal cycle",
		Kind:    "concept",
		Aliases: []string{"seasons", "seasonal cycle"},
		Brief:   "The yearly cycle by which Earth's climate varies at a given location — the phenomenon that the 'seasons are caused by Earth-Sun distance' misconception is about.",
	}}

	AtmosphericEntryHeating = Concept{&Entity{
		ID:      "concept-atmospheric-entry-heating",
		Name:    "Atmospheric entry heating",
		Kind:    "concept",
		Aliases: []string{"reentry heating", "meteor heating"},
		Brief:   "The mechanism by which objects entering a planetary atmosphere at high speed become intensely hot — the phenomenon that the 'friction-caused reentry heating' misconception is about.",
	}}

	ArtificialStructureSpaceVisibility = Concept{&Entity{
		ID:      "concept-artificial-structure-space-visibility",
		Name:    "Visibility of artificial structures from space",
		Kind:    "concept",
		Brief:   "The question of which, if any, human-made structures can be seen from orbit or from the Moon — the phenomenon that the 'Great Wall is the only human-made object visible from space' misconception is about.",
	}}
)

// -----------------------------------------------------------------------------
// Corrected-fact hypotheses. Each entity's Name is the correction.
// -----------------------------------------------------------------------------

var (
	SeasonsCausedByAxialTilt = Hypothesis{&Entity{
		ID:    "hyp-seasons-axial-tilt",
		Name:  "Seasons are caused by Earth's 23.4-degree axial tilt, not by orbital distance",
		Kind:  "hypothesis",
		Brief: "Each hemisphere is tilted towards the Sun in its respective summer, producing longer days and more direct sunlight; the opposite is true in winter. Earth reaches perihelion (closest to the Sun) in January and aphelion (farthest) in July, so orbital eccentricity slightly opposes the seasonal temperature trend in the Northern Hemisphere. The commonly-held belief that seasons are caused by varying Earth-Sun distance is false, and winze records this as a scientifically-accepted correction tagged with CorrectsCommonMisconception.",
	}}

	AtmosphericEntryHeatFromAdiabaticCompression = Hypothesis{&Entity{
		ID:    "hyp-atmospheric-entry-adiabatic-compression",
		Name:  "Atmospheric-entry heat comes primarily from adiabatic compression of air ahead of the object, not from friction",
		Kind:  "hypothesis",
		Brief: "When a meteor or spacecraft enters the atmosphere at hypersonic speed, the air immediately in front of it is compressed so rapidly that it cannot dissipate heat, and its temperature rises adiabatically into a shock layer that transfers heat to the object. Frictional heating between the object's surface and the airflow is a minor contribution. The commonly-held belief that reentry heating is primarily a friction effect is false.",
	}}

	GreatWallNotUniquelyVisibleFromSpace = Hypothesis{&Entity{
		ID:    "hyp-great-wall-not-uniquely-visible",
		Name:  "The Great Wall of China is not the only human-made object visible from space, and is not visible from the Moon at all",
		Kind:  "hypothesis",
		Brief: "No Apollo astronaut reported seeing any specific human-made object from the Moon, and Earth-orbiting astronauts can see the Great Wall only with magnification. City lights, by contrast, are easily visible on the night side of Earth from orbit. Many other human-made structures are visible from low Earth orbit; the Great Wall is not uniquely distinguished. The commonly-held belief that the Great Wall is the only human-made object visible from space (or visible from the Moon) is false.",
	}}
)

// -----------------------------------------------------------------------------
// Claims. Each corrected-fact hypothesis gets two claims: a TheoryOf
// connecting it to the Concept it is about, and a CorrectsCommonMisconception
// tag marking it as a member of the list of common-misconception corrections.
// -----------------------------------------------------------------------------

var (
	SeasonsHypothesisAboutEarthSeasons = TheoryOf{
		Subject: SeasonsCausedByAxialTilt,
		Object:  EarthSeasons,
		Prov:    misconceptionsSource,
	}

	SeasonsCorrectsMisconception = CorrectsCommonMisconception{
		Subject: SeasonsCausedByAxialTilt,
		Prov:    misconceptionsSource,
	}

	AtmosphericEntryHypothesisAboutEntryHeating = TheoryOf{
		Subject: AtmosphericEntryHeatFromAdiabaticCompression,
		Object:  AtmosphericEntryHeating,
		Prov:    misconceptionsSource,
	}

	AtmosphericEntryCorrectsMisconception = CorrectsCommonMisconception{
		Subject: AtmosphericEntryHeatFromAdiabaticCompression,
		Prov:    misconceptionsSource,
	}

	GreatWallHypothesisAboutSpaceVisibility = TheoryOf{
		Subject: GreatWallNotUniquelyVisibleFromSpace,
		Object:  ArtificialStructureSpaceVisibility,
		Prov:    misconceptionsSource,
	}

	GreatWallCorrectsMisconception = CorrectsCommonMisconception{
		Subject: GreatWallNotUniquelyVisibleFromSpace,
		Prov:    misconceptionsSource,
	}
)
