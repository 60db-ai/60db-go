package sixtydb

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// lastRequest is the most recent request the fake API received.
type lastRequest struct {
	method, uri, body string
	header            http.Header
	form              *multipart.Form
}

// fakeAPI answers every request with status, content type ctype and body resp, recording the request.
func fakeAPI(t *testing.T, status int, ctype, resp string) (*Client, *lastRequest) {
	t.Helper()
	last := &lastRequest{}
	c := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		*last = lastRequest{method: req.Method, uri: req.URL.RequestURI(), header: req.Header.Clone()}
		if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/") {
			if err := req.ParseMultipartForm(32 << 20); err != nil {
				t.Error(err)
			}
			last.form = req.MultipartForm
		} else {
			b, _ := io.ReadAll(req.Body)
			last.body = string(b)
		}
		if ctype != "" {
			w.Header().Set("Content-Type", ctype)
		}
		w.WriteHeader(status)
		io.WriteString(w, resp)
	})
	return c, last
}

// sameJSON reports whether the raw body got equals want (nil want = no body).
func sameJSON(t *testing.T, got string, want any) bool {
	t.Helper()
	if want == nil {
		return got == ""
	}
	wb, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var g, w any
	if json.Unmarshal([]byte(got), &g) != nil || json.Unmarshal(wb, &w) != nil {
		return false
	}
	return reflect.DeepEqual(g, w)
}

type routeCase struct {
	name        string
	call        func() (any, error)
	method, uri string
	body        any // nil = no request body
}

type m = map[string]any

func runRoutes(t *testing.T, c *Client, last *lastRequest, cases []routeCase) {
	t.Helper()
	for _, tc := range cases {
		out, err := tc.call()
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if last.method != tc.method || last.uri != tc.uri {
			t.Errorf("%s: want %s %s, got %s %s", tc.name, tc.method, tc.uri, last.method, last.uri)
		}
		if !sameJSON(t, last.body, tc.body) {
			t.Errorf("%s: body %s", tc.name, last.body)
		}
		if d, ok := out.(map[string]any); ok && d["ok"] != true {
			t.Errorf("%s: data not unwrapped: %v", tc.name, d)
		}
	}
}

