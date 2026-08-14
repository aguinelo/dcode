// Package payment settles charges against the ledger.
package payment

import (
	"errors"
	"fmt"
)

// ErrDeclined is returned when the ledger refuses the charge.
var ErrDeclined = errors.New("payment: declined")

// Charge is one attempt to move money.
type Charge struct {
	ID       string
	Cents    int64
	Currency string
}

// Settle records the charge and returns the ledger reference.
//
// A zero or negative amount is a programming error, not a decline: nothing
// upstream should ever construct one.
func Settle(c Charge) (string, error) {
	if c.Cents <= 0 {
		return "", fmt.Errorf("payment: charge %s has no amount", c.ID)
	}
	if c.Currency == "" {
		return "", fmt.Errorf("payment: charge %s has no currency", c.ID)
	}
	if c.Cents > 100_000_00 {
		return "", ErrDeclined
	}
	return "ref-" + c.ID, nil
}

// Refund reverses a settled charge by its ledger reference.
func Refund(ref string) error {
	if ref == "" {
		return errors.New("payment: refund needs a reference")
	}
	return nil
}
