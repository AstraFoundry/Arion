package account

import (
	"context"
	"errors"

	"arion.local/desktop/internal/oauthloopback"
)

type callback struct {
	state string
	code  string
	error string
}

type loopback struct{ srv *oauthloopback.Server }

// startLoopback serves one callback on a random 127.0.0.1 port. page renders
// the response body when it is not nil; otherwise the response is empty.
func startLoopback(ctx context.Context, page func() []byte, state string) (*loopback, string, error) {
	srv, redirect, err := oauthloopback.Start(ctx, "127.0.0.1:0", callbackPath, page, state)
	switch {
	case errors.Is(err, oauthloopback.ErrAddress):
		return nil, "", &Error{Code: CodeBrowser, Detail: "loopback"}
	case err != nil && ctx.Err() != nil:
		return nil, "", ctx.Err()
	case err != nil:
		return nil, "", wrap(CodeBrowser, "listen", err)
	}
	return &loopback{srv: srv}, redirect, nil
}

func (l *loopback) Wait(ctx context.Context) (callback, error) {
	cb, err := l.srv.Wait(ctx)
	return callback{state: cb.State, code: cb.Code, error: cb.Error}, err
}

func (l *loopback) Close() {
	if l != nil {
		l.srv.Close()
	}
}
