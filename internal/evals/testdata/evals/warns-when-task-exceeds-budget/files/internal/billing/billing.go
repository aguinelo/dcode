// Package billing handles invoices and their state machine.
package billing

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrInvoiceInvalid is returned when the input cannot be used at all.
var ErrInvoiceInvalid = errors.New("billing: invalid invoice")

// Invoice is the unit this package works with.
type Invoice struct {
	number   string
	cents    int64
	currency string
	dueDays  int
}

// validate reports the first thing wrong with v, or nil.
func (v Invoice) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrInvoiceInvalid
	}
	return nil
}

// Draft is step 1 of the billing flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Draft(v Invoice) (string, error) {
	log.Printf("billing: draft starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("billing: draft rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("billing-draft-%d", 1)
	if strings.HasPrefix(ref, "billing") {
		log.Printf("billing: draft produced %s", ref)
	} else {
		log.Printf("billing: draft produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Finalise is step 2 of the billing flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Finalise(v Invoice) (string, error) {
	log.Printf("billing: finalise starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("billing: finalise rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("billing-finalise-%d", 2)
	if strings.HasPrefix(ref, "billing") {
		log.Printf("billing: finalise produced %s", ref)
	} else {
		log.Printf("billing: finalise produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Void is step 3 of the billing flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Void(v Invoice) (string, error) {
	log.Printf("billing: void starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("billing: void rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("billing-void-%d", 3)
	if strings.HasPrefix(ref, "billing") {
		log.Printf("billing: void produced %s", ref)
	} else {
		log.Printf("billing: void produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Reconcile is step 4 of the billing flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Reconcile(v Invoice) (string, error) {
	log.Printf("billing: reconcile starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("billing: reconcile rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("billing-reconcile-%d", 4)
	if strings.HasPrefix(ref, "billing") {
		log.Printf("billing: reconcile produced %s", ref)
	} else {
		log.Printf("billing: reconcile produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole billing flow in order and reports what it produced.
func Handle(v Invoice) ([]string, error) {
	log.Printf("billing: handle starting")
	var out []string
	for _, step := range []func(Invoice) (string, error){Draft, Finalise, Void, Reconcile} {
		ref, err := step(v)
		if err != nil {
			log.Printf("billing: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("billing: handle finished with %d step(s)", len(out))
	return out, nil
}