func TestAgentsNewRoutes(t *testing.T) {
	ctx := context.Background()
	c, last := fakeAPI(t, 200, "application/json", `{"success":true,"data":{"ok":true}}`)
	a := c.Agents
	vars := map[string]string{"name": "Asha"}
	inline := m{"to": "+1555", "phone_line_id": "l1", "agent": m{"script": "s", "voice_id": "v"}, "settings": m{"max_duration_secs": 60}}
	byo := m{"kind": "byo", "name": "t", "domain": "sip.x", "numbers": []any{m{"number": "+1", "agent_id": "a"}},
		"allowed_addresses": []any{"1.2.3.4"}, "allowed_numbers": []any{"+1"}, "normalize_numbers": true,
		"default_country_code": "91", "bridge_transfers": true}
	runRoutes(t, c, last, []routeCase{
		{"WebCall", func() (any, error) { return a.WebCall(ctx, "a1", "v=0\r\n") }, "POST", "/agents/a1/web-calls", m{"offer": "v=0\r\n"}},
		{"BackgroundSounds", func() (any, error) { return a.BackgroundSounds(ctx) }, "GET", "/agents/background-sounds", nil},
		{"Settings.Get", func() (any, error) { return a.Settings.Get(ctx) }, "GET", "/agents/settings", nil},
		{"Settings.Update", func() (any, error) {
			return a.Settings.Update(ctx, m{"event_webhook_url": "https://x.test/e", "tool_webhook_url": nil})
		}, "PUT", "/agents/settings", m{"event_webhook_url": "https://x.test/e", "tool_webhook_url": nil}},
		{"Settings.Rotate", func() (any, error) { return a.Settings.RotateWebhookSecret(ctx) }, "POST", "/agents/settings/rotate-webhook-secret", nil},
		{"Call+variables", func() (any, error) {
			return a.Call(ctx, "a1", PlaceCallParams{To: "+1555", PhoneLineID: "l1", Variables: vars})
		}, "POST", "/agents/a1/calls", m{"to": "+1555", "phone_line_id": "l1", "variables": m{"name": "Asha"}}},
		{"StreamCallWithParams", func() (any, error) {
			return a.StreamCallWithParams(ctx, "a1", StreamCallParams{Variables: vars, Metadata: m{"k": "v"}})
		}, "POST", "/agents/a1/stream-calls", m{"variables": m{"name": "Asha"}, "metadata": m{"k": "v"}}},
		{"StreamCall", func() (any, error) { return a.StreamCall(ctx, "a1", m{"idle_ms": 1}, nil) }, "POST", "/agents/a1/stream-calls", m{"overrides": m{"idle_ms": 1}}},
		{"Calls.List sentiment", func() (any, error) {
			return a.Calls.List(ctx, AgentCallListParams{Sentiment: "positive", Limit: 5})
		}, "GET", "/agents/calls?limit=5&sentiment=positive", nil},
		{"Calls.PlaceInline", func() (any, error) { return a.Calls.PlaceInline(ctx, inline, "") }, "POST", "/agents/calls", inline},
		{"Calls.Usage", func() (any, error) { return a.Calls.Usage(ctx, "c1") }, "GET", "/agents/calls/c1/usage", nil},
		{"Calls.ToolEvents", func() (any, error) { return a.Calls.ToolEvents(ctx, "c1") }, "GET", "/agents/calls/c1/tool-events", nil},
		{"Calls.Result", func() (any, error) { return a.Calls.Result(ctx, "c1") }, "GET", "/agents/calls/c1/result", nil},
		{"Calls.EndStream", func() (any, error) { return a.Calls.EndStream(ctx, "c1") }, "DELETE", "/agents/stream-calls/c1", nil},
		{"Calls.HangUpWeb", func() (any, error) { return a.Calls.HangUpWeb(ctx, "c1") }, "DELETE", "/agents/web-calls/c1", nil},
		{"PhoneLines.Setup", func() (any, error) { return a.PhoneLines.Setup(ctx, "l 1") }, "GET", "/agents/phone-lines/l%201/setup", nil},
		{"PhoneLines.Check", func() (any, error) {
			return a.PhoneLines.Check(ctx, "l1", m{"called_number": "+1555", "source_ip": "1.2.3.4"})
		}, "POST", "/agents/phone-lines/l1/check", m{"called_number": "+1555", "source_ip": "1.2.3.4"}},
		{"PhoneLines.Create byo", func() (any, error) { return a.PhoneLines.Create(ctx, byo) }, "POST", "/agents/phone-lines", byo},
		{"Builder.GetDraft", func() (any, error) { return a.Builder.GetDraft(ctx, "d1") }, "GET", "/agents/builder/drafts/d1", nil},
	})
}

