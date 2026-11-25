package sixtydb

import (
	"context"
	"testing"
)

func TestIntegrationRoutes(t *testing.T) {
	ctx := context.Background()
	c, last := fakeAPI(t, 200, "application/json", `{"success":true,"data":{"ok":true}}`)
	s := c.Agents.Integrations
	assign := []map[string]any{{"agent_id": "a1", "tools": []any{"search"}}}
	runRoutes(t, c, last, []routeCase{
		{"Catalog", func() (any, error) { return s.Catalog(ctx) }, "GET", "/agents/integrations/catalog", nil},
		{"List", func() (any, error) { return s.List(ctx) }, "GET", "/agents/integrations", nil},
		{"Create", func() (any, error) {
			return s.Create(ctx, m{"provider": "hubspot", "name": "CRM", "fields": m{"api_key": "k"}})
		},
			"POST", "/agents/integrations", m{"provider": "hubspot", "name": "CRM", "fields": m{"api_key": "k"}}},
		{"Update", func() (any, error) { return s.Update(ctx, "i1", m{"name": "CRM 2"}) }, "PATCH", "/agents/integrations/i1", m{"name": "CRM 2"}},
		{"Delete", func() (any, error) { return s.Delete(ctx, "i1") }, "DELETE", "/agents/integrations/i1", nil},
		{"Test", func() (any, error) { return s.Test(ctx, "i1") }, "POST", "/agents/integrations/i1/test", nil},
		{"RotateSecret", func() (any, error) { return s.RotateSecret(ctx, "i1") }, "POST", "/agents/integrations/i1/rotate-secret", nil},
		{"Tools", func() (any, error) { return s.Tools(ctx, "i1") }, "GET", "/agents/integrations/i1/tools", nil},
		{"AssignAgents", func() (any, error) { return s.AssignAgents(ctx, "i1", assign) }, "PUT", "/agents/integrations/i1/agents", m{"agents": assign}},
		{"AssignAgents nil", func() (any, error) { return s.AssignAgents(ctx, "i1", nil) }, "PUT", "/agents/integrations/i1/agents", m{"agents": []any{}}},
		{"RunAction none", func() (any, error) { return s.RunAction(ctx, "i1", "search_contact", nil) },
			"POST", "/agents/integrations/i1/actions/search_contact", m{"arguments": m{}}},
		{"RunAction args", func() (any, error) { return s.RunAction(ctx, "i1", "search_contact", m{"email": "a@b.c"}) },
			"POST", "/agents/integrations/i1/actions/search_contact", m{"arguments": m{"email": "a@b.c"}}},
		{"Request", func() (any, error) { return s.Request(ctx, m{"name": "Zoho", "details": "d"}) },
			"POST", "/agents/integrations/requests", m{"name": "Zoho", "details": "d"}},
		{"GoogleSheetTabs", func() (any, error) {
			return s.GoogleSheetTabs(ctx, "i1", "https://docs.google.com/spreadsheets/d/abc/edit")
		},
			"GET", "/agents/integrations/i1/google-sheets/tabs?spreadsheet_id=https%3A%2F%2Fdocs.google.com%2Fspreadsheets%2Fd%2Fabc%2Fedit", nil},
		{"GoogleSheetColumns", func() (any, error) { return s.GoogleSheetColumns(ctx, "i1", "abc", "Sheet 1") },
			"GET", "/agents/integrations/i1/google-sheets/columns?sheet=Sheet+1&spreadsheet_id=abc", nil},
	})
}

