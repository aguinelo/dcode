package policy

import "testing"

// A command that leaves the machine is worth a question for a reason none of
// the others share: the sandbox does not go with it.
//
// Everything else on the list destroys work or publishes irreversibly, and the
// sandbox still bounds where it can happen. `ssh host 'systemctl stop postgres'`
// happens somewhere containment has no reach at all — so on that side, the
// question IS the whole mechanism, and it has to fire on the crossing rather
// than on what is being done over there.
func TestACommandThatLeavesTheMachineAsks(t *testing.T) {
	rules := DefaultRules()

	for _, cmd := range []string{
		"ssh deploy@prod 'systemctl restart nginx'",
		"ssh prod uptime",
		"scp build.tar deploy@prod:/srv/",
		"rsync -a dist/ deploy@prod:/srv/www/",
		"kubectl exec -it api-0 -- sh",
		"ansible-playbook deploy.yml",
		"aws ssm start-session --target i-0123",
		"docker -H tcp://10.0.0.5:2375 ps",
	} {
		if _, ok := rules.MatchCommand(cmd); !ok {
			t.Errorf("a command that leaves the machine must ask: %q", cmd)
		}
	}
}

// And the ordinary work of a repository does not, including the git commands
// that reach a remote through ssh without being remote execution themselves.
//
// The rule reads the command that was declared, not what its subprocesses do.
// Saying so is the point: this is attention, and attention that fires on every
// push is attention nobody reads.
func TestOrdinaryWorkStillDoesNotAsk(t *testing.T) {
	rules := DefaultRules()

	for _, cmd := range []string{
		"git push origin main",
		"git pull",
		"go test ./...",
		"make check",
		"npm run build",
		"cat internal/policy/rules.go",
		// Named for the shape, not the word: a file with ssh in its name is
		// not a connection to anywhere.
		"cat ~/.ssh/config",
		"grep ssh internal/sandbox/sockets.go",
	} {
		if _, ok := rules.MatchCommand(cmd); ok {
			t.Errorf("ordinary work must not ask: %q", cmd)
		}
	}
}
