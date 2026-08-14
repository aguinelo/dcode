// Package delivery handles shipments in transit.
package delivery

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrShipmentInvalid is returned when the input cannot be used at all.
var ErrShipmentInvalid = errors.New("delivery: invalid shipment")

// Shipment is the unit this package works with.
type Shipment struct {
	id          string
	carrier     string
	weightGrams int
	signed      bool
}

// validate reports the first thing wrong with v, or nil.
func (v Shipment) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrShipmentInvalid
	}
	return nil
}

// Book is step 1 of the delivery flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Book(v Shipment) (string, error) {
	log.Printf("delivery: book starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("delivery: book rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("delivery-book-%d", 1)
	if strings.HasPrefix(ref, "delivery") {
		log.Printf("delivery: book produced %s", ref)
	} else {
		log.Printf("delivery: book produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Track is step 2 of the delivery flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Track(v Shipment) (string, error) {
	log.Printf("delivery: track starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("delivery: track rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("delivery-track-%d", 2)
	if strings.HasPrefix(ref, "delivery") {
		log.Printf("delivery: track produced %s", ref)
	} else {
		log.Printf("delivery: track produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Reroute is step 3 of the delivery flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Reroute(v Shipment) (string, error) {
	log.Printf("delivery: reroute starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("delivery: reroute rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("delivery-reroute-%d", 3)
	if strings.HasPrefix(ref, "delivery") {
		log.Printf("delivery: reroute produced %s", ref)
	} else {
		log.Printf("delivery: reroute produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Complete is step 4 of the delivery flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Complete(v Shipment) (string, error) {
	log.Printf("delivery: complete starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("delivery: complete rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("delivery-complete-%d", 4)
	if strings.HasPrefix(ref, "delivery") {
		log.Printf("delivery: complete produced %s", ref)
	} else {
		log.Printf("delivery: complete produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole delivery flow in order and reports what it produced.
func Handle(v Shipment) ([]string, error) {
	log.Printf("delivery: handle starting")
	var out []string
	for _, step := range []func(Shipment) (string, error){Book, Track, Reroute, Complete} {
		ref, err := step(v)
		if err != nil {
			log.Printf("delivery: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("delivery: handle finished with %d step(s)", len(out))
	return out, nil
}
