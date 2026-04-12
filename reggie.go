package winze

// First real ingest target: npc-reggie-tsosie.md from the Stope corpus.
// Hand-encoded by the founding session as ground truth for what an ingest
// worker should eventually produce automatically.
//
// v0.2: entities are declared directly in their role type so predicate
// slots are compile-checked. The previous v0 used raw *Entity and let a
// Church-Rock-Spill-as-Subject-of-WorksFor claim build, which surfaced
// finding #1 (BinaryRelation[*Entity, *Entity] was too loose).

var reggieSource = Provenance{
	Origin:     "Stope reference corpus / npc-reggie-tsosie.md",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze founding session",
	Quote:      "Junior environmental field monitoring technician. Works for a small subcontracting firm (3-4 people) that holds a monitoring subcontract from what the paperwork describes as 'Hasper Land Holdings / Environmental Compliance Division.'",
}

// -----------------------------------------------------------------------------
// Entities referenced by the Reggie doc, declared in their primary roles.
// -----------------------------------------------------------------------------

var (
	Reggie = Person{&Entity{
		ID:    "reggie-tsosie",
		Name:  "Reggie Tsosie",
		Kind:  "character",
		Brief: "Junior environmental field monitoring technician working a quarterly Blue Mountains water-quality site outside Quamash. Navajo Nation background; reads landscapes for contamination the way other people read weather.",
	}}

	ReggieEmployer = Organization{&Entity{
		ID:    "reggie-employer",
		Name:  "Reggie's subcontracting firm",
		Kind:  "organization",
		Brief: "Small 3-4 person subcontracting firm that holds the Hasper monitoring subcontract. Employs Reggie. Company truck is an F-250.",
	}}

	HasperLandHoldings = Organization{&Entity{
		ID:      "hasper-land-holdings",
		Name:    "Hasper Land Holdings / Environmental Compliance Division",
		Kind:    "organization",
		Aliases: []string{"Hasper Land Holdings", "Hasper"},
		Brief:   "The nominal client on Reggie's monitoring paperwork. The player eventually learns what it actually is; Reggie's response when told is recognition, not surprise.",
	}}

	BlueMountainsSite = Place{&Entity{
		ID:      "wq-site-01",
		Name:    "WQ-site-01",
		Kind:    "place",
		Aliases: []string{"Blue Mountains monitoring site", "the site"},
		Brief:   "The single monitoring location in the Blue Mountains outside Quamash that Reggie visits quarterly. ~40 miles of mixed highway and dirt road from the nearest motel town.",
	}}

	Quamash = Place{&Entity{
		ID:    "quamash",
		Name:  "Quamash",
		Kind:  "place",
		Brief: "Stope's setting town. The Blue Mountains monitoring site is roughly 40 miles out.",
	}}

	JohnDay = Place{&Entity{
		ID:    "john-day",
		Name:  "John Day",
		Kind:  "place",
		Brief: "The town where Reggie drops FedEx overnight shipments of samples. Drop-off closes at 5pm.",
	}}

	PortlandLab = Organization{&Entity{
		ID:    "portland-lab",
		Name:  "Portland analytical lab",
		Kind:  "organization",
		Brief: "The analytical lab Reggie ships samples to. Returns data packages in 10-14 days.",
	}}

	EXO2 = Instrument{&Entity{
		ID:      "ysi-exo2",
		Name:    "YSI EXO2 multiparameter sonde",
		Kind:    "instrument",
		Aliases: []string{"EXO2", "the sonde"},
		Brief:   "Standard field instrument. Measures temperature, specific conductance, dissolved oxygen, pH, turbidity, depth. Bluetooth download.",
	}}

	AquaTROLL = Instrument{&Entity{
		ID:    "aqua-troll-500",
		Name:  "In-Situ Aqua TROLL 500",
		Kind:  "instrument",
		Brief: "Pressure transducer / data logger deployed in a monitoring well. Optical-cable download. Continuous water level and temperature.",
	}}

	OrionStar = Instrument{&Entity{
		ID:    "orion-star-a329",
		Name:  "Orion Star A329",
		Kind:  "instrument",
		Brief: "Handheld pH/conductivity/DO meter. Used for field blanks and cross-checks.",
	}}

	TCProbe = Instrument{&Entity{
		ID:      "tc-probe",
		Name:    "TC probe",
		Kind:    "instrument",
		Aliases: []string{"the TC probe", "fourth instrument"},
		Brief:   "The mysterious fourth instrument. Reggie logs its readings per a three-page calibration procedure without knowing what TC measures. Supervisor said 'just log it, it's what the client wants.'",
	}}

	ChurchRockMill = Facility{&Entity{
		ID:    "church-rock-mill",
		Name:  "Church Rock uranium mill",
		Kind:  "facility",
		Brief: "Uranium mill operated by United Nuclear Corporation. Its tailings dam breached July 16 1979.",
	}}

	ChurchRockSpill = Event{&Entity{
		ID:    "church-rock-spill",
		Name:  "Church Rock uranium mill spill",
		Kind:  "event",
		Brief: "July 16 1979 breach of United Nuclear Corporation's Church Rock tailings dam. 93 million gallons of radioactive wastewater into the Rio Puerco. Largest release of radioactive material in US history by volume; received a fraction of the Three Mile Island coverage the same year.",
	}}

	CRUMP = Organization{&Entity{
		ID:      "crump",
		Name:    "Church Rock Uranium Monitoring Project",
		Kind:    "organization",
		Aliases: []string{"CRUMP"},
		Brief:   "Community-initiated monitoring project, 2003-present. Started because waiting for federal response meant waiting. The thing Reggie actually knows about monitoring — not a federal program.",
	}}

	UnitedNuclear = Organization{&Entity{
		ID:    "united-nuclear-corp",
		Name:  "United Nuclear Corporation",
		Kind:  "organization",
		Brief: "Operator of the Church Rock mill whose tailings dam breached in 1979.",
	}}

	RioPuerco = Place{&Entity{
		ID:    "rio-puerco",
		Name:  "Rio Puerco",
		Kind:  "place",
		Brief: "The river contaminated downstream by the Church Rock spill.",
	}}

	ChurchRockArea = Place{&Entity{
		ID:    "church-rock-area",
		Name:  "Church Rock area, Navajo Nation",
		Kind:  "place",
		Brief: "Site of the 1979 uranium mill spill and, starting 2003, of the community-initiated CRUMP monitoring project.",
	}}

	RadioactiveWastewater = Substance{&Entity{
		ID:    "radioactive-wastewater",
		Name:  "radioactive wastewater",
		Kind:  "substance",
		Brief: "Mill-tailings slurry released in the Church Rock spill. Substance is now a first-class role type.",
	}}
)

