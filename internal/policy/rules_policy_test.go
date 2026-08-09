package policy

import "testing"

func inside(a Access) bool { return a.Rel != "" || a.Path == "/w" }

// A rule adds a question; it never grants one. Otherwise the boundary would be
// negotiable by configuration, which is the opposite of a boundary.
func TestARuleNeverRescuesSomethingContainmentRefused(t *testing.T) {
	rules := Rules{ConfirmWrite: []string{"**"}, ConfirmRead: []string{"**"}}

	// Read-only refuses a write. A rule matching it cannot turn that into a
	// question the user could say yes to.
	got := Evaluate(
		Request{Tool: "write", Paths: []Access{{Path: "/w/a", Rel: "a", Write: true}}},
		ModeReadOnly, PolicyOnRequest, rules, inside)
	if got.Decision != DecisionDeny {
		t.Errorf("got %s, want deny", got.Decision)
	}

	// A write outside the workspace already escalates on containment, and it
	// must keep saying so rather than being relabelled as a rule.
	got = Evaluate(
		Request{Tool: "write", Paths: []Access{{Path: "/elsewhere", Write: true}}},
		ModeWorkspaceWrite, PolicyOnRequest, rules,
		func(a Access) bool { return a.Rel != "" })
	if got.Boundary != BoundaryFilesystemWrit {
		t.Errorf("containment must answer first, got %s", got.Boundary)
	}
}

// The question the sandbox cannot ask: inside the workspace, some writes have a
// longer tail than others.
func TestARuleAsksInsideTheWorkspace(t *testing.T) {
	rules := DefaultRules()

	ordinary := Evaluate(
		Request{Tool: "write", Paths: []Access{{Path: "/w/src/main.go", Rel: "src/main.go", Write: true}}},
		ModeWorkspaceWrite, PolicyOnRequest, rules, inside)
	if ordinary.Decision != DecisionAllow {
		t.Fatalf("ordinary work must not ask, got %s", ordinary.Decision)
	}

	hook := Evaluate(
		Request{Tool: "write", Paths: []Access{
			{Path: "/w/.git/hooks/pre-commit", Rel: ".git/hooks/pre-commit", Write: true}}},
		ModeWorkspaceWrite, PolicyOnRequest, rules, inside)
	if hook.Decision != DecisionEscalate {
		t.Fatalf("a git hook must ask, got %s", hook.Decision)
	}
	if hook.Rule != ".git/**" {
		t.Errorf("the rule must be named, got %q", hook.Rule)
	}
	// Named so nobody reads a rule as a boundary the sandbox enforced.
	if hook.Boundary != BoundaryPathRuleWrite {
		t.Errorf("got %s", hook.Boundary)
	}
	// Consent to a rule nobody can see is consent to nothing.
	if hook.Reason == "" {
		t.Error("the reason must say what matched")
	}
}

// Reading a secret sends it to the model provider, off this machine — and the
// workspace is exactly where a .env lives.
func TestReadingASecretAsks(t *testing.T) {
	rules := DefaultRules()
	got := Evaluate(
		Request{Tool: "read", Paths: []Access{{Path: "/w/.env", Rel: ".env"}}},
		ModeWorkspaceWrite, PolicyOnRequest, rules, inside)
	if got.Decision != DecisionEscalate || got.Boundary != BoundaryPathRuleRead {
		t.Fatalf("got %s/%s", got.Decision, got.Boundary)
	}
	// And an ordinary read stays silent, or the rule is just noise.
	quiet := Evaluate(
		Request{Tool: "read", Paths: []Access{{Path: "/w/README.md", Rel: "README.md"}}},
		ModeWorkspaceWrite, PolicyOnRequest, rules, inside)
	if quiet.Decision != DecisionAllow {
		t.Errorf("got %s", quiet.Decision)
	}
}

// A command rule pauses; it does not contain. The verdict says which, by its
// boundary name.
func TestACommandRuleAsks(t *testing.T) {
	rules := Rules{ConfirmCommand: []string{"rm -rf*"}}
	got := Evaluate(
		Request{Tool: "bash", Command: "rm -rf build/"},
		ModeWorkspaceWrite, PolicyOnRequest, rules, inside)
	if got.Decision != DecisionEscalate || got.Boundary != BoundaryCommandRule {
		t.Fatalf("got %s/%s", got.Decision, got.Boundary)
	}
	if got.Rule != "rm -rf*" {
		t.Errorf("got %q", got.Rule)
	}
}

// `never` means never asked, and a rule is a question — so the policy axis
// still decides whether anyone is asked at all.
func TestTheApprovalPolicyStillGovernsRules(t *testing.T) {
	rules := DefaultRules()
	req := Request{Tool: "write", Paths: []Access{
		{Path: "/w/.git/config", Rel: ".git/config", Write: true}}}

	if got := Evaluate(req, ModeWorkspaceWrite, PolicyNever, rules, inside); got.Decision == DecisionEscalate {
		t.Errorf("policy `never` asks nobody, got %s", got.Decision)
	}
	if got := Evaluate(req, ModeWorkspaceWrite, PolicyOnRequest, rules, inside); got.Decision != DecisionEscalate {
		t.Errorf("got %s", got.Decision)
	}
}

