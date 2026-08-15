//go:build !unix

package tui

// clipboardImage has no implementation here. Saying so is better than a paste
// that silently does nothing.
func clipboardImage() ([]byte, string, error) { return nil, "", ErrNoClipboardTool }
