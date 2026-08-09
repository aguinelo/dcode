package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestArtifactNameIsTheStableFormat(t *testing.T) {
	for _, tc := range []struct{ version, goos, goarch, want string }{
		{"v0.3.1", "darwin", "arm64", "dcode_0.3.1_darwin_arm64.tar.gz"},
		{"0.3.1", "linux", "amd64", "dcode_0.3.1_linux_amd64.tar.gz"},
		{"v1.0.0", "windows", "amd64", "dcode_1.0.0_windows_amd64.zip"},
	} {
		if got := ArtifactName(tc.version, tc.goos, tc.goarch); got != tc.want {
			t.Errorf("got %q want %q", got, tc.want)
		}
	}
}

func TestParseChecksums(t *testing.T) {
	sum := strings.Repeat("a", 64)
	got, err := ParseChecksums([]byte(
		"# comment\n" + sum + "  dcode_1.0.0_linux_amd64.tar.gz\n\n" +
			sum + "  *dcode_1.0.0_darwin_arm64.tar.gz\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	// The `*` marks binary mode in the sha256sum format; it is not part of the
	// filename.
	if _, ok := got["dcode_1.0.0_darwin_arm64.tar.gz"]; !ok {
		t.Errorf("got %v", got)
	}
}

func TestParseChecksumsRejectsGarbage(t *testing.T) {
	for name, body := range map[string]string{
		"empty":      "\n\n",
		"one field":  "abc\n",
		"short hash": "abc  file\n",
		"not hex":    strings.Repeat("z", 64) + "  file\n",
	} {
		if _, err := ParseChecksums([]byte(body)); err == nil {
			t.Errorf("%s: must be rejected", name)
		}
	}
}

func TestVerifySHA256(t *testing.T) {
	data := []byte("hello")
	sum := sha256.Sum256(data)
	if err := VerifySHA256(data, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(data, strings.Repeat("0", 64)); err == nil {
		t.Error("a mismatch must be reported")
	}
}

// ---------- archives ----------

func tarGz(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for name, body := range entries {
		if err := tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	zw.Close()
	return buf.Bytes()
}

func zipOf(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	zw.Close()
	return buf.Bytes()
}

func TestExtractBinaryFindsTheBinaryInBothFormats(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "dcode")
	if err := ExtractBinary("x.tar.gz", tarGz(t, map[string]string{
		"README.md": "docs", "dcode": "#!/bin/sh\n",
	}), dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "#!/bin/sh\n" {
		t.Fatalf("got %q, %v", got, err)
	}
	fi, _ := os.Stat(dest)
	if fi.Mode().Perm()&0o100 == 0 {
		t.Errorf("the extracted binary must be executable, got %v", fi.Mode())
	}

	dest2 := filepath.Join(t.TempDir(), "dcode.exe")
	if err := ExtractBinary("x.zip", zipOf(t, map[string]string{
		"dcode.exe": "MZ", "readme": "x",
	}), dest2); err != nil {
		t.Fatal(err)
	}
}

// An archive is remote input, and `../` in a member name is the oldest way
// there is to write outside the directory you were given.
func TestExtractBinaryRefusesAnUnsafePath(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "dcode")
	err := ExtractBinary("x.tar.gz", tarGz(t, map[string]string{"../evil": "x"}), dest)
	if err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("got %v", err)
	}
	err = ExtractBinary("x.zip", zipOf(t, map[string]string{"../evil": "x"}), dest)
	if err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("got %v", err)
	}
}

func TestExtractBinaryReportsAMissingBinaryAndABadArchive(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "dcode")
	if err := ExtractBinary("x.tar.gz", tarGz(t, map[string]string{"README": "x"}), dest); err == nil {
		t.Error("an archive with no dcode binary must be rejected")
	}
	if err := ExtractBinary("x.zip", zipOf(t, map[string]string{"README": "x"}), dest); err == nil {
		t.Error("an archive with no dcode binary must be rejected")
	}
	if err := ExtractBinary("x.tar.gz", []byte("not gzip"), dest); err == nil {
		t.Error("a corrupt archive must be rejected")
	}
	if err := ExtractBinary("x.zip", []byte("not zip"), dest); err == nil {
		t.Error("a corrupt archive must be rejected")
	}
}

