package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aguinelo/dcode/internal/policy"
)

// A Chromium reaches its first frame inside the sandbox.
//
// Asked of the KERNEL and of a real browser, not of the profile text. The
// profile can be read and agreed with while the thing it governs still dies:
// this failure was a SIGSEGV before Chromium drew anything, so there was no
// denial to assert on and no error in the tool's output — the screen showed a
// stack trace and an exit code, and nothing said the boundary had refused.
//
// Everything Chromium-based was affected: Playwright, Puppeteer, Lighthouse, an
// Electron app under test. It was found by a person watching an agent try three
// different browsers, fail at all three, and reading that as timidity.
func TestAChromiumReachesItsFirstFrameInsideTheSandbox(t *testing.T) {
	bin, headless := findChromium()
	if bin == "" {
		t.Skip("no Chromium found; nothing to ask the kernel about")
	}
	s, err := New(Config{AllowNetwork: func() bool { return false }}, policy.ModeWorkspaceWrite)
	if err != nil {
		t.Skipf("no sandbox available: %v", err)
	}

	ws := t.TempDir()
	r := Runner{Sandbox: s, Mode: policy.ModeWorkspaceWrite}

	// about:blank and a virtual time budget: the page is not the point, getting
	// to a page at all is. No network is granted, deliberately — what this asks
	// is whether the browser STARTS under the boundary, not whether it browses.
	flags := "--headless=new"
	if headless {
		flags = ""
	}
	out, _, err := r.Run(context.Background(), ws, fmt.Sprintf(
		"%q %s --disable-gpu --no-sandbox --user-data-dir=%q --virtual-time-budget=1500 --dump-dom about:blank 2>&1",
		bin, flags, filepath.Join(ws, "profile")))
	if err != nil {
		t.Fatalf("running under the sandbox failed outright: %v", err)
	}

	// The crash is the regression this guards. It comes back the moment either
	// half of the grant is dropped, and it comes back as a signal rather than
	// as a message — which is exactly why it needs a test rather than a reader.
	if strings.Contains(out, "SEGV") || strings.Contains(out, "Received signal 11") {
		t.Fatalf("Chromium crashed inside the sandbox; the grant it needs is gone:\n%s", out)
	}

	// A full Chrome writes its crash database under ~/Library whatever
	// --user-data-dir says, and the sandbox refuses that — correctly. Reaching
	// a frame is asserted of the headless shell, which is what Playwright and
	// every agent-driven browser actually launch. Somebody who wants the full
	// browser grants that path by name, which is what a named grant is for.
	if headless && !strings.Contains(out, "<html") {
		t.Fatalf("the headless shell did not reach a frame:\n%s", out)
	}
}

// findChromium looks for a browser to ask, preferring the headless shell — it is
// what Playwright launches, and it was the first thing to crash.
func findChromium() (path string, headless bool) {
	if p, err := exec.LookPath("chrome-headless-shell"); err == nil {
		return p, true
	}
	// Where Playwright puts it, in a project or in the shared cache.
	for _, root := range []string{
		filepath.Join(os.Getenv("HOME"), "Library", "Caches", "ms-playwright"),
	} {
		if p := findUnder(root, "chrome-headless-shell"); p != "" {
			return p, true
		}
	}
	for _, p := range []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, false
		}
	}
	return "", false
}

func findUnder(root, name string) string {
	var found string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if !d.IsDir() && d.Name() == name {
			found = p
		}
		return nil
	})
	return found
}
