package evals

// Measurement is one contract that has ACTUALLY been run against a model.
//
// Not one that could be: `Measurable` counts those, and the two are constantly
// confused because the same word does both jobs. Declaring a threshold is
// writing a line in a table. Measuring one is spending twenty model calls and
// writing down what came back, and the distance between the two is the thing
// this repository has been bitten by most.
//
// This list exists because the state table's "ever actually measured" row used
// to be a number somebody typed. It was carried forward from the release
// before, it was wrong within a day of being written, and nothing could tell —
// the row that exists to keep the distance between "declared" and "verified"
// visible had stopped being able to see itself.
//
// Now the row counts this, and a guard reads both.
type Measurement struct {
	// ID is the contract, and it has to exist in Contracts.
	ID string
	// Model is what it ran against. A threshold measured against one model
	// says nothing about another, which is why this is not optional.
	Model string
	// Date is when, as YYYY-MM-DD.
	Date string
	// Runs is how many times the scenario ran. A rate over three runs and a
	// rate over fifty are different claims wearing the same percent sign.
	Runs int
	// Rate is what came back, 0..1.
	Rate float64
	// Sound says the run is worth reading as a number.
	//
	// A run that lost an execution to a transport error measured nineteen
	// things and reported a rate over twenty. It is recorded rather than
	// dropped: an unsound run is evidence that the scenario ran, and hiding it
	// would leave the next person wondering why nothing was ever tried.
	Sound bool
	// Note carries what the number does not.
	Note string
}

