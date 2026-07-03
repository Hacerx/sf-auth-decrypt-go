package authdecrypt

import "strings"

type ErrorKind string

const (
	KindNotFound      ErrorKind = "not_found"
	KindMissingKey    ErrorKind = "missing_key"
	KindInvalidFormat ErrorKind = "invalid_format"
	KindKeychain      ErrorKind = "keychain"
	KindDecrypt       ErrorKind = "decrypt"
	KindFile          ErrorKind = "file"
	KindConfig        ErrorKind = "config"
)

var (
	ErrNotFound      = &Error{Kind: KindNotFound}
	ErrMissingKey    = &Error{Kind: KindMissingKey}
	ErrInvalidFormat = &Error{Kind: KindInvalidFormat}
	ErrKeychain      = &Error{Kind: KindKeychain}
	ErrDecrypt       = &Error{Kind: KindDecrypt}
	ErrFile          = &Error{Kind: KindFile}
	ErrConfig        = &Error{Kind: KindConfig}
)

type Error struct {
	Kind ErrorKind
	Op   string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return "authdecrypt: unknown error"
	}

	parts := []string{"authdecrypt"}
	if e.Op != "" {
		parts = append(parts, e.Op)
	}
	if e.Kind != "" {
		parts = append(parts, string(e.Kind))
	} else {
		parts = append(parts, "unknown")
	}

	return strings.Join(parts, ": ")
}

func (e *Error) Unwrap() error {
	if e != nil {
		return e.Err
	}
	return nil
}

func (e *Error) Is(target error) bool {
	targetErr, ok := target.(*Error)
	return ok && e != nil && targetErr.Kind != "" && e.Kind == targetErr.Kind
}

func newError(kind ErrorKind, op string, err error) *Error {
	return &Error{Kind: kind, Op: op, Err: err}
}
