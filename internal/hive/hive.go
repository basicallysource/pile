// Package hive reads the pieces sorters reported to a Hive (the sorters'
// shared backend) into SQLite files of pile's own, and reads them back as
// records. It only ever reads from Hive. There are two ways in:
//
//   - The fleet pull (fleet.go, fleetstore.go): signed in as a Hive user
//     (session.go), every machine that user may read, from
//     GET /api/machines/{id}/pieces, into hive.sqlite. Hive gives no
//     person's corrections there.
//   - The records pull (machines.go, machinestore.go): with an API key of
//     scope records:read (key.go), the key owner's own machines, each piece
//     with people's color verdicts and the color models' answers
//     (color.go), and each machine's lots and sorting runs, from
//     /api/records, into records.sqlite. pile runs it at start and again
//     every few minutes.
package hive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ErrNotFound is Hive's 404: no such thing, or not one this user may see.
var ErrNotFound = errors.New("not found")

// userAgent names pile on every request: the proxy in front of a Hive may
// refuse a client's default one.
const userAgent = "pile"

// pause is the wait between pages, so a pull never crowds the sorters
// reporting in.
const pause = 150 * time.Millisecond

// seconds reads a time Hive gave (ISO 8601) as unix seconds; ok is false
// when there is none or it does not read. One without an offset is UTC.
func seconds(s *string) (float64, bool) {
	if s == nil {
		return 0, false
	}
	t, err := time.Parse(time.RFC3339Nano, *s)
	if err != nil {
		if t, err = time.Parse("2006-01-02T15:04:05.999999999", *s); err != nil {
			return 0, false
		}
	}
	return float64(t.UnixNano()) / 1e9, true
}

// column is a time Hive gave as unix seconds for a column: null when there is
// none or it does not read.
func column(s *string) any {
	if t, ok := seconds(s); ok {
		return t
	}
	return nil
}

// statusError is an answer other than 200 and 404, with the start of its
// body.
type statusError struct {
	method, path string
	code         int
	body         string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("%s %s: status %d: %s", e.method, e.path, e.code, e.body)
}

// conn is one Hive's address and the HTTP client requests to it go through.
type conn struct {
	base *url.URL
	http *http.Client
}

func newConn(base string, jar http.CookieJar) (conn, error) {
	u, err := url.Parse(base)
	if err != nil {
		return conn{}, err
	}
	return conn{base: u, http: &http.Client{Jar: jar, Timeout: 2 * time.Minute}}, nil
}

// do sends a request (body as JSON when there is one, after set adds its
// headers) and reads a 200's JSON into out. Another status is ErrNotFound or
// a *statusError.
func (c conn) do(ctx context.Context, method, path string, body, out any, set func(*http.Request)) error {
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
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	set(req)
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
		return &statusError{method: method, path: ref.Path, code: res.StatusCode, body: string(msg)}
	case out == nil:
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
