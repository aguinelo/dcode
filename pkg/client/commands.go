package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/aguinelo/dcode/internal/protocol"
)

// ListCommands asks which user-defined slash commands a client in workspace
// offers, the user's and the project's, with the text each one sends.
func (c *Client) ListCommands(ctx context.Context, workspace string) (protocol.CommandsResponse, error) {
	var out protocol.CommandsResponse
	if err := c.do(ctx, http.MethodGet, "/commands?workspace="+url.QueryEscape(workspace), nil, &out); err != nil {
		return protocol.CommandsResponse{}, err
	}
	return out, nil
}
