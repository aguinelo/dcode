// Package search handles the search front end.
package search

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrQueryInvalid is returned when the input cannot be used at all.
var ErrQueryInvalid = errors.New("search: invalid query")

// Query is the unit this package works with.
type Query struct {
	text    string
	page    int
	perPage int
	fuzzy   bool
}

// validate reports the first thing wrong with v, or nil.
func (v Query) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrQueryInvalid
	}
	return nil
}

// Parse is step 1 of the search flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Parse(v Query) (string, error) {
	log.Printf("search: parse starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("search: parse rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("search-parse-%d", 1)
	if strings.HasPrefix(ref, "search") {
		log.Printf("search: parse produced %s", ref)
	} else {
		log.Printf("search: parse produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Rank is step 2 of the search flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Rank(v Query) (string, error) {
	log.Printf("search: rank starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("search: rank rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("search-rank-%d", 2)
	if strings.HasPrefix(ref, "search") {
		log.Printf("search: rank produced %s", ref)
	} else {
		log.Printf("search: rank produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Paginate is step 3 of the search flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Paginate(v Query) (string, error) {
	log.Printf("search: paginate starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("search: paginate rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("search-paginate-%d", 3)
	if strings.HasPrefix(ref, "search") {
		log.Printf("search: paginate produced %s", ref)
	} else {
		log.Printf("search: paginate produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Suggest is step 4 of the search flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Suggest(v Query) (string, error) {
	log.Printf("search: suggest starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("search: suggest rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("search-suggest-%d", 4)
	if strings.HasPrefix(ref, "search") {
		log.Printf("search: suggest produced %s", ref)
	} else {
		log.Printf("search: suggest produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole search flow in order and reports what it produced.
func Handle(v Query) ([]string, error) {
	log.Printf("search: handle starting")
	var out []string
	for _, step := range []func(Query) (string, error){Parse, Rank, Paginate, Suggest} {
		ref, err := step(v)
		if err != nil {
			log.Printf("search: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("search: handle finished with %d step(s)", len(out))
	return out, nil
}
