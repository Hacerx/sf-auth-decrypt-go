package keychain

import (
	"context"
	"errors"
)

const (
	ServiceSFDX  = "sfdx"
	AccountLocal = "local"
)

var (
	ErrMissingKey = errors.New("keychain key missing")
	ErrKeychain   = errors.New("keychain lookup failed")
)

type Provider interface {
	Key(ctx context.Context, service, account string) ([]byte, error)
}

type CommandRunner interface {
	Run(ctx context.Context, program string, args ...string) (CommandResult, error)
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}
