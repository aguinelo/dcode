package tools

import "context"

// How a long-running tool says how far it has got.
//
// Carried on the context rather than on State, and that is the whole design:
// State is per SESSION and shared, while a report belongs to one call. Two
// calls running in parallel would otherwise write their counts through the same
// field, and the screen would show one scan's progress under the other's name.
// A context is already per call and already threaded through Execute.

type progressKey struct{}

// Report says how far a call has got. Total 0 means the total is not known yet,
// which is honest for a walk that has not finished enumerating.
type Report func(kind string, done, total int)

// WithProgress attaches a reporter to a call's context.
func WithProgress(ctx context.Context, r Report) context.Context {
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, progressKey{}, r)
}

// Progress returns the reporter, and never nil.
//
// A no-op rather than a nil check at every call site: a tool should say how far
// it has got without first asking whether anybody is listening, and a nil
// dereference in a scan is a crash in the middle of somebody's work.
func Progress(ctx context.Context) Report {
	if r, ok := ctx.Value(progressKey{}).(Report); ok {
		return r
	}
	return func(string, int, int) {}
}

// ProgressStep is how often a scan reports.
//
// Every file would put one event per file into the log and the record — a scan
// of ten thousand files writing ten thousand lines nobody reads. Every
// twenty-five keeps a hundred-file scan honest at four updates and a large one
// from flooding, and the final count arrives with the result either way.
const ProgressStep = 25

// Reporter counts and reports on the step, so a caller writes one line in its
// loop instead of the same arithmetic in each tool.
func Reporter(ctx context.Context, kind string, total int) func(done int) {
	report := Progress(ctx)
	return func(done int) {
		// The first is worth sending: it is what turns "something is happening"
		// into "this many so far", and on a small scan it may be the only one.
		if done == 1 || done%ProgressStep == 0 {
			report(kind, done, total)
		}
	}
}
