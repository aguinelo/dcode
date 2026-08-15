//go:build unix && !darwin

package tui

import (
	"context"
	"os/exec"
	"time"
)

// clipboardImage asks whichever clipboard tool this desktop ships.
//
// Wayland first, then X11, because a Wayland session usually has xclip
// installed and answering from the wrong clipboard is worse than not answering.
// Neither present is a real answer — "this machine has no way to read the
// clipboard" — and the caller says so rather than pretending there was no
// image.
func clipboardImage() ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, c := range [][]string{
		{"wl-paste", "--type", "image/png", "--no-newline"},
		{"xclip", "-selection", "clipboard", "-t", "image/png", "-o"},
	} {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		out, err := exec.CommandContext(ctx, c[0], c[1:]...).Output()
		if err != nil || len(out) == 0 {
			// The tool is there and the clipboard holds no image. Trying the
			// next one would read a different clipboard and answer a question
			// nobody asked.
			return nil, "", ErrNoImageInClipboard
		}
		return out, "image/png", nil
	}
	return nil, "", ErrNoClipboardTool
}
