package session

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PruneBudget is how much history is kept.
//
// Two budgets rather than one, and both have to hold. Age alone lets a busy
// fortnight fill a disk inside the window; size alone deletes this morning's
// work on a busy day. Neither is wrong, and neither is enough.
type PruneBudget struct {
	// MaxAge drops records older than this. Zero means age is not a policy.
	MaxAge time.Duration
	// MaxBytes caps the directory, oldest removed first. Zero means size is
	// not a policy.
	MaxBytes int64
	// KeepAtLeast is the floor, and it beats both budgets.
	//
	// Someone who used dcode twice last year should still find those two
	// sessions. A cleanup that empties the directory the first time it runs is
	// a cleanup nobody trusts again, and the whole point of the record is
	// being able to go back to it.
	KeepAtLeast int
	// Live are session ids currently open. A session being written is not
	// garbage, however old its first line is.
	Live map[string]bool
}

// zero reports whether any policy is set at all.
//
// Nothing configured means nothing removed. A zero budget reads as "no policy",
// never as "delete everything" — the second reading loses a user's history on a
// typo, and there is no undo for it.
func (b PruneBudget) zero() bool { return b.MaxAge <= 0 && b.MaxBytes <= 0 }

// Prune removes old session records and reports how many went.
//
// It runs when a session opens rather than on a timer: nothing should be
// deleting a person's history while the program is not running, and a readdir
// on the way into a session is cheap enough that nobody notices.
//
// The order is deliberate. Age first, because "older than a month" is the rule
// a person can predict. Size second, oldest first, because a cap has to be met
// somehow and the oldest is the one least likely to be wanted. The floor last,
// because it overrides both.
func Prune(dir string, b PruneBudget) (int, error) {
	if dir == "" || b.zero() {
		return 0, nil
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		// A fresh install prunes nothing and says so quietly.
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	type record struct {
		name string
		size int64
		mod  time.Time
	}
	var records []record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		if b.Live[strings.TrimSuffix(e.Name(), ".jsonl")] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		records = append(records, record{e.Name(), info.Size(), info.ModTime()})
	}
	// Newest first, so the floor is counted from the end a person cares about.
	sort.Slice(records, func(i, j int) bool { return records[i].mod.After(records[j].mod) })

	var total int64
	var doomed []string
	for i, r := range records {
		if i < b.KeepAtLeast {
			total += r.size
			continue
		}
		switch {
		case b.MaxAge > 0 && time.Since(r.mod) > b.MaxAge:
			doomed = append(doomed, r.name)
		case b.MaxBytes > 0 && total+r.size > b.MaxBytes:
			doomed = append(doomed, r.name)
		default:
			total += r.size
		}
	}

	removed := 0
	for _, name := range doomed {
		if err := os.Remove(filepath.Join(dir, name)); err == nil {
			removed++
		}
	}
	return removed, nil
}
