package sixtydb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// recorder captures requests and answers with a canned body.
type recorder struct {
	method, uri, ctype, body string
	form                     *http.Request
}

func recClient(t *testing.T, resp string) (*Client, *recorder) {
	r := &recorder{}
	c := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		r.method, r.uri, r.ctype = req.Method, req.URL.RequestURI(), req.Header.Get("Content-Type")
		if strings.HasPrefix(r.ctype, "multipart/") {
			req.ParseMultipartForm(1 << 20)
			r.form = req
		} else {
			b, _ := io.ReadAll(req.Body)
			r.body = string(b)
		}
		io.WriteString(w, resp)
	})
	return c, r
}

func TestJudgeShapes(t *testing.T) {
	ctx := context.Background()
	c, r := recClient(t, `{"data":{"x":1},"pagination":{"total":3}}`)
	d, err := c.Judge.Evaluate(ctx, map[string]any{"state": "s"})
	if err != nil || d.(map[string]any)["x"] != 1.0 || r.uri != "/judge/evaluate" || r.method != "POST" {
		t.Fatalf("%v %v %+v", d, err, r)
	}
	c.Judge.Extract(ctx, nil)
	if r.uri != "/judge/extract" {
		t.Fatal(r.uri)
	}
	c.Judge.Usage(ctx, "")
	if r.uri != "/judge/usage?period=current_month" {
		t.Fatal(r.uri)
	}
	c.Judge.Rubrics.Update(ctx, "a/b", map[string]any{"name": "n"})
	if r.method != "PATCH" || r.uri != "/judge/rubrics/a%2Fb" {
		t.Fatalf("%+v", r)
	}
	if err := c.Judge.Rubrics.Delete(ctx, "r1"); err != nil || r.method != "DELETE" {
		t.Fatal(err)
	}
	runs, err := c.Judge.Runs.List(ctx, map[string]any{"needs_review": true, "limit": 5})
	if err != nil || runs["pagination"] == nil || runs["runs"] == nil {
		t.Fatalf("%v %v", runs, err)
	}
	if !strings.HasPrefix(r.uri, "/judge/evaluations?") || !strings.Contains(r.uri, "needs_review=true") || !strings.Contains(r.uri, "limit=5") {
		t.Fatal(r.uri)
	}
	c.Judge.Runs.List(ctx, map[string]any{"needs_review": false})
	if r.uri != "/judge/evaluations" {
		t.Fatal(r.uri)
	}
	c.Judge.Runs.Delete(ctx, "e1")
	if r.uri != "/judge/evaluations/e1" {
		t.Fatal(r.uri)
	}
}

func TestMusicShapes(t *testing.T) {
	ctx := context.Background()
	c, r := recClient(t, `{"success":true}`)
	c.Music.Create(ctx, map[string]any{"prompt": "p"}, nil)
	if r.uri != "/songs" || r.ctype != "application/json" || !strings.Contains(r.body, `"prompt":"p"`) {
		t.Fatalf("%+v", r)
	}
	c.Music.Create(ctx, map[string]any{"prompt": "p", "instrumental": true}, &UploadFile{Data: []byte("la la")})
	f := r.form.MultipartForm
	if r.form.FormValue("instrumental") != "true" || len(f.File["lyrics_file"]) != 1 || f.File["lyrics_file"][0].Filename != "lyrics.txt" {
		t.Fatalf("%+v", f)
	}
	c.Music.CreateVoice(ctx, "v", UploadFile{Filename: "a.mp3", Data: []byte("ID3x")})
	f = r.form.MultipartForm
	if r.uri != "/songs/voices" || r.form.FormValue("consent") != "true" || r.form.FormValue("name") != "v" ||
		len(f.File["voice_reference"]) != 1 || f.File["voice_reference"][0].Header.Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("%+v", f)
	}
	c.Music.List(ctx, map[string]any{"liked": true, "limit": 2})
	if !strings.Contains(r.uri, "liked=true") || !strings.HasPrefix(r.uri, "/songs?") {
		t.Fatal(r.uri)
	}
	c.Music.Update(ctx, "s1", map[string]any{"liked": true})
	if r.method != "PATCH" || r.uri != "/songs/s1" {
		t.Fatal(r.method, r.uri)
	}
	c.Music.Health(ctx)
	if r.uri != "/songs/health" {
		t.Fatal(r.uri)
	}
	c.Music.Download(ctx, "s1")
	if r.uri != "/songs/s1/download" {
		t.Fatal(r.uri)
	}
}

