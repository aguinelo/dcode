//go:build unix

package app

import (
	"io/fs"
	"syscall"
)

// ownerOf reports who owns a file, where the platform can say.
func ownerOf(fi fs.FileInfo) (uid int, known bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return int(st.Uid), true
}
