package provider

import (
	"regexp"
	"strings"
	"sync"
)

// Credential redaction.
//
// A key that reaches a log line, an error message or an event is the leak that
// only gets discovered after publication. Rather than trusting every call site
// to remember, every string that could reach a user goes through sanitize.

var (
	secretsMu sync.RWMutex
	secrets   []string
)

// RegisterSecret marks a value to be redacted from any outgoing text. Called
// once at startup with the API key. Values shorter than 8 characters are
// ignored: redacting a short string would blank out unrelated text.
func RegisterSecret(v string) {
	if len(v) < 8 {
		return
	}
	secretsMu.Lock()
	defer secretsMu.Unlock()
	for _, s := range secrets {
		if s == v {
			return
		}
	}
	secrets = append(secrets, v)
}

// ClearSecrets drops every registered secret. Test-only in practice.
func ClearSecrets() {
	secretsMu.Lock()
	defer secretsMu.Unlock()
	secrets = nil
}

// bearerLike catches credentials that never went through RegisterSecret —
// echoed back by a provider, or embedded in a URL.
var bearerLike = regexp.MustCompile(
	`(?i)(bearer\s+|api[_-]?key["'\s:=]+|token["'\s:=]+|sk-)[A-Za-z0-9_\-\.]{12,}`)

// sanitize removes known secrets and anything shaped like one.
func sanitize(s string) string {
	if s == "" {
		return s
	}
	secretsMu.RLock()
	known := make([]string, len(secrets))
	copy(known, secrets)
	secretsMu.RUnlock()

	for _, sec := range known {
		s = strings.ReplaceAll(s, sec, "[redacted]")
	}
	return bearerLike.ReplaceAllString(s, "[redacted]")
}

// Sanitize is the exported form, for transports and families building messages.
func Sanitize(s string) string { return sanitize(s) }
