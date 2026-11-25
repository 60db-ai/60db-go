package sixtydb

import "context"

// DialerWebhooks manages dialer webhooks. Delete is a soft disable: the name stays reserved.
type DialerWebhooks struct{ c *Client }

func whPath(id string) string { return "/dialer/webhooks/" + pathEscape(id) }

func (w *DialerWebhooks) List(ctx context.Context) (map[string]any, error) {
	return w.c.do(ctx, "GET", "/dialer/webhooks", nil, timeoutDefault)
}

// Create params: name, url, event_types (required); description, status, timeout_ms,
// rate_limit_per_minute, max_attempts, max_event_age_seconds.
func (w *DialerWebhooks) Create(ctx context.Context, params map[string]any) (map[string]any, error) {
	return w.c.do(ctx, "POST", "/dialer/webhooks", params, timeoutDefault)
}

func (w *DialerWebhooks) Update(ctx context.Context, id string, params map[string]any) (map[string]any, error) {
	return w.c.do(ctx, "PATCH", whPath(id), params, timeoutDefault)
}

func (w *DialerWebhooks) Delete(ctx context.Context, id string) (map[string]any, error) {
	return w.c.do(ctx, "DELETE", whPath(id), nil, timeoutDefault)
}

func (w *DialerWebhooks) RotateSecret(ctx context.Context, id string) (map[string]any, error) {
	return w.c.do(ctx, "POST", whPath(id)+"/rotate-secret", nil, timeoutDefault)
}

// Test sends a test event (subscribe to "webhook.test" first).
func (w *DialerWebhooks) Test(ctx context.Context, id string) (map[string]any, error) {
	return w.c.do(ctx, "POST", whPath(id)+"/test", nil, timeoutDefault)
}

func (w *DialerWebhooks) Deliveries(ctx context.Context, id string) (map[string]any, error) {
	return w.c.do(ctx, "GET", whPath(id)+"/deliveries", nil, timeoutDefault)
}
