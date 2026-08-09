package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	xterm "github.com/charmbracelet/x/term"

	"github.com/aguinelo/dcode/internal/app"
	"github.com/aguinelo/dcode/internal/config"
	"github.com/aguinelo/dcode/internal/credential"
)

// runLogin stores, shows or removes a model credential.
//
// The secret is never a command-line argument, in any form. An argument lands
// in the shell history and is visible in `ps` to every process on the machine
// for as long as the command runs — so the only way in is a prompt that does
// not echo, or a pipe.
func runLogin(args []string) error {
	fs := flag.NewFlagSet("dcode login", flag.ContinueOnError)
	var (
		family = fs.String("family", "", "which family this key belongs to (default: the one the model resolves to)")
		list   = fs.Bool("list", false, "list stored credentials, masked")
		remove = fs.Bool("delete", false, "remove the stored credential")
		reveal = fs.Bool("reveal", false, "print the stored key in full — it will appear on screen and in scrollback")
	)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `dcode login — store the model credential outside the files you sync

The key is read from a prompt that does not echo, or from a pipe. It is never
taken as an argument: an argument lands in shell history and is visible in `+"`ps`"+`.

Examples:
  dcode login                       store the key for the current model's family
  dcode login --family claude       store a second key for another family
  dcode login --list                what is stored, masked
  dcode login --reveal              print the key in full, on purpose
  dcode login --delete

The store is chosen by `+"`credential.backend`"+` (keychain, file, or empty
to pick one), so what writes and what reads always agree.

Flags:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf(
			"a key is never passed as an argument — it would land in your shell history " +
				"and be visible in `ps`. Run `dcode login` and paste it at the prompt")
	}

	ws, err := resolveWorkspace("")
	if err != nil {
		return err
	}
	opts, _, err := loadOptions(ws)
	if err != nil {
		return err
	}
	roots, err := config.DiscoverRoots(os.Getenv)
	if err != nil {
		return err
	}
	// The same store every reader uses. A flag here and nothing on the readers
	// would file the secret somewhere nothing looks for it.
	store, err := credential.Open(credential.Options{
		StateDir: roots.State, Backend: opts.CredentialBackend,
	})
	if err != nil {
		return err
	}

	name := *family
	if name == "" {
		if name = app.CredentialName(opts); name == "" {
			return fmt.Errorf(
				"no family claims the model %q, so there is nowhere to file the key. "+
					"Name one with --family", opts.Model)
		}
	}

	switch {
	case *list:
		return listCredentials(store, name)
	case *remove:
		if err := store.Delete(name); err != nil {
			if errors.Is(err, credential.ErrNotFound) {
				fmt.Printf("nothing was stored for %s\n", name)
				return nil
			}
			return err
		}
		fmt.Printf("removed the credential for %s from %s\n", name, store.Where())
		return nil
	case *reveal:
		return revealCredential(store, name)
	}

	return storeCredential(store, name)
}

func storeCredential(store credential.Store, name string) error {
	secret, err := readSecret(fmt.Sprintf("key for %s (it will not be shown): ", name))
	if err != nil {
		return err
	}
	if secret == "" {
		return fmt.Errorf("nothing was entered, so nothing was stored")
	}
	if err := store.Set(name, secret); err != nil {
		return err
	}

	fmt.Printf("\nstored for %s\n", name)
	fmt.Printf("  %-12s %s\n", "value", credential.Mask(secret))
	fmt.Printf("  %-12s %s\n", "fingerprint", credential.Fingerprint(secret))
	fmt.Printf("  %-12s %s\n", "in", store.Where())
	if strings.Contains(store.Where(), credential.FileName) {
		// Where the file is the only option, saying what it does and does not
		// protect against is part of the contract.
		fmt.Println("\n  There is no keychain here, so this is a file only you can read.")
		fmt.Println("  It keeps the key out of anything you commit or sync; it does not")
		fmt.Println("  hide it from a process running as you.")
	}
	return nil
}

func listCredentials(store credential.Store, current string) error {
	names, err := store.List()
	if err != nil {
		return err
	}
	// A backend that cannot enumerate still knows about the name in hand, and
	// reporting nothing at all would read as "nothing is stored".
	if len(names) == 0 {
		if _, err := store.Get(current); err == nil {
			names = []string{current}
		}
	}
	if len(names) == 0 {
		fmt.Printf("no credentials in %s\n", store.Where())
		fmt.Println("Store one with `dcode login`.")
		return nil
	}

	fmt.Printf("credentials in %s\n\n", store.Where())
	for _, n := range names {
		secret, err := store.Get(n)
		if err != nil {
			continue
		}
		mark := "  "
		if n == current {
			mark = "* "
		}
		fmt.Printf("%s%-14s %-24s %s\n", mark, n, credential.Mask(secret), credential.Fingerprint(secret))
	}
	return nil
}

// revealCredential prints the secret in full.
//
// It exists because recovering a key you stored is a real need, and refusing it
// only sends people to a worse workaround. It is a separate flag rather than the
// default so that disclosure is a thing someone chose, once, rather than
// something every `config` leaves in scrollback.
func revealCredential(store credential.Store, name string) error {
	secret, err := store.Get(name)
	if err != nil {
		if errors.Is(err, credential.ErrNotFound) {
			return fmt.Errorf("nothing is stored for %s", name)
		}
		return err
	}
	if xterm.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintf(os.Stderr, "# %s — this will stay in your scrollback\n", name)
	}
	fmt.Println(secret)
	return nil
}

// readSecret takes the key from a prompt without echo, or from a pipe.
//
// The pipe form is what makes this usable from a script or a secret manager
// without ever putting the value on a command line.
func readSecret(prompt string) (string, error) {
	if !xterm.IsTerminal(os.Stdin.Fd()) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}

	fmt.Print(prompt)
	raw, err := xterm.ReadPassword(os.Stdin.Fd())
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}
