package tools

import (
	"os"
	"sort"
)

// snapshot is a file as it stood before a turn touched it.
type snapshot struct {
	content []byte
	existed bool
}

// BeginTurn starts a new set of undoable changes.
//
// A new turn replaces the old set rather than adding to it. Keeping them would
// let one undo reach back through work the person already read and accepted,
// and "undo" that goes further than the last thing is not undo.
func (s *State) BeginTurn() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snaps = map[string]snapshot{}
}

// Snapshot records how a file stood before this turn changes it.
//
// Called by the tools that write, before they write. Only the first call per
// path per turn is kept: the first is what the turn started from, and a later
// one would record a state the turn itself produced.
//
// A file that is not there is recorded as not there, which is what makes
// undoing a creation mean removing it.
func (s *State) Snapshot(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snaps == nil {
		s.snaps = map[string]snapshot{}
	}
	if _, ok := s.snaps[path]; ok {
		return
	}
	body, err := os.ReadFile(path)
	if err != nil {
		s.snaps[path] = snapshot{existed: false}
		return
	}
	s.snaps[path] = snapshot{content: body, existed: true}
}

// BeginCycle marks a point inside a turn that UndoCycle can come back to.
//
// A turn holds many verification cycles, and BeginTurn's snapshot set spans all
// of them. Undoing at turn scope after one bad cycle would throw away every
// good cycle that came before it, which is why the loop could not be given the
// undo it already had.
//
// It takes a SECOND layer of snapshots — how each already-touched path stands
// right now — and leaves the turn's own layer alone. One layer was tried and is
// wrong: a path written in cycle one and again in cycle two would keep cycle
// two's content, so the bad cycle's write survived the undo of the bad cycle.
// Two layers because they answer different questions: the turn's is "how did
// this look before the model started", the cycle's is "how did it look before
// this attempt".
func (s *State) BeginCycle() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cycleSnaps = make(map[string]snapshot, len(s.snaps))
	for p := range s.snaps {
		body, err := os.ReadFile(p)
		if err != nil {
			s.cycleSnaps[p] = snapshot{existed: false}
			continue
		}
		s.cycleSnaps[p] = snapshot{content: body, existed: true}
	}
	s.cycleWrote = map[string]struct{}{}
}

// UndoCycle puts back what changed since the last BeginCycle.
//
// Only what this cycle wrote. A path the turn touched earlier and this cycle
// left alone is not restored, because there is nothing about it to undo.
//
// Without a BeginCycle it undoes nothing and says so by returning nothing. A
// cycle boundary nobody marked is not a boundary, and guessing one would undo
// the whole turn under a name that promises less.
func (s *State) UndoCycle() (restored, refused []string, err error) {
	s.mu.Lock()
	if s.cycleWrote == nil {
		s.mu.Unlock()
		return nil, nil, nil
	}
	// The turn's layer stays where it is; this builds the cycle's view and
	// hands it to the same restore the person's undo uses.
	turn := s.snaps
	cycle := map[string]snapshot{}
	for p := range s.cycleWrote {
		if sn, ok := s.cycleSnaps[p]; ok {
			cycle[p] = sn
			continue
		}
		// First touched in this cycle: the turn's snapshot IS the cycle's, and
		// for a file created here it records that there was nothing.
		if sn, ok := turn[p]; ok {
			cycle[p] = sn
		}
	}
	s.snaps = cycle
	s.mu.Unlock()

	restored, refused, err = s.Undo()

	s.mu.Lock()
	s.snaps = turn
	s.cycleSnaps = nil
	s.cycleWrote = nil
	s.mu.Unlock()
	return restored, refused, err
}