func TestMusicWaitForCompletion(t *testing.T) {
	var n int32
	c := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		st := "running"
		if atomic.AddInt32(&n, 1) >= 3 {
			st = "succeeded"
		}
		io.WriteString(w, `{"data":{"id":"s","status":"`+st+`"}}`)
	})
	song, err := c.Music.WaitForCompletion(context.Background(), "s", &MusicWaitOptions{Interval: 5 * time.Millisecond})
	if err != nil || song["status"] != "succeeded" || n != 3 {
		t.Fatalf("%v %v %d", song, err, n)
	}
	// timeout
	c2 := newTestClient(t, func(w http.ResponseWriter, req *http.Request) { io.WriteString(w, `{"data":{"status":"running"}}`) })
	_, err = c2.Music.WaitForCompletion(context.Background(), "s", &MusicWaitOptions{Interval: 5 * time.Millisecond, Timeout: 20 * time.Millisecond})
	var te *Error
	if !errors.As(err, &te) || te.Code != "TIMEOUT" {
		t.Fatal(err)
	}
	// ctx cancel
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = c2.Music.WaitForCompletion(ctx, "s", &MusicWaitOptions{Interval: 10 * time.Millisecond, Timeout: time.Minute})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestAgentsShapes(t *testing.T) {
	ctx := context.Background()
	c, r := recClient(t, `{"data":{"id":"a1"}}`)
	d, err := c.Agents.Get(ctx, "a 1")
	if err != nil || d.(map[string]any)["id"] != "a1" || r.uri != "/agents/a%201" {
		t.Fatalf("%v %v %s", d, err, r.uri)
	}
	c.Agents.Calls.List(ctx, AgentCallListParams{AgentID: "a", Status: "completed", From: "2026-01-01", Limit: 10})
	for _, want := range []string{"agent_id=a", "status=completed", "from=2026-01-01", "limit=10"} {
		if !strings.Contains(r.uri, want) {
			t.Errorf("%s missing in %s", want, r.uri)
		}
	}
	c.Agents.Calls.Recording(ctx, "c1", true)
	if r.uri != "/agents/calls/c1/recording?mix=1" {
		t.Fatal(r.uri)
	}
	c.Agents.PhoneLines.Create(ctx, map[string]any{"kind": "byo", "callerId": "+1", "inboundNumber": "+2", "agentId": "a"})
	for _, want := range []string{`"caller_id":"+1"`, `"inbound_number":"+2"`, `"agent_id":"a"`} {
		if !strings.Contains(r.body, want) {
			t.Errorf("%s missing in %s", want, r.body)
		}
	}
	c.Agents.Call(ctx, "a1", PlaceCallParams{To: "+1", PhoneLineID: "l1"})
	if r.uri != "/agents/a1/calls" || !strings.Contains(r.body, `"phone_line_id":"l1"`) {
		t.Fatalf("%+v", r)
	}
	c.Agents.RotateWebhookSecret(ctx, "a1")
	if r.uri != "/agents/a1/rotate-webhook-secret" {
		t.Fatal(r.uri)
	}
	c.Agents.StreamCall(ctx, "a1", nil, nil)
	c.Agents.ToolEvents(ctx, "a1")
	if r.uri != "/agents/a1/tool-events" {
		t.Fatal(r.uri)
	}
}

func TestAgentsErrorShape(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(403)
		io.WriteString(w, `{"success":false,"message":"denied","code":"POLICY_DENY"}`)
	})
	_, err := c.Agents.List(context.Background(), nil)
	var e *Error
	if !errors.As(err, &e) || e.Status != 403 || e.Code != "POLICY_DENY" {
		t.Fatal(err)
	}
	_, err = c.Agents.Calls.Recording(context.Background(), "x", false)
	if !errors.As(err, &e) || e.Code != "POLICY_DENY" {
		t.Fatal(err)
	}
}