// -----------------------------------------------------------------------------
// Temporal markers.
// -----------------------------------------------------------------------------

var (
	Era1979    = &TemporalMarker{Era: "1979-07-16"}
	EraCRUMP   = &TemporalMarker{Era: "2003-present"}
	EraPresent = &TemporalMarker{Era: "game-present"}
)

// -----------------------------------------------------------------------------
// Claims extracted from the Reggie doc. Each is a named var of a specific
// predicate type so defn sees it as a first-class entry in the graph.
// -----------------------------------------------------------------------------

var (
	ReggieWorksForFirm = WorksFor{
		Subject: Reggie,
		Object:  ReggieEmployer,
		When:    EraPresent,
		Prov:    reggieSource,
	}

	FirmContractsWithHasper = HoldsContractWith{
		Subject: ReggieEmployer,
		Object:  HasperLandHoldings,
		When:    EraPresent,
		Prov:    reggieSource,
	}

	SiteNearQuamash = LocatedNear{
		Subject: BlueMountainsSite,
		Object:  Quamash,
		Prov:    reggieSource,
	}

	SiteMonitoredByReggie = MonitoredBy{
		Subject: BlueMountainsSite,
		Object:  Reggie,
		When:    EraPresent,
		Prov:    reggieSource,
	}

	ReggieOperatesEXO2      = Operates{Subject: Reggie, Object: EXO2, When: EraPresent, Prov: reggieSource}
	ReggieOperatesAquaTROLL = Operates{Subject: Reggie, Object: AquaTROLL, When: EraPresent, Prov: reggieSource}
	ReggieOperatesOrionStar = Operates{Subject: Reggie, Object: OrionStar, When: EraPresent, Prov: reggieSource}
	ReggieOperatesTCProbe   = Operates{Subject: Reggie, Object: TCProbe, When: EraPresent, Prov: reggieSource}

	UNCRunsChurchRockMill = RunsFacility{
		Subject: UnitedNuclear,
		Object:  ChurchRockMill,
		When:    Era1979,
		Prov:    reggieSource,
	}

	ChurchRockMillCausedSpill = CausedEvent{
		Subject: ChurchRockMill,
		Object:  ChurchRockSpill,
		When:    Era1979,
		Prov:    reggieSource,
	}

	SpillReleasedWastewater = Released{
		Subject: ChurchRockSpill,
		Object:  RadioactiveWastewater,
		When:    Era1979,
		Prov:    reggieSource,
	}

	// Orphan-cleanup claims added after the first orphan-report lint run
	// flagged CRUMP, JohnDay, PortlandLab, RioPuerco, EraCRUMP, and
	// ChurchRockArea as created-but-never-claimed-about.

	SiteNearJohnDay = LocatedNear{
		Subject: BlueMountainsSite,
		Object:  JohnDay,
		Prov:    reggieSource,
	}

	ReggieShipsSamplesToPortland = ShipsSamplesTo{
		Subject: Reggie,
		Object:  PortlandLab,
		When:    EraPresent,
		Prov:    reggieSource,
	}

	SpillContaminatedRioPuerco = Contaminates{
		Subject: ChurchRockSpill,
		Object:  RioPuerco,
		When:    Era1979,
		Prov:    reggieSource,
	}

	ChurchRockAreaMonitoredByCRUMP = MonitoredByOrg{
		Subject: ChurchRockArea,
		Object:  CRUMP,
		When:    EraCRUMP,
		Prov:    reggieSource,
	}
)
