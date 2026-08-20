package update

import "strings"

// The installer that a release publishes carries the SHA-256 of every artifact
// in that release, between two markers, written by scripts/installer.sh from
// the checksums file the pipeline had already signed and verified.
//
// It matters because of where it then lives. The checksums file travels from
// the same host as the tarball, so on its own it cannot rule out a substituted
// release: whoever replaces one replaces the other and the pair stays
// consistent with itself. The installer is committed to main, where a release
// asset can be swapped leaving no public trace and a tracked line cannot,
// because changing it is a commit.
//
// So the binary reads the same file the install script carries in itself, and
// gets the same second route — which is what lets `dcode update` stop asking
// for cosign.
const (
	pinnedBegin = "# BEGIN PINNED"
	pinnedEnd   = "# END PINNED"
)

// PinnedDigest returns the SHA-256 an installer carries for one artifact, or
// empty when it carries none.
//
// Only what sits between the markers counts. Reading a digest from anywhere in
// the file would let an unrelated line — a comment, an example, an error
// message — decide what gets installed.
func PinnedDigest(script []byte, artifact string) string {
	inside := false
	for _, line := range strings.Split(string(script), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, pinnedBegin):
			inside = true
			continue
		case strings.HasPrefix(trimmed, pinnedEnd):
			return ""
		}
		if !inside || !strings.Contains(line, artifact+")") {
			continue
		}
		for _, field := range strings.Fields(line) {
			if isSHA256(field) {
				return field
			}
		}
	}
	return ""
}

// isSHA256 reports whether s is exactly a lowercase hex digest. Scanning the
// line for one rather than parsing the shell syntax around it: the surrounding
// `case` arm is the generator's business and may be reformatted, while what a
// digest looks like will not change.
func isSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
