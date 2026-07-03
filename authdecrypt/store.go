package authdecrypt

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/hacerx/sf-auth-decrypt-go/internal/cryptoutil"
	"github.com/hacerx/sf-auth-decrypt-go/internal/keychain"
)

type Store struct{ options options }

func NewStore(opts ...Option) (*Store, error) {
	cfg, err := defaultOptions()
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}
	cfg.withDefaults()
	return &Store{options: cfg}, nil
}

func (s *Store) Aliases(ctx context.Context) (AliasFile, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	aliases := make(AliasFile)
	for _, adapter := range s.options.adapters {
		got, err := adapter.Aliases(ctx)
		if err != nil {
			return nil, wrapIfNeeded(KindFile, "read aliases", err)
		}
		for k, v := range got {
			aliases[k] = v
		}
	}
	return aliases, nil
}

func (s *Store) Orgs(ctx context.Context) ([]OrgRecord, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	var orgs []OrgRecord
	decrypt := s.decryptor()
	for _, adapter := range s.options.adapters {
		got, err := adapter.Orgs(ctx)
		if err != nil {
			return nil, wrapIfNeeded(KindFile, "read orgs", err)
		}
		for _, org := range got {
			decrypted, err := decrypt(ctx, org)
			if err != nil {
				return nil, err
			}
			orgs = append(orgs, decrypted)
		}
	}
	return orgs, nil
}

func (s *Store) OrgFromFile(ctx context.Context, path string) (OrgRecord, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(path) == "" {
		return nil, newError(KindNotFound, "read org from file", nil)
	}
	for _, adapter := range s.options.adapters {
		org, err := adapter.OrgFromFile(ctx, path)
		if err == nil {
			return s.decryptor()(ctx, org)
		}
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, fs.ErrNotExist) {
			return nil, wrapIfNeeded(KindFile, "read org from file", err)
		}
	}
	return nil, newError(KindNotFound, "read org from file", nil)
}

func (s *Store) OrgMap(ctx context.Context) (map[string]OrgRecord, error) {
	orgs, err := s.Orgs(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]OrgRecord, len(orgs))
	for _, org := range orgs {
		if username, ok := orgUsername(org); ok {
			m[username] = org
		}
	}
	return m, nil
}

func (s *Store) ResolveOrg(ctx context.Context, selector string) (OrgRecord, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, newError(KindNotFound, "resolve org", nil)
	}
	aliases, err := s.Aliases(ctx)
	if err != nil {
		return nil, err
	}
	if username, ok := aliases[selector]; ok {
		selector = username
	}
	orgMap, err := s.OrgMap(ctx)
	if err != nil {
		return nil, err
	}
	if org, ok := orgMap[selector]; ok {
		return org, nil
	}
	return nil, newError(KindNotFound, "resolve org", nil)
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return newError(KindFile, "context", err)
	}
	return nil
}

func wrapIfNeeded(kind ErrorKind, op string, err error) error {
	var authErr *Error
	if errors.As(err, &authErr) {
		return err
	}
	return newError(kind, op, err)
}

func orgUsername(org OrgRecord) (string, bool) {
	for _, key := range []string{"username", "userName"} {
		if value, ok := org[key].(string); ok && value != "" {
			return value, true
		}
	}
	return "", false
}

func (s *Store) decryptor() func(context.Context, OrgRecord) (OrgRecord, error) {
	var key []byte
	var keyLoaded bool
	return func(ctx context.Context, org OrgRecord) (OrgRecord, error) {
		if !hasEncryptedField(org) {
			return cloneRecord(org), nil
		}
		if !keyLoaded {
			got, err := s.options.keyProvider.Key(ctx, keychain.ServiceSFDX, keychain.AccountLocal)
			if err != nil {
				if errors.Is(err, keychain.ErrMissingKey) {
					return nil, newError(KindMissingKey, "retrieve key", err)
				}
				return nil, newError(KindKeychain, "retrieve key", err)
			}
			key = append([]byte(nil), got...)
			keyLoaded = true
		}
		decrypted, err := cryptoutil.DecryptTopLevelFields(org, key)
		if err != nil {
			if errors.Is(err, cryptoutil.ErrInvalidFormat) {
				return nil, newError(KindInvalidFormat, "decrypt org", err)
			}
			return nil, newError(KindDecrypt, "decrypt org", err)
		}
		return OrgRecord(decrypted), nil
	}
}

func hasEncryptedField(org OrgRecord) bool {
	for _, value := range org {
		stringValue, ok := value.(string)
		if ok && cryptoutil.LooksEncrypted(stringValue) {
			return true
		}
	}
	return false
}

func cloneRecord(org OrgRecord) OrgRecord {
	if org == nil {
		return nil
	}
	clone := make(OrgRecord, len(org))
	for k, v := range org {
		clone[k] = v
	}
	return clone
}
