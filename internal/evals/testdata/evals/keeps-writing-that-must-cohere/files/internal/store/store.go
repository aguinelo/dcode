// Package store is the one place that knows how records are kept.
package store

// Keeper is what everything else in this repository writes through.
type Keeper interface {
	// Put stores one record under key.
	Put(key string, body []byte) error
}

type memory map[string][]byte

// Put stores one record under key.
func (m memory) Put(key string, body []byte) error {
	m[key] = body
	return nil
}
