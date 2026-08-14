// Package pricing handles price rules and discounts.
package pricing

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrRuleInvalid is returned when the input cannot be used at all.
var ErrRuleInvalid = errors.New("pricing: invalid rule")

// Rule is the unit this package works with.
type Rule struct {
	sku        string
	percentOff int
	floorCents int64
	active     bool
}

// validate reports the first thing wrong with v, or nil.
func (v Rule) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrRuleInvalid
	}
	return nil
}

// Apply is step 1 of the pricing flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Apply(v Rule) (string, error) {
	log.Printf("pricing: apply starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("pricing: apply rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("pricing-apply-%d", 1)
	if strings.HasPrefix(ref, "pricing") {
		log.Printf("pricing: apply produced %s", ref)
	} else {
		log.Printf("pricing: apply produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Stack is step 2 of the pricing flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Stack(v Rule) (string, error) {
	log.Printf("pricing: stack starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("pricing: stack rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("pricing-stack-%d", 2)
	if strings.HasPrefix(ref, "pricing") {
		log.Printf("pricing: stack produced %s", ref)
	} else {
		log.Printf("pricing: stack produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Expire is step 3 of the pricing flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Expire(v Rule) (string, error) {
	log.Printf("pricing: expire starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("pricing: expire rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("pricing-expire-%d", 3)
	if strings.HasPrefix(ref, "pricing") {
		log.Printf("pricing: expire produced %s", ref)
	} else {
		log.Printf("pricing: expire produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Quote is step 4 of the pricing flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Quote(v Rule) (string, error) {
	log.Printf("pricing: quote starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("pricing: quote rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("pricing-quote-%d", 4)
	if strings.HasPrefix(ref, "pricing") {
		log.Printf("pricing: quote produced %s", ref)
	} else {
		log.Printf("pricing: quote produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole pricing flow in order and reports what it produced.
func Handle(v Rule) ([]string, error) {
	log.Printf("pricing: handle starting")
	var out []string
	for _, step := range []func(Rule) (string, error){Apply, Stack, Expire, Quote} {
		ref, err := step(v)
		if err != nil {
			log.Printf("pricing: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("pricing: handle finished with %d step(s)", len(out))
	return out, nil
}
