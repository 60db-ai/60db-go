package sixtydb

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// withQuery appends non-empty params to path as a query string (keys sorted by url.Values).
func withQuery(path string, params map[string]any) string {
	q := url.Values{}
	for k, v := range params {
		if v == nil {
			continue
		}
		if s := fmt.Sprint(v); s != "" {
			q.Set(k, s)
		}
	}
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

// doData sends a request and returns the body's `data` field (nil when absent).
func (c *Client) doData(ctx context.Context, method, path string, payload any, timeout time.Duration) (any, error) {
	return c.doDataWith(ctx, method, path, payload, nil, timeout)
}

// poll calls fn every interval until it reports done, the timeout passes, or ctx ends.
// onTimeout builds the error returned when the deadline is reached.
func poll(ctx context.Context, interval, timeout time.Duration, onTimeout func() error,
	fn func() (done bool, result map[string]any, err error)) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for {
		done, res, err := fn()
		if err != nil {
			return nil, err
		}
		if done {
			return res, nil
		}
		if time.Now().Add(interval).After(deadline) {
			return nil, onTimeout()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}
