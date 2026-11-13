package mcp

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

// initializeLine is the first message of every session, as a client sends it on standard input.
const initializeLine = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18",` +
	`"capabilities":{},"clientInfo":{"name":"test","version":"0"}}}` + "\n"

// TestServeAnswersInitializeOverPipes checks the real stdio path: Serve reads newline-delimited JSON-RPC from its input and
// writes the reply to its output. The client waits for the reply before it closes its input, as a real client does, because
// the SDK ends the session at end of input and drops a request that is still in flight.
func TestServeAnswersInitializeOverPipes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	done := make(chan error, 1)
	go func() {
		err := Serve(ctx, emptyService(t), Options{Version: "test"}, inR, outW)
		_ = outW.Close()
		done <- err
	}()

	if _, err := io.WriteString(inW, initializeLine); err != nil {
		t.Fatal(err)
	}
	reply, err := bufio.NewReader(outR).ReadString('\n')
	if err != nil {
		t.Fatalf("no initialize reply: %v", err)
	}
	if !strings.Contains(reply, `"protocolVersion"`) || !strings.Contains(reply, `"jsonrpc":"2.0"`) {
		t.Fatalf("reply is not an initialize answer: %q", reply)
	}
	if err := inW.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("Serve returned %v, want nil after the input ended", err)
	}
}

// TestServeReturnsWhenInputEnds checks that a reader with nothing in it ends the session without an answer.
func TestServeReturnsWhenInputEnds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var out bytes.Buffer
	if err := Serve(ctx, emptyService(t), Options{}, strings.NewReader(""), &out); err != nil {
		t.Fatalf("Serve returned %v, want nil", err)
	}
	if out.Len() != 0 {
		t.Fatalf("standard output is %q, want nothing", out.String())
	}
}
