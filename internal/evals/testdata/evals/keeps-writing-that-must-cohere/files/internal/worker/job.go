// Package worker runs the background jobs.
package worker

import "example.com/app/internal/store"

// Job writes its result through a Keeper.
type Job struct {
	Records store.Keeper
}

// Finish writes what the job produced.
func (j Job) Finish(key string, out []byte) error {
	return j.Records.Put(key, out)
}
