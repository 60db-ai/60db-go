package sixtydb

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestPagedEnvelopeKeepsPagination(t *testing.T) {
	ctx := context.Background()
	c, _ := fakeAPI(t, 200, "application/json",
		`{"success":true,"data":[{"id":"x"}],"pagination":{"total":7,"limit":1,"offset":2}}`)
	lists := map[string]func() (*Page, error){
		"campaigns":  func() (*Page, error) { return c.Agents.Campaigns.List(ctx, nil) },
		"recipients": func() (*Page, error) { return c.Agents.Campaigns.Recipients.List(ctx, "k1", nil) },
		"dnc":        func() (*Page, error) { return c.Agents.DoNotCall.List(ctx, nil) },
		"sims":       func() (*Page, error) { return c.Agents.Simulations.List(ctx, "a1", nil) },
		"items":      func() (*Page, error) { return c.CallAnalysis.Batches.Items(ctx, "b1", nil) },
	}
	for name, list := range lists {
		p, err := list()
		if err != nil || len(p.Data) != 1 || p.Data[0].(map[string]any)["id"] != "x" {
			t.Errorf("%s: %+v %v", name, p, err)
			continue
		}
		if p.Pagination["total"] != 7.0 || p.Pagination["limit"] != 1.0 || p.Pagination["offset"] != 2.0 {
			t.Errorf("%s: pagination %v", name, p.Pagination)
		}
	}
	// an empty list is [] (not nil)
	c2, _ := fakeAPI(t, 200, "application/json", `{"success":true,"data":[],"pagination":{"total":0}}`)
	if p, err := c2.Agents.DoNotCall.List(ctx, nil); err != nil || p.Data == nil || len(p.Data) != 0 || p.Pagination["total"] != 0.0 {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestCampaignCSVReturnsString(t *testing.T) {
	ctx := context.Background()
	csv := "phone_number,name\n+15550001,Asha\n"
	c, last := fakeAPI(t, 200, "text/csv; charset=utf-8", csv)
	got, err := c.Agents.Campaigns.Recipients.Template(ctx, "k1")
	if err != nil || got != csv || last.uri != "/agents/campaigns/k1/recipients/template" {
		t.Fatalf("%q %v %s", got, err, last.uri)
	}
	got, err = c.Agents.Campaigns.Recipients.Export(ctx, "k1", map[string]any{"sentiment": "negative"})
	if err != nil || got != csv || last.uri != "/agents/campaigns/k1/recipients/export?sentiment=negative" {
		t.Fatalf("%q %v %s", got, err, last.uri)
	}
	// an error before the body is the JSON envelope
	c2, _ := fakeAPI(t, 404, "application/json", `{"success":false,"message":"Campaign not found","code":"CAMPAIGN_NOT_FOUND"}`)
	_, err = c2.Agents.Campaigns.Recipients.Export(ctx, "k1", nil)
	var e *Error
	if !errors.As(err, &e) || e.Status != 404 || e.Code != "CAMPAIGN_NOT_FOUND" {
		t.Fatal(err)
	}
}

func TestIdempotencyKeyHeader(t *testing.T) {
	ctx := context.Background()
	c, last := fakeAPI(t, 201, "application/json", `{"success":true,"data":{"id":"x"}}`)
	sends := map[string]func(key string) (any, error){
		"campaigns.create": func(k string) (any, error) { return c.Agents.Campaigns.Create(ctx, m{"name": "n"}, k) },
		"recipients.add": func(k string) (any, error) {
			return c.Agents.Campaigns.Recipients.Add(ctx, "k1", m{"csv_text": "x"}, k)
		},
		"calls.placeInline": func(k string) (any, error) { return c.Agents.Calls.PlaceInline(ctx, m{"to": "+1"}, k) },
	}
	for name, send := range sends {
		d, err := send("retry-abc_123")
		if err != nil || d.(map[string]any)["id"] != "x" || last.header.Get("Idempotency-Key") != "retry-abc_123" {
			t.Errorf("%s: %v %v header=%q", name, d, err, last.header.Get("Idempotency-Key"))
		}
		if last.header.Get("Content-Type") != "application/json" || last.header.Get("Authorization") != "Bearer k" {
			t.Errorf("%s: headers %v", name, last.header)
		}
		if _, err := send(""); err != nil || len(last.header.Values("Idempotency-Key")) != 0 {
			t.Errorf("%s: empty key must send no header (%v)", name, last.header)
		}
	}
	// other writes never send it
	c.Agents.Campaigns.Pause(ctx, "k1")
	if len(last.header.Values("Idempotency-Key")) != 0 {
		t.Fatal("pause sent Idempotency-Key")
	}
}

func TestBuilderCreateDraftMultipart(t *testing.T) {
	ctx := context.Background()
	c, last := fakeAPI(t, 202, "application/json", `{"success":true,"data":{"id":"d1","status":"queued"}}`)
	path := filepath.Join(t.TempDir(), "faq.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4 test"), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := c.Agents.Builder.CreateDraft(ctx, AgentDraftParams{
		Description: "Books dental appointments", Direction: "inbound", Language: "en",
		WebsiteURL: "https://clinic.test", Files: []UploadFile{{Filename: "notes.md", Data: []byte("# hours")}},
		FilePaths: []string{path},
	})
	if err != nil || d.(map[string]any)["id"] != "d1" || d.(map[string]any)["status"] != "queued" {
		t.Fatalf("%v %v", d, err)
	}
	if last.method != "POST" || last.uri != "/agents/builder/drafts" || last.form == nil {
		t.Fatalf("%s %s %v", last.method, last.uri, last.form)
	}
	v := last.form.Value
	if v["description"][0] != "Books dental appointments" || v["direction"][0] != "inbound" ||
		v["language"][0] != "en" || v["website_url"][0] != "https://clinic.test" {
		t.Errorf("fields %v", v)
	}
	if _, sent := v["text"]; sent {
		t.Error("unset text must not be sent")
	}
	files := last.form.File["files"]
	if len(files) != 2 || len(last.form.File) != 1 {
		t.Fatalf("files %v", last.form.File)
	}
	if files[0].Filename != "notes.md" || files[0].Header.Get("Content-Type") != "text/markdown" ||
		files[1].Filename != "faq.pdf" || files[1].Header.Get("Content-Type") != "application/pdf" {
		t.Errorf("parts %+v %+v", files[0], files[1])
	}
	f, _ := files[1].Open()
	b, _ := io.ReadAll(f)
	f.Close()
	if string(b) != "%PDF-1.4 test" {
		t.Errorf("file bytes %q", b)
	}
	if documentMIMEOf("deck.PPTX") != "application/vnd.openxmlformats-officedocument.presentationml.presentation" ||
		documentMIMEOf("a.csv") != "text/csv" {
		t.Error("document MIME")
	}
	if _, err := c.Agents.Builder.CreateDraft(ctx, AgentDraftParams{FilePaths: []string{path + ".missing"}}); err == nil {
		t.Fatal("missing file must error")
	}
}

func TestNewResourcesErrorEnvelope(t *testing.T) {
	ctx := context.Background()
	c, _ := fakeAPI(t, 400, "application/json", `{"success":false,"message":"csv_text is required","code":"VALIDATION_ERROR","field":"csv_text"}`)
	calls := map[string]func() error{
		"campaigns.create": func() error { _, err := c.Agents.Campaigns.Create(ctx, m{}, "k"); return err },
		"campaigns.list":   func() error { _, err := c.Agents.Campaigns.List(ctx, nil); return err },
		"template":         func() error { _, err := c.Agents.Campaigns.Recipients.Template(ctx, "k1"); return err },
		"dnc.add":          func() error { _, err := c.Agents.DoNotCall.Add(ctx, m{}); return err },
		"integrations":     func() error { _, err := c.Agents.Integrations.Create(ctx, m{}); return err },
		"simulations":      func() error { _, err := c.Agents.Simulations.Start(ctx, "a1", m{}); return err },
		"builder":          func() error { _, err := c.Agents.Builder.CreateDraft(ctx, AgentDraftParams{}); return err },
		"settings":         func() error { _, err := c.Agents.Settings.Update(ctx, m{}); return err },
		"call analysis":    func() error { _, err := c.CallAnalysis.Batches.Create(ctx, "z1", m{}); return err },
	}
	for name, call := range calls {
		var e *Error
		if err := call(); !errors.As(err, &e) || e.Status != 400 || e.Code != "VALIDATION_ERROR" ||
			e.Field != "csv_text" || e.Message != "csv_text is required" {
			t.Errorf("%s: %v", name, err)
		}
	}
	c2, _ := fakeAPI(t, 403, "application/json", `{"success":false,"message":"no","code":"DA_API_KEY_SCOPE_MISSING"}`)
	var e *Error
	if _, err := c2.CallAnalysis.Analyzers.List(ctx); !errors.As(err, &e) || e.Status != 403 || e.Code != "DA_API_KEY_SCOPE_MISSING" {
		t.Fatal(err)
	}
	c3, _ := fakeAPI(t, 502, "text/html", "<html>bad gateway</html>")
	if _, err := c3.Agents.Simulations.List(ctx, "a1", nil); !errors.As(err, &e) || e.Message != "HTTP 502" || e.Status != 502 {
		t.Fatal(err)
	}
}

// sequence serves bodies[i] for the i-th request (the last one repeats); "503"/"429" answer that status.
func sequence(t *testing.T, bodies ...string) (*Client, *int32) {
	var n int32
	c := newTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		i := int(atomic.AddInt32(&n, 1)) - 1
		if i >= len(bodies) {
			i = len(bodies) - 1
		}
		switch bodies[i] {
		case "503":
			w.WriteHeader(503)
		case "429":
			w.WriteHeader(429)
		case "404":
			w.WriteHeader(404)
		default:
			io.WriteString(w, bodies[i])
		}
	})
	return c, &n
}

