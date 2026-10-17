package loop

// ContinueTurns numbers the next turn after n, for an engine built for a
// session reopened from its record. A reopened session keeps its id, so it
// keeps its turn ids too: starting again at t1 would give two turns of one
// log the same name, and an approval id is built from it.
//
// Called before the first turn, never during one; n below the count already
// reached is ignored, since numbering backwards is the collision this avoids.
func (e *Engine) ContinueTurns(n int) {
	if n > e.turnSeq {
		e.turnSeq = n
	}
}
