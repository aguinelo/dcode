// Package api serves the outside world.
package api

import "example.com/app/internal/store"

// Handler answers requests by writing through a Keeper.
type Handler struct {
	Records store.Keeper
}

// Save writes one uploaded body.
func (h Handler) Save(key string, body []byte) error {
	return h.Records.Put(key, body)
}