// full-access still asks, and this is the case where it matters most.
//
// The two axes are orthogonal (ADR-02): full-access is the *sandbox* saying
// everything is possible, and a rule is attention, which lives on the approval
// axis. Someone who set full-access and left the policy at `on-request` asked
// to be consulted — and with nothing else holding the line, the pause before a
// git hook is the only thing left.
//
// Turning the questions off is what `never` is for, and the test below pins
// that it still works.
func TestFullAccessStillAsksWhereARuleFires(t *testing.T) {
	got := Evaluate(
		Request{Tool: "write", Paths: []Access{{Path: "/w/.git/config", Rel: ".git/config", Write: true}}},
		ModeFullAccess, PolicyOnRequest, DefaultRules(), inside)
	if got.Decision != DecisionEscalate {
		t.Errorf("got %s, want escalate", got.Decision)
	}

	// And ordinary work in full-access stays silent, as it always did.
	quiet := Evaluate(
		Request{Tool: "write", Paths: []Access{{Path: "/w/src/main.go", Rel: "src/main.go", Write: true}}},
		ModeFullAccess, PolicyOnRequest, DefaultRules(), inside)
	if quiet.Decision != DecisionAllow {
		t.Errorf("got %s", quiet.Decision)
	}

	// `never` is how the questions are turned off, in any mode.
	off := Evaluate(
		Request{Tool: "write", Paths: []Access{{Path: "/w/.git/config", Rel: ".git/config", Write: true}}},
		ModeFullAccess, PolicyNever, DefaultRules(), inside)
	if off.Decision != DecisionAllow {
		t.Errorf("got %s", off.Decision)
	}
}

// A write and a read in one call is a write as far as the question goes: it is
// the one with the longer tail.
func TestAWriteIsAskedAboutBeforeARead(t *testing.T) {
	rules := Rules{ConfirmWrite: []string{"**/*.go"}, ConfirmRead: []string{"**/*.md"}}
	got := Evaluate(
		Request{Tool: "edit", Paths: []Access{
			{Path: "/w/README.md", Rel: "README.md"},
			{Path: "/w/main.go", Rel: "main.go", Write: true},
		}},
		ModeWorkspaceWrite, PolicyOnRequest, rules, inside)
	if got.Boundary != BoundaryPathRuleWrite {
		t.Errorf("got %s", got.Boundary)
	}
}

// A path outside the workspace has no relative form, and a rule written against
// the workspace must not be tested against it.
func TestARuleIgnoresWhatIsOutsideTheWorkspace(t *testing.T) {
	rules := Rules{ConfirmRead: []string{"**"}}
	got := Evaluate(
		Request{Tool: "read", Paths: []Access{{Path: "/elsewhere/x", Rel: ""}}},
		ModeWorkspaceWrite, PolicyOnRequest, rules,
		func(a Access) bool { return a.Rel != "" })
	// Containment answers, and it answers with its own boundary.
	if got.Boundary == BoundaryPathRuleRead {
		t.Errorf("a workspace rule must not claim something outside it: %+v", got)
	}
}

// A rule is a request for a person's attention. With `never` there is no
// person, so there is no question — and turning an unaskable question into a
// denial would make `never` more restrictive than `on-request`, which is the
// opposite of what the name says.
//
// The sandbox is untouched either way, and the sandbox is what contains.
func TestNeverDoesNotTurnARuleIntoADenial(t *testing.T) {
	req := Request{Tool: "write", Paths: []Access{
		{Path: "/w/.git/config", Rel: ".git/config", Write: true}}}

	got := Evaluate(req, ModeWorkspaceWrite, PolicyNever, DefaultRules(), inside)
	if got.Decision != DecisionAllow {
		t.Fatalf("got %s, want allow", got.Decision)
	}

	// And `never` stays at least as permissive as `on-request` for anything a
	// rule touches, which is the property that was wrong.
	asked := Evaluate(req, ModeWorkspaceWrite, PolicyOnRequest, DefaultRules(), inside)
	if asked.Decision != DecisionEscalate {
		t.Fatalf("setup: on-request should ask, got %s", asked.Decision)
	}

	// What `never` does deny is unchanged: a real crossing with nobody to ask.
	outside := Evaluate(
		Request{Tool: "write", Paths: []Access{{Path: "/elsewhere", Write: true}}},
		ModeWorkspaceWrite, PolicyNever, DefaultRules(),
		func(a Access) bool { return a.Rel != "" })
	if outside.Decision != DecisionDeny {
		t.Errorf("containment with nobody to ask still denies, got %s", outside.Decision)
	}
}
