package report

import (
	"testing"

	"example/stats"
)

func TestRenderReadsTheSummary(t *testing.T) {
	var s stats.Summary
	if Render(s) == "" {
		t.Fatal("empty")
	}
}
