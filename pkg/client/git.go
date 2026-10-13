package client

import (
	"context"
	"net/http"
	"net/url"

	"github.com/aguinelo/dcode/internal/protocol"
)

// ReadGit asks where workspace is in git now — branch, head, top of the work
// tree, whether it is a linked worktree — and which files changed there since
// the last commit.
func (c *Client) ReadGit(ctx context.Context, workspace string) (protocol.GitResponse, error) {
	var out protocol.GitResponse
	if err := c.do(ctx, http.MethodGet, "/git?workspace="+url.QueryEscape(workspace), nil, &out); err != nil {
		return protocol.GitResponse{}, err
	}
	return out, nil
}

// ReadGitDiff asks for one changed file's unified diff, path as ReadGit lists it.
func (c *Client) ReadGitDiff(ctx context.Context, workspace, path string) (protocol.GitDiffResponse, error) {
	var out protocol.GitDiffResponse
	q := url.Values{"workspace": {workspace}, "path": {path}}
	if err := c.do(ctx, http.MethodGet, "/git/diff?"+q.Encode(), nil, &out); err != nil {
		return protocol.GitDiffResponse{}, err
	}
	return out, nil
}
