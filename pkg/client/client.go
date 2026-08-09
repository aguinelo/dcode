// Package client is the reference client for the dcode protocol.
//
// It is the first consumer of the contract. If the API feels awkward here, the
// protocol is what needs fixing, not this package.
//
// It holds no session state: only a read position, which is what lets a client
// die and rejoin without the session noticing.
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// Client talks to a dcode daemon over a Unix socket.
type Client struct {
	http *http.Client
	base string
}

// New builds a client for a socket path.
func New(socketPath string) *Client {
	return &Client{
		base: "http://dcode/" + protocol.Version,
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", socketPath)
				},
			},
			// No overall timeout: an event stream is meant to stay open for the
			// life of a session.
		},
	}
}

// Health reports whether a daemon is answering.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://dcode/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer drain(resp)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("client: daemon reported %s", resp.Status)
	}
	return nil
}

// CreateSession opens a session.
func (c *Client) CreateSession(ctx context.Context, req protocol.CreateSessionRequest) (protocol.Session, error) {
	var out protocol.Session
	err := c.do(ctx, http.MethodPost, "/sessions", req, &out)
	return out, err
}

// ListSessions returns live sessions.
func (c *Client) ListSessions(ctx context.Context) ([]protocol.Session, error) {
	var out []protocol.Session
	err := c.do(ctx, http.MethodGet, "/sessions", nil, &out)
	return out, err
}

// GetSession returns one session.
func (c *Client) GetSession(ctx context.Context, id string) (protocol.Session, error) {
	var out protocol.Session
	err := c.do(ctx, http.MethodGet, "/sessions/"+id, nil, &out)
	return out, err
}

// DeleteSession closes a session.
func (c *Client) DeleteSession(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/sessions/"+id, nil, nil)
}

// Submit sends user input.
func (c *Client) Submit(ctx context.Context, id, text string) error {
	return c.do(ctx, http.MethodPost, "/sessions/"+id+"/turns",
		protocol.SubmitTurnRequest{Text: text}, nil)
}

// Interrupt cancels the running turn.
func (c *Client) Interrupt(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPost, "/sessions/"+id+"/interrupt", nil, nil)
}

// Resolve answers a pending approval.
func (c *Client) Resolve(ctx context.Context, id, approvalID string, d protocol.ApprovalDecision) error {
	return c.do(ctx, http.MethodPost,
		"/sessions/"+id+"/approvals/"+approvalID,
		protocol.ResolveApprovalRequest{Decision: d}, nil)
}

// Subscribe streams events from `from`, reconnecting on its own.
//
// Reconnection resumes at the last sequence seen plus one, which is why the
// read position is the only state a client keeps. Reconnecting therefore
// behaves exactly like connecting for the first time.
func (c *Client) Subscribe(ctx context.Context, id string, from uint64) (<-chan protocol.Event, <-chan error) {
	events := make(chan protocol.Event, 256)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		next := from
		backoff := 200 * time.Millisecond

		for {
			n, err := c.streamOnce(ctx, id, next, events)
			if n > 0 {
				next = n + 1
				backoff = 200 * time.Millisecond // progress resets the backoff
			}
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				// A session that no longer exists, or history that has been
				// dropped, cannot be recovered by retrying.
				if pe, ok := protocol.AsError(err); ok &&
					(pe.Code == protocol.CodeSessionNotFound || pe.Code == protocol.CodeEventsExpired) {
					errs <- err
					return
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 5*time.Second {
				backoff *= 2
			}
		}
	}()
	return events, errs
}

// streamOnce consumes one connection, returning the last sequence delivered.
func (c *Client) streamOnce(ctx context.Context, id string, from uint64, out chan<- protocol.Event) (uint64, error) {
	url := fmt.Sprintf("%s/sessions/%s/events?from=%d", c.base, id, from)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, decodeError(resp)
	}

	var last uint64
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue // id: lines and : ping comments carry nothing to decode
		}
		var ev protocol.Event
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &ev); err != nil {
			continue
		}
		select {
		case out <- ev:
			last = ev.Seq
		case <-ctx.Done():
			return last, ctx.Err()
		}
	}
	return last, sc.Err()
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer drain(resp)

	if resp.StatusCode >= 400 {
		return decodeError(resp)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// decodeError preserves the server's code so callers can branch on it rather
// than on a status number or a message.
func decodeError(resp *http.Response) error {
	var pe protocol.Error
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&pe); err != nil || pe.Code == "" {
		return protocol.Errorf(protocol.CodeInternal, "daemon reported %s", resp.Status)
	}
	return &pe
}

func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
	resp.Body.Close()
}
