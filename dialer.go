package sixtydb

import "context"

// DialerService is available as Client.Dialer. Responses are the full JSON body.
type DialerService struct {
	c          *Client
	Numbers    *DialerNumbers
	Calls      *DialerCalls
	Recordings *DialerRecordings
	Billing    *DialerBilling
	Webhooks   *DialerWebhooks
}

func newDialerService(c *Client) *DialerService {
	return &DialerService{c: c, Numbers: &DialerNumbers{c}, Calls: &DialerCalls{c},
		Recordings: &DialerRecordings{c}, Billing: &DialerBilling{c}, Webhooks: &DialerWebhooks{c}}
}

func (s *DialerService) Status(ctx context.Context) (map[string]any, error) {
	return s.c.do(ctx, "GET", "/dialer/status", nil, timeoutDefault)
}

// DialerNumbers manages numbers. Buy spends credits.
type DialerNumbers struct{ c *Client }

func (n *DialerNumbers) Search(ctx context.Context) (map[string]any, error) {
	return n.c.do(ctx, "GET", "/dialer/pool", nil, timeoutDefault)
}

// Buy allocates a number (params: number, assign_for, outbound_trunk_id, inbound_trunk_id, channel_limit).
func (n *DialerNumbers) Buy(ctx context.Context, params map[string]any) (map[string]any, error) {
	if params == nil {
		params = map[string]any{}
	}
	return n.c.do(ctx, "POST", "/dialer/pool/allocate", params, timeoutDefault)
}

func (n *DialerNumbers) List(ctx context.Context) (map[string]any, error) {
	return n.c.do(ctx, "GET", "/dialer/numbers", nil, timeoutDefault)
}

func (n *DialerNumbers) Release(ctx context.Context, e164 string) (map[string]any, error) {
	return n.c.do(ctx, "DELETE", "/dialer/numbers/"+pathEscape(e164), nil, timeoutDefault)
}

func (n *DialerNumbers) SetCallerID(ctx context.Context, e164 string) (map[string]any, error) {
	return n.c.do(ctx, "PUT", "/dialer/caller-id", map[string]any{"caller_id": e164}, timeoutDefault)
}

// DialerCalls reads dialer call logs.
type DialerCalls struct{ c *Client }

// List returns call logs (params: limit, offset).
func (d *DialerCalls) List(ctx context.Context, params map[string]any) (map[string]any, error) {
	return d.c.do(ctx, "GET", withQuery("/dialer/logs", params), nil, timeoutDefault)
}

func (d *DialerCalls) Get(ctx context.Context, id string) (map[string]any, error) {
	return d.c.do(ctx, "GET", "/dialer/calls/"+pathEscape(id), nil, timeoutDefault)
}

func (d *DialerCalls) Live(ctx context.Context) (map[string]any, error) {
	return d.c.do(ctx, "GET", "/dialer/live", nil, timeoutDefault)
}

func (d *DialerCalls) LineStatus(ctx context.Context) (map[string]any, error) {
	return d.c.do(ctx, "GET", "/dialer/calls/line-status", nil, timeoutDefault)
}

// Reserve registers a call; direction may be "" (omitted).
func (d *DialerCalls) Reserve(ctx context.Context, to, from, sipCallID, direction string) (map[string]any, error) {
	body := map[string]any{"to": to, "from": from, "sip_call_id": sipCallID}
	if direction != "" {
		body["direction"] = direction
	}
	return d.c.do(ctx, "POST", "/dialer/calls", body, timeoutDefault)
}

// DialerRecordings manages call recordings.
type DialerRecordings struct{ c *Client }

// List filters: direction, status, e164, limit, offset.
func (r *DialerRecordings) List(ctx context.Context, params map[string]any) (map[string]any, error) {
	return r.c.do(ctx, "GET", withQuery("/dialer/recordings", params), nil, timeoutDefault)
}

func (r *DialerRecordings) PlaybackURL(ctx context.Context, id string) (map[string]any, error) {
	return r.c.do(ctx, "POST", "/dialer/recordings/"+pathEscape(id)+"/playback-url", nil, timeoutDefault)
}

func (r *DialerRecordings) Transcript(ctx context.Context, id string) (map[string]any, error) {
	return r.c.do(ctx, "POST", "/dialer/recordings/"+pathEscape(id)+"/transcript", nil, timeoutLong)
}

func (r *DialerRecordings) Delete(ctx context.Context, id string) (map[string]any, error) {
	return r.c.do(ctx, "DELETE", "/dialer/recordings/"+pathEscape(id), nil, timeoutDefault)
}

func (r *DialerRecordings) SetEnabled(ctx context.Context, enabled bool) (map[string]any, error) {
	return r.c.do(ctx, "PUT", "/dialer/recording", map[string]any{"enabled": enabled}, timeoutDefault)
}

// DialerBilling reads dialer billing.
type DialerBilling struct{ c *Client }

func (b *DialerBilling) Subscriptions(ctx context.Context) (map[string]any, error) {
	return b.c.do(ctx, "GET", "/dialer/subscriptions", nil, timeoutDefault)
}

// Usage params: limit.
func (b *DialerBilling) Usage(ctx context.Context, params map[string]any) (map[string]any, error) {
	return b.c.do(ctx, "GET", withQuery("/dialer/usage", params), nil, timeoutDefault)
}
