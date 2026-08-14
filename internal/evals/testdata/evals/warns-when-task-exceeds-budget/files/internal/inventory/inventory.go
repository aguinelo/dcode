// Package inventory handles stock levels and reservations.
package inventory

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrStockInvalid is returned when the input cannot be used at all.
var ErrStockInvalid = errors.New("inventory: invalid stock")

// Stock is the unit this package works with.
type Stock struct {
	sku       string
	onHand    int
	reserved  int
	reorderAt int
}

// validate reports the first thing wrong with v, or nil.
func (v Stock) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrStockInvalid
	}
	return nil
}

// Reserve is step 1 of the inventory flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Reserve(v Stock) (string, error) {
	log.Printf("inventory: reserve starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("inventory: reserve rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("inventory-reserve-%d", 1)
	if strings.HasPrefix(ref, "inventory") {
		log.Printf("inventory: reserve produced %s", ref)
	} else {
		log.Printf("inventory: reserve produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Release is step 2 of the inventory flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Release(v Stock) (string, error) {
	log.Printf("inventory: release starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("inventory: release rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("inventory-release-%d", 2)
	if strings.HasPrefix(ref, "inventory") {
		log.Printf("inventory: release produced %s", ref)
	} else {
		log.Printf("inventory: release produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Receive is step 3 of the inventory flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Receive(v Stock) (string, error) {
	log.Printf("inventory: receive starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("inventory: receive rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("inventory-receive-%d", 3)
	if strings.HasPrefix(ref, "inventory") {
		log.Printf("inventory: receive produced %s", ref)
	} else {
		log.Printf("inventory: receive produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Count is step 4 of the inventory flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Count(v Stock) (string, error) {
	log.Printf("inventory: count starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("inventory: count rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("inventory-count-%d", 4)
	if strings.HasPrefix(ref, "inventory") {
		log.Printf("inventory: count produced %s", ref)
	} else {
		log.Printf("inventory: count produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole inventory flow in order and reports what it produced.
func Handle(v Stock) ([]string, error) {
	log.Printf("inventory: handle starting")
	var out []string
	for _, step := range []func(Stock) (string, error){Reserve, Release, Receive, Count} {
		ref, err := step(v)
		if err != nil {
			log.Printf("inventory: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("inventory: handle finished with %d step(s)", len(out))
	return out, nil
}
