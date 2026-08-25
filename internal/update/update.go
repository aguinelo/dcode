// Package update installs a newer dcode over the running one.
//
// Nothing here ever runs on its own. An agent that replaces its own binary
// without being asked defeats every audit the rest of this program is built to
// support — so the passive notice never applies anything, and Apply is only
// reachable from an explicit `dcode update` (RN-3).
//
// Spec: docs/specs/architecture/distribution/202608072352-*.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/version"
)

// Release is one published version and everything needed to install it.
type Release struct {
	Version   string              `json:"version"`
	Artifacts map[string]Artifact `json:"artifacts"`
}

// Artifact is one platform's build.
type Artifact struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// Updater is the update surface. Two operations, deliberately: everything that
// looks is separate from the one thing that writes.
type Updater interface {
	// Latest reports the most recent release. A network error is NOT fatal to
	// the passive notice (RN-4).
	Latest(ctx context.Context) (Release, error)

	// Apply downloads, VERIFIES and swaps the binary atomically (RN-5).
	Apply(ctx context.Context, r Release) error
}

// PlatformKey is the map key inside a Release.
func PlatformKey(goos, goarch string) string { return goos + "_" + goarch }

// ArtifactName is the stable artifact filename. Third-party scripts depend on
// this format, which is why it carries the same stability promise as the binary
// name itself.
func ArtifactName(version, goos, goarch string) string {
	version = strings.TrimPrefix(version, "v")
	if goos == "windows" {
		return fmt.Sprintf("dcode_%s_windows_%s.zip", version, goarch)
	}
	return fmt.Sprintf("dcode_%s_%s_%s.tar.gz", version, goos, goarch)
}

// SupportedPlatforms is the published matrix.
//
// It is exactly the set that has a sandbox backend, and it has to be. Windows
// was in this list and in the release workflow, and internal/sandbox has no
// backend for it — so New fails closed there and the binary cannot create a
// session at all. Publishing an artefact that cannot run is worse than not
// publishing one: someone downloads it, it installs, it verifies, and then it
// refuses the first thing they ask, with an error about a sandbox they never
// chose.
//
// The sandbox spec puts Windows out of the MVP by name. This list now says the
// same thing, and TestThePublishedMatrixIsExactlyWhatCanRun keeps the two from
// drifting apart again.
var SupportedPlatforms = []string{
	"darwin_amd64", "darwin_arm64",
	"linux_amd64", "linux_arm64",
}

// ParseChecksums reads a `sha256␠␠filename` list into a map keyed by filename.
//
// The signature covers this one file rather than each artifact: one signature
// for the whole release, while verification stays per-artifact through the
// hashes it carries.
func ParseChecksums(data []byte) (map[string]string, error) {
	out := map[string]string{}
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("checksums.txt:%d: expected `<sha256>  <file>`, got %q", i+1, line)
		}
		sum := strings.ToLower(fields[0])
		if len(sum) != 64 {
			return nil, fmt.Errorf("checksums.txt:%d: %q is not a sha-256 digest", i+1, fields[0])
		}
		if _, err := hex.DecodeString(sum); err != nil {
			return nil, fmt.Errorf("checksums.txt:%d: %q is not hexadecimal", i+1, fields[0])
		}
		out[strings.TrimPrefix(fields[1], "*")] = sum
	}
	if len(out) == 0 {
		return nil, errors.New("checksums.txt is empty")
	}
	return out, nil
}

// ---------- signature ----------

// Verifier checks that a checksums file was signed by the project's release
// identity.
type Verifier interface {
	Verify(ctx context.Context, checksums, signature, certificate []byte) error
}

// ErrNoVerifier says the signature could not be checked on this machine.
//
// It used to fail the update on its own, which made cosign a requirement for
// updating at all — a package nobody installs, demanded of a machine that
// already has a working dcode. What actually needs covering is a substituted
// release, and the digest committed to the installer covers it too. So this is
// now one route being unavailable rather than a verdict, and Apply refuses only
// when neither route held.
var ErrNoVerifier = errors.New("the release signature could not be checked here")

