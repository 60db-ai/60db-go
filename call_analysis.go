package sixtydb

import "context"

// CallAnalysisService is available as Client.CallAnalysis (base path /deep-analytics). It needs a user
// API key with the call_analysis permission (else 403 DA_API_KEY_SCOPE_MISSING). An analyzer is an
// analysis agent; ids are UUIDs. Methods return the API's `data`; failures are *Error.
type CallAnalysisService struct {
	c         *Client
	Analyzers *CallAnalyzers
	Sources   *CallAnalysisSources
	Batches   *CallAnalysisBatches
	AgentFix  *CallAnalysisAgentFix
}

// CallAnalyzers is available as Client.CallAnalysis.Analyzers.
type CallAnalyzers struct{ c *Client }

// CallAnalysisSources is available as Client.CallAnalysis.Sources: the calls an analyzer reads.
type CallAnalysisSources struct{ c *Client }

// CallAnalysisBatches is available as Client.CallAnalysis.Batches: recordings brought in via CSV.
type CallAnalysisBatches struct{ c *Client }

// CallAnalysisAgentFix is available as Client.CallAnalysis.AgentFix: fix a 60db agent from one call's analysis.
type CallAnalysisAgentFix struct{ c *Client }

func newCallAnalysisService(c *Client) *CallAnalysisService {
	return &CallAnalysisService{c: c, Analyzers: &CallAnalyzers{c}, Sources: &CallAnalysisSources{c},
		Batches: &CallAnalysisBatches{c}, AgentFix: &CallAnalysisAgentFix{c}}
}

func analyzerPath(id string) string { return "/deep-analytics/analysis-agents/" + pathEscape(id) }
func importBatchPath(id string) string {
	return "/deep-analytics/imports/batches/" + pathEscape(id)
}
func importItemPath(batchID, itemID string) string {
	return importBatchPath(batchID) + "/items/" + pathEscape(itemID)
}
func daCallPath(id string) string { return "/deep-analytics/calls/" + pathEscape(id) }

// callIDsBody is {"call_ids": ids}, sending [] for nil.
func callIDsBody(ids []string) map[string]any {
	if ids == nil {
		ids = []string{}
	}
	return map[string]any{"call_ids": ids}
}

func (a *CallAnalyzers) List(ctx context.Context) (any, error) {
	return a.c.doData(ctx, "GET", "/deep-analytics/analysis-agents", nil, timeoutDefault)
}

// Create makes an analyzer (params: name, instructions; description, knowledge_collection_id, key_terms, language).
func (a *CallAnalyzers) Create(ctx context.Context, params map[string]any) (any, error) {
	return a.c.doData(ctx, "POST", "/deep-analytics/analysis-agents", params, timeoutDefault)
}

func (a *CallAnalyzers) Get(ctx context.Context, id string) (any, error) {
	return a.c.doData(ctx, "GET", analyzerPath(id), nil, timeoutDefault)
}

// Update changes any Create field (sent as PUT).
func (a *CallAnalyzers) Update(ctx context.Context, id string, patch map[string]any) (any, error) {
	return a.c.doData(ctx, "PUT", analyzerPath(id), patch, timeoutDefault)
}

// Archive archives an analyzer.
func (a *CallAnalyzers) Archive(ctx context.Context, id string) (any, error) {
	return a.c.doData(ctx, "DELETE", analyzerPath(id), nil, timeoutDefault)
}

// DialerNumbers lists the 60db Dialer numbers an analyzer can read calls from.
func (s *CallAnalysisService) DialerNumbers(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/deep-analytics/analysis-agents/dialer-numbers", nil, timeoutDefault)
}

func (s *CallAnalysisSources) Get(ctx context.Context, analyzerID string) (any, error) {
	return s.c.doData(ctx, "GET", analyzerPath(analyzerID)+"/sources", nil, timeoutDefault)
}

// Set chooses the calls an analyzer reads (params: agent_ids; dialer_number_ids, auto_new, min_seconds,
// backfill_days, self_heal_mode).
func (s *CallAnalysisSources) Set(ctx context.Context, analyzerID string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "PUT", analyzerPath(analyzerID)+"/sources", params, timeoutDefault)
}

// AnalyzeAgentCalls queues 60db AI agent calls for analysis.
func (s *CallAnalysisService) AnalyzeAgentCalls(ctx context.Context, analyzerID string, callIDs []string) (any, error) {
	return s.c.doData(ctx, "POST", analyzerPath(analyzerID)+"/agent-calls", callIDsBody(callIDs), timeoutDefault)
}

// AnalyzeDialerCalls queues 60db Dialer calls for analysis.
func (s *CallAnalysisService) AnalyzeDialerCalls(ctx context.Context, analyzerID string, callIDs []string) (any, error) {
	return s.c.doData(ctx, "POST", analyzerPath(analyzerID)+"/dialer-calls", callIDsBody(callIDs), timeoutDefault)
}

// AnalyzeAll analyzes every ready recording of the analyzer; confirmCount must match the count to analyze.
func (s *CallAnalysisService) AnalyzeAll(ctx context.Context, analyzerID string, confirmCount int) (any, error) {
	return s.c.doData(ctx, "POST", analyzerPath(analyzerID)+"/analyze", map[string]any{"confirm_count": confirmCount}, timeoutDefault)
}

