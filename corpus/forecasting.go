package winze

// Eighth public-corpus ingest: Wikipedia's Forecasting article.
// Paired with the preceding predictive_processing.go (PubMed 23663408,
// Clark 2013) per the user's observation that the two sources are
// closely related but operate on "slightly different time scales" —
// predictive processing is a theory of how brains implicitly predict
// at milliseconds-to-seconds in neural substrate, whereas forecasting
// is the deliberate human practice of making explicit claims about
// future events at days-to-years horizons. Both are claims-about-
// prediction, but they live in completely different schema
// neighbourhoods. The pairing was deliberate: ingesting the Clark
// paper first seeded the cognitive-substrate vocabulary, so the
// Forecasting slice can reason from inside a graph that already
// knows what "prediction" means at one time scale while asserting
// it at another.
//
// Schema forcing functions earned by this slice:
//
//   - None. Zero new predicates, zero new roles, zero new pragmas.
//     This is the second slice in a row (after predictive_processing.go)
//     to earn nothing, and the third overall (the common-misconceptions
//     slice was the first). At three occurrences the "ingest earns no
//     new primitives" pattern is probably worth a name: call it a
//     **vocabulary-fit ingest** — a slice whose discipline-win is
//     that the existing schema already fits the source cleanly, so
//     inventing structure would bloat the predicate graph without
//     yield. The alternative — forcing structure when the source does
//     not commit to it — is exactly the failure mode CorrectsCommonMisconception
//     was designed to name for the misconceptions slice.
//
// Deferred schema surface (not forced by this slice, captured for a
// later ingest that does assert a specific dated forecast):
//
//   - A `Predicts[Person, Event]` or `Predicts[Person, Claim]`
//     predicate, where the Event is in the future and the predicate
//     carries a credence value and a resolution state. Not earned by
//     the Forecasting Wikipedia article itself because the article is
//     a meta-level treatment of forecasting as a concept — it does
//     not assert any specific dated forecast. An ingest that reads
//     Tetlock's Good Judgment Project data, a specific Philip Tetlock
//     2005 "Expert Political Judgment" score, or a FiveThirtyEight
//     political forecast would force this schema. Until then the
//     discipline says don't invent it.
//
//   - A `Credence` value struct (parallel to EnergyReading and
//     EnglishRendering) attaching a probability to a forecast. Same
//     deferral reason: the Wikipedia article talks about credence in
//     the abstract but does not cite a specific numerical credence
//     the ingest could anchor to.
//
//   - A `ResolvedAs[Forecast, Outcome]` functional predicate with a
//     //winze:functional pragma — a forecast has exactly one ground-
//     truth outcome once time passes. This is the first deferred
//     predicate that would explicitly use the functional machinery
//     for a *temporally* resolved claim, which is a shape winze has
//     not seen yet (FormedAt, EnergyEstimate, EnglishTranslationOf
//     are all atemporal in their function: the true value just IS).
//
//   - **Wide-range time-scale representation for resolution dates.**
//     Per user observation 2026-04-11: the Clark/Forecasting pairing
//     reveals that any future Predicts schema will need to span
//     milliseconds (neural prediction-error minimisation), seconds
//     (perceptual prediction), days (weather nowcast), years
//     (economic and political forecasts), and decades+ (climate and
//     technology forecasting) within a single representation. The
//     current TemporalMarker struct (non-entity value type used for
//     Lake Cheko's formation date in tunguska.go) is the obvious
//     candidate to promote, but it was introduced for geological-
//     era-scale claims and its value field is a free-text string
//     rather than a typed-interval. A unified TemporalMarker that
//     carries both a point-or-interval value AND a precision/scale
//     annotation (so a millisecond-level prediction error and a
//     decade-scale climate forecast can coexist without losing
//     information) is the right design point when a specific-forecast
//     ingest forces the predicate family. Until then this is a
//     recorded constraint, not live schema: the design decision will
//     be worth making only when there is a specific forecast with a
//     specific resolution horizon on the other side of it.
//
// Content shape earned:
//
//   - Forecasting/prediction is a polyvalent-term case. Hydrology
//     uses "forecast" for a dated estimate and "prediction" for a
//     general one; general usage collapses them. Mirrors the
//     Nondualism/IsPolyvalentTerm pattern exactly, zero new code.
//
//   - Philip Tetlock's calibration framing is a specific theory of
//     what makes a good forecaster (10% credences hit 10% of the
//     time). Encoded via the existing Proposes + TheoryOf +
//     //winze:contested machinery, which means a future slice that
//     reads a counter-theory (Friedman/Tetlock "hedgehog vs fox",
//     or the Philip Meehl clinical-vs-statistical-prediction
//     tradition) will surface automatically on the contested-concept
//     rule without any touches to the lint binary.
//
//   - Qualitative vs quantitative method taxonomy reuses the
//     BelongsTo pattern from cognitive_biases.go, validating it on a
//     third subject domain (after cognitive bias families and
//     Jean le Flambeur book→series membership). Three occurrences
//     is where BelongsTo becomes a load-bearing cross-ingest predicate
//     rather than a cognitive-biases convenience.

