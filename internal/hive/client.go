// Package hive pulls the pieces sorters reported to a Hive (the sorters'
// shared backend) into a SQLite file of pile's own, hive.sqlite, and reads
// them back as records. It only ever reads from Hive.
//
// It signs in as a Hive user and pulls every machine that user may read: their
// own, or with an admin's sign-in every machine on the Hive. A machine's
// pieces come from GET /api/machines/{id}/pieces, newest first, a page at a
// time.
package hive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

// Client is a signed-in session with one Hive. Its access token lasts minutes;
// it refreshes it when Hive says it ran out.
type Client struct {
	base *url.URL
	http *http.Client
	// The signed-in user's id.
	UserID string
}

// ErrNotFound is Hive's 404: no such thing, or not one this user may see.
var ErrNotFound = errors.New("not found")

// SignIn signs in to the Hive at base (https://hive.example.org) with a
// user's email and password.
func SignIn(ctx context.Context, base, email, password string) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	jar, _ := cookiejar.New(nil)
	c := &Client{base: u, http: &http.Client{Jar: jar, Timeout: 2 * time.Minute}}
	var me struct {
		ID string `json:"id"`
	}
	if err := c.do(ctx, http.MethodPost, "/api/auth/login", map[string]string{"email": email, "password": password}, &me); err != nil {
		return nil, fmt.Errorf("sign in: %w", err)
	}
	c.UserID = me.ID
	return c, nil
}

// Get reads path (with its query) into out, refreshing the session once if it
// ran out.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	err := c.do(ctx, http.MethodGet, path, nil, out)
	var st statusError
	if errors.As(err, &st) && st == http.StatusUnauthorized {
		if err := c.refresh(ctx); err != nil {
			return err
		}
		err = c.do(ctx, http.MethodGet, path, nil, out)
	}
	return err
}

// refresh trades the refresh cookie for a new session; Hive wants the CSRF
// cookie echoed in a header on every request that is not a read.
func (c *Client) refresh(ctx context.Context) error {
	if err := c.do(ctx, http.MethodPost, "/api/auth/refresh", struct{}{}, nil); err != nil {
		return fmt.Errorf("refresh the session: %w", err)
	}
	return nil
}

type statusError int

func (s statusError) Error() string { return fmt.Sprintf("status %d", int(s)) }

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	ref, err := url.Parse(path)
	if err != nil {
		return err
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base.ResolveReference(ref).String(), rd)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		for _, k := range c.http.Jar.Cookies(c.base) {
			if k.Name == "csrf_token" {
				req.Header.Set("X-CSRF-Token", k.Value)
			}
		}
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case res.StatusCode != http.StatusOK:
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 300))
		return fmt.Errorf("%s %s: %w: %s", method, ref.Path, statusError(res.StatusCode), msg)
	case out == nil:
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
