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

// ReadGitPR asks for the PR of the branch workspace has checked out now, its
// checks summed up, and the base a new PR would target.
func (c *Client) ReadGitPR(ctx context.Context, workspace string) (protocol.GitPRResponse, error) {
	var out protocol.GitPRResponse
	if err := c.do(ctx, http.MethodGet, "/git/pr?workspace="+url.QueryEscape(workspace), nil, &out); err != nil {
		return protocol.GitPRResponse{}, err
	}
	return out, nil
}

// CreateGitPR pushes the workspace's branch and opens a PR for it, as req
// says. Send only what a person confirmed: this is the one git route that
// changes something outside the machine.
func (c *Client) CreateGitPR(ctx context.Context, req protocol.CreateGitPRRequest) (protocol.GitPRResponse, error) {
	var out protocol.GitPRResponse
	if err := c.do(ctx, http.MethodPost, "/git/pr", req, &out); err != nil {
		return protocol.GitPRResponse{}, err
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