// ---------- runs-before-swap ----------

func TestCheckRunsAcceptsAMatchingVersionAndRejectsTheRest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixtures are not portable to windows")
	}
	dir := t.TempDir()

	good := filepath.Join(dir, "good")
	writeScript(t, good, "echo 1.2.3")
	if err := CheckRuns(context.Background(), good, "v1.2.3"); err != nil {
		t.Fatal(err)
	}

	// This is what stops a working binary being replaced by one that reports a
	// different version — a wrong build slipped into the release.
	wrong := filepath.Join(dir, "wrong")
	writeScript(t, wrong, "echo 9.9.9")
	if err := CheckRuns(context.Background(), wrong, "v1.2.3"); err == nil {
		t.Error("a version mismatch must abort the update")
	}

	// And this is what stops a binary that cannot run here at all.
	broken := filepath.Join(dir, "broken")
	writeScript(t, broken, "exit 1")
	if err := CheckRuns(context.Background(), broken, "v1.2.3"); err == nil {
		t.Error("a binary that does not run must abort the update")
	}
}

func writeScript(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// ---------- signature ----------

// "Installed, but unverified" is the worst of both worlds: the user ends up
// with a binary and the impression that it went fine.
func TestVerifierFailsClosedWithoutCosign(t *testing.T) {
	v := CosignVerifier{Look: func(string) (string, error) { return "", errors.New("nope") }}
	err := v.Verify(context.Background(), []byte("sums"), []byte("sig"), []byte("pem"))
	if !errors.Is(err, ErrNoVerifier) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "will not install") {
		t.Errorf("the refusal must be explicit: %v", err)
	}
}

func TestVerifierRefusesAReleaseMissingItsSignature(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixtures are not portable to windows")
	}
	bin := filepath.Join(t.TempDir(), "cosign")
	writeScript(t, bin, "exit 0")
	v := CosignVerifier{Path: bin}
	if err := v.Verify(context.Background(), []byte("sums"), nil, []byte("pem")); err == nil {
		t.Error("a release with no signature cannot be verified")
	}
}

func TestVerifierRunsCosignAndReportsItsVerdict(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixtures are not portable to windows")
	}
	dir := t.TempDir()
	ok := filepath.Join(dir, "cosign-ok")
	writeScript(t, ok, "exit 0")
	if err := (CosignVerifier{Path: ok}).Verify(
		context.Background(), []byte("s"), []byte("g"), []byte("p")); err != nil {
		t.Fatal(err)
	}

	bad := filepath.Join(dir, "cosign-bad")
	writeScript(t, bad, "echo tampered >&2; exit 1")
	err := (CosignVerifier{Path: bad}).Verify(
		context.Background(), []byte("s"), []byte("g"), []byte("p"))
	if err == nil || !strings.Contains(err.Error(), "did not verify") {
		t.Fatalf("got %v", err)
	}
}

// ---------- the updater end to end ----------

type fixture struct {
	server  *httptest.Server
	archive []byte
	sum     string
	version string
}

