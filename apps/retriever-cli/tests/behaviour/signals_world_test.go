package behaviour

import "os"

// noSignalNotify is the signal source of the in-process protocol sessions: it registers nothing, so a session never
// changes the process's signal handlers. Each session ends by its context, which a step cancels with a signal cause.
func noSignalNotify(chan<- os.Signal, ...os.Signal) {}

// noSignalStop releases the channel of noSignalNotify, which registered nothing.
func noSignalStop(chan<- os.Signal) {}
