package loop

import (
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
	"github.com/aguinelo/dcode/internal/provider"
	"github.com/aguinelo/dcode/internal/tools"
)

func networkRequest() policy.Request {
	return policy.Request{Tool: "bash", Command: "make check", Network: true}
}

// A configuration that wired no grant asks rather than assumes.
//
// The zero value of a field is the one place a permission can be granted by
// nobody at all: every caller that forgets it would be silently reaching the
// network. The safe reading of "nothing was said" is never yes.
func TestAnUnwiredGrantAsksRatherThanAssumes(t *testing.T) {
	e, _ := newEngine(t, &scriptedProvider{}, tools.NewRegistry())
	if e.cfg.NetworkGrant != nil {
		t.Fatal("the helper wired a grant; this asserts the absence of one")
	}
	if v := e.evaluate(networkRequest()); v.Decision != policy.DecisionEscalate {
		t.Errorf("an unwired configuration decided %s without asking", v.Decision)
	}
}

// And a wired grant is honoured, so ordinary shell work runs.
func TestAWiredGrantLetsOrdinaryWorkRun(t *testing.T) {
	e, _ := newEngine(t, &scriptedProvider{}, tools.NewRegistry(), func(c *Config) {
		c.NetworkGrant = policy.GrantedNetwork{}
	})
	if v := e.evaluate(networkRequest()); v.Decision != policy.DecisionAllow {
		t.Errorf("a granted network still asked: %s — %s", v.Decision, v.Reason)
	}
}

var _ provider.Provider = (*scriptedProvider)(nil)