func TestWaitForDraft(t *testing.T) {
	ctx := context.Background()
	fast := &AgentWaitOptions{Interval: 5 * time.Millisecond}
	c, n := sequence(t, "503", "429", `{"data":{"id":"d1","status":"reading"}}`,
		`{"data":{"id":"d1","status":"ready","draft":{"agent":{"name":"A"}}}}`)
	d, err := c.Agents.Builder.WaitForDraft(ctx, "d1", fast)
	if err != nil || d["status"] != "ready" || d["draft"] == nil || *n != 4 {
		t.Fatalf("%v %v %d", d, err, *n)
	}
	// failed is terminal and returned, not thrown
	c, n = sequence(t, `{"data":{"status":"queued"}}`, `{"data":{"status":"failed","error":{"code":"EXTRACT_FAILED"}}}`)
	d, err = c.Agents.Builder.WaitForDraft(ctx, "d1", fast)
	if err != nil || d["status"] != "failed" || *n != 2 {
		t.Fatalf("%v %v %d", d, err, *n)
	}
	var e *Error
	c, _ = sequence(t, `{"data":{"status":"writing"}}`)
	if _, err = c.Agents.Builder.WaitForDraft(ctx, "d1", &AgentWaitOptions{Interval: 5 * time.Millisecond, Timeout: 20 * time.Millisecond}); !errors.As(err, &e) || e.Code != "TIMEOUT" {
		t.Fatal(err)
	}
	c, _ = sequence(t, "404")
	if _, err = c.Agents.Builder.WaitForDraft(ctx, "d1", fast); !errors.As(err, &e) || e.Status != 404 {
		t.Fatal(err)
	}
}

