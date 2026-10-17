package app

import (
	"errors"
	"sync"
	"time"

	"github.com/aguinelo/dcode/internal/loop/qualifier"
)

// MaxProposals is how many times one qualifying turn may propose.
const MaxProposals = 8

// ErrTooManyProposals is a qualifying turn that tried to propose past
// MaxProposals.
var ErrTooManyProposals = errors.New("app: too many proposals")

// Proposal is a definition of done the model derived, recorded and not yet
// written down.
//
// Recorded rather than written because the turn that produces it runs in plan
// mode: working out what you will be measured by is reading, and read-only
// denies every write with no exception. The loop takes it from here — it
// measures the criteria under the boundary the WORK will run under, which is
// also the only place they can actually run, and writes the file.
type Proposal struct {
	Spec      string
	Criteria  []qualifier.Proposed
	Protected []string
	Sequence  int
	At        time.Time
}

// Proposals is where a qualifying session keeps what the model proposed.
type Proposals struct {
	mu      sync.Mutex
	now     func() time.Time
	current *Proposal
}

// Take removes what was recorded and returns it.
func (p *Proposals) Take() ([]*Proposal, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.current
	p.current = nil
	if out == nil {
		return nil, nil
	}
	return []*Proposal{out}, nil
}

func (p *Proposals) put(v *Proposal) error {
	p.mu.Lock()
	p.current = v
	p.mu.Unlock()
	return nil
}