func TestCampaignAndDoNotCallRoutes(t *testing.T) {
	ctx := context.Background()
	c, last := fakeAPI(t, 200, "application/json", `{"success":true,"data":{"ok":true}}`)
	k, r, d := c.Agents.Campaigns, c.Agents.Campaigns.Recipients, c.Agents.DoNotCall
	create := m{"name": "n", "agent_id": "a1", "phone_line_id": "l1", "timezone": "Asia/Kolkata", "csv_text": "phone\n+1\n",
		"calling_windows": []any{m{"days": []any{1, 2}, "start": "09:00", "end": "18:00"}}, "max_consecutive_failures": nil}
	add := m{"recipients": []any{m{"phone": "+1", "variables": m{"n": "A"}}}}
	runRoutes(t, c, last, []routeCase{
		{"List", func() (any, error) { return k.List(ctx, m{"status": "running", "q": "x", "limit": 10, "offset": 20}) },
			"GET", "/agents/campaigns?limit=10&offset=20&q=x&status=running", nil},
		{"Summary", func() (any, error) { return k.Summary(ctx) }, "GET", "/agents/campaigns/summary", nil},
		{"PreviewRecipients", func() (any, error) { return k.PreviewRecipients(ctx, m{"csv_text": "phone\n+1\n", "agent_id": "a1"}) },
			"POST", "/agents/campaigns/recipients/preview", m{"csv_text": "phone\n+1\n", "agent_id": "a1"}},
		{"Create", func() (any, error) { return k.Create(ctx, create, "") }, "POST", "/agents/campaigns", create},
		{"Get", func() (any, error) { return k.Get(ctx, "k1") }, "GET", "/agents/campaigns/k1", nil},
		{"Analytics", func() (any, error) { return k.Analytics(ctx, "k1", m{"from": "2026-01-01", "to": "2026-01-31"}) },
			"GET", "/agents/campaigns/k1/analytics?from=2026-01-01&to=2026-01-31", nil},
		{"Pause", func() (any, error) { return k.Pause(ctx, "k1") }, "POST", "/agents/campaigns/k1/pause", nil},
		{"Resume", func() (any, error) { return k.Resume(ctx, "k1") }, "POST", "/agents/campaigns/k1/resume", nil},
		{"Cancel", func() (any, error) { return k.Cancel(ctx, "k1") }, "POST", "/agents/campaigns/k1/cancel", nil},
		{"Recipients.List", func() (any, error) { return r.List(ctx, "k1", m{"status": "callback", "sentiment": "positive"}) },
			"GET", "/agents/campaigns/k1/recipients?sentiment=positive&status=callback", nil},
		{"Recipients.Add", func() (any, error) { return r.Add(ctx, "k1", add, "") }, "POST", "/agents/campaigns/k1/recipients", add},
		{"Recipients.Preview", func() (any, error) { return r.Preview(ctx, "k1", m{"csv_text": "phone\n+1\n"}) },
			"POST", "/agents/campaigns/k1/recipients/preview", m{"csv_text": "phone\n+1\n"}},
		{"Recipients.Template", func() (any, error) { return r.Template(ctx, "k1") }, "GET", "/agents/campaigns/k1/recipients/template", nil},
		{"Recipients.Export", func() (any, error) { return r.Export(ctx, "k1", m{"status": "completed", "q": "+91"}) },
			"GET", "/agents/campaigns/k1/recipients/export?q=%2B91&status=completed", nil},
		{"Recipients.Calls", func() (any, error) { return r.Calls(ctx, "k1", "r1") }, "GET", "/agents/campaigns/k1/recipients/r1/calls", nil},
		{"Recipients.CancelCallback", func() (any, error) { return r.CancelCallback(ctx, "k1", "r1") },
			"POST", "/agents/campaigns/k1/recipients/r1/callback/cancel", nil},
		{"Recipients.CallbackNow", func() (any, error) { return r.CallbackNow(ctx, "k1", "r1") },
			"POST", "/agents/campaigns/k1/recipients/r1/callback/call-now", nil},
		{"DNC.List", func() (any, error) { return d.List(ctx, m{"q": "+91"}) }, "GET", "/agents/do-not-call?q=%2B91", nil},
		{"DNC.Add", func() (any, error) { return d.Add(ctx, m{"phone": "+15550001", "reason": "asked"}) },
			"POST", "/agents/do-not-call", m{"phone": "+15550001", "reason": "asked"}},
		{"DNC.Remove", func() (any, error) { return d.Remove(ctx, "e1") }, "DELETE", "/agents/do-not-call/e1", nil},
		{"DNC.RemoveByPhone", func() (any, error) { return d.RemoveByPhone(ctx, "+919876543210") },
			"DELETE", "/agents/do-not-call?phone=%2B919876543210", nil},
	})
}