// CosignVerifier verifies with the cosign binary.
//
// Shelling out rather than linking sigstore: the verification path is then the
// same one a user can run by hand to reproduce the result, and the trust root
// is a tool they can audit independently of this program.
type CosignVerifier struct {
	// Identity and Issuer pin who may sign. Keyless OIDC signing removes the
	// long-lived private key, which would otherwise be the highest-value item
	// in the entire repository.
	Identity string
	Issuer   string
	// Path overrides the cosign binary, for tests.
	Path string
	// Look overrides binary lookup, for tests.
	Look func(string) (string, error)
}

// DefaultIdentity is the workflow allowed to sign releases.
const (
	DefaultIdentity = "https://github.com/aguinelo/dcode/.github/workflows/release.yml@refs/heads/main"
	DefaultIssuer   = "https://token.actions.githubusercontent.com"
)

func (v CosignVerifier) Verify(ctx context.Context, checksums, signature, certificate []byte) error {
	look := v.Look
	if look == nil {
		look = exec.LookPath
	}
	bin := v.Path
	if bin == "" {
		p, err := look("cosign")
		if err != nil {
			return ErrNoVerifier
		}
		bin = p
	}

	dir, err := os.MkdirTemp("", "dcode-verify")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	paths := map[string][]byte{
		"checksums.txt":     checksums,
		"checksums.txt.sig": signature,
		"checksums.txt.pem": certificate,
	}
	for name, data := range paths {
		if len(data) == 0 {
			return fmt.Errorf("update: %s is missing from the release", name)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			return err
		}
	}

	identity, issuer := v.Identity, v.Issuer
	if identity == "" {
		identity = DefaultIdentity
	}
	if issuer == "" {
		issuer = DefaultIssuer
	}

	cmd := exec.CommandContext(ctx, bin, "verify-blob",
		"--certificate", filepath.Join(dir, "checksums.txt.pem"),
		"--signature", filepath.Join(dir, "checksums.txt.sig"),
		"--certificate-identity", identity,
		"--certificate-oidc-issuer", issuer,
		filepath.Join(dir, "checksums.txt"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update: the release signature did not verify: %w\n%s", err, out)
	}
	return nil
}

// ---------- verification helpers ----------

// VerifySHA256 compares a payload against an expected digest.
func VerifySHA256(data []byte, want string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("update: checksum mismatch — expected %s, got %s", want, got)
	}
	return nil
}

// ExtractBinary pulls the dcode binary out of a release archive.
//
// Entry paths are checked rather than trusted: an archive is remote input, and
// a `../` in a member name is the oldest way there is to write outside the
// directory you were given.
func ExtractBinary(archiveName string, data []byte, dest string) error {
	if strings.HasSuffix(archiveName, ".zip") {
		return extractZip(data, dest)
	}
	return extractTarGz(data, dest)
}

func binaryEntry(name string) bool {
	base := filepath.Base(filepath.Clean(name))
	return base == "dcode" || base == "dcode.exe"
}

func safeEntry(name string) bool {
	clean := filepath.Clean(name)
	return !filepath.IsAbs(clean) && !strings.HasPrefix(clean, "..")
}

func extractTarGz(data []byte, dest string) error {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("update: the archive is not valid gzip: %w", err)
	}
	defer zr.Close()

	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("update: the archive is not a valid tar: %w", err)
		}
		if !safeEntry(h.Name) {
			return fmt.Errorf("update: the archive contains an unsafe path %q", h.Name)
		}
		if h.Typeflag != tar.TypeReg || !binaryEntry(h.Name) {
			continue
		}
		return writeExecutable(dest, tr)
	}
	return errors.New("update: the archive does not contain a dcode binary")
}

