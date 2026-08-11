package main

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/app"
)

// `dcode config` lists what is stored, never what it is. A tool that prints a
// key to prove it has one is a tool that puts it in a scrollback, a screen
// share and a bug report.
//
// Asserted on every path, including the ones that give up early: a machine with
// no keyring prints "(no store: …)" and returns, and the property has to hold
// there too — that branch is the one a CI runner takes.
func TestTheCredentialIsNeverPrintedOnAnyPath(t *testing.T) {
	const secret = "sk-live-0123456789abcdefghij"
	for _, from := range []string{"keychain", "file", "DCODE_API_KEY", ""} {
		out, _ := capture(t, func() {
			printCredentials(app.Options{APIKey: secret, CredentialFrom: from})
		})
		if strings.Contains(out, secret) {
			t.Fatalf("from=%q printed the credential:\n%s", from, out)
		}
		if strings.Contains(out, "0123456789") {
			t.Fatalf("from=%q printed a recognisable run of it:\n%s", from, out)
		}
	}
}

// The environment wins over the store, so it has to be named — otherwise the
// screen reports a stored key that is not the one in use.
//
// This is the one path that does not depend on a keyring existing, which is why
// it is the one that asserts the provenance. The earlier version of this test
// asserted it for a stored credential and passed only on a machine with a
// keychain: green on macOS, red on Linux.
func TestTheEnvironmentCredentialSaysItIsInUse(t *testing.T) {
	out, _ := capture(t, func() {
		printCredentials(app.Options{
			APIKey: "sk-live-0123456789abcdefghij", CredentialFrom: "DCODE_API_KEY",
		})
	})
	if !strings.Contains(out, "DCODE_API_KEY") {
		t.Errorf("the output does not say the environment is in use:\n%s", out)
	}
	if !strings.Contains(out, "being ignored") {
		t.Errorf("the output does not say the stored one is not in use:\n%s", out)
	}
}

func TestNoCredentialSaysSoRatherThanPrintingNothing(t *testing.T) {
	out, _ := capture(t, func() { printCredentials(app.Options{}) })
	if strings.TrimSpace(out) == "" {
		t.Fatal("with no credential the command said nothing at all, which reads as a bug")
	}
}

// `dcode config <key>` for a key nobody has heard of should list the surface
// rather than fail, because a typo is the most likely reason to be here.
func TestConfigForAnUnknownKeyIsNotAnError(t *testing.T) {
	_, _ = capture(t, func() {
		if err := runConfig([]string{"model.nmae"}); err != nil {
			t.Errorf("a mistyped key returned an error: %v", err)
		}
	})
}
