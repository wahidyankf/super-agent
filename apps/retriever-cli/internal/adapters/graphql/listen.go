package graphql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	usecases "github.com/wahidyankf/super-agent/libs/retriever-runtime/sources"
)

const (
	loopbackIP = "127.0.0.1"
	maxPort    = 65535

	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	// writeTimeout covers an export that waits out Outline's Retry-After: up to three waits of 60 seconds.
	writeTimeout = 5 * time.Minute
	idleTimeout  = 60 * time.Second
)

// shutdownTimeout bounds the graceful shutdown. A variable so a unit test can end it at once, without a real wait.
var shutdownTimeout = 5 * time.Second

var (
	errPortRange   = errors.New("port must be between 0 and 65535")
	errNotLoopback = errors.New("the listener must be bound to loopback")
	errNilArgument = errors.New("a listener and a handler are required")
)

// LoopbackAddr returns the address retriever serve binds at port: 127.0.0.1 with that port. Listen binds exactly
// this address, and the unit level reports it, so this one function decides the bind host.
func LoopbackAddr(port int) (*net.TCPAddr, error) {
	if port < 0 || port > maxPort {
		return nil, errPortRange
	}
	return &net.TCPAddr{IP: net.ParseIP(loopbackIP), Port: port}, nil
}

// Listen binds a TCP listener on 127.0.0.1 at port. Port 0 picks a free port. The function takes no host
// argument, so it cannot bind a wider address.
func Listen(port int) (net.Listener, error) {
	addr, err := LoopbackAddr(port)
	if err != nil {
		return nil, err
	}
	return net.Listen("tcp", addr.String())
}

// ListenFunc opens the listener of a port. The composition root passes Listen, which binds a real loopback socket. A test
// passes an in-memory listener, so the same serve path runs with no socket.
type ListenFunc func(port int) (net.Listener, error)

// Run is the body of retriever serve. It opens the listener at port with listen, mounts the GraphQL handler over svc,
// calls ready with the bound address, and serves until ctx ends. The composition root and the behaviour tests both call
// it, so every level runs the same wiring.
func Run(ctx context.Context, svc usecases.Service, port int, listen ListenFunc, logger *slog.Logger, ready func(addr string)) error {
	listener, err := listen(port)
	if err != nil {
		return err
	}
	return serveOn(ctx, listener, svc, logger, ready)
}

// serveOn mounts the GraphQL handler over svc on an already bound listener, calls ready with its address, and serves until
// ctx ends. A listener whose address is not TCP is refused by the handler, and the listener is closed then.
func serveOn(ctx context.Context, listener net.Listener, svc usecases.Service, logger *slog.Logger, ready func(addr string)) error {
	handler, err := NewHandler(Deps{Service: svc, Logger: logger}, WithListener(listener))
	if err != nil {
		return errors.Join(err, listener.Close())
	}
	ready(listener.Addr().String())
	return Serve(ctx, listener, handler)
}

// Serve serves handler on l until ctx is cancelled, then shuts down gracefully. It refuses a listener that
// is not bound to loopback, before it accepts anything. It returns nil after a clean shutdown.
func Serve(ctx context.Context, l net.Listener, handler http.Handler) error {
	if l == nil || handler == nil {
		return errNilArgument
	}
	if !isLoopback(l.Addr()) {
		return errNotLoopback
	}
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(l) }()

	select {
	case err := <-serveErr:
		// Serve returns ErrServerClosed only after Shutdown, and Shutdown runs only in the ctx branch below, so a value here
		// is always a listener failure.
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	shutdownErr := srv.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		shutdownErr = errors.Join(shutdownErr, srv.Close())
	}
	<-serveErr // Serve returns http.ErrServerClosed once Shutdown or Close has run.
	if shutdownErr != nil {
		return fmt.Errorf("shutdown of the GraphQL server: %w", shutdownErr)
	}
	return nil
}

// isLoopback reports whether addr is a TCP address on a loopback IP.
func isLoopback(addr net.Addr) bool {
	tcp, ok := addr.(*net.TCPAddr)
	return ok && tcp.IP.IsLoopback()
}
