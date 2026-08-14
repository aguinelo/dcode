// Package shipping handles shipping rates by zone.
package shipping

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrRateInvalid is returned when the input cannot be used at all.
var ErrRateInvalid = errors.New("shipping: invalid rate")

// Rate is the unit this package works with.
type Rate struct {
	zone    string
	cents   int64
	days    int
	express bool
}

// validate reports the first thing wrong with v, or nil.
func (v Rate) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrRateInvalid
	}
	return nil
}

// Quote is step 1 of the shipping flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Quote(v Rate) (string, error) {
	log.Printf("shipping: quote starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("shipping: quote rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("shipping-quote-%d", 1)
	if strings.HasPrefix(ref, "shipping") {
		log.Printf("shipping: quote produced %s", ref)
	} else {
		log.Printf("shipping: quote produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Compare is step 2 of the shipping flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Compare(v Rate) (string, error) {
	log.Printf("shipping: compare starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("shipping: compare rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("shipping-compare-%d", 2)
	if strings.HasPrefix(ref, "shipping") {
		log.Printf("shipping: compare produced %s", ref)
	} else {
		log.Printf("shipping: compare produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Select is step 3 of the shipping flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Select(v Rate) (string, error) {
	log.Printf("shipping: select starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("shipping: select rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("shipping-select-%d", 3)
	if strings.HasPrefix(ref, "shipping") {
		log.Printf("shipping: select produced %s", ref)
	} else {
		log.Printf("shipping: select produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Refund is step 4 of the shipping flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Refund(v Rate) (string, error) {
	log.Printf("shipping: refund starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("shipping: refund rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("shipping-refund-%d", 4)
	if strings.HasPrefix(ref, "shipping") {
		log.Printf("shipping: refund produced %s", ref)
	} else {
		log.Printf("shipping: refund produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole shipping flow in order and reports what it produced.
func Handle(v Rate) ([]string, error) {
	log.Printf("shipping: handle starting")
	var out []string
	for _, step := range []func(Rate) (string, error){Quote, Compare, Select, Refund} {
		ref, err := step(v)
		if err != nil {
			log.Printf("shipping: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("shipping: handle finished with %d step(s)", len(out))
	return out, nil
}
