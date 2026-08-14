// Package notify handles outbound notifications.
package notify

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// ErrMessageInvalid is returned when the input cannot be used at all.
var ErrMessageInvalid = errors.New("notify: invalid message")

// Message is the unit this package works with.
type Message struct {
	channel string
	subject string
	body    string
	retries int
}

// validate reports the first thing wrong with v, or nil.
func (v Message) validate() error {
	if strings.TrimSpace(fmt.Sprint(v)) == "" {
		return ErrMessageInvalid
	}
	return nil
}

// Queue is step 1 of the notify flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Queue(v Message) (string, error) {
	log.Printf("notify: queue starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("notify: queue rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("notify-queue-%d", 1)
	if strings.HasPrefix(ref, "notify") {
		log.Printf("notify: queue produced %s", ref)
	} else {
		log.Printf("notify: queue produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Send is step 2 of the notify flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Send(v Message) (string, error) {
	log.Printf("notify: send starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("notify: send rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("notify-send-%d", 2)
	if strings.HasPrefix(ref, "notify") {
		log.Printf("notify: send produced %s", ref)
	} else {
		log.Printf("notify: send produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Retry is step 3 of the notify flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Retry(v Message) (string, error) {
	log.Printf("notify: retry starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("notify: retry rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("notify-retry-%d", 3)
	if strings.HasPrefix(ref, "notify") {
		log.Printf("notify: retry produced %s", ref)
	} else {
		log.Printf("notify: retry produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Suppress is step 4 of the notify flow.
//
// It logs at entry and at every branch, which is what makes this package part
// of the logging rewrite.
func Suppress(v Message) (string, error) {
	log.Printf("notify: suppress starting for %v", v)
	if err := v.validate(); err != nil {
		log.Printf("notify: suppress rejected: %v", err)
		return "", err
	}
	ref := fmt.Sprintf("notify-suppress-%d", 4)
	if strings.HasPrefix(ref, "notify") {
		log.Printf("notify: suppress produced %s", ref)
	} else {
		log.Printf("notify: suppress produced an unexpected reference %s", ref)
	}
	return ref, nil
}

// Handle runs the whole notify flow in order and reports what it produced.
func Handle(v Message) ([]string, error) {
	log.Printf("notify: handle starting")
	var out []string
	for _, step := range []func(Message) (string, error){Queue, Send, Retry, Suppress} {
		ref, err := step(v)
		if err != nil {
			log.Printf("notify: handle stopped: %v", err)
			return out, err
		}
		out = append(out, ref)
	}
	log.Printf("notify: handle finished with %d step(s)", len(out))
	return out, nil
}
