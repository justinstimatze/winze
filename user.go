package winze

// The user. Encoded in winze as a dogfood test: winze's thesis says
// markdown-for-persistence is the dodge, and an external memory/
// directory of .md files is exactly that. If winze is a KB substrate,
// session-level working-style knowledge should live here as first-class
// typed claims, not in /home/<user>/.claude/projects/.../memory/*.md.
//
// The assertions below were extracted from the 2026-04-11 founding and
// continuation sessions. Provenance.Quote holds the specific line the
// claim was inferred from so that even when the session transcript is
// gone (sources are transient, dec-prose-is-io) the audit record remains.
//
// This is a small seed slice. Future sessions will add or revise style
// claims as the user's preferences surface in new contexts. Contradictions
// between a new reading and the ones below will be caught by the
// value-conflict lint rule if they share a functional-predicate pattern,
// or by explicit KnownDispute annotations otherwise.

var userSource = Provenance{
	Origin:     "conversation 2026-04-11 / winze founding and continuation sessions",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze session 2 (Tunguska ingest + dogfood memory test)",
	Quote:      "this is your project now / nah bro your pick / obviously if at some point we have feature requests or bug reports or feedback for defn and adit they are also our projects / I definitely don't expect a tidy predicate vocab / continue bro",
}

// The user themselves. Name inferred from filesystem paths
// (/home/justin/Documents/defn, /home/justin/Documents/adit). Update if
// the inference turns out wrong.
var UserJustin = Person{&Entity{
	ID:      "user-justin",
	Name:    "Justin",
	Kind:    "person",
	Aliases: []string{"bro"},
	Brief:   "The user and director of winze. Name inferred from /home/justin/ paths in related project directories. See the collaboration-style claims below for durable working-style facts carried across sessions.",
}}

// -----------------------------------------------------------------------------
// Style claims. Each is a distinct UnaryClaim type (see predicates.go)
// so the predicate type NAME is the content.
// -----------------------------------------------------------------------------

var (
	UserGrantsBroadAuthority = GrantsBroadAuthorityOverWinze{
		Subject: UserJustin,
		Prov:    userSource,
	}

	UserPrefersTerseResponses = PrefersTerseResponses{
		Subject: UserJustin,
		Prov:    userSource,
	}

	UserPushesBackOnOverengineering = PushesBackOnOverengineering{
		Subject: UserJustin,
		Prov:    userSource,
	}

	UserPrefersOrganicSchemaGrowth = PrefersOrganicSchemaGrowth{
		Subject: UserJustin,
		Prov:    userSource,
	}
)