func TestWaitForRun(t *testing.T) {
	ctx := context.Background()
	fast := &AgentWaitOptions{Interval: 5 * time.Millisecond}
	for _, final := range []string{"finished", "cancelled", "stopped_budget"} {
		c, n := sequence(t, `{"data":{"state":"queued"}}`, "503", `{"data":{"state":"running"}}`, `{"data":{"state":"paused"}}`,
			`{"data":{"id":"run1","state":"`+final+`"}}`)
		run, err := c.Agents.Simulations.WaitForRun(ctx, "run1", fast)
		if err != nil || run["state"] != final || *n != 5 {
			t.Fatalf("%s: %v %v %d", final, run, err, *n)
		}
	}
	var e *Error
	c, _ := sequence(t, `{"data":{"state":"running"}}`)
	if _, err := c.Agents.Simulations.WaitForRun(ctx, "run1", &AgentWaitOptions{Interval: 5 * time.Millisecond, Timeout: 20 * time.Millisecond}); !errors.As(err, &e) || e.Code != "TIMEOUT" {
		t.Fatal(err)
	}
	c, _ = sequence(t, "404")
	if _, err := c.Agents.Simulations.WaitForRun(ctx, "run1", fast); !errors.As(err, &e) || e.Status != 404 {
		t.Fatal(err)
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	c, _ = sequence(t, `{"data":{"state":"running"}}`)
	if _, err := c.Agents.Simulations.WaitForRun(cctx, "run1", &AgentWaitOptions{Interval: 10 * time.Millisecond}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}
