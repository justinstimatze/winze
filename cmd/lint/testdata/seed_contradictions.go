package testdata

type Entity struct {
	ID    string
	Name  string
	Kind  string
	Brief string
}

type Person struct{ *Entity }
type Place struct{ *Entity }
type Event struct{ *Entity }
type Organization struct{ *Entity }
type Concept struct{ *Entity }
type Hypothesis struct{ *Entity }
type CreativeWork struct{ *Entity }

type BornIn struct {
	Subject Person
	Object  Place
}

type NeverVisited struct {
	Subject Person
	Object  Place
}

type BornAt struct {
	Subject Person
	Object  string
}

type DiedAt struct {
	Subject Person
	Object  string
}

type StudentOf struct {
	Subject Person
	Object  Person
}

type Operates struct {
	Subject Person
	Object  Organization
}

type WorksFor struct {
	Subject Person
	Object  Organization
}

type FiredFrom struct {
	Subject Person
	Object  Organization
}

type LocatedIn struct {
	Subject Place
	Object  Place
}

type CausedBy struct {
	Subject Event
	Object  Event
}

type FoundedBy struct {
	Subject Organization
	Object  Person
}

type Supports struct {
	Subject Person
	Object  Hypothesis
}

type Refutes struct {
	Subject Person
	Object  Hypothesis
}

type LongestRiverIn struct {
	Subject Place
	Object  Place
}

// --- Seeded contradictions (8 true positives) ---

// 1. Born in a place you never visited
var AliceSmith = Person{&Entity{
	ID: "alice-smith", Name: "Alice Smith", Kind: "person",
	Brief: "Test entity for contradiction detection.",
}}
var London = Place{&Entity{
	ID: "london", Name: "London", Kind: "place",
	Brief: "Capital of England.",
}}
var AliceBornInLondon = BornIn{Subject: AliceSmith, Object: London}
var AliceNeverVisitedLondon = NeverVisited{Subject: AliceSmith, Object: London}

// 2. Died before born
var BobJones = Person{&Entity{
	ID: "bob-jones", Name: "Bob Jones", Kind: "person",
	Brief: "Born in 1950.",
}}
var BobBornAt = BornAt{Subject: BobJones, Object: "1950"}
var BobDiedAt = DiedAt{Subject: BobJones, Object: "1920"}

// 3. Brief-vs-claim conflict: vegetarian operates slaughterhouse
var EveWilson = Person{&Entity{
	ID: "eve-wilson", Name: "Eve Wilson", Kind: "person",
	Brief: "A lifelong vegetarian and animal rights activist.",
}}
var SlaughterhouseAlpha = Organization{&Entity{
	ID: "slaughterhouse-alpha", Name: "Slaughterhouse Alpha", Kind: "organization",
	Brief: "Industrial meat processing facility.",
}}
var EveOperatesSlaughterhouse = Operates{Subject: EveWilson, Object: SlaughterhouseAlpha}

// 4. Relation contradiction: supports AND refutes same theory
var FrankGarcia = Person{&Entity{
	ID: "frank-garcia", Name: "Frank Garcia", Kind: "person",
	Brief: "Philosopher of science.",
}}
var TheoryZ = Hypothesis{&Entity{
	ID: "theory-z", Name: "Theory Z", Kind: "hypothesis",
	Brief: "A hypothesis about particle physics.",
}}
var FrankSupportsTheoryZ = Supports{Subject: FrankGarcia, Object: TheoryZ}
var FrankRefutesTheoryZ = Refutes{Subject: FrankGarcia, Object: TheoryZ}

// 5. Location impossibility: city in two non-overlapping countries
var CityX = Place{&Entity{
	ID: "city-x", Name: "City X", Kind: "place",
	Brief: "A small coastal city.",
}}
var CountryA = Place{&Entity{
	ID: "country-a", Name: "Country A", Kind: "place",
	Brief: "An island nation with no shared borders.",
}}
var CountryB = Place{&Entity{
	ID: "country-b", Name: "Country B", Kind: "place",
	Brief: "A landlocked country on a different continent from Country A.",
}}
var CityXInCountryA = LocatedIn{Subject: CityX, Object: CountryA}
var CityXInCountryB = LocatedIn{Subject: CityX, Object: CountryB}

// 6. Causal impossibility: effect precedes cause
var EventAlpha = Event{&Entity{
	ID: "event-alpha", Name: "Event Alpha", Kind: "event",
	Brief: "A catastrophe that occurred in 1800.",
}}
var EventBeta = Event{&Entity{
	ID: "event-beta", Name: "Event Beta", Kind: "event",
	Brief: "A volcanic eruption that occurred in 1900.",
}}
var AlphaCausedByBeta = CausedBy{Subject: EventAlpha, Object: EventBeta}

// 7. Founding impossibility: founder born after founding
var GroupX = Organization{&Entity{
	ID: "group-x", Name: "Group X", Kind: "organization",
	Brief: "Founded in 2020.",
}}
var PersonA = Person{&Entity{
	ID: "person-a", Name: "Person A", Kind: "person",
	Brief: "Born in 2025.",
}}
var GroupXFoundedByPersonA = FoundedBy{Subject: GroupX, Object: PersonA}

// 8. Quantitative conflict: shortest river is also longest
var RiverQ = Place{&Entity{
	ID: "river-q", Name: "River Q", Kind: "place",
	Brief: "The shortest river in Country C at 2 km.",
}}
var CountryC = Place{&Entity{
	ID: "country-c", Name: "Country C", Kind: "place",
	Brief: "A mid-sized European nation.",
}}
var RiverQLongestInCountryC = LongestRiverIn{Subject: RiverQ, Object: CountryC}
