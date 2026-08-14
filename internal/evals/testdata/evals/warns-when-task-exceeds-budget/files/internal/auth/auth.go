// Package auth handles credentials and session lifetime.
package auth

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrSessionInvalid is returned when the input cannot be used at all.
var ErrSessionInvalid = errors.New("auth: invalid session")

// Session is the unit this package works with.
type Session struct {
	token     string
	userID    string
	scopes    []string
	expiresIn int
}

// validate reports the first thing wrong with v, or nil.
func (v Session) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrSessionInvalid
	}
	return nil
}

// Issue is step 1 of the auth flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Issue(v Session) (string, error) {
	log.Printf("auth: issue starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("auth: issue rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("auth-issue-%d", 1)
	if strings.HasPrefix(ref, "auth") {
		log.Printf("auth: issue produced %s", ref)
	} else {
		log.Printf("auth: issue produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Verify is step 2 of the auth flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Verify(v Session) (string, error) {
	log.Printf("auth: verify starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("auth: verify rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("auth-verify-%d", 2)
	if strings.HasPrefix(ref, "auth") {
		log.Printf("auth: verify produced %s", ref)
	} else {
		log.Printf("auth: verify produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Revoke is step 3 of the auth flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Revoke(v Session) (string, error) {
	log.Printf("auth: revoke starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("auth: revoke rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("auth-revoke-%d", 3)
	if strings.HasPrefix(ref, "auth") {
		log.Printf("auth: revoke produced %s", ref)
	} else {
		log.Printf("auth: revoke produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Refresh is step 4 of the auth flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Refresh(v Session) (string, error) {
	log.Printf("auth: refresh starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("auth: refresh rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("auth-refresh-%d", 4)
	if strings.HasPrefix(ref, "auth") {
		log.Printf("auth: refresh produced %s", ref)
	} else {
		log.Printf("auth: refresh produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole auth flow in order and reports what it produced.
func Handle(v Session) ([]string, error) {
	log.Printf("auth: handle starting")
	var out []string
	for _, step := range []func(Session) (string, error){Issue, Verify, Revoke, Refresh} {
		ref, err := step(v)
		if err != nil {
			log.Printf("auth: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("auth: handle finished with %d step(s)", len(out))
	return out, nil
}