func TestSimulationRoutes(t *testing.T) {
	ctx := context.Background()
	c, last := fakeAPI(t, 200, "application/json", `{"success":true,"data":{"ok":true}}`)
	s := c.Agents.Simulations
	start := m{"mode": "generate", "count": 9, "max_credits": 100, "estimated_credits_per_call": 2}
	fix := m{"base_hash": "h", "script": "s"}
	runRoutes(t, c, last, []routeCase{
		{"List", func() (any, error) { return s.List(ctx, "a1", m{"limit": 5}) }, "GET", "/agents/a1/simulations?limit=5", nil},
		{"Start", func() (any, error) { return s.Start(ctx, "a1", start) }, "POST", "/agents/a1/simulations", start},
		{"Estimate", func() (any, error) { return s.Estimate(ctx, "a1", m{"repeats": 2}) }, "POST", "/agents/a1/simulations/estimate", m{"repeats": 2}},
		{"Get", func() (any, error) { return s.Get(ctx, "run1") }, "GET", "/agents/simulations/run1", nil},
		{"Pause", func() (any, error) { return s.Pause(ctx, "run1") }, "POST", "/agents/simulations/run1/pause", nil},
		{"Resume", func() (any, error) { return s.Resume(ctx, "run1") }, "POST", "/agents/simulations/run1/resume", nil},
		{"Cancel", func() (any, error) { return s.Cancel(ctx, "run1") }, "POST", "/agents/simulations/run1/cancel", nil},
		{"CreateDraft", func() (any, error) { return s.CreateDraft(ctx, "run1") }, "POST", "/agents/simulations/run1/draft", nil},
		{"UpdateDraft", func() (any, error) { return s.UpdateDraft(ctx, "dr1", "new script") },
			"PUT", "/agents/simulation-drafts/dr1", m{"script": "new script"}},
		{"Rerun", func() (any, error) { return s.Rerun(ctx, "run1", m{"agent": "current", "repeats": 3}) },
			"POST", "/agents/simulations/run1/rerun", m{"agent": "current", "repeats": 3}},
		{"RerunCall", func() (any, error) { return s.RerunCall(ctx, "run1", "call1", m{"draft_id": "dr1"}) },
			"POST", "/agents/simulations/run1/calls/call1/rerun", m{"draft_id": "dr1"}},
		{"SuggestFix", func() (any, error) { return s.SuggestFix(ctx, "run1") }, "POST", "/agents/simulations/run1/fix", nil},
		{"ApplyFix", func() (any, error) { return s.ApplyFix(ctx, "run1", fix) }, "POST", "/agents/simulations/run1/fix/apply", fix},
		{"UndoFix", func() (any, error) { return s.UndoFix(ctx, "run1", fix) }, "POST", "/agents/simulations/run1/fix/undo", fix},
		{"Evals", func() (any, error) { return s.Evals(ctx, "a1") }, "GET", "/agents/a1/evals", nil},
		{"Metrics", func() (any, error) {
			return s.Metrics(ctx, "a1", m{"source": "production", "agent_version": 3, "from": 1700000000})
		}, "GET", "/agents/a1/metrics?agent_version=3&from=1700000000&source=production", nil},
		{"CompareMetrics", func() (any, error) {
			return s.CompareMetrics(ctx, "a1", m{"source": "simulation", "version_a": 1, "version_b": 2})
		}, "GET", "/agents/a1/metrics/compare?source=simulation&version_a=1&version_b=2", nil},
		{"RunGolden", func() (any, error) { return s.RunGolden(ctx, "a1", "tone check") }, "POST", "/agents/a1/evals/tone%20check/golden", nil},
		{"GetGoldenRun", func() (any, error) { return s.GetGoldenRun(ctx, "tone", "g1") }, "GET", "/agents/evals/tone/golden/runs/g1", nil},
	})
}

