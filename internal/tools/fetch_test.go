package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fetchWith(h http.HandlerFunc) (Fetch, *httptest.Server) {
	srv := httptest.NewServer(h)
	return Fetch{Client: srv.Client(), Limit: 64 << 10}, srv
}

func runFetch(t *testing.T, f Fetch, s *State, url string) Result {
	t.Helper()
	in, err := json.Marshal(FetchInput{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.Execute(context.Background(), in, s)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// Reading a library's changelog is ordinary programming work, and dcode could
// not do it. The reason it was out of scope — a network tool with no permission
// model is a hole — stopped being true when grants and the policy were built.
func TestFetchReturnsTheDocument(t *testing.T) {
	f, srv := fetchWith(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("Go 1.26 removes the old loop variable semantics."))
	})
	defer srv.Close()
	s, _ := setup(t)

	res := runFetch(t, f, s, srv.URL)
	if res.IsError {
		t.Fatalf("fetching failed: %s", res.Output)
	}
	if !strings.Contains(res.Output, "loop variable") {
		t.Errorf("the document did not come back: %q", res.Output)
	}
	// A model that cannot tell a quotation from a summary will produce both,
	// so the answer says where it came from.
	if !strings.Contains(res.Output, srv.URL) {
		t.Errorf("the source is not in the answer: %q", res.Output)
	}
}

// Every call declares the crossing, whatever the URL looks like. Reading the
// string to decide is the reasoning bash already rejected: there is no reading
// of it that answers whether this one reaches out.
func TestFetchAlwaysDeclaresTheNetwork(t *testing.T) {
	for _, u := range []string{"https://example.com", "http://localhost:8080/health"} {
		req, err := Fetch{}.Declare([]byte(`{"url":"` + u + `"}`))
		if err != nil {
			t.Fatal(err)
		}
		if !req.Network {
			t.Errorf("%s was declared unable to reach the network", u)
		}
		if len(req.Paths) != 0 {
			t.Errorf("%s declared a path it does not touch: %v", u, req.Paths)
		}
	}
}

// A binary body is refused rather than decoded into the context window. An
// image or a tarball as text is thousands of tokens of noise and no answer.
func TestFetchRefusesABodyThatIsNotText(t *testing.T) {
	f, srv := fetchWith(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte{0x00, 0x01, 0x02})
	})
	defer srv.Close()
	s, _ := setup(t)

	res := runFetch(t, f, s, srv.URL)
	if !res.IsError {
		t.Fatal("a binary body was read into the context")
	}
	if !strings.Contains(res.Output, "octet-stream") {
		t.Errorf("the refusal does not say what it got: %q", res.Output)
	}
}

// Truncation is declared, like every other output in this codebase.
func TestFetchDeclaresItsTruncation(t *testing.T) {
	f, srv := fetchWith(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Repeat("x", 4096)))
	})
	defer srv.Close()
	f.Limit = 512
	s, _ := setup(t)

	res := runFetch(t, f, s, srv.URL)
	if res.IsError {
		t.Fatalf("failed: %s", res.Output)
	}
	if !res.Truncated {
		t.Error("a cut body did not report being cut")
	}
	if !strings.Contains(res.Output, "truncated") {
		t.Errorf("the cut is not declared in the text: %q", res.Output)
	}
}

// A status that is not success is the answer, not a failure to ask. The model
// reads it and decides, exactly as it does with a non-zero exit.
func TestFetchReportsTheStatusItGot(t *testing.T) {
	f, srv := fetchWith(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	})
	defer srv.Close()
	s, _ := setup(t)

	res := runFetch(t, f, s, srv.URL)
	if !strings.Contains(res.Output, "404") {
		t.Errorf("the status is not in the answer: %q", res.Output)
	}
}

// Only http and https. A file:// URL would read the disk through a tool whose
// whole declaration says it touches no path, which is the policy gate answering
// a question it was never asked.
func TestFetchRefusesASchemeThatIsNotWeb(t *testing.T) {
	s, _ := setup(t)
	for _, u := range []string{"file:///etc/passwd", "ftp://example.com", "not a url"} {
		res := runFetch(t, Fetch{Limit: 1024}, s, u)
		if !res.IsError {
			t.Errorf("%q was accepted", u)
		}
	}
}
