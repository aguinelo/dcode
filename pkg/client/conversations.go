package client

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// ListConversations lists every conversation the daemon knows, live and
// recorded, newest activity first. An empty workspace lists them all.
func (c *Client) ListConversations(ctx context.Context, workspace string) ([]protocol.Conversation, error) {
	path := "/conversations"
	if workspace != "" {
		path += "?workspace=" + url.QueryEscape(workspace)
	}
	var out protocol.ListConversationsResponse
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out.Conversations, err
}

// WatchConversations streams the list: a snapshot, then each change.
//
// A dropped connection is reconnected, and the new one opens with a new
// snapshot, which a client applies the way it applied the first: the list has
// no position to resume from, because the snapshot is the position. A refusal
// the daemon will give again is sent on the error channel, and the watch ends.
func (c *Client) WatchConversations(ctx context.Context) (<-chan protocol.ConversationChange, <-chan error) {
	changes := make(chan protocol.ConversationChange, 64)
	errs := make(chan error, 1)

	go func() {
		defer close(changes)
		defer close(errs)
		backoff := 200 * time.Millisecond
		for {
			got, err := c.watchOnce(ctx, changes)
			if ctx.Err() != nil {
				return
			}
			if got {
				backoff = 200 * time.Millisecond
			}
			if _, refused := protocol.AsError(err); refused {
				errs <- err
				return
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
	return changes, errs
}

// watchOnce consumes one connection, and reports whether anything came on it.
func (c *Client) watchOnce(ctx context.Context, out chan<- protocol.ConversationChange) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/conversations/events", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, decodeError(resp)
	}

	got := false
	sc := bufio.NewScanner(resp.Body)
	// A snapshot carries the whole list, which outgrows the default line.
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var ch protocol.ConversationChange
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &ch); err != nil {
			continue
		}
		select {
		case out <- ch:
			got = true
		case <-ctx.Done():
			return got, ctx.Err()
		}
	}
	return got, sc.Err()
}
