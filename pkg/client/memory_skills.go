package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/aguinelo/dcode/internal/protocol"
)

// ReadMemory asks what a session in workspace reads as memory: every entry of
// its memory file, which of them a session shows, and the blocks that are not
// memories, with why.
func (c *Client) ReadMemory(ctx context.Context, workspace string) (protocol.MemoryResponse, error) {
	var out protocol.MemoryResponse
	if err := c.do(ctx, http.MethodGet, "/memory?workspace="+url.QueryEscape(workspace), nil, &out); err != nil {
		return protocol.MemoryResponse{}, err
	}
	return out, nil
}

// ListSkills asks which skills a session in workspace has available, the
// user's and the project's, and what was said while loading them.
func (c *Client) ListSkills(ctx context.Context, workspace string) (protocol.SkillsResponse, error) {
	var out protocol.SkillsResponse
	if err := c.do(ctx, http.MethodGet, "/skills?workspace="+url.QueryEscape(workspace), nil, &out); err != nil {
		return protocol.SkillsResponse{}, err
	}
	return out, nil
}
