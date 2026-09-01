package evals

import (
	"crypto/sha256"
	"encoding/hex"
)

// PromptFingerprint identifies the prefix a scenario runs against.
//
// Short on purpose: it is written by hand into a Measurement, and a
// sixty-four-character hash in a table nobody can check is a hash nobody
// checks. Twelve hex characters distinguish every prompt this suite has ever
// had, several times over.
func PromptFingerprint(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])[:12]
}

// Fingerprint is the prefix this scenario would run against now.
func (f Fixture) Fingerprint() (string, error) {
	prompt, err := f.Prompt("")
	if err != nil {
		return "", err
	}
	return PromptFingerprint(prompt), nil
}

// unverifiable counts the measurements that cannot say which prompt they saw.
//
// Reported rather than enforced. Making a prompt change fail the build would
// mean every prompt PR carrying a re-measurement of fifty-three contracts, and
// a rule nobody can afford is a rule that gets switched off. What this buys is
// that the distance is VISIBLE — the same job the "ever actually measured" row
// does for the distance between declared and verified.
func unverifiable(ms []Measurement) int {
	n := 0
	for _, m := range ms {
		if m.Prompt == "" {
			n++
		}
	}
	return n
}
