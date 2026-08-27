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
// Five, against forty-three contracts that need a model. That ratio is the
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
}
