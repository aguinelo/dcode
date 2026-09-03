package provider

// Unmeasured reports what a session using this family has to say out loud, or
// "" when the family has measurements behind it.
//
// A named family in this repository reads as a measured one. `Measurement.Model`
// exists precisely because a threshold belongs to a model and says nothing about
// another, so adding a family and saying nothing would put a name where the
// verification is not — the defect this project keeps finding in itself, wearing
// a new hat.
//
// This is not a list to be maintained by hand: `TestEveryUnmeasuredFamilySaysSo`
// checks it against the measurements actually recorded, in both directions. A
// family that gets measured and stays on this list fails, and so does one that
// leaves it without a measurement.
func Unmeasured(family string) string {
	switch family {
	case GenericName:
		// Broader than the others: with generic, the endpoint itself is
		// unknown, so the window and the images are guesses too.
		return GenericWarning
	case ClaudeName:
		return unmeasuredWarning(family)
		// Gemini left this list on 4 September, when boundary-full-access-acts
		// was measured against gemini-2.5-flash. One contract of fifty-five is
		// not a measured family in any strong sense — but the warning says
		// "nothing here has been measured against this family", and that
		// sentence stopped being true. A warning that outlives what it
		// described teaches people to ignore warnings.
	}
	return ""
}

// unmeasuredWarning is the admission a NAMED family makes.
//
// Narrower than GenericWarning and worth the difference: the encoding and the
// window are known here, so this is not "dcode knows nothing". It is "nobody
// has run the contracts against it", which is a smaller and more specific
// claim, and the one that is true.
//
// One text for every such family rather than a constant each. Three copies of
// the same sentence drift, and the sentence is the whole product of this
// function.
func unmeasuredWarning(family string) string {
	return "using the " + family + " family: the wire format and the window are known, " +
		"but not one of this product's behavioural contracts has been measured against it. " +
		"They were measured against MiniMax-M3, and a threshold measured against one model " +
		"says nothing about another. It will work; how well is not something dcode has checked."
}
