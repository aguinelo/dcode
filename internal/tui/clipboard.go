package tui

import (
	"errors"
	"os"
)

// ErrNoImageInClipboard means the clipboard holds something, and it is not a
// picture. Distinct from a failure to look, because the two want different
// answers: one is "paste some text then", the other is "this machine has no
// way to read the clipboard".
var ErrNoImageInClipboard = errors.New("no image in the clipboard")

// ErrNoClipboardTool means this machine has no way to read the clipboard at
// all. A different answer from an empty clipboard, and it needs a different
// response: install one, or use /image with a path.
var ErrNoClipboardTool = errors.New("no clipboard tool on this machine")

// ClipboardImage returns the picture on the clipboard, as bytes.
//
// The terminal cannot help here, and that is the whole difficulty. A paste
// arrives as bracketed text or as a key press; an image on the clipboard
// produces nothing either way. So dcode has to go and ask the operating system
// itself, which is why this is one file per platform shelling out to the tool
// that platform ships.
//
// This runs in the CLIENT, not in the agent, so it is not behind the sandbox
// and does not need to be: the person pressed the key, and reading their own
// clipboard is the thing they asked for.
func ClipboardImage() ([]byte, string, error) {
	return clipboardImage()
}

// tempPNG is where a platform that can only write to a file puts it.
func tempPNG() (string, func(), error) {
	f, err := os.CreateTemp("", "dcode-*.png")
	if err != nil {
		return "", func() {}, err
	}
	name := f.Name()
	_ = f.Close()
	return name, func() { _ = os.Remove(name) }, nil
}