var forecastingSource = Provenance{
	Origin:     "Wikipedia (zim 2025-12) / Forecasting",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "Forecasting is the process of making predictions based on past and present data. Later these can be compared with what actually happens. [...] Usage can vary between areas of application: for example, in hydrology the terms 'forecast' and 'forecasting' are sometimes reserved for estimates of values at certain specific future times, while the term 'prediction' is used for more general estimates, such as the number of times floods will occur over a long period. [...] Risk and uncertainty are central to forecasting and prediction; it is generally considered a good practice to indicate the degree of uncertainty attaching to forecasts. [...] Forecasting can be described as predicting what the future will look like, whereas planning predicts what the future should look like. [...] In Philip E. Tetlock's Superforecasting: The Art and Science of Prediction, he discusses forecasting as a method of improving the ability to make decisions. A person can become better calibrated — i.e. having things they give 10% credence to happening 10% of the time.",
}

// -----------------------------------------------------------------------------
// Meta-concepts and method families.
// -----------------------------------------------------------------------------

var (
	Forecasting = Concept{&Entity{
		ID:      "concept-forecasting",
		Name:    "Forecasting",
		Kind:    "concept",
		Aliases: []string{"forecasts"},
		Brief:   "Process of predicting future events based on historical and current data, distinct from planning (which prescribes what should happen) and budgeting (fixed resource allocation). Incorporates uncertainty quantification as core practice.",
	}}

	Prediction = Concept{&Entity{
		ID:      "concept-prediction",
		Name:    "Prediction",
		Kind:    "concept",
		Aliases: []string{"predictions"},
		Brief:   "Concept of estimating a future or unknown value or event. In hydrology, prediction covers general estimates (e.g., frequency) while forecast denotes specific future times; in general usage the terms overlap.",
	}}

	QualitativeForecasting = Concept{&Entity{
		ID:    "concept-qualitative-forecasting",
		Name:  "Qualitative forecasting",
		Kind:  "concept",
		Brief: "Forecasting method based on expert judgment and subjective opinion, used when historical data is unavailable for intermediate to long-range decisions.",
	}}

	QuantitativeForecasting = Concept{&Entity{
		ID:    "concept-quantitative-forecasting",
		Name:  "Quantitative forecasting",
		Kind:  "concept",
		Brief: "Forecasting methods using formal statistical techniques applied to time-series, cross-sectional, or longitudinal data to generate predictions, contrasting with judgment-based qualitative approaches.",
	}}
)

// -----------------------------------------------------------------------------
// People and hypotheses.
// -----------------------------------------------------------------------------

var (
	PhilipTetlock = Person{&Entity{
		ID:    "philip-tetlock",
		Name:  "Philip E. Tetlock",
		Kind:  "person",
		Brief: "American-Canadian political scientist known for research on expert judgment and forecasting accuracy. Author of Superforecasting: The Art and Science of Prediction.",
	}}

	TetlockCalibrationFraming = Hypothesis{&Entity{
		ID:    "hyp-tetlock-calibration-framing",
		Name:  "Good forecasting is a trainable calibration skill — a well-calibrated forecaster's N%-credence claims are true N% of the time",
		Kind:  "hypothesis",
		Brief: "Forecasting framework positing that calibrated probability assignment—where predicted likelihoods match actual occurrence rates—is a trainable discipline transferable across domains, more important than domain expertise.",
	}}
)

// -----------------------------------------------------------------------------
// Claims.
// -----------------------------------------------------------------------------

var (
	ForecastingIsPolyvalent = IsPolyvalentTerm{
		Subject: Forecasting,
		Prov:    forecastingSource,
	}

	QualitativeForecastingBelongsToForecasting = BelongsTo{
		Subject: QualitativeForecasting,
		Object:  Forecasting,
		Prov:    forecastingSource,
	}
	QuantitativeForecastingBelongsToForecasting = BelongsTo{
		Subject: QuantitativeForecasting,
		Object:  Forecasting,
		Prov:    forecastingSource,
	}

	TetlockProposesCalibrationFraming = Proposes{
		Subject: PhilipTetlock,
		Object:  TetlockCalibrationFraming,
		Prov:    forecastingSource,
	}

	TetlockCalibrationTheoryOfForecasting = TheoryOf{
		Subject: TetlockCalibrationFraming,
		Object:  Forecasting,
		Prov:    forecastingSource,
	}

	// Prediction is wired into the graph via its polyvalent-pair relation
	// to Forecasting — the two concepts exist in winze specifically
	// because the hydrology-vs-general-usage disagreement about their
	// boundary is what the IsPolyvalentTerm tag on Forecasting is
	// referring to. DerivedFrom is the closest fit: Prediction is the
	// more general concept from which the narrower hydrological
	// forecast is distinguished.
	ForecastingDerivedFromPrediction = DerivedFrom{
		Subject: Forecasting,
		Object:  Prediction,
		Prov:    forecastingSource,
	}
)