func TestCallAnalysisRoutes(t *testing.T) {
	ctx := context.Background()
	c, last := fakeAPI(t, 200, "application/json", `{"success":true,"data":{"ok":true}}`)
	ca := c.CallAnalysis
	const az = "/deep-analytics/analysis-agents/z1"
	const bt = "/deep-analytics/imports/batches/b1"
	create := m{"name": "QA", "instructions": "i", "key_terms": []any{"refund"}}
	sources := m{"agent_ids": []any{"a1"}, "auto_new": false, "min_seconds": 30}
	batch := m{"source_kind": "csv_urls", "csv_text": "url\nhttps://x\n"}
	runRoutes(t, c, last, []routeCase{
		{"Analyzers.List", func() (any, error) { return ca.Analyzers.List(ctx) }, "GET", "/deep-analytics/analysis-agents", nil},
		{"Analyzers.Create", func() (any, error) { return ca.Analyzers.Create(ctx, create) }, "POST", "/deep-analytics/analysis-agents", create},
		{"Analyzers.Get", func() (any, error) { return ca.Analyzers.Get(ctx, "z1") }, "GET", az, nil},
		{"Analyzers.Update", func() (any, error) { return ca.Analyzers.Update(ctx, "z1", m{"name": "QA2"}) }, "PUT", az, m{"name": "QA2"}},
		{"Analyzers.Archive", func() (any, error) { return ca.Analyzers.Archive(ctx, "z1") }, "DELETE", az, nil},
		{"DialerNumbers", func() (any, error) { return ca.DialerNumbers(ctx) }, "GET", "/deep-analytics/analysis-agents/dialer-numbers", nil},
		{"Sources.Get", func() (any, error) { return ca.Sources.Get(ctx, "z1") }, "GET", az + "/sources", nil},
		{"Sources.Set", func() (any, error) { return ca.Sources.Set(ctx, "z1", sources) }, "PUT", az + "/sources", sources},
		{"AnalyzeAgentCalls", func() (any, error) { return ca.AnalyzeAgentCalls(ctx, "z1", []string{"c1", "c2"}) },
			"POST", az + "/agent-calls", m{"call_ids": []any{"c1", "c2"}}},
		{"AnalyzeDialerCalls nil", func() (any, error) { return ca.AnalyzeDialerCalls(ctx, "z1", nil) },
			"POST", az + "/dialer-calls", m{"call_ids": []any{}}},
		{"Batches.Create", func() (any, error) { return ca.Batches.Create(ctx, "z1", batch) }, "POST", az + "/batches", batch},
		{"Batches.Get", func() (any, error) { return ca.Batches.Get(ctx, "b1") }, "GET", bt, nil},
		{"Batches.Items", func() (any, error) { return ca.Batches.Items(ctx, "b1", m{"state": "ready", "limit": 10}) },
			"GET", bt + "/items?limit=10&state=ready", nil},
		{"Batches.CreateUploadSlot", func() (any, error) {
			return ca.Batches.CreateUploadSlot(ctx, "b1", "it1", m{"size_bytes": 1024, "content_type": "audio/wav"})
		}, "POST", bt + "/items/it1/upload-slot", m{"size_bytes": 1024, "content_type": "audio/wav"}},
		{"Batches.ConfirmUpload", func() (any, error) { return ca.Batches.ConfirmUpload(ctx, "b1", "it1", m{"receipt_ref": "r"}) },
			"POST", bt + "/items/it1/asset-receipt", m{"receipt_ref": "r"}},
		{"Batches.FetchRecording", func() (any, error) { return ca.Batches.FetchRecording(ctx, "b1", "it1") },
			"POST", bt + "/items/it1/fetch-recording", nil},
		{"Batches.FetchRecordings nil", func() (any, error) { return ca.Batches.FetchRecordings(ctx, "b1", nil) },
			"POST", bt + "/fetch-recordings", m{}},
		{"Batches.FetchRecordings", func() (any, error) { return ca.Batches.FetchRecordings(ctx, "b1", m{"limit": 25}) },
			"POST", bt + "/fetch-recordings", m{"limit": 25}},
		{"Batches.Analyze", func() (any, error) { return ca.Batches.Analyze(ctx, "z1", "b1", 12) },
			"POST", az + "/batches/b1/analyze", m{"confirm_count": 12}},
		{"AnalyzeAll", func() (any, error) { return ca.AnalyzeAll(ctx, "z1", 40) }, "POST", az + "/analyze", m{"confirm_count": 40}},
		{"Recordings", func() (any, error) { return ca.Recordings(ctx, "z1", m{"severity": "high", "q": "refund", "limit": 5}) },
			"GET", az + "/recordings?limit=5&q=refund&severity=high", nil},
		{"Problems", func() (any, error) { return ca.Problems(ctx, "z1", m{"from": "2026-01-01", "to": "2026-02-01"}) },
			"GET", az + "/problems?from=2026-01-01&to=2026-02-01", nil},
		{"Analytics", func() (any, error) { return ca.Analytics(ctx, "z1", m{"days": 7}) }, "GET", az + "/analytics?days=7", nil},
		{"Overview", func() (any, error) { return ca.Overview(ctx, nil) }, "GET", "/deep-analytics/analysis-agents-analytics", nil},
		{"Overview days", func() (any, error) { return ca.Overview(ctx, m{"days": 90}) }, "GET", "/deep-analytics/analysis-agents-analytics?days=90", nil},
		{"GetCall", func() (any, error) { return ca.GetCall(ctx, "c1") }, "GET", "/deep-analytics/calls/c1/analysis", nil},
		{"AgentFix.Suggest", func() (any, error) { return ca.AgentFix.Suggest(ctx, "c1") }, "POST", "/deep-analytics/calls/c1/agent-fix/suggest", nil},
		{"AgentFix.Apply", func() (any, error) { return ca.AgentFix.Apply(ctx, "c1", m{"base_hash": "h", "script": "s"}) },
			"POST", "/deep-analytics/calls/c1/agent-fix/apply", m{"base_hash": "h", "script": "s"}},
		{"AgentFix.Undo", func() (any, error) { return ca.AgentFix.Undo(ctx, "c1", "f1") }, "POST", "/deep-analytics/calls/c1/agent-fix/f1/undo", nil},
	})
}
