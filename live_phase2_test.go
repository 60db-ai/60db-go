package sixtydb

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func asMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T (%v)", v, v)
	}
	return m
}

// idOf finds an id in a result object or its nested `data`.
func idOf(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := asString(m[k]); s != "" {
			return s
		}
	}
	if d, ok := m["data"].(map[string]any); ok {
		return idOf(d, keys...)
	}
	return ""
}

// firstList finds the first array of objects in a response body.
func firstList(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case map[string]any:
		for _, k := range []string{"data", "songs", "items", "rows"} {
			if l := firstList(x[k]); l != nil {
				return l
			}
		}
	}
	return nil
}

func TestLivePhase2(t *testing.T) {
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	choice := map[string]any{"type": "choice", "criteria": map[string]any{
		"positive": "The text is positive", "negative": "The text is negative", "unknown": "Cannot tell"}}

	t.Run("Judge", func(t *testing.T) {
		for name, f := range map[string]func(context.Context) (any, error){
			"models": c.Judge.Models, "health": c.Judge.Health,
			"rubrics": c.Judge.Rubrics.List,
		} {
			if _, err := f(ctx); err != nil {
				t.Errorf("%s: %v", name, err)
			}
		}
		if _, err := c.Judge.Usage(ctx, ""); err != nil {
			t.Errorf("usage: %v", err)
		}
		ev, err := c.Judge.Evaluate(ctx, map[string]any{"state": "I love this product!", "save": false,
			"questions": map[string]any{"sentiment": choice}})
		if err != nil || ev == nil {
			t.Fatalf("evaluate: %v", err)
		}
		t.Logf("evaluate: %v", ev)
		ex, err := c.Judge.Extract(ctx, map[string]any{"text": "Please cancel my order", "save": false,
			"schema": map[string]any{
				"intents":    map[string]any{"cancel_order": "Customer wants to cancel"},
				"operations": map[string]any{"cancel": "Cancel something"}}})
		if err != nil || ex == nil {
			t.Fatalf("extract: %v", err)
		}

		r, err := c.Judge.Rubrics.Create(ctx, map[string]any{"name": fmt.Sprintf("sdk-go-%d", time.Now().Unix()),
			"questions": map[string]any{"sentiment": choice}})
		if err != nil {
			t.Fatalf("rubric create: %v", err)
		}
		rid := idOf(asMap(t, r), "id", "rubric_id")
		if rid == "" {
			t.Fatalf("no rubric id: %v", r)
		}
		defer func() {
			if err := c.Judge.Rubrics.Delete(ctx, rid); err != nil {
				t.Errorf("rubric delete: %v", err)
			}
		}()
		if _, err := c.Judge.Rubrics.Get(ctx, rid); err != nil {
			t.Errorf("rubric get: %v", err)
		}
		if _, err := c.Judge.Rubrics.Update(ctx, rid, map[string]any{"description": "updated by go sdk"}); err != nil {
			t.Errorf("rubric update: %v", err)
		}
		run, err := c.Judge.Evaluate(ctx, map[string]any{"state": "Terrible service.", "rubric_id": rid, "label": "sdk-go"})
		if err != nil {
			t.Fatalf("evaluate by id: %v", err)
		}
		runID := idOf(asMap(t, run), "run_id", "id", "evaluation_id")
		list, err := c.Judge.Runs.List(ctx, map[string]any{"rubric_id": rid, "limit": 10})
		if err != nil || list["pagination"] == nil {
			t.Fatalf("runs list: %v %v", list, err)
		}
		if runID == "" { // fall back to the run listed for our fresh rubric
			if l := firstList(list["runs"]); len(l) > 0 {
				runID = idOf(asMap(t, l[0]), "id", "run_id")
			}
		}
		if runID == "" {
			t.Fatalf("no run id: %v", run)
		}
		if _, err := c.Judge.Runs.Get(ctx, runID); err != nil {
			t.Errorf("run get: %v", err)
		}
		if err := c.Judge.Runs.Delete(ctx, runID); err != nil {
			t.Errorf("run delete: %v", err)
		}
	})

	t.Run("MusicReadOnly", func(t *testing.T) {
		if _, err := c.Music.ListVoices(ctx); err != nil {
			t.Errorf("voices: %v", err)
		}
		if _, err := c.Music.Health(ctx); err != nil {
			t.Errorf("health: %v", err)
		}
		l, err := c.Music.List(ctx, map[string]any{"limit": 20})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		for _, s := range firstList(l) {
			song, _ := s.(map[string]any)
			if asString(song["status"]) != "succeeded" {
				continue
			}
			id := asString(song["id"])
			if _, err := c.Music.Get(ctx, id); err != nil {
				t.Fatalf("get: %v", err)
			}
			b, err := c.Music.Download(ctx, id)
			if err != nil || len(b) < 100 {
				t.Fatalf("download: %v len=%d", err, len(b))
			}
			t.Logf("downloaded %d bytes", len(b))
			return
		}
		t.Log("no succeeded song to download")
	})

	t.Run("DialerReadOnly", func(t *testing.T) {
		for name, f := range map[string]func(context.Context) (map[string]any, error){
			"status": c.Dialer.Status, "numbers": c.Dialer.Numbers.List, "live": c.Dialer.Calls.Live,
			"line-status": c.Dialer.Calls.LineStatus, "subscriptions": c.Dialer.Billing.Subscriptions,
			"webhooks": c.Dialer.Webhooks.List,
		} {
			if _, err := f(ctx); err != nil {
				t.Errorf("%s: %v", name, err)
			}
		}
		if _, err := c.Dialer.Calls.List(ctx, map[string]any{"limit": 5}); err != nil {
			t.Errorf("logs: %v", err)
		}
		if _, err := c.Dialer.Recordings.List(ctx, map[string]any{"limit": 5}); err != nil {
			t.Errorf("recordings: %v", err)
		}
		if _, err := c.Dialer.Billing.Usage(ctx, map[string]any{"limit": 5}); err != nil {
			t.Errorf("usage: %v", err)
		}
	})

	t.Run("DialerWebhook", func(t *testing.T) {
		w := c.Dialer.Webhooks
		cr, err := w.Create(ctx, map[string]any{"name": fmt.Sprintf("sdk-go-%d", time.Now().UnixNano()),
			"url": "https://example.com/60db-go-sdk-test", "event_types": []string{"webhook.test"}})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		id := idOf(cr, "id", "webhook_id")
		if id == "" {
			t.Fatalf("no webhook id: %v", cr)
		}
		defer func() {
			if _, err := w.Delete(ctx, id); err != nil {
				t.Errorf("delete: %v", err)
			}
		}()
		if _, err := w.Update(ctx, id, map[string]any{"description": "go sdk"}); err != nil {
			t.Errorf("update: %v", err)
		}
		if _, err := w.RotateSecret(ctx, id); err != nil {
			t.Errorf("rotate: %v", err)
		}
		if _, err := w.Test(ctx, id); err != nil {
			t.Errorf("test: %v", err)
		}
		if _, err := w.Deliveries(ctx, id); err != nil {
			t.Errorf("deliveries: %v", err)
		}
	})

	t.Run("AgentsPolicyDeny", func(t *testing.T) {
		_, err := c.Agents.List(ctx, nil)
		if e := wantStatus(t, err, 403); e.Code != "POLICY_DENY" {
			t.Errorf("list code %q", e.Code)
		}
		_, err = c.Agents.Get(ctx, "does-not-exist")
		if e := wantStatus(t, err, 403); e.Code != "POLICY_DENY" {
			t.Errorf("get code %q", e.Code)
		}
	})
}
