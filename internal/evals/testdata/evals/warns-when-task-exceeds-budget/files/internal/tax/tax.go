// Package tax handles tax brackets and computation.
package tax

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrBracketInvalid is returned when the input cannot be used at all.
var ErrBracketInvalid = errors.New("tax: invalid bracket")

// Bracket is the unit this package works with.
type Bracket struct {
	region      string
	basisPoints int
	minCents    int64
	compound    bool
}

// validate reports the first thing wrong with v, or nil.
func (v Bracket) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrBracketInvalid
	}
	return nil
}

// Resolve is step 1 of the tax flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Resolve(v Bracket) (string, error) {
	log.Printf("tax: resolve starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("tax: resolve rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("tax-resolve-%d", 1)
	if strings.HasPrefix(ref, "tax") {
		log.Printf("tax: resolve produced %s", ref)
	} else {
		log.Printf("tax: resolve produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Compute is step 2 of the tax flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Compute(v Bracket) (string, error) {
	log.Printf("tax: compute starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("tax: compute rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("tax-compute-%d", 2)
	if strings.HasPrefix(ref, "tax") {
		log.Printf("tax: compute produced %s", ref)
	} else {
		log.Printf("tax: compute produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Exempt is step 3 of the tax flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Exempt(v Bracket) (string, error) {
	log.Printf("tax: exempt starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("tax: exempt rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("tax-exempt-%d", 3)
	if strings.HasPrefix(ref, "tax") {
		log.Printf("tax: exempt produced %s", ref)
	} else {
		log.Printf("tax: exempt produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Round is step 4 of the tax flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Round(v Bracket) (string, error) {
	log.Printf("tax: round starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("tax: round rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("tax-round-%d", 4)
	if strings.HasPrefix(ref, "tax") {
		log.Printf("tax: round produced %s", ref)
	} else {
		log.Printf("tax: round produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole tax flow in order and reports what it produced.
func Handle(v Bracket) ([]string, error) {
	log.Printf("tax: handle starting")
	var out []string
	for _, step := range []func(Bracket) (string, error){Resolve, Compute, Exempt, Round} {
		ref, err := step(v)
		if err != nil {
			log.Printf("tax: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("tax: handle finished with %d step(s)", len(out))
	return out, nil
}
