//go:build darwin

package tui

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// clipboardImage asks AppleScript for the clipboard as a PNG.
//
// Through a temp file rather than stdout: `the clipboard as «class PNGf»`
// returns AppleScript's hex-data literal, and parsing that back is work with a
// failure mode for every malformed byte. Writing the data to a file is the same
// script saying what it means.
//
// No cgo, deliberately. Reading the pasteboard natively means AppKit, and this
// binary is built with CGO_ENABLED=0 so it cross-compiles — a decision ADR-01
// makes and one screenshot is not worth reopening.
func clipboardImage() ([]byte, string, error) {
	path, cleanup, err := tempPNG()
	if err != nil {
		return nil, "", err
	}
	defer cleanup()

	const script = `on run argv
	set p to POSIX file (item 1 of argv)
	try
		set d to the clipboard as «class PNGf»
	on error
		return "none"
	end try
	set f to open for access p with write permission
	set eof f to 0
	write d to f
	close access f
	return "ok"
end run`

	// A clipboard read that hangs would hang the interface, and the person
	// pressed a key expecting an answer now.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "osascript", "-e", script, path).Output()
	if err != nil {
		return nil, "", err
	}
	if string(trimSpace(out)) == "none" {
		return nil, "", ErrNoImageInClipboard
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	if len(body) == 0 {
		return nil, "", ErrNoImageInClipboard
	}
	return body, "image/png", nil
}

func trimSpace(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\r' || b[j-1] == '\t') {
		j--
	}
	return b[i:j]
}
