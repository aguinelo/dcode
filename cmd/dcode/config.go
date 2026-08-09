package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/credential"
)

// runConfig shows the effective configuration, including which credential is in
// use — masked.
//
// Masked rather than hidden: "configured: yes" cannot tell you that you pasted
// the key from the wrong account, which is the question someone is actually
// asking when they come here. Masked rather than printed: this screen ends up in
// screenshots, screen shares, scrollback and recordings, and printing the key by
// default hands it to all of them at once. `dcode login --reveal` is the way to
// see it in full, once, on purpose.
func runConfig(args []string) error {
	fs := flag.NewFlagSet("dcode config", flag.ContinueOnError)
	workspace := fs.String("workspace", "", "workspace root (default: current directory)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `dcode config — the effective configuration and where each value came from

  dcode config              everything
  dcode config <key>        one key
  dcode login --reveal      print the credential in full, on purpose

Flags:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	ws, err := resolveWorkspace(*workspace)
	if err != nil {
		return err
	}
	opts, resolved, err := loadOptions(ws)
	if err != nil {
		return err
	}
	if key := fs.Arg(0); key != "" {
		return printConfig(resolved, key)
	}

	fmt.Printf("workspace  %s\n\n", ws)

	fmt.Println("model")
	for _, k := range []string{"model.name", "model.transport", "model.base_url"} {
		v, ok := resolved.Get(k)
		if !ok || v.Value == "" {
			continue
		}
		fmt.Printf("  %-16s %-28s %s\n", short(k), v.Value, origin(v))
	}
	if name := app.CredentialName(opts); name != "" {
		fmt.Printf("  %-16s %s\n", "family", name)
	}

	fmt.Println("\nsandbox")
	for _, k := range []string{"sandbox.mode", "sandbox.approval_policy", "sandbox.allow_network"} {
		if v, ok := resolved.Get(k); ok {
			fmt.Printf("  %-16s %-28s %s\n", short(k), v.Value, origin(v))
		}
	}

	fmt.Println("\ncredentials")
	printCredentials(opts)
	return nil
}

func short(key string) string {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			return key[i+1:]
		}
	}
	return key
}

func origin(v config.Value) string {
	s := fmt.Sprintf("%s (%s)", v.Source, v.Origin)
	if v.Locked {
		s += " · locked"
	}
	return s
}

// printCredentials lists what is stored, never what it is.
func printCredentials(opts app.Options) {
	roots, err := config.DiscoverRoots(os.Getenv)
	if err != nil {
		fmt.Println("  (state directory unavailable)")
		return
	}
	store, err := credential.Open(credential.Options{
		StateDir: roots.State, Backend: opts.CredentialBackend,
	})
	if err != nil {
		fmt.Printf("  (no store: %v)\n", err)
		return
	}

	current := app.CredentialName(opts)

	// The environment wins over the store, so it is reported first or the
	// screen would name a key that is not the one in use.
	if opts.CredentialFrom == "DCODE_API_KEY" {
		fmt.Printf("  %-14s %-24s %-10s %s\n", current,
			credential.Mask(opts.APIKey), credential.Fingerprint(opts.APIKey), "DCODE_API_KEY")
		fmt.Println("\n  The environment is in use; anything stored is being ignored.")
		return
	}

	names, _ := store.List()
	if len(names) == 0 && current != "" {
		if _, err := store.Get(current); err == nil {
			names = []string{current}
		}
	}
	if len(names) == 0 {
		fmt.Printf("  nothing stored in %s\n", store.Where())
		fmt.Println("  Store one with `dcode login`.")
		return
	}

	for _, n := range names {
		secret, err := store.Get(n)
		if err != nil {
			continue
		}
		mark := "  "
		if n == current {
			mark = "* "
		}
		fmt.Printf("%s%-14s %-24s %-10s %s\n", mark, n,
			credential.Mask(secret), credential.Fingerprint(secret), store.Where())
	}
	fmt.Println("\n  * is the one this model uses. `dcode login --reveal` prints it in full.")
}
