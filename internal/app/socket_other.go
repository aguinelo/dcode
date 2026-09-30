//go:build !unix

package app

import "io/fs"

// ownerOf cannot say who owns a file here: there is no uid to compare with.
func ownerOf(fs.FileInfo) (uid int, known bool) { return 0, false }
