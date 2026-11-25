package sixtydb

import (
	"context"
	"time"
)

// AgentCalls is available as Client.Agents.Calls.
type AgentCalls struct{ c *Client }

// AgentPhoneLines is available as Client.Agents.PhoneLines.
type AgentPhoneLines struct{ c *Client }

// AgentCallListParams filters Calls.List. From and To are a DATE range (ISO 8601), not numbers.
// Sentiment is the call's sentiment label.
type AgentCallListParams struct {
	AgentID, Status, Direction, From, To string
	Limit, Offset                        int
	Sentiment                            string
}

func callPath(id string) string { return "/agents/calls/" + pathEscape(id) }

func (s *AgentCalls) List(ctx context.Context, p AgentCallListParams) (any, error) {
	q := map[string]any{"agent_id": p.AgentID, "status": p.Status, "direction": p.Direction, "from": p.From, "to": p.To,
		"sentiment": p.Sentiment}
	if p.Limit > 0 {
		q["limit"] = p.Limit
	}
	if p.Offset > 0 {
		q["offset"] = p.Offset
	}
	return s.c.doData(ctx, "GET", withQuery("/agents/calls", q), nil, timeoutDefault)
}

func (s *AgentCalls) Get(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", callPath(id), nil, timeoutDefault)
}

// PlaceInline places a call that carries its own agent (rings a real phone, spends credits). body: to,
// phone_line_id or sip{domain, username, password, caller_id}, agent{script, voice_id, ...},
// settings{max_duration_secs, ...}, metadata, variables. idempotencyKey ("" for none) makes a retry safe.
func (s *AgentCalls) PlaceInline(ctx context.Context, body map[string]any, idempotencyKey string) (any, error) {
	return s.c.doDataWith(ctx, "POST", "/agents/calls", body, idempotencyHeader(idempotencyKey), timeoutDefault)
}

// Usage returns the credits a call used: measured, total, services, turns, phone, price_usd.
func (s *AgentCalls) Usage(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", callPath(id)+"/usage", nil, timeoutDefault)
}

// ToolEvents lists the tool calls made during this call.
func (s *AgentCalls) ToolEvents(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", callPath(id)+"/tool-events", nil, timeoutDefault)
}

// Result returns the call's per-turn latency slices and eval results.
func (s *AgentCalls) Result(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", callPath(id)+"/result", nil, timeoutDefault)
}

// EndStream ends a running stream call; the result holds call and status "ending".
func (s *AgentCalls) EndStream(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "DELETE", "/agents/stream-calls/"+pathEscape(id), nil, timeoutDefault)
}

// HangUpWeb hangs up a browser (web) call.
func (s *AgentCalls) HangUpWeb(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "DELETE", "/agents/web-calls/"+pathEscape(id), nil, timeoutDefault)
}

func (s *AgentCalls) Transcript(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", callPath(id)+"/transcript", nil, timeoutDefault)
}

// Recording returns the WAV bytes (caller left, agent right; mix=true for one channel).
func (s *AgentCalls) Recording(ctx context.Context, id string, mix bool) ([]byte, error) {
	path := callPath(id) + "/recording"
	if mix {
		path += "?mix=1"
	}
	b, _, err := s.c.raw(ctx, "GET", path, nil, timeoutLong)
	return b, err
}

func (s *AgentCalls) DeleteRecording(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "DELETE", callPath(id)+"/recording", nil, timeoutDefault)
}

// AgentWaitOptions tunes the agents waiters. Zero values mean each waiter's defaults:
// WaitForCompletion 2 s / 20 min, WaitForDraft 2 s / 5 min, WaitForRun 4 s / 30 min.
type AgentWaitOptions struct{ Interval, Timeout time.Duration }

func transient(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Status == 429 || e.Status >= 500
	}
	return true // network error
}

// WaitForCompletion polls until the call is completed or failed and returns it. 429, 5xx
// and network errors are retried; timeout returns an *Error with Code "TIMEOUT".
func (s *AgentCalls) WaitForCompletion(ctx context.Context, id string, opts *AgentWaitOptions) (map[string]any, error) {
	return waitFor(ctx, opts, 2*time.Second, 20*time.Minute, "Timed out waiting for the call to end",
		func() (any, error) { return s.Get(ctx, id) },
		func(call map[string]any) bool {
			st := asString(call["status"])
			return st == "completed" || st == "failed"
		})
}

// waitFor polls get until done reports true for the returned object, the timeout passes (an *Error
// with Code "TIMEOUT") or ctx ends. 429, 5xx and network errors are retried; other errors abort.
func waitFor(ctx context.Context, opts *AgentWaitOptions, interval, timeout time.Duration, timeoutMsg string,
	get func() (any, error), done func(obj map[string]any) bool) (map[string]any, error) {
	if opts != nil {
		if opts.Interval > 0 {
			interval = opts.Interval
		}
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
	}
	return poll(ctx, interval, timeout,
		func() error { return &Error{Code: "TIMEOUT", Message: timeoutMsg} },
		func() (bool, map[string]any, error) {
			d, err := get()
			if err != nil {
				if ctx.Err() != nil || !transient(err) {
					return false, nil, err
				}
				return false, nil, nil
			}
			obj, _ := d.(map[string]any)
			return done(obj), obj, nil
		})
}

func (s *AgentPhoneLines) List(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/phone-lines", nil, timeoutDefault)
}

func (s *AgentPhoneLines) AvailableNumbers(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/phone-lines/available-numbers", nil, timeoutDefault)
}

func (s *AgentPhoneLines) Get(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/phone-lines/"+pathEscape(id), nil, timeoutDefault)
}

// snakePhoneLine maps agentId, callerId and inboundNumber to the API's snake_case names.
func snakePhoneLine(p map[string]any) map[string]any {
	ren := map[string]string{"agentId": "agent_id", "callerId": "caller_id", "inboundNumber": "inbound_number"}
	out := make(map[string]any, len(p))
	for k, v := range p {
		if n, ok := ren[k]; ok {
			k = n
		}
		out[k] = v
	}
	return out
}

// Create adds a line. kind "sixtydb": number, name, agentId, inbound. kind "byo": name, domain,
// username, password (both optional for an IP-trust carrier), callerId, inboundNumber, agentId, and the
// trunk fields numbers[{number, agent_id}], allowed_addresses[], allowed_numbers[], normalize_numbers,
// default_country_code, bridge_transfers. camelCase keys agentId, callerId, inboundNumber are mapped to snake_case.
func (s *AgentPhoneLines) Create(ctx context.Context, p map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", "/agents/phone-lines", snakePhoneLine(p), timeoutDefault)
}

func (s *AgentPhoneLines) Update(ctx context.Context, id string, p map[string]any) (any, error) {
	return s.c.doData(ctx, "PATCH", "/agents/phone-lines/"+pathEscape(id), snakePhoneLine(p), timeoutDefault)
}

func (s *AgentPhoneLines) Delete(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "DELETE", "/agents/phone-lines/"+pathEscape(id), nil, timeoutDefault)
}

// Setup returns the setup guide for a line: what to give the carrier.
func (s *AgentPhoneLines) Setup(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/phone-lines/"+pathEscape(id)+"/setup", nil, timeoutDefault)
}

// Check reports how an incoming call to the line would be answered (params: called_number, source_ip, caller_number).
func (s *AgentPhoneLines) Check(ctx context.Context, id string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", "/agents/phone-lines/"+pathEscape(id)+"/check", params, timeoutDefault)
}
