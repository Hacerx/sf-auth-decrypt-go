package authdecrypt

import "context"

type Client struct {
	store *Store
}

func New(opts ...Option) (*Client, error) {
	store, err := NewStore(opts...)
	if err != nil {
		return nil, err
	}
	return &Client{store: store}, nil
}

func (c *Client) Aliases(ctx context.Context) (AliasFile, error) {
	if err := c.valid("read aliases"); err != nil {
		return nil, err
	}
	return c.store.Aliases(ctx)
}

func (c *Client) Orgs(ctx context.Context) ([]OrgRecord, error) {
	if err := c.valid("read orgs"); err != nil {
		return nil, err
	}
	return c.store.Orgs(ctx)
}

func (c *Client) OrgFromFile(ctx context.Context, path string) (OrgRecord, error) {
	if err := c.valid("read org from file"); err != nil {
		return nil, err
	}
	return c.store.OrgFromFile(ctx, path)
}

func (c *Client) OrgMap(ctx context.Context) (map[string]OrgRecord, error) {
	if err := c.valid("read org map"); err != nil {
		return nil, err
	}
	return c.store.OrgMap(ctx)
}

func (c *Client) ResolveOrg(ctx context.Context, selector string) (OrgRecord, error) {
	if err := c.valid("resolve org"); err != nil {
		return nil, err
	}
	return c.store.ResolveOrg(ctx, selector)
}

func (c *Client) valid(op string) error {
	if c == nil || c.store == nil {
		return newError(KindConfig, op, nil)
	}
	return nil
}
