package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/policy"
)

// Fetch reads a document off the web.
//
// The tool suite put network access out of scope from the beginning, and the
// reason was sound at the time: a network tool without a permission model is a
// hole. That premise is gone. Consent is asked once per project and kept in
// grants, the policy already treats network as an axis of its own, and the
// evaluator already refuses what was not granted.
//
// A fetch, not a browser. It returns the text at a URL, which is what reading a
// changelog or a library's documentation needs. Search is a different
// capability with a different failure mode — a model given a search box will
// answer from whatever came back first — and it waits for evidence that
// fetching alone is not enough.
//
// Unlike bash, this runs in this process rather than behind the sandbox, so the
// gate is the policy verdict and not the operating system. That is the same
// guarantee every other tool here relies on: `read` is kept inside the
// workspace by the resolver, not by seatbelt. bash is the exception because a
// shell command is opaque, and this one is not.
type Fetch struct {
	// Client is injected so a test never reaches the network and so a
	// deployment can supply its own timeouts and proxy.
	Client *http.Client
	// Limit caps the body. A document that does not fit is truncated and says
	// so, like every other output here.
	Limit int
}

// FetchInput is the argument shape.
type FetchInput struct {
	URL string `json:"url"`
}

func (Fetch) Name() string { return "fetch" }

func (Fetch) Description() string {
	return "Read the text at a URL — documentation, a changelog, an error message someone published. " +
		"Reaching the network crosses a boundary the user is asked about, so use it when the answer " +
		"is genuinely out there and not when a file in the workspace has it. " +
		"Returns text; a binary body is refused rather than read. " +
		"The answer carries the URL it came from, and you should say so when you use it."
}

func (Fetch) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` +
		`"url":{"type":"string","description":"An http or https URL."}},` +
		`"required":["url"]}`)
}

// Declare reports the crossing and no path.
//
// Always the network, whatever the URL looks like. Reading the string to decide
// is the reasoning bash already rejected: there is no reading of it that
// answers whether this one reaches out, because all of them do.
func (f Fetch) Declare(input json.RawMessage) (policy.Request, error) {
	var in FetchInput
	if err := decode(f.Name(), input, &in); err != nil {
		return policy.Request{}, err
	}
	return policy.Request{Tool: f.Name(), Network: true, Command: in.URL}, nil
}

func (f Fetch) Execute(ctx context.Context, input json.RawMessage, s *State) (Result, error) {
	var in FetchInput
	if err := decode(f.Name(), input, &in); err != nil {
		return err.(*ToolError).Result(), nil
	}

	u, perr := url.Parse(strings.TrimSpace(in.URL))
	if perr != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		// file:// would read the disk through a tool whose declaration says it
		// touches no path — the policy gate answering a question nobody asked
		// it. Only the two schemes this tool is about.
		return errf(f.Name(), CodeBadInput,
			"Give an http or https URL, or use `read` for a file on disk.",
			"%q is not a web address", in.URL).Result(), nil
	}

	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if rerr != nil {
		return errf(f.Name(), CodeBadInput, "", "could not build the request: %v", rerr).Result(), nil
	}
	resp, derr := client.Do(req)
	if derr != nil {
		return errf(f.Name(), CodeNotFound,
			"Check the address, or say that you could not reach it.",
			"could not reach %s: %v", u.Host, derr).Result(), nil
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !isReadable(ct) {
		return errf(f.Name(), CodeBadInput,
			"This tool reads text. Say what you could not read.",
			"%s returned %s, which is not text", u.Host, firstToken(ct)).Result(), nil
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 256 << 10
	}
	body, ierr := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	if ierr != nil {
		return errf(f.Name(), CodeNotFound, "", "could not read the body: %v", ierr).Result(), nil
	}

	text := string(body)
	truncated := false
	if len(text) > limit {
		text = text[:limit]
		truncated = true
	}

	// The source rides with the text. A model that cannot tell a quotation
	// from its own summary will produce both, and the URL is what lets a
	// reader check which one they got.
	head := fmt.Sprintf("%s · %s", u.String(), resp.Status)
	if truncated {
		head += fmt.Sprintf(" · truncated at %d bytes", limit)
	}
	return Result{
		Output:    head + "\n\n" + text,
		Truncated: truncated,
		Meta:      Meta{Lines: countLines(text)},
	}, nil
}

// isReadable reports whether a content type is text this tool should return.
//
// An empty type is allowed: plenty of servers send none, and refusing them
// would refuse exactly the small static pages most worth reading.
func isReadable(ct string) bool {
	t := firstToken(ct)
	switch {
	case t == "":
		return true
	case strings.HasPrefix(t, "text/"):
		return true
	case t == "application/json", t == "application/xml", t == "application/xhtml+xml":
		return true
	case strings.HasSuffix(t, "+json"), strings.HasSuffix(t, "+xml"):
		return true
	}
	return false
}

func firstToken(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}
