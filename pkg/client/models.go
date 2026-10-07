package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/aguinelo/dcode/internal/protocol"
)

// ListModels asks which models a session in workspace can ask for: the one it
// gets when it asks for none, and every profile it can name, each saying
// whether its family has measurements behind it. An empty workspace asks about
// the configuration the daemon started with.
func (c *Client) ListModels(ctx context.Context, workspace string) (protocol.ModelsResponse, error) {
	path := "/models"
	if workspace != "" {
		path += "?workspace=" + url.QueryEscape(workspace)
	}
	var out protocol.ModelsResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return protocol.ModelsResponse{}, err
	}
	return out, nil
}
