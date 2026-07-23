# Existing predicate families

Full definitions live in `predicates.go`; `winze-query --schema .` prints the
current signatures. This is the grouped reference.

**Attribution:** Proposes, Disputes, ProposesOrg, DisputesOrg
**Acceptance:** Accepts, AcceptsOrg, EarlyFormulationOf
**Theory:** TheoryOf (//winze:contested), HypothesisExplains
**Cross-domain analogy:** StructurallyAnalogousTo — two hypotheses from different clusters with the same epistemic structure (neither explains nor causes the other); symmetric; source-required
**Taxonomy:** BelongsTo, DerivedFrom, IsCognitiveBias, IsPolyvalentTerm, CorrectsCommonMisconception
**Authorship:** Authored, AuthoredOrg, CommentaryOn, AppearsIn
**Fiction:** IsFictionalWork, IsFictional
**Spatial:** LocatedIn, LocatedNear, OccurredAt
**People:** InfluencedBy, WorksFor, AffiliatedWith, InvestigatedBy
**Prediction:** Predicts, Credence, ResolvedAs (//winze:functional)
**Functional (//winze:functional):** FormedAt, EnergyEstimate, EnglishTranslationOf
**Investigation (Tunguska-domain, low corpus usage):** LedExpedition, FundedBy, CausedEvent, Operates, RunsFacility, Released, Contaminates, HoldsContractWith, MonitoredBy, MonitoredByOrg, ShipsSamplesTo
**Audit (KB self-mutation history):** AbsorbedAlternate (`UnaryClaim[*Entity]`, PROV-O alternateOf) — written by `winze-edit merge` to record that an entity was folded into the Subject survivor; absorbed identity lives in `Provenance.Quote`
**User:** GrantsBroadAuthorityOverWinze, PrefersTerseResponses, PushesBackOnOverengineering, PrefersOrganicSchemaGrowth
