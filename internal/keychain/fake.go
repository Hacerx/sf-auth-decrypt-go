package keychain

import "context"

type StaticProvider struct {
	Value []byte
	Err   error
}

func (p StaticProvider) Key(ctx context.Context, service, account string) ([]byte, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	if p.Err != nil {
		return nil, p.Err
	}
	return append([]byte(nil), p.Value...), nil
}

type RecordingProvider struct {
	Value   []byte
	Err     error
	Calls   int
	Service string
	Account string
}

func (p *RecordingProvider) Key(ctx context.Context, service, account string) ([]byte, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	p.Calls++
	p.Service = service
	p.Account = account
	if p.Err != nil {
		return nil, p.Err
	}
	return append([]byte(nil), p.Value...), nil
}
