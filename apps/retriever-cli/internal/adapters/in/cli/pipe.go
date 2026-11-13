package cli

import (
	"errors"
	"io"
	"syscall"
)

// exitBrokenPipe is the status of a command whose standard output or standard error has no reader. The built binary ends
// by SIGPIPE, which the shell reports as 128+13. The in-process form of the same ending returns that status.
const exitBrokenPipe = 141

// pipeWatch passes writes to w and records whether one met a closed reader. A closed reader fails the write with EPIPE.
type pipeWatch struct {
	w      io.Writer
	broken bool
}

// Write writes p to the wrapped writer, and records a write that met a closed reader.
func (p *pipeWatch) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	if errors.Is(err, syscall.EPIPE) {
		p.broken = true
	}
	return n, err
}

// orDiscard returns w, or a writer that drops every byte when w is nil.
func orDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}
