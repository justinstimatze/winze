package winze

// The user. Encoded as a dogfood test: collaboration-style preferences
// as first-class typed claims. This file demonstrates the pattern for
// encoding meta-KB knowledge (about the KB's maintainer, not the domain).
// starter.go is the template for domain knowledge.
//
// Style claims are UnaryClaim types whose predicate NAME is the content.
// Provenance.Quote holds the specific source fragment so the audit record
// survives even when the source transcript is gone (dec-prose-is-io).

var userSource = Provenance{
	Origin:     "conversation 2026-04-11",
	IngestedAt: "2026-04-11",
	IngestedBy: "winze",
	Quote:      "this is your project now / I definitely don't expect a tidy predicate vocab / continue",
}

var UserAlice = Person{&Entity{
	ID:    "user-alice",
	Name:  "Alice",
	Kind:  "person",
	Brief: "The user and director of this winze instance. See the collaboration-style claims below for durable working-style preferences.",
}}

var (
	UserGrantsBroadAuthority = GrantsBroadAuthorityOverWinze{
		Subject: UserAlice,
		Prov:    userSource,
	}

	UserPrefersTerseResponses = PrefersTerseResponses{
		Subject: UserAlice,
		Prov:    userSource,
	}

	UserPushesBackOnOverengineering = PushesBackOnOverengineering{
		Subject: UserAlice,
		Prov:    userSource,
	}

	UserPrefersOrganicSchemaGrowth = PrefersOrganicSchemaGrowth{
		Subject: UserAlice,
		Prov:    userSource,
	}
)
