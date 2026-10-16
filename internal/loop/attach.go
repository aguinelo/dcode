package loop

import (
	"context"

	ce "github.com/aguinelo/dcode/internal/contextengine"
)

// Attachments are what a turn carries besides its text.
type Attachments struct {
	Images    []ce.Image
	Documents []ce.Document
}

// RunAttached runs a turn with what was attached to it.
func (e *Engine) RunAttached(ctx context.Context, input string, att Attachments) (Outcome, error) {
	return e.Run(ctx, input, att.Images...)
}

// AcceptsPDF reports whether this engine's model reads a PDF.
func (e *Engine) AcceptsPDF() bool { return false }
