package credential

import (
	"errors"
	"strings"
	"testing"
)

// fakeRun records what the backend asked the OS to do, without a keychain.
type fakeRun struct {
	calls [][]string
	out   []byte
	err   error
}

func (f *fakeRun) run(name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	return f.out, f.err
}

// A name is validated before it reaches a command line, in every operation.
// Without it, a crafted name becomes extra arguments to the keychain tool.
func TestEveryKeychainOperationValidatesTheNameFirst(t *testing.T) {
	bad := []string{"", "  ", "a b", "a\nb", "--wat", "a/b", strings.Repeat("x", 300)}

	for _, name := range bad {
		t.Run(name, func(t *testing.T) {
			m := &macKeychain{bin: "security"}
			f := &fakeRun{}
			m.run = f.run
			if err := m.Set(name, "s"); err == nil {
				t.Error("mac Set accepted it")
			}
			if err := m.Delete(name); err == nil {
				t.Error("mac Delete accepted it")
			}
			if _, err := m.Get(name); err == nil {
				t.Error("mac Get accepted it")
			}

			s := &secretTool{bin: "secret-tool"}
			s.run = f.run
			if err := s.Delete(name); err == nil {
				t.Error("secret-tool Delete accepted it")
			}
			if _, err := s.Get(name); err == nil {
				t.Error("secret-tool Get accepted it")
			}

			if len(f.calls) != 0 {
				t.Errorf("a rejected name still reached the command line: %v", f.calls)
			}
		})
	}
}

func TestDeletingSomethingAbsentReportsNotFound(t *testing.T) {
	f := &fakeRun{err: errors.New("could not be found")}

	m := &macKeychain{bin: "security"}
	m.run = f.run
	if err := m.Delete("openai"); !errors.Is(err, ErrNotFound) {
		t.Errorf("mac Delete = %v, want ErrNotFound", err)
	}

	s := &secretTool{bin: "secret-tool"}
	s.run = f.run
	if err := s.Delete("openai"); !errors.Is(err, ErrNotFound) {
		t.Errorf("secret-tool Delete = %v, want ErrNotFound", err)
	}
}

func TestDeletingSomethingPresentSucceeds(t *testing.T) {
	f := &fakeRun{}
	m := &macKeychain{bin: "security"}
	m.run = f.run
	if err := m.Delete("openai"); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 1 || f.calls[0][1] != "delete-generic-password" {
		t.Fatalf("called %v", f.calls)
	}
}

// A refused write is reported with what the tool said. "It did not work" sends
// someone to read a keychain by hand.
func TestARefusedWriteCarriesTheReason(t *testing.T) {
	f := &fakeRun{err: errors.New("exit 1"), out: []byte("User interaction is not allowed.")}
	m := &macKeychain{bin: "security"}
	m.run = f.run

	err := m.Set("openai", "sk-test")
	if err == nil {
		t.Fatal("a refused write reported success")
	}
	if !strings.Contains(err.Error(), "User interaction is not allowed") {
		t.Errorf("the reason was lost: %v", err)
	}
	// And the secret is not in it.
	if strings.Contains(err.Error(), "sk-test") {
		t.Fatal("the credential leaked into the error message")
	}
}

// Dumping the keychain asks for authorisation and is refused more often than
// not. An empty list is the honest answer: this backend cannot enumerate, and
// guessing would report credentials that may not exist.
func TestAKeychainThatWillNotEnumerateReportsNothingRatherThanGuessing(t *testing.T) {
	f := &fakeRun{err: errors.New("refused")}
	m := &macKeychain{bin: "security"}
	m.run = f.run

	got, err := m.List()
	if err != nil {
		t.Fatalf("a refused dump became an error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

// The secret goes down a pipe and never onto a command line, where anything
// that can list processes would see it.
func TestTheSecretNeverReachesACommandLine(t *testing.T) {
	var gotStdin string
	var gotArgs []string
	s := &secretTool{bin: "secret-tool"}
	s.runIn = func(stdin, name string, args ...string) ([]byte, error) {
		gotStdin = stdin
		gotArgs = append([]string{name}, args...)
		return nil, nil
	}

	if err := s.Set("openai", "sk-live-do-not-log"); err != nil {
		t.Fatal(err)
	}
	if gotStdin != "sk-live-do-not-log" {
		t.Errorf("the secret did not go down stdin: %q", gotStdin)
	}
	for _, a := range gotArgs {
		if strings.Contains(a, "sk-live") {
			t.Fatalf("the secret appeared on the command line: %v", gotArgs)
		}
	}
}

func TestARefusedKeyringWriteCarriesTheReasonAndNotTheSecret(t *testing.T) {
	s := &secretTool{bin: "secret-tool"}
	s.runIn = func(string, string, ...string) ([]byte, error) {
		return []byte("no keyring daemon"), errors.New("exit 1")
	}
	err := s.Set("openai", "sk-live-do-not-log")
	if err == nil {
		t.Fatal("a refused write reported success")
	}
	if !strings.Contains(err.Error(), "no keyring daemon") {
		t.Errorf("the reason was lost: %v", err)
	}
	if strings.Contains(err.Error(), "sk-live") {
		t.Fatal("the credential leaked into the error message")
	}
}

// The scraping is pure so it is testable without a keychain, and it has to be:
// getting it wrong reports credentials for the wrong service.
func TestAccountsAreScrapedOnlyForThisService(t *testing.T) {
	dump := `keychain: "/Users/a/Library/Keychains/login.keychain-db"
    "acct"<blob>="openai"
    "svce"<blob>="dcode"
keychain: "/Users/a/Library/Keychains/login.keychain-db"
    "acct"<blob>="someone-elses"
    "svce"<blob>="another-app"
keychain: "/Users/a/Library/Keychains/login.keychain-db"
    "acct"<blob>="anthropic"
    "svce"<blob>="dcode"
`
	got := accountsFor(dump, "dcode")
	if strings.Join(got, ",") != "anthropic,openai" {
		t.Fatalf("got %v, want this service's accounts, sorted", got)
	}
	if accountsFor(dump, "nothing-uses-this") != nil {
		t.Error("a service with no entries produced names")
	}
}

func TestListReturnsWhatTheDumpHeld(t *testing.T) {
	f := &fakeRun{out: []byte("keychain: x\n    \"acct\"<blob>=\"openai\"\n    \"svce\"<blob>=\"dcode\"\n")}
	m := &macKeychain{bin: "security"}
	m.run = f.run
	got, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "openai" {
		t.Fatalf("got %v", got)
	}
}

func TestBetweenHandlesAMissingDelimiter(t *testing.T) {
	if got := between("no markers here", `"acct"<blob>="`, `"`); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := between(`x"acct"<blob>="openai"y`, `"acct"<blob>="`, `"`); got != "openai" {
		t.Errorf("got %q", got)
	}
}

// runWithStdin is the real path the injected one stands in for. Exercised once
// against a command that exists everywhere, so the branch is not only reachable
// through a fake.
func TestRunWithStdinFeedsTheCommand(t *testing.T) {
	out, err := runWithStdin("hello\n", "cat")
	if err != nil {
		t.Skipf("cat is not available: %v", err)
	}
	if strings.TrimSpace(string(out)) != "hello" {
		t.Fatalf("got %q, want the stdin echoed back", out)
	}
}