func TestAgentCallWait(t *testing.T) {
	var n int32
	c := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch atomic.AddInt32(&n, 1) {
		case 1:
			w.WriteHeader(503)
		case 2:
			w.WriteHeader(429)
		case 3:
			io.WriteString(w, `{"data":{"status":"in_progress"}}`)
		default:
			io.WriteString(w, `{"data":{"status":"completed"}}`)
		}
	})
	call, err := c.Agents.Calls.WaitForCompletion(context.Background(), "c", &AgentWaitOptions{Interval: 5 * time.Millisecond})
	if err != nil || call["status"] != "completed" || n != 4 {
		t.Fatalf("%v %v %d", call, err, n)
	}
	// non-transient error aborts
	c2 := newTestClient(t, func(w http.ResponseWriter, req *http.Request) { w.WriteHeader(404) })
	_, err = c2.Agents.Calls.WaitForCompletion(context.Background(), "c", &AgentWaitOptions{Interval: time.Millisecond})
	var e *Error
	if !errors.As(err, &e) || e.Status != 404 {
		t.Fatal(err)
	}
	// timeout
	c3 := newTestClient(t, func(w http.ResponseWriter, req *http.Request) { io.WriteString(w, `{"data":{"status":"queued"}}`) })
	_, err = c3.Agents.Calls.WaitForCompletion(context.Background(), "c", &AgentWaitOptions{Interval: 5 * time.Millisecond, Timeout: 20 * time.Millisecond})
	if !errors.As(err, &e) || e.Code != "TIMEOUT" {
		t.Fatal(err)
	}
	// ctx cancel
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = c3.Agents.Calls.WaitForCompletion(ctx, "c", &AgentWaitOptions{Interval: 10 * time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestDialerShapes(t *testing.T) {
	ctx := context.Background()
	c, r := recClient(t, `{"success":true}`)
	cases := []struct {
		call         func()
		method, path string
	}{
		{func() { c.Dialer.Status(ctx) }, "GET", "/dialer/status"},
		{func() { c.Dialer.Numbers.Search(ctx) }, "GET", "/dialer/pool"},
		{func() { c.Dialer.Numbers.Buy(ctx, nil) }, "POST", "/dialer/pool/allocate"},
		{func() { c.Dialer.Numbers.Release(ctx, "+1555") }, "DELETE", "/dialer/numbers/%2B1555"},
		{func() { c.Dialer.Numbers.SetCallerID(ctx, "+1") }, "PUT", "/dialer/caller-id"},
		{func() { c.Dialer.Calls.List(ctx, map[string]any{"limit": 3}) }, "GET", "/dialer/logs?limit=3"},
		{func() { c.Dialer.Calls.LineStatus(ctx) }, "GET", "/dialer/calls/line-status"},
		{func() { c.Dialer.Calls.Reserve(ctx, "+1", "+2", "sip", "") }, "POST", "/dialer/calls"},
		{func() { c.Dialer.Recordings.PlaybackURL(ctx, "r") }, "POST", "/dialer/recordings/r/playback-url"},
		{func() { c.Dialer.Recordings.SetEnabled(ctx, true) }, "PUT", "/dialer/recording"},
		{func() { c.Dialer.Billing.Subscriptions(ctx) }, "GET", "/dialer/subscriptions"},
		{func() { c.Dialer.Billing.Usage(ctx, nil) }, "GET", "/dialer/usage"},
		{func() { c.Dialer.Webhooks.Update(ctx, "w", map[string]any{"status": "paused"}) }, "PATCH", "/dialer/webhooks/w"},
		{func() { c.Dialer.Webhooks.RotateSecret(ctx, "w") }, "POST", "/dialer/webhooks/w/rotate-secret"},
		{func() { c.Dialer.Webhooks.Test(ctx, "w") }, "POST", "/dialer/webhooks/w/test"},
		{func() { c.Dialer.Webhooks.Deliveries(ctx, "w") }, "GET", "/dialer/webhooks/w/deliveries"},
	}
	for _, tc := range cases {
		tc.call()
		if r.method != tc.method || r.uri != tc.path {
			t.Errorf("want %s %s, got %s %s", tc.method, tc.path, r.method, r.uri)
		}
	}
	c.Dialer.Calls.Reserve(ctx, "+1", "+2", "sip", "inbound")
	if !strings.Contains(r.body, `"sip_call_id":"sip"`) || !strings.Contains(r.body, `"direction":"inbound"`) {
		t.Fatal(r.body)
	}
}
