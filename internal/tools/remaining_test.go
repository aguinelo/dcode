package tools

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// A cut result says how much it left out. `Truncated` alone tells the model
// there is more and not whether it is a line or a megabyte — the difference
// between reading on and narrowing the question. `fetch`, `process` and a
// background `bash` set the flag and left Remaining at zero.
func TestTruncatedOutputAlwaysSaysWhatRemains(t *testing.T) {
	const body, limit = 4096, 512

	t.Run("fetch with a declared length", func(t *testing.T) {
		f, srv := fetchWith(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", strconv.Itoa(body))
			_, _ = w.Write([]byte(strings.Repeat("x", body)))
		})
		defer srv.Close()
		f.Limit = limit
		s, _ := setup(t)
		res := runFetch(t, f, s, srv.URL)
		if !res.Truncated || res.Remaining != body-limit {
			t.Errorf("Truncated=%v Remaining=%d, want true and %d", res.Truncated, res.Remaining, body-limit)
		}
	})

	t.Run("fetch with no declared length", func(t *testing.T) {
		f, srv := fetchWith(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			for i := 0; i < body/64; i++ {
				_, _ = w.Write([]byte(strings.Repeat("y", 64)))
				w.(http.Flusher).Flush() // chunked: the length is not known up front
			}
		})
		defer srv.Close()
		f.Limit = limit
		s, _ := setup(t)
		res := runFetch(t, f, s, srv.URL)
		if !res.Truncated || res.Remaining != body-limit {
			t.Errorf("Truncated=%v Remaining=%d, want true and %d", res.Truncated, res.Remaining, body-limit)
		}
		if !strings.Contains(res.Output, strconv.Itoa(body-limit)) {
			t.Errorf("the text does not say how much was left out: %q", res.Output[:120])
		}
	})

	out := strings.Repeat("noise\n", 200) + "THE LAST LINE\n"
	checkTail := func(t *testing.T, res Result) {
		t.Helper()
		if !res.Truncated {
			t.Fatal("output was cut without saying so")
		}
		if res.Remaining <= 0 || res.Remaining < len(out)-200 || res.Remaining >= len(out) {
			t.Errorf("Remaining=%d for %d bytes kept to a 200-byte window", res.Remaining, len(out))
		}
	}

	t.Run("process", func(t *testing.T) {
		s, _ := setup(t)
		s.Limits.MaxToolOutput = 200
		bg := &fakeBackground{}
		runBash(t, backgroundBash(bg), s, `{"command":"noisy","background":true}`)
		bg.handle(0).write(out)
		checkTail(t, runProcess(t, s, `{"id":"bg1"}`))
	})

	t.Run("bash in the background", func(t *testing.T) {
		s, _ := setup(t)
		s.Limits.MaxToolOutput = 200
		bg := &fakeBackground{seed: []*fakeHandle{{out: out}}}
		checkTail(t, runBash(t, backgroundBash(bg), s, `{"command":"noisy","background":true}`))
	})
}