func extractZip(data []byte, dest string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("update: the archive is not a valid zip: %w", err)
	}
	for _, f := range zr.File {
		if !safeEntry(f.Name) {
			return fmt.Errorf("update: the archive contains an unsafe path %q", f.Name)
		}
		if f.FileInfo().IsDir() || !binaryEntry(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return writeExecutable(dest, rc)
	}
	return errors.New("update: the archive does not contain a dcode binary")
}

func writeExecutable(dest string, r io.Reader) error {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// CheckRuns executes the candidate binary and confirms it reports the version
// it claims to be.
//
// This runs before the swap, which is what stops a working binary from being
// replaced by one that does not run on this machine — a wrong architecture, a
// missing library, a truncated download that still hashed correctly because it
// was the wrong file all along.
func CheckRuns(ctx context.Context, path, wantVersion string) error {
	cmd := exec.CommandContext(ctx, path, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update: the downloaded binary does not run here: %w\n%s", err, out)
	}
	want := strings.TrimPrefix(wantVersion, "v")
	if !strings.Contains(string(out), want) {
		return fmt.Errorf("update: the downloaded binary reports %q, which does not contain the expected version %s",
			strings.TrimSpace(string(out)), wantVersion)
	}
	return nil
}

// ---------- the GitHub updater ----------

// DefaultBaseURL is the release origin.
const DefaultBaseURL = "https://github.com/aguinelo/dcode/releases"

// DefaultAPIURL is where release metadata is read from.
const DefaultAPIURL = "https://api.github.com/repos/aguinelo/dcode/releases"

// DefaultInstallerURL is the installer on the default branch. The pipeline
// rewrites its pinned block after publishing, so it carries the digests of the
// latest release — and it is in git history, which is what makes it a route
// independent of the release assets.
const DefaultInstallerURL = "https://raw.githubusercontent.com/aguinelo/dcode/main/install.sh"

// Config configures the updater.
type Config struct {
	// APIURL and BaseURL exist for an internal mirror. Signature verification
	// is required against any origin — a mirror is not a reason to trust.
	APIURL   string
	BaseURL  string
	Channel  string
	HTTP     *http.Client
	Verifier Verifier
	// InstallerURL is where the installer that carries this release's digests
	// is read from. It is the second route to the expected SHA-256, and the
	// only one that does not travel with the artifact.
	InstallerURL string
	// TargetPath is the binary to replace. Empty means the running one.
	TargetPath string
	// Pin refuses to change version when set.
	Pin string
	// GOOS and GOARCH override the platform, for tests.
	GOOS, GOARCH string
	// AllowLocalOverwrite lets a caller replace a binary the release pipeline
	// did not publish. Only the explicit `--force` sets it: the default has to
	// be the refusal, or the guard is decoration.
	AllowLocalOverwrite bool
}

// GitHub is the released-artifact updater.
type GitHub struct{ cfg Config }

// NewGitHub builds the updater.
func NewGitHub(cfg Config) *GitHub {
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPIURL
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 60 * time.Second}
	}
	if cfg.Verifier == nil {
		cfg.Verifier = CosignVerifier{}
	}
	if cfg.InstallerURL == "" {
		cfg.InstallerURL = DefaultInstallerURL
	}
	if cfg.GOOS == "" {
		cfg.GOOS = runtime.GOOS
	}
	if cfg.GOARCH == "" {
		cfg.GOARCH = runtime.GOARCH
	}
	if cfg.Channel == "" {
		cfg.Channel = "stable"
	}
	return &GitHub{cfg: cfg}
}

type ghRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// Latest reports the newest release on the configured channel.
func (g *GitHub) Latest(ctx context.Context) (Release, error) {
	body, err := g.get(ctx, g.cfg.APIURL)
	if err != nil {
		return Release{}, err
	}
	var list []ghRelease
	if err := json.Unmarshal(body, &list); err != nil {
		return Release{}, fmt.Errorf("update: could not read the release list: %w", err)
	}

	var picked *ghRelease
	for i := range list {
		r := &list[i]
		if r.Draft {
			continue
		}
		// A prerelease is never the default and never updates on its own.
		if r.Prerelease && g.cfg.Channel != "prerelease" {
			continue
		}
		picked = r
		break
	}
	if picked == nil {
		return Release{}, fmt.Errorf("update: no release found on the %s channel", g.cfg.Channel)
	}

	rel := Release{Version: picked.TagName, Artifacts: map[string]Artifact{}}
	assets := map[string]string{}
	for _, a := range picked.Assets {
		assets[a.Name] = a.URL
	}

	checksums, err := g.fetchAsset(ctx, assets, "checksums.txt")
	if err != nil {
		return Release{}, err
	}
	sums, err := ParseChecksums(checksums)
	if err != nil {
		return Release{}, err
	}

	for _, key := range SupportedPlatforms {
		parts := strings.SplitN(key, "_", 2)
		name := ArtifactName(picked.TagName, parts[0], parts[1])
		url, ok := assets[name]
		if !ok {
			continue
		}
		rel.Artifacts[key] = Artifact{URL: url, SHA256: sums[name]}
	}
	if len(rel.Artifacts) == 0 {
		return Release{}, fmt.Errorf("update: release %s publishes no artifacts this tool recognises", picked.TagName)
	}
	return rel, nil
}

// ErrLocalBuild is returned when the running binary was not published by the
// release pipeline.
//
// Refusing is the useful answer rather than the cautious one: a build from the
// working tree is normally *ahead* of the last tag, so replacing it with the
// latest release is a downgrade wearing the word "update". And a binary we did
// not publish is one we cannot reason about at all.
var ErrLocalBuild = errors.New(
	"this is a local build, not a published release, so `update` would replace it " +
		"with something older. Rebuild with `make install`, or install a release " +
		"with the install script")