func newFixture(t *testing.T, version string, prerelease bool) *fixture {
	t.Helper()
	f := &fixture{version: version}
	f.archive = tarGz(t, map[string]string{
		"dcode": "#!/bin/sh\necho " + strings.TrimPrefix(version, "v") + "\n",
	})
	sum := sha256.Sum256(f.archive)
	f.sum = hex.EncodeToString(sum[:])

	name := ArtifactName(version, runtime.GOOS, runtime.GOARCH)
	mux := http.NewServeMux()
	mux.HandleFunc("/releases", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"tag_name":%q,"draft":false,"prerelease":%t,"assets":[
			{"name":%q,"browser_download_url":"%s/a/%s"},
			{"name":"checksums.txt","browser_download_url":"%s/a/checksums.txt"}]}]`,
			version, prerelease, name, f.base(), name, f.base())
	})
	mux.HandleFunc("/a/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  %s\n", f.sum, name)
	})
	mux.HandleFunc("/a/"+name, func(w http.ResponseWriter, r *http.Request) {
		w.Write(f.archive)
	})
	mux.HandleFunc("/dl/download/"+version+"/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("blob"))
	})
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fixture) base() string {
	if f.server == nil {
		return ""
	}
	return f.server.URL
}

type okVerifier struct{ called bool }

func (v *okVerifier) Verify(context.Context, []byte, []byte, []byte) error {
	v.called = true
	return nil
}

type failVerifier struct{}

func (failVerifier) Verify(context.Context, []byte, []byte, []byte) error {
	return errors.New("signature did not verify")
}

func newUpdater(t *testing.T, f *fixture, target string, mut ...func(*Config)) *GitHub {
	t.Helper()
	cfg := Config{
		APIURL:     f.server.URL + "/releases",
		BaseURL:    f.server.URL + "/dl",
		Verifier:   &okVerifier{},
		TargetPath: target,
	}
	for _, m := range mut {
		m(&cfg)
	}
	return NewGitHub(cfg)
}

func TestLatestReadsTheReleaseAndItsChecksums(t *testing.T) {
	f := newFixture(t, "v1.2.3", false)
	rel, err := newUpdater(t, f, "").Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version != "v1.2.3" {
		t.Errorf("got %q", rel.Version)
	}
	art, ok := rel.Artifacts[PlatformKey(runtime.GOOS, runtime.GOARCH)]
	if !ok {
		t.Fatalf("got %v", rel.Artifacts)
	}
	if art.SHA256 != f.sum {
		t.Errorf("got %q want %q", art.SHA256, f.sum)
	}
}

// A prerelease is never the default and never updates on its own.
func TestLatestSkipsAPrereleaseOnTheStableChannel(t *testing.T) {
	f := newFixture(t, "v2.0.0-rc1", true)
	if _, err := newUpdater(t, f, "").Latest(context.Background()); err == nil {
		t.Error("a prerelease must not be offered on the stable channel")
	}
	u := newUpdater(t, f, "", func(c *Config) { c.Channel = "prerelease" })
	if _, err := u.Latest(context.Background()); err != nil {
		t.Fatalf("the prerelease channel must see it: %v", err)
	}
}

func TestApplyReplacesTheBinaryOnlyAfterEveryCheckPasses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixtures are not portable to windows")
	}
	f := newFixture(t, "v1.2.3", false)
	target := filepath.Join(t.TempDir(), "dcode")
	writeScript(t, target, "echo 0.0.1")

	v := &okVerifier{}
	u := newUpdater(t, f, target, func(c *Config) { c.Verifier = v })
	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := u.Apply(context.Background(), rel); err != nil {
		t.Fatal(err)
	}
	if !v.called {
		t.Error("the signature must be verified on every apply")
	}
	got, _ := os.ReadFile(target)
	if !strings.Contains(string(got), "1.2.3") {
		t.Errorf("the binary was not replaced:\n%s", got)
	}
	// Staging happens beside the target so the swap is a rename within one
	// filesystem; nothing may be left behind.
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Errorf("the staging directory must be cleaned up, found %d entries", len(entries))
	}
}

// Every failure has to leave the machine with a working dcode on it.
func TestApplyLeavesTheCurrentBinaryIntactOnEveryFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixtures are not portable to windows")
	}
	const originalBody = "#!/bin/sh\necho 0.0.1\n"

	for name, mut := range map[string]func(*Config, *Release){
		"bad signature": func(c *Config, r *Release) { c.Verifier = failVerifier{} },
		"bad checksum": func(c *Config, r *Release) {
			key := PlatformKey(runtime.GOOS, runtime.GOARCH)
			art := r.Artifacts[key]
			art.SHA256 = strings.Repeat("0", 64)
			r.Artifacts[key] = art
		},
		"no checksum": func(c *Config, r *Release) {
			key := PlatformKey(runtime.GOOS, runtime.GOARCH)
			r.Artifacts[key] = Artifact{URL: r.Artifacts[key].URL}
		},
		"wrong platform": func(c *Config, r *Release) {
			c.GOOS, c.GOARCH = "plan9", "sparc"
		},
		"pinned": func(c *Config, r *Release) { c.Pin = "v0.0.1" },
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, "v1.2.3", false)
			target := filepath.Join(t.TempDir(), "dcode")
			if err := os.WriteFile(target, []byte(originalBody), 0o755); err != nil {
				t.Fatal(err)
			}

			base := newUpdater(t, f, target)
			rel, err := base.Latest(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			cfg := Config{
				APIURL: f.server.URL + "/releases", BaseURL: f.server.URL + "/dl",
				Verifier: &okVerifier{}, TargetPath: target,
			}
			mut(&cfg, &rel)
			if err := NewGitHub(cfg).Apply(context.Background(), rel); err == nil {
				t.Fatal("this must not install")
			}

			got, _ := os.ReadFile(target)
			if string(got) != originalBody {
				t.Errorf("the current binary was disturbed:\n%s", got)
			}
			entries, _ := os.ReadDir(filepath.Dir(target))
			if len(entries) != 1 {
				t.Errorf("a failed apply must leave no residue, found %d entries", len(entries))
			}
		})
	}
}

func TestApplyRefusesABinaryThatDoesNotRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fixtures are not portable to windows")
	}
	// The artifact hashes correctly and is signed; it simply reports the wrong
	// version, which is exactly the case the run-before-swap check exists for.
	archive := tarGz(t, map[string]string{"dcode": "#!/bin/sh\necho 0.0.9\n"})
	sum := sha256.Sum256(archive)
	name := ArtifactName("v1.2.3", runtime.GOOS, runtime.GOARCH)

	mux := http.NewServeMux()
	mux.HandleFunc("/a/"+name, func(w http.ResponseWriter, r *http.Request) { w.Write(archive) })
	mux.HandleFunc("/dl/download/v1.2.3/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("blob"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	target := filepath.Join(t.TempDir(), "dcode")
	writeScript(t, target, "echo 0.0.1")

	u := NewGitHub(Config{
		BaseURL: srv.URL + "/dl", Verifier: &okVerifier{}, TargetPath: target,
	})
	err := u.Apply(context.Background(), Release{
		Version: "v1.2.3",
		Artifacts: map[string]Artifact{
			PlatformKey(runtime.GOOS, runtime.GOARCH): {
				URL: srv.URL + "/a/" + name, SHA256: hex.EncodeToString(sum[:]),
			},
		},
	})
	if err == nil {
		t.Fatal("a binary reporting the wrong version must not be installed")
	}
	got, _ := os.ReadFile(target)
	if !strings.Contains(string(got), "0.0.1") {
		t.Errorf("the working binary was replaced by one that does not match:\n%s", got)
	}
}

func TestLatestReportsATransportFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()
	u := NewGitHub(Config{APIURL: srv.URL})
	if _, err := u.Latest(context.Background()); err == nil {
		t.Error("a failing origin must be reported")
	}
}

func TestLatestRejectsAReleaseWithNoUsableArtifacts(t *testing.T) {
	sum := strings.Repeat("a", 64)
	mux := http.NewServeMux()
	mux.HandleFunc("/releases", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"tag_name":"v1.0.0","assets":[
			{"name":"checksums.txt","browser_download_url":"http://%s/c"}]}]`, r.Host)
	})
	mux.HandleFunc("/c", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  something-else.tar.gz\n", sum)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	u := NewGitHub(Config{APIURL: srv.URL + "/releases"})
	if _, err := u.Latest(context.Background()); err == nil {
		t.Error("a release with nothing recognisable must be rejected")
	}
}

