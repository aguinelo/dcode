package evals

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// freshness splits the measurements that recorded a fingerprint into those
// whose prompt is still the one today's build sends and those whose prompt has
// changed since.
//
// today answers the fingerprint a measurement's scenario would be sent now,
// under its family. Passed in rather than computed here because answering it
// builds a workspace on disk and resolves the model's family, and neither
// belongs in the arithmetic.
//
// A measurement with no fingerprint is in neither count: unverifiable already
// counts it, and "not recorded" read as "current" is the reading the field
// exists to refuse. Nothing here invents a fingerprint for it either — a hash
// computed today is a hash of today's prompt, not of the one it saw.
//
// Still reported rather than enforced, for the reason unverifiable gives. A
// stale count that rises is what a prompt change costs, made visible.
func freshness(ms []Measurement, today func(Measurement) (string, error)) (current, stale int, err error) {
	for _, m := range ms {
		if m.Prompt == "" {
			continue
		}
		now, err := today(m)
		if err != nil {
			return 0, 0, fmt.Errorf("%s: %w", m.ID, err)
		}
		if now == m.Prompt {
			current++
		} else {
			stale++
		}
	}
	return current, stale, nil
}
