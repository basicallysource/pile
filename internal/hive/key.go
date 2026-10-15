package hive

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// KeyClient reads one Hive's records API with an API key of scope
// records:read, which sees only the key owner's own machines. The key goes
// only in the Authorization header: never in a URL, a log line or an error.
type KeyClient struct {
	conn
	key string
}

// NewKeyClient is a client of the Hive at base (https://hive.example.org)
// with an API key.
func NewKeyClient(base, key string) (*KeyClient, error) {
	if key == "" {
		return nil, errors.New("no API key")
	}
	c, err := newConn(base, nil)
	if err != nil {
		return nil, err
	}
	return &KeyClient{conn: c, key: key}, nil
}

// Get reads path (with its query) into out.
func (c *KeyClient) Get(ctx context.Context, path string, out any) error {
	err := c.do(ctx, http.MethodGet, path, nil, out, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+c.key)
	})
	st := (*statusError)(nil)
	if !errors.As(err, &st) {
		return err
	}
	switch {
	case st.code == http.StatusForbidden && strings.Contains(st.body, "error code: 1010"):
		// Cloudflare refusing the client by its signature, before Hive.
		return fmt.Errorf("GET %s: the proxy in front of Hive blocked this client (403, error code 1010), before Hive saw the key", st.path)
	case st.code == http.StatusUnauthorized || st.code == http.StatusForbidden:
		return fmt.Errorf("GET %s: Hive refused the API key (status %d): it is wrong, revoked, or lacks the records:read scope", st.path, st.code)
	}
	st.body = strings.ReplaceAll(st.body, c.key, "[key]")
	return st
}
