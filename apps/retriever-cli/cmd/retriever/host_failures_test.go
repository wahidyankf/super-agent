package main

import (
	"errors"

	"github.com/wahidyankf/super-agent/libs/retriever-runtime/envfile"
)

// errFakeGetwd is the fault a fake working directory returns.
var errFakeGetwd = errors.New("fake working directory fault")

// failingEnvFile is an env file opener that always fails, so a test can check the wiring's error path.
func failingEnvFile(string) (*envfile.Credentials, error) {
	return nil, errFakeGetwd
}
