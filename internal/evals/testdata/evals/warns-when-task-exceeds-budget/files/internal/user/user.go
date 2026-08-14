// Package user handles user profiles.
package user

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrProfileInvalid is returned when the input cannot be used at all.
var ErrProfileInvalid = errors.New("user: invalid profile")

// Profile is the unit this package works with.
type Profile struct {
	id       string
	email    string
	locale   string
	verified bool
}

// validate reports the first thing wrong with v, or nil.
func (v Profile) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrProfileInvalid
	}
	return nil
}

// Create is step 1 of the user flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Create(v Profile) (string, error) {
	log.Printf("user: create starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("user: create rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("user-create-%d", 1)
	if strings.HasPrefix(ref, "user") {
		log.Printf("user: create produced %s", ref)
	} else {
		log.Printf("user: create produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Verify is step 2 of the user flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Verify(v Profile) (string, error) {
	log.Printf("user: verify starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("user: verify rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("user-verify-%d", 2)
	if strings.HasPrefix(ref, "user") {
		log.Printf("user: verify produced %s", ref)
	} else {
		log.Printf("user: verify produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Rename is step 3 of the user flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Rename(v Profile) (string, error) {
	log.Printf("user: rename starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("user: rename rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("user-rename-%d", 3)
	if strings.HasPrefix(ref, "user") {
		log.Printf("user: rename produced %s", ref)
	} else {
		log.Printf("user: rename produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Deactivate is step 4 of the user flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Deactivate(v Profile) (string, error) {
	log.Printf("user: deactivate starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("user: deactivate rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("user-deactivate-%d", 4)
	if strings.HasPrefix(ref, "user") {
		log.Printf("user: deactivate produced %s", ref)
	} else {
		log.Printf("user: deactivate produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole user flow in order and reports what it produced.
func Handle(v Profile) ([]string, error) {
	log.Printf("user: handle starting")
	var out []string
	for _, step := range []func(Profile) (string, error){Create, Verify, Rename, Deactivate} {
		ref, err := step(v)
		if err != nil {
			log.Printf("user: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("user: handle finished with %d step(s)", len(out))
	return out, nil
}