// Recordings lists analyzed recordings (params: status, category, severity, q, limit, offset).
func (s *CallAnalysisService) Recordings(ctx context.Context, analyzerID string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "GET", withQuery(analyzerPath(analyzerID)+"/recordings", params), nil, timeoutDefault)
}

// Problems summarizes the problems found (params: from, to).
func (s *CallAnalysisService) Problems(ctx context.Context, analyzerID string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "GET", withQuery(analyzerPath(analyzerID)+"/problems", params), nil, timeoutDefault)
}

// Analytics returns one analyzer's analytics (params: days).
func (s *CallAnalysisService) Analytics(ctx context.Context, analyzerID string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "GET", withQuery(analyzerPath(analyzerID)+"/analytics", params), nil, timeoutDefault)
}

// Overview returns analytics across all analyzers (params: days).
func (s *CallAnalysisService) Overview(ctx context.Context, params map[string]any) (any, error) {
	return s.c.doData(ctx, "GET", withQuery("/deep-analytics/analysis-agents-analytics", params), nil, timeoutDefault)
}

// GetCall returns one call's analysis.
func (s *CallAnalysisService) GetCall(ctx context.Context, callID string) (any, error) {
	return s.c.doData(ctx, "GET", daCallPath(callID)+"/analysis", nil, timeoutDefault)
}

// Create brings recordings in via CSV (params: source_kind direct_upload|csv_urls, csv_text; name,
// mapping, default_agent_id).
func (b *CallAnalysisBatches) Create(ctx context.Context, analyzerID string, params map[string]any) (any, error) {
	return b.c.doData(ctx, "POST", analyzerPath(analyzerID)+"/batches", params, timeoutLong)
}

func (b *CallAnalysisBatches) Get(ctx context.Context, batchID string) (any, error) {
	return b.c.doData(ctx, "GET", importBatchPath(batchID), nil, timeoutDefault)
}

// Items returns a page of a batch's rows (params: state, limit, offset).
func (b *CallAnalysisBatches) Items(ctx context.Context, batchID string, params map[string]any) (*Page, error) {
	return b.c.doPage(ctx, withQuery(importBatchPath(batchID)+"/items", params), timeoutDefault)
}

// CreateUploadSlot returns a presigned form POST for one row's recording (params: size_bytes,
// content_type). The result holds upload_id, expires_at, max_size_bytes, content_type and
// upload{method, url, fields}; upload the file there, then call ConfirmUpload.
func (b *CallAnalysisBatches) CreateUploadSlot(ctx context.Context, batchID, itemID string, params map[string]any) (any, error) {
	return b.c.doData(ctx, "POST", importItemPath(batchID, itemID)+"/upload-slot", params, timeoutDefault)
}

// ConfirmUpload claims an uploaded recording (params: receipt_ref; upload_id).
func (b *CallAnalysisBatches) ConfirmUpload(ctx context.Context, batchID, itemID string, params map[string]any) (any, error) {
	return b.c.doData(ctx, "POST", importItemPath(batchID, itemID)+"/asset-receipt", params, timeoutDefault)
}

// FetchRecording fetches one row's recording URL.
func (b *CallAnalysisBatches) FetchRecording(ctx context.Context, batchID, itemID string) (any, error) {
	return b.c.doData(ctx, "POST", importItemPath(batchID, itemID)+"/fetch-recording", nil, timeoutLong)
}

// FetchRecordings fetches the recording URLs of many rows (params: limit; nil sends {}).
func (b *CallAnalysisBatches) FetchRecordings(ctx context.Context, batchID string, params map[string]any) (any, error) {
	if params == nil {
		params = map[string]any{}
	}
	return b.c.doData(ctx, "POST", importBatchPath(batchID)+"/fetch-recordings", params, timeoutLong)
}

// Analyze analyzes a batch's ready recordings; confirmCount must match the count to analyze.
func (b *CallAnalysisBatches) Analyze(ctx context.Context, analyzerID, batchID string, confirmCount int) (any, error) {
	return b.c.doData(ctx, "POST", analyzerPath(analyzerID)+"/batches/"+pathEscape(batchID)+"/analyze",
		map[string]any{"confirm_count": confirmCount}, timeoutDefault)
}

// Suggest proposes a script fix for the 60db agent from this call's analysis.
func (f *CallAnalysisAgentFix) Suggest(ctx context.Context, callID string) (any, error) {
	return f.c.doData(ctx, "POST", daCallPath(callID)+"/agent-fix/suggest", nil, timeoutLong)
}

// Apply saves a suggested script (params: base_hash, script).
func (f *CallAnalysisAgentFix) Apply(ctx context.Context, callID string, params map[string]any) (any, error) {
	return f.c.doData(ctx, "POST", daCallPath(callID)+"/agent-fix/apply", params, timeoutDefault)
}

// Undo reverts an applied fix.
func (f *CallAnalysisAgentFix) Undo(ctx context.Context, callID, fixID string) (any, error) {
	return f.c.doData(ctx, "POST", daCallPath(callID)+"/agent-fix/"+pathEscape(fixID)+"/undo", nil, timeoutDefault)
}
