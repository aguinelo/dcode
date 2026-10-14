package client

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aguinelo/dcode/internal/protocol"
)

// StartLoop asks the daemon to run `/loop`, with args as the person typed them
// after the command. The daemon parses them and conducts the run; closing this
// client does not stop it.
func (c *Client) StartLoop(ctx context.Context, req protocol.StartLoopRequest) (protocol.LoopRun, error) {
	var out protocol.LoopRun
	err := c.do(ctx, http.MethodPost, "/loops", req, &out)
	return out, err
}

// ListLoops lists the runs the daemon holds, oldest first. An empty workspace
// lists them all.
func (c *Client) ListLoops(ctx context.Context, workspace string) ([]protocol.LoopRun, error) {
	path := "/loops"
	if workspace != "" {
		path += "?workspace=" + url.QueryEscape(workspace)
	}
	var out protocol.ListLoopsResponse
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out.Loops, err
}

// ContinueLoop starts the work of a run waiting for its proposal to be
// reviewed. Another client having continued it first is refused with
// loop_not_awaiting_review.
func (c *Client) ContinueLoop(ctx context.Context, id string) (protocol.LoopRun, error) {
	var out protocol.LoopRun
	err := c.do(ctx, http.MethodPost, "/loops/"+url.PathEscape(id)+"/continue", nil, &out)
	return out, err
}

// StopLoop stops a run now (protocol.StopNow) or after the folder it is in
// (protocol.StopAfterSpec).
func (c *Client) StopLoop(ctx context.Context, id, when string) (protocol.LoopRun, error) {
	var out protocol.LoopRun
	err := c.do(ctx, http.MethodPost, "/loops/"+url.PathEscape(id)+"/stop", protocol.StopLoopRequest{When: when}, &out)
	return out, err
}

// ErrUnreadableLoopFrame is a frame of the stream of loops this client could
// not read. It ends the watch: a list missing a change is a list that lies.
var ErrUnreadableLoopFrame = errors.New("a frame of the list of loops could not be read")

// WatchLoops streams the runs: a snapshot, then each change. Reconnecting
// opens with a new snapshot, as WatchConversations does; a refusal the daemon
// will give again is sent on the error channel, and the watch ends.
func (c *Client) WatchLoops(ctx context.Context) (<-chan protocol.LoopChange, <-chan error) {
	changes := make(chan protocol.LoopChange, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(changes)
		defer close(errs)
		backoff := 200 * time.Millisecond
		for {
			got, err := c.watchLoopsOnce(ctx, changes)
			if ctx.Err() != nil {
				return
			}
			if got {
				backoff = 200 * time.Millisecond
			}
			if _, refused := protocol.AsError(err); refused || errors.Is(err, ErrUnreadableLoopFrame) {
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

func (c *Client) watchLoopsOnce(ctx context.Context, out chan<- protocol.LoopChange) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/loops/events", nil)
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
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var ch protocol.LoopChange
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &ch); err != nil {
			// Skipping it would leave a list that silently disagrees with the
			// daemon; the watch ends and says why.
			return got, fmt.Errorf("%w: %v", ErrUnreadableLoopFrame, err)
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
