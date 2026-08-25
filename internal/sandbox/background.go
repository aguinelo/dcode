package sandbox

import (
	"context"
	"errors"
	"os/exec"
	"sync"
)

// Proc is a command that was started and is not being waited on.
//
// It satisfies the Handle the tools package defines. The two are kept
// structurally compatible rather than by import: tools must not depend on
// sandbox, or the boundary would be describable only in terms of the thing it
// confines.
type Proc struct {
	cmd *exec.Cmd
	buf *lockedBuffer

	mu   sync.Mutex
	code int
	done bool
}

// Output is everything the command has written so far.
func (p *Proc) Output() string { return p.buf.String() }

// Exited reports the exit code and whether it has exited at all.
func (p *Proc) Exited() (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.code, p.done
}

// Stop ends the command and everything it started.
//
// The group, not the process. A server is almost never the process that was
// launched: `npm start` is a shell that execs npm that spawns node, and killing
// the wrapper leaves the thing holding the port alive with nobody left who
// knows its name. That is the orphan the whole lifetime decision exists to
// prevent, so the kill has to reach as far as the start did.
func (p *Proc) Stop() {
	p.mu.Lock()
	done := p.done
	p.mu.Unlock()
	if done || p.cmd.Process == nil {
		return
	}
	killGroup(p.cmd)
}

// Start launches a command and returns before it finishes.
//
// ctx belongs to the turn, and the command deliberately does not: a server
// started in one turn is meant to still be there in the next, and binding it to
// CommandContext would kill it the moment the turn that asked for it ended.
// What owns it instead is the session's tool state, which stops every process
// it holds when it closes. Ownership is the whole of the lifetime rule — there
// is no cleanup handler to forget to register.
func (r Runner) Start(ctx context.Context, workdir, command string) (*Proc, error) {
	if r.Sandbox == nil {
		return nil, ErrUnavailable
	}
	// A turn already abandoned gets nothing started on its behalf.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cmd, err := r.Sandbox.Wrap(context.Background(), workdir, command, r.mode())
	if err != nil {
		return nil, err
	}
	setGroup(cmd)

	buf := &lockedBuffer{}
	cmd.Stdout, cmd.Stderr = buf, buf
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	p := &Proc{cmd: cmd, buf: buf}
	go func() {
		err := cmd.Wait()
		code := 0
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				code = ee.ExitCode()
			} else {
				code = -1
			}
		}
		p.mu.Lock()
		p.code, p.done = code, true
		p.mu.Unlock()
	}()
	return p, nil
}