// Undo puts back what the last turn changed, and reports what it would not
// touch.
//
// A file that changed on disk after the turn left it is refused, never
// overwritten. Undoing over somebody's own edit would throw away their work to
// restore something older, which is the opposite of what undo is for — and it
// is the same invariant `edit` already enforces when it refuses a file that
// moved under it.
//
// Refusing is per file rather than all-or-nothing. Seven files changed and one
// edited by hand should still give six back, and saying which one did not go is
// more useful than refusing everything on account of it.
func (s *State) Undo() (restored, refused []string, err error) {
	s.mu.Lock()
	snaps := make(map[string]snapshot, len(s.snaps))
	for p, sn := range s.snaps {
		snaps[p] = sn
	}
	files := make(map[string]fileState, len(s.files))
	for p, f := range s.files {
		files[p] = f
	}
	s.mu.Unlock()

	for path, sn := range snaps {
		if !changedByTheTurn(path, files) {
			continue
		}
		if movedSince(path, files) {
			refused = append(refused, path)
			continue
		}
		if !sn.existed {
			if rerr := os.Remove(path); rerr != nil && !os.IsNotExist(rerr) {
				return restored, refused, rerr
			}
			restored = append(restored, path)
			continue
		}
		if werr := os.WriteFile(path, sn.content, 0o644); werr != nil {
			return restored, refused, werr
		}
		restored = append(restored, path)
	}

	// Sorted so two runs over the same turn report in the same order, which is
	// what makes the list readable and the tests honest.
	sort.Strings(restored)
	sort.Strings(refused)

	s.mu.Lock()
	s.snaps = map[string]snapshot{}
	s.mu.Unlock()
	return restored, refused, nil
}

// changedByTheTurn reports whether the turn actually wrote this path.
//
// A snapshot is taken before a write is attempted, and an attempt can fail — a
// denied path, a match that was ambiguous. Undoing what was never changed would
// rewrite a file for no reason and count it as work undone.
func changedByTheTurn(path string, files map[string]fileState) bool {
	_, ok := files[path]
	return ok
}

// movedSince reports whether the file differs from what the turn left.
func movedSince(path string, files map[string]fileState) bool {
	body, err := os.ReadFile(path)
	if err != nil {
		// Gone. Restoring over an absence is not clobbering anything, and a
		// turn that created a file somebody then deleted is already undone.
		return !os.IsNotExist(err)
	}
	return hashOf(string(body)) != files[path].hash
}

// Adopt takes over what a delegated child recorded, so the turn that asked for
// the work is the turn that can undo it.
//
// Undo is per turn and delegation happens inside one. Without this the parent's
// undo would reach everything except the part it delegated, and undoing half of
// a division of work leaves a tree nobody designed.
//
// Three things move, and each for its own reason:
//
//   - the snapshots, so there is something to put back. The parent's own
//     snapshot wins where both have one: the first of the turn is what the turn
//     started from, and the child's later one records a state the turn itself
//     produced.
//   - the file records, because undo refuses a file that moved since the turn
//     left it, and that judgement needs the hash of what was actually written.
//     Here the child's wins, because the child wrote last and the disk agrees
//     with it.
//   - the written set, because "did this session change anything" is the fact
//     the definition of done reads, and work done by a child is still work done.
func (s *State) Adopt(child *State) {
	if child == nil {
		return
	}

	child.mu.Lock()
	snaps := make(map[string]snapshot, len(child.snaps))
	for p, sn := range child.snaps {
		snaps[p] = sn
	}
	files := make(map[string]fileState, len(child.files))
	for p, f := range child.files {
		files[p] = f
	}
	written := make([]string, 0, len(child.written))
	for p := range child.written {
		written = append(written, p)
	}
	child.mu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snaps == nil {
		s.snaps = map[string]snapshot{}
	}
	for p, sn := range snaps {
		if _, ok := s.snaps[p]; ok {
			continue
		}
		s.snaps[p] = sn
	}
	if s.files == nil {
		s.files = map[string]fileState{}
	}
	for p, f := range files {
		s.files[p] = f
	}
	if s.written == nil {
		s.written = map[string]struct{}{}
	}
	for _, p := range written {
		if _, ok := s.written[p]; !ok {
			s.writeSeq++
		}
		s.written[p] = struct{}{}
	}
}
