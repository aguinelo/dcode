// Package catalog handles the product catalogue.
package catalog

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrItemInvalid is returned when the input cannot be used at all.
var ErrItemInvalid = errors.New("catalog: invalid item")

// Item is the unit this package works with.
type Item struct {
	sku   string
	title string
	cents int64
	stock int
}

// validate reports the first thing wrong with v, or nil.
func (v Item) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrItemInvalid
	}
	return nil
}

// Add is step 1 of the catalog flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Add(v Item) (string, error) {
	log.Printf("catalog: add starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("catalog: add rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("catalog-add-%d", 1)
	if strings.HasPrefix(ref, "catalog") {
		log.Printf("catalog: add produced %s", ref)
	} else {
		log.Printf("catalog: add produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Reprice is step 2 of the catalog flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Reprice(v Item) (string, error) {
	log.Printf("catalog: reprice starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("catalog: reprice rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("catalog-reprice-%d", 2)
	if strings.HasPrefix(ref, "catalog") {
		log.Printf("catalog: reprice produced %s", ref)
	} else {
		log.Printf("catalog: reprice produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Retire is step 3 of the catalog flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Retire(v Item) (string, error) {
	log.Printf("catalog: retire starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("catalog: retire rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("catalog-retire-%d", 3)
	if strings.HasPrefix(ref, "catalog") {
		log.Printf("catalog: retire produced %s", ref)
	} else {
		log.Printf("catalog: retire produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Search is step 4 of the catalog flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Search(v Item) (string, error) {
	log.Printf("catalog: search starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("catalog: search rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("catalog-search-%d", 4)
	if strings.HasPrefix(ref, "catalog") {
		log.Printf("catalog: search produced %s", ref)
	} else {
		log.Printf("catalog: search produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole catalog flow in order and reports what it produced.
func Handle(v Item) ([]string, error) {
	log.Printf("catalog: handle starting")
	var out []string
	for _, step := range []func(Item) (string, error){Add, Reprice, Retire, Search} {
		ref, err := step(v)
		if err != nil {
			log.Printf("catalog: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("catalog: handle finished with %d step(s)", len(out))
	return out, nil
}
