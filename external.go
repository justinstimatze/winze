package winze

// External naming oracle (per dec-no-upper-ontology).
//
// This is NOT an imported ontology. It is a lookup table of well-known term
// names from Schema.org, WordNet, and Wikidata class labels, used at ingest
// time by a lint rule that asks: "you invented a new type — is there already
// a standard name for it?" If yes, prefer the standard name before the type
// is promoted out of pending/.
//
// The table is deliberately flat and tiny for v0. It is not a hierarchy and
// carries no semantic commitments beyond "this name exists out there." Add
// entries when a real promotion is blocked by a missing lookup, not
// speculatively.
//
// Source "winze-native" is used when a role type is legitimately invented
// by winze itself and has no clean external match. Registering it here is
// not cheating — it's the difference between "silently invented" (caught
// by the naming oracle as ungrounded) and "explicitly winze-local
// vocabulary" (passes the oracle because it has been acknowledged). The
// oracle's job is to make invention visible, not to force everything to
// be external.

// ExternalTerm records a well-known term winze ingest can defer to.
// Source is the namespace it comes from ("schema.org", "wordnet", "wikidata").
// URL is the canonical page for the term so a human can audit.
type ExternalTerm struct {
	Name   string
	Source string
	URL    string
	Brief  string
}

var ExternalTerms = []*ExternalTerm{
	{Name: "Person", Source: "schema.org", URL: "https://schema.org/Person", Brief: "A human being."},
	{Name: "Organization", Source: "schema.org", URL: "https://schema.org/Organization", Brief: "An organization of any kind."},
	{Name: "Place", Source: "schema.org", URL: "https://schema.org/Place", Brief: "Entities with some fixed physical extent."},
	{Name: "Event", Source: "schema.org", URL: "https://schema.org/Event", Brief: "An event occurring at a specific time and location."},
	{Name: "Role", Source: "schema.org", URL: "https://schema.org/Role", Brief: "Wrapper for a time-bounded role an entity plays (e.g., employment)."},
	{Name: "Product", Source: "schema.org", URL: "https://schema.org/Product", Brief: "Anything that is made available for sale."},
	{Name: "CreativeWork", Source: "schema.org", URL: "https://schema.org/CreativeWork", Brief: "The most generic kind of creative work, including books, movies, photographs, software programs, etc."},
	{Name: "Substance", Source: "wikidata", URL: "https://www.wikidata.org/wiki/Q11344", Brief: "Chemical substance / material."},
	{Name: "Facility", Source: "wordnet", URL: "https://en-word.net/lemma/facility", Brief: "A building or complex built to serve a specific function."},
	{Name: "Instrument", Source: "wordnet", URL: "https://en-word.net/lemma/instrument", Brief: "A device used for measurement, performance, or other craft; any tool whose purpose is to produce a reading or an effect."},
	{Name: "Hypothesis", Source: "wikidata", URL: "https://www.wikidata.org/wiki/Q41719", Brief: "A proposed explanation for a phenomenon. Reified so that proposer/disputant/subject relations can attach."},
	{Name: "Concept", Source: "wikidata", URL: "https://www.wikidata.org/wiki/Q151885", Brief: "An abstract idea or a mental image representing a class of things; a term whose definition is itself the subject of claims. Used for corpus entities that are words/ideas rather than concrete referents (e.g., 'nondualism', 'advaita')."},

	// winze-native vocabulary — legitimately invented for design/authoring
	// prose where no clean external term exists. Registered here so the
	// naming oracle treats invention as explicit, not silent.
	{Name: "DesignLayer", Source: "winze-native", URL: "", Brief: "An interpretive layer of a creative work (Stope Layer 1/2/3). No standard external term."},
	{Name: "Phase", Source: "winze-native", URL: "", Brief: "A production or arc-scoped stage of a creative work. Schema.org has no equivalent at this granularity."},
	{Name: "ProtectedLine", Source: "winze-native", URL: "", Brief: "A specific authored string that is load-bearing across multiple interpretive layers and must not be edited."},
	{Name: "NeverAnswered", Source: "winze-native", URL: "", Brief: "A question the creative work commits to never resolving. The absence is authorial."},
	{Name: "AuthorialPolicy", Source: "winze-native", URL: "", Brief: "A rule applied at authoring time that rejects or revises content matching a condition. Loose match to Schema.org Rule but Rule is not in core."},
	{Name: "Reading", Source: "winze-native", URL: "", Brief: "A specific interpretation of a ProtectedLine at a given DesignLayer or Phase. Reification of a 3-ary relation."},
}
