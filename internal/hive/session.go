package hive

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
)

// Session is a Hive user signed in to one Hive, for the fleet pull. Its
// access token lasts minutes; it refreshes it when Hive says it ran out.
type Session struct {
	conn
	// The signed-in user's id.
	UserID string
}

// SignIn signs in to the Hive at base (https://hive.example.org) with a
// user's email and password.
func SignIn(ctx context.Context, base, email, password string) (*Session, error) {
	jar, _ := cookiejar.New(nil)
	c, err := newConn(base, jar)
	if err != nil {
		return nil, err
	}
	s := &Session{conn: c}
	var me struct {
		ID string `json:"id"`
	}
	if err := s.send(ctx, http.MethodPost, "/api/auth/login", map[string]string{"email": email, "password": password}, &me); err != nil {
		return nil, fmt.Errorf("sign in: %w", err)
	}
	s.UserID = me.ID
	return s, nil
}

// Get reads path (with its query) into out, refreshing the session once if it
// ran out.
func (s *Session) Get(ctx context.Context, path string, out any) error {
	err := s.send(ctx, http.MethodGet, path, nil, out)
	if st := (*statusError)(nil); errors.As(err, &st) && st.code == http.StatusUnauthorized {
		if err := s.refresh(ctx); err != nil {
			return err
		}
		err = s.send(ctx, http.MethodGet, path, nil, out)
	}
	return err
}

// refresh trades the refresh cookie for a new session.
func (s *Session) refresh(ctx context.Context) error {
	if err := s.send(ctx, http.MethodPost, "/api/auth/refresh", struct{}{}, nil); err != nil {
		return fmt.Errorf("refresh the session: %w", err)
	}
	return nil
}

// send is a request in the session: Hive wants the CSRF cookie echoed in a
// header on every request that is not a read.
func (s *Session) send(ctx context.Context, method, path string, body, out any) error {
	return s.do(ctx, method, path, body, out, func(req *http.Request) {
		if body == nil {
			return
		}
		for _, k := range s.http.Jar.Cookies(s.base) {
			if k.Name == "csrf_token" {
				req.Header.Set("X-CSRF-Token", k.Value)
			}
		}
	})
}
