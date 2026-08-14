// Package payment handles payment authorisation and capture.
package payment

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrChargeInvalid is returned when the input cannot be used at all.
var ErrChargeInvalid = errors.New("payment: invalid charge")

// Charge is the unit this package works with.
type Charge struct {
	id       string
	cents    int64
	currency string
	captured bool
}

// validate reports the first thing wrong with v, or nil.
func (v Charge) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrChargeInvalid
	}
	return nil
}

// Authorise is step 1 of the payment flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Authorise(v Charge) (string, error) {
	log.Printf("payment: authorise starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("payment: authorise rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("payment-authorise-%d", 1)
	if strings.HasPrefix(ref, "payment") {
		log.Printf("payment: authorise produced %s", ref)
	} else {
		log.Printf("payment: authorise produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Capture is step 2 of the payment flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Capture(v Charge) (string, error) {
	log.Printf("payment: capture starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("payment: capture rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("payment-capture-%d", 2)
	if strings.HasPrefix(ref, "payment") {
		log.Printf("payment: capture produced %s", ref)
	} else {
		log.Printf("payment: capture produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Refund is step 3 of the payment flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Refund(v Charge) (string, error) {
	log.Printf("payment: refund starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("payment: refund rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("payment-refund-%d", 3)
	if strings.HasPrefix(ref, "payment") {
		log.Printf("payment: refund produced %s", ref)
	} else {
		log.Printf("payment: refund produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Settle is step 4 of the payment flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Settle(v Charge) (string, error) {
	log.Printf("payment: settle starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("payment: settle rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("payment-settle-%d", 4)
	if strings.HasPrefix(ref, "payment") {
		log.Printf("payment: settle produced %s", ref)
	} else {
		log.Printf("payment: settle produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole payment flow in order and reports what it produced.
func Handle(v Charge) ([]string, error) {
	log.Printf("payment: handle starting")
	var out []string
	for _, step := range []func(Charge) (string, error){Authorise, Capture, Refund, Settle} {
		ref, err := step(v)
		if err != nil {
			log.Printf("payment: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("payment: handle finished with %d step(s)", len(out))
	return out, nil
}