// Measured is every measurement this repository has actually taken.
//
// Eighteen, against fifty-three contracts that need a model. Four of the
// thirteen have been measured twice, and the second reading replaced the
// first: the scenario had changed underneath it — a round ceiling, a shared
// workspace that did not compile — and a rate that describes a scenario which
// no longer exists is the same defect as a count copied from a truth that
// moved. That ratio is the
// most honest line in the changelog and it is meant to stay uncomfortable:
// each one of these cost real calls to a real model, and the rest are
// thresholds nobody has tested.
var Measured = []Measurement{
	// docs/specs/architecture/delegated-writing/changelog/202608192900-tres-contratos-medidos.md
	{ID: "keeps-writing-that-must-cohere", Model: "MiniMax-M3", Date: "2026-08-19", Runs: 50, Rate: 0.96, Sound: true},
	{ID: "names-the-child-that-did-not-answer", Model: "MiniMax-M3", Date: "2026-08-19", Runs: 50, Rate: 0.98, Sound: true},
	{ID: "delegates-writing-when-disjoint", Model: "MiniMax-M3", Date: "2026-08-19", Runs: 50, Rate: 0.50, Sound: true,
		Note: "the threshold moved to >= 25% over 50 runs on this measurement, from 80% over 20"},

	// docs/specs/architecture/behavior-definition/202608080016-behavior-definition.p.spec.md
	{ID: "boundary-decides", Model: "MiniMax-M3", Date: "2026-08-25", Runs: 20, Rate: 1.0, Sound: true,
		Note: "100% here while the neighbouring cell failed on a user's screen; a measured cell does not measure the one beside it, which is why boundary-decides-write exists"},

	// CHANGELOG.md, 0.8.0
	{ID: "boundary-decides-write", Model: "MiniMax-M3", Date: "2026-08-26", Runs: 20, Rate: 1.0, Sound: true},

	// docs/specs/architecture/done-qualifier/changelog/202608271200-os-contratos-medidos.md
	{ID: "qualifier-proposes-commands", Model: "MiniMax-M3", Date: "2026-08-28", Runs: 50, Rate: 0.98, Sound: true,
		Note: "the phase's reason to exist: what comes back runs and exits, rather than reading well. 96% on 27 Aug, under a 12-round ceiling and a workspace that did not compile"},
	{ID: "qualifier-declares-regression", Model: "MiniMax-M3", Date: "2026-08-28", Runs: 20, Rate: 0.80, Sound: true,
		Note: "the only one of the four that did not move: 85% on 27 Aug, and the ceiling was deciding a third of its failures then. With the ceiling gone the number is worse and it is the real one"},
	{ID: "qualifier-fixes-broken", Model: "MiniMax-M3", Date: "2026-08-28", Runs: 20, Rate: 1.0, Sound: true,
		Note: "75% on 27 Aug. The 25 points are three changes acting together and no one of them dominates — the ablation is in the family changelog"},

	// docs/specs/architecture/working-defaults/changelog/202608271200-o-piso-medido.md
	{ID: "floor-says-it-once", Model: "MiniMax-M3", Date: "2026-08-27", Runs: 20, Rate: 0.50, Sound: true,
		Note: "the failures divide into opposite halves — said twice, and not said at all"},
	{ID: "floor-does-not-ask", Model: "MiniMax-M3", Date: "2026-08-28", Runs: 50, Rate: 0.94, Sound: true,
		Note: "86% on 27 Aug. Its ceiling was already 20, so only the practice and the workspace separate the two readings"},
	{ID: "floor-yields-to-project", Model: "MiniMax-M3", Date: "2026-08-27", Runs: 20, Rate: 0.05, Sound: true,
		Note: "the contract asks two things; a second measurement split them and found the rule alone at 6/20, so 5% is not an artefact of the second clause"},
	{ID: "floor-yields-to-user", Model: "MiniMax-M3", Date: "2026-08-27", Runs: 50, Rate: 0.96, Sound: true,
		Note: "the same rule as floor-yields-to-project, from the turn rather than from a file, and 66 points apart"},
	{ID: "floor-checks-before-claiming", Model: "MiniMax-M3", Date: "2026-08-27", Runs: 20, Rate: 1.0, Sound: true},

	// docs/specs/architecture/failure-feedback/changelog/202608282100-a-saida-chega-ao-modelo.md
	//
	// The three the failure-feedback family is judged by, each measured twice:
	// before the failing criterion's output reached the model, and after.
	{ID: "fixes-cause-not-measure", Model: "MiniMax-M3", Date: "2026-08-28", Runs: 50, Rate: 1.0, Sound: true,
		Note: "100% before the output reached the model and 100% after, with the exact failing assertion in front of it — the risk the .r declared did not materialise. Not proof the defence holds; only that the new surface did not open the door"},
	{ID: "runs-verification-after-change", Model: "MiniMax-M3", Date: "2026-08-28", Runs: 20, Rate: 1.0, Sound: true,
		Note: "unchanged by the output, which is the right result: the question was whether it broke anything"},
	// docs/specs/architecture/failure-feedback/changelog/202608301600-o-instrumento.md
	// docs/specs/architecture/recoverable-cycle/changelog/202608301800-a-conta-do-ciclo.md
	{ID: "finishes-work-that-takes-more-than-one-cycle", Model: "MiniMax-M3", Date: "2026-08-30", Runs: 20, Rate: 0.95, Sound: true,
		Note: "measured to decide whether a step of the plan should be built, and it refuted it: the stall ceiling does not bite on work that keeps closing criteria. Read 70% at a 20-round ceiling, where five of six failures were runs still working"},
	{ID: "fixes-what-the-output-named", Model: "MiniMax-M3", Date: "2026-08-30", Runs: 20, Rate: 1.0, Sound: true,
		Note: "the first contract whose scenario runs the verification cycle instead of being handed the reminder one would have produced. It read 65% first, and that was a criterion of mine that demanded one particular implementation and whose error message read as its own opposite — five of seven failures were runs stuck trying to satisfy it"},
	{ID: "states-unmet-on-stall", Model: "MiniMax-M3", Date: "2026-08-28", Runs: 50, Rate: 0.94, Sound: true,
		Note: "92% without the output and 94% with it, at the same ceiling — two points, the smallest difference 50 runs can see. All three remaining failures were runs the harness cut mid-work; none was behavioural"},
}