func TestLatestRejectsAReleaseWithoutChecksums(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"tag_name":"v1.0.0","assets":[]}]`)
	}))
	defer srv.Close()
	u := NewGitHub(Config{APIURL: srv.URL})
	_, err := u.Latest(context.Background())
	if err == nil || !strings.Contains(err.Error(), "checksums.txt") {
		t.Fatalf("got %v", err)
	}
}

func TestLatestRejectsAGarbledResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json`)
	}))
	defer srv.Close()
	if _, err := NewGitHub(Config{APIURL: srv.URL}).Latest(context.Background()); err == nil {
		t.Error("a garbled release list must be rejected")
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"tag_name":"v1","draft":true,"assets":[]}]`)
	}))
	defer empty.Close()
	if _, err := NewGitHub(Config{APIURL: empty.URL}).Latest(context.Background()); err == nil {
		t.Error("a list of nothing but drafts must be rejected")
	}
}

func TestDefaultsAreFilledIn(t *testing.T) {
	u := NewGitHub(Config{})
	if u.cfg.APIURL != DefaultAPIURL || u.cfg.BaseURL != DefaultBaseURL {
		t.Errorf("got %+v", u.cfg)
	}
	if u.cfg.Channel != "stable" {
		t.Errorf("stable is the only safe default, got %q", u.cfg.Channel)
	}
	if u.cfg.GOOS != runtime.GOOS || u.cfg.GOARCH != runtime.GOARCH {
		t.Errorf("got %s/%s", u.cfg.GOOS, u.cfg.GOARCH)
	}
	if _, ok := u.cfg.Verifier.(CosignVerifier); !ok {
		t.Errorf("verification must be on by default, got %T", u.cfg.Verifier)
	}
}

func TestApplyReportsAnUnreachableArtifactAndOrigin(t *testing.T) {
	target := filepath.Join(t.TempDir(), "dcode")
	if err := os.WriteFile(target, []byte("original"), 0o755); err != nil {
		t.Fatal(err)
	}
	rel := Release{
		Version: "v1.0.0",
		Artifacts: map[string]Artifact{
			PlatformKey(runtime.GOOS, runtime.GOARCH): {
				URL: "http://127.0.0.1:1/never", SHA256: strings.Repeat("a", 64),
			},
		},
	}
	u := NewGitHub(Config{BaseURL: "http://127.0.0.1:1", Verifier: &okVerifier{}, TargetPath: target})
	if err := u.Apply(context.Background(), rel); err == nil {
		t.Fatal("an unreachable artifact must abort")
	}
	got, _ := os.ReadFile(target)
	if string(got) != "original" {
		t.Errorf("the current binary was disturbed: %q", got)
	}
}

// The signature material is fetched from the release origin; any of the three
// files being unreachable aborts before anything is written.
func TestVerifySignatureAbortsWhenTheMaterialIsMissing(t *testing.T) {
	for _, missing := range []string{"checksums.txt", "checksums.txt.sig", "checksums.txt.pem"} {
		mux := http.NewServeMux()
		mux.HandleFunc("/dl/download/v1.0.0/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, missing) {
				http.Error(w, "gone", http.StatusNotFound)
				return
			}
			w.Write([]byte("blob"))
		})
		srv := httptest.NewServer(mux)
		u := NewGitHub(Config{BaseURL: srv.URL + "/dl", Verifier: &okVerifier{}})
		if err := u.verifySignature(context.Background(), Release{Version: "v1.0.0"}); err == nil {
			t.Errorf("%s missing must abort", missing)
		}
		srv.Close()
	}
}

func TestWriteExecutableReportsAnUnwritableDestination(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "no-such-dir", "dcode")
	if err := ExtractBinary("x.tar.gz", tarGz(t, map[string]string{"dcode": "x"}), dest); err == nil {
		t.Error("an unwritable destination must be reported")
	}
	if err := ExtractBinary("x.zip", zipOf(t, map[string]string{"dcode": "x"}), dest); err == nil {
		t.Error("an unwritable destination must be reported")
	}
}