// Apply installs a release over the current binary.
func (g *GitHub) Apply(ctx context.Context, r Release) error {
	if !g.cfg.AllowLocalOverwrite && !version.IsRelease() {
		return ErrLocalBuild
	}
	// An update is something newer. The check that finds the release is cached
	// for a day, so a binary that has moved on since — the one this very
	// command installs — would otherwise be offered its own predecessor and
	// take it.
	if cur := version.Version; version.IsRelease() && Newer(cur, r.Version) {
		return fmt.Errorf(
			"update: %s is older than the %s you are running, and going backwards "+
				"is not an update. Install it deliberately with the install script "+
				"if that is what you want", r.Version, cur)
	}
	if g.cfg.Pin != "" && strings.TrimPrefix(g.cfg.Pin, "v") != strings.TrimPrefix(r.Version, "v") {
		return fmt.Errorf(
			"update: this installation is pinned to %s by DCODE_PIN_VERSION, so it will not move to %s",
			g.cfg.Pin, r.Version)
	}

	key := PlatformKey(g.cfg.GOOS, g.cfg.GOARCH)
	art, ok := r.Artifacts[key]
	if !ok {
		return fmt.Errorf("update: release %s has no artifact for %s. Supported: %s",
			r.Version, key, strings.Join(SupportedPlatforms, ", "))
	}
	if art.SHA256 == "" {
		return fmt.Errorf("update: release %s lists no checksum for %s, so it cannot be verified", r.Version, key)
	}

	target := g.cfg.TargetPath
	if target == "" {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		// Through the symlink: replacing the link would leave the real binary
		// untouched and the update silently ineffective.
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		target = exe
	}

	// The staging directory sits beside the target so the final step is a
	// rename within one filesystem. Across filesystems a rename is a copy, and
	// a copy that is interrupted leaves a partial binary — a machine with no
	// working dcode on it.
	stage, err := os.MkdirTemp(filepath.Dir(target), ".dcode-update")
	if err != nil {
		return fmt.Errorf("update: cannot stage next to %s: %w", target, err)
	}
	defer os.RemoveAll(stage)

	archive, err := g.get(ctx, art.URL)
	if err != nil {
		return err
	}
	if err := VerifySHA256(archive, art.SHA256); err != nil {
		return err
	}

	name := ArtifactName(r.Version, g.cfg.GOOS, g.cfg.GOARCH)

	// The checksum above came from the release, alongside the artifact, so it
	// catches a corrupted download and not a substituted release. Two things
	// cover substitution, and EITHER is enough: a digest that reached here by a
	// route independent of the release, and the signature.
	covered := false

	if digest := g.carriedDigest(ctx, name); digest != "" {
		if err := VerifySHA256(archive, digest); err != nil {
			return fmt.Errorf("update: %s does not match the digest committed to the "+
				"installer, which is the copy that did not travel with it: %w", name, err)
		}
		covered = true
	}

	switch err := g.verifySignature(ctx, r); {
	case err == nil:
		covered = true
	case !errors.Is(err, ErrNoVerifier):
		// A signature that fails is a verdict, not a missing route. Making a
		// check optional must not make it decorative.
		return err
	}

	// Refusing here is not the refusal cosign used to force. There is a working
	// binary on this machine, so stopping costs a version and keeps everything,
	// and the way forward is a release rather than a package to install.
	if !covered {
		return fmt.Errorf("update: nothing here could rule out a substituted release of %s. "+
			"The installer at %s carries no digest for %s, and the signature could not be "+
			"checked on this machine. The latest release is the one whose digests that "+
			"installer carries", r.Version, g.cfg.InstallerURL, name)
	}

	candidate := filepath.Join(stage, "dcode")
	if g.cfg.GOOS == "windows" {
		candidate += ".exe"
	}
	if err := ExtractBinary(name, archive, candidate); err != nil {
		return err
	}
	if err := CheckRuns(ctx, candidate, r.Version); err != nil {
		return err
	}

	// Last: everything that can fail has already failed by here, so a failure
	// at any earlier step leaves the current binary untouched.
	if err := os.Rename(candidate, target); err != nil {
		return fmt.Errorf("update: could not replace %s: %w", target, err)
	}
	return nil
}

// carriedDigest reads the digest the published installer carries for an
// artifact, or empty when there is none to be had.
//
// Every failure is empty rather than an error: an unreachable raw host, an
// installer pinned to another release, a block that cannot be parsed. None of
// them is a verdict about the artifact — they mean this route is unavailable,
// and Apply decides what that costs once it knows about the other one.
func (g *GitHub) carriedDigest(ctx context.Context, artifact string) string {
	script, err := g.get(ctx, g.cfg.InstallerURL)
	if err != nil {
		return ""
	}
	return PinnedDigest(script, artifact)
}

func (g *GitHub) verifySignature(ctx context.Context, r Release) error {
	base := strings.TrimSuffix(g.cfg.BaseURL, "/") + "/download/" + r.Version + "/"
	checksums, err := g.get(ctx, base+"checksums.txt")
	if err != nil {
		return err
	}
	sig, err := g.get(ctx, base+"checksums.txt.sig")
	if err != nil {
		return err
	}
	cert, err := g.get(ctx, base+"checksums.txt.pem")
	if err != nil {
		return err
	}
	return g.cfg.Verifier.Verify(ctx, checksums, sig, cert)
}

func (g *GitHub) fetchAsset(ctx context.Context, assets map[string]string, name string) ([]byte, error) {
	url, ok := assets[name]
	if !ok {
		return nil, fmt.Errorf("update: the release does not publish %s, so it cannot be verified", name)
	}
	return g.get(ctx, url)
}

func (g *GitHub) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream, application/json")
	resp, err := g.cfg.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update: %s returned %s", url, resp.Status)
	}
	// Bounded: a release artifact has a known order of magnitude, and an
	// unbounded read from a remote is a memory-exhaustion primitive.
	return io.ReadAll(io.LimitReader(resp.Body, 256<<20))
}
