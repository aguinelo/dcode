package main

import (
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/app"
)

// `dcode config` lists what is stored, never what it is. A tool that prints a
// key to prove it has one is a tool that puts it in a scrollback, a screen
// share and a bug report.
func TestCredentialsAreListedAndNeverPrinted(t *testing.T) {
	const secret = "sk-live-0123456789abcdefghij"
	out, _ := capture(t, func() {
		printCredentials(app.Options{APIKey: secret, CredentialFrom: "keychain"})
	})
	if strings.Contains(out, secret) {
		t.Fatalf("the credential was printed:\n%s", out)
	}
	if strings.Contains(out, "0123456789") {
		t.Fatalf("a recognisable run of the credential was printed:\n%s", out)
	}
	if !strings.Contains(out, "keychain") {
		t.Errorf("the output does not say where it came from:\n%s", out)
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
