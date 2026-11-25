package sixtydb

import (
	"context"
	"time"
)

// AgentSimulations is available as Client.Agents.Simulations: simulation runs, evals, metrics and
// golden runs. Starting or re-running a simulation, golden runs and fix suggestions spend credits.
type AgentSimulations struct{ c *Client }

func simulationPath(runID string) string { return "/agents/simulations/" + pathEscape(runID) }

// List returns a page of an agent's runs, newest first (params: limit, offset).
func (s *AgentSimulations) List(ctx context.Context, agentID string, params map[string]any) (*Page, error) {
	return s.c.doPage(ctx, withQuery(agentPath(agentID)+"/simulations", params), timeoutDefault)
}

// Start starts a run (spends credits). body: mode "generate" (count, sample_rate) or "custom"
// (scenario); repeats, priority, max_concurrency, max_calls, max_minutes, max_credits,
// estimated_credits_per_call, baseline_run_id.
func (s *AgentSimulations) Start(ctx context.Context, agentID string, body map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", agentPath(agentID)+"/simulations", body, timeoutDefault)
}

// Estimate is a free dry run of Start (body: scenarios, repeats, priority, max_calls, max_credits,
// estimated_credits_per_call).
func (s *AgentSimulations) Estimate(ctx context.Context, agentID string, body map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", agentPath(agentID)+"/simulations/estimate", body, timeoutDefault)
}

// Get returns a run's progress and report. state: running, paused, queued, cancelled, finished, stopped_budget.
func (s *AgentSimulations) Get(ctx context.Context, runID string) (any, error) {
	return s.c.doData(ctx, "GET", simulationPath(runID), nil, timeoutDefault)
}

// WaitForRun polls Get until state is finished, cancelled or stopped_budget and returns the run.
// 429, 5xx and network errors are retried; timeout returns an *Error with Code "TIMEOUT".
func (s *AgentSimulations) WaitForRun(ctx context.Context, runID string, opts *AgentWaitOptions) (map[string]any, error) {
	return waitFor(ctx, opts, 4*time.Second, 30*time.Minute, "Timed out waiting for the simulation run",
		func() (any, error) { return s.Get(ctx, runID) },
		func(run map[string]any) bool {
			st := asString(run["state"])
			return st == "finished" || st == "cancelled" || st == "stopped_budget"
		})
}

func (s *AgentSimulations) Pause(ctx context.Context, runID string) (any, error) {
	return s.c.doData(ctx, "POST", simulationPath(runID)+"/pause", nil, timeoutDefault)
}

// Resume resumes a paused run (needs a Pro or Business plan; spends credits).
func (s *AgentSimulations) Resume(ctx context.Context, runID string) (any, error) {
	return s.c.doData(ctx, "POST", simulationPath(runID)+"/resume", nil, timeoutDefault)
}

func (s *AgentSimulations) Cancel(ctx context.Context, runID string) (any, error) {
	return s.c.doData(ctx, "POST", simulationPath(runID)+"/cancel", nil, timeoutDefault)
}

// CreateDraft creates an unpublished draft of the agent from a run.
func (s *AgentSimulations) CreateDraft(ctx context.Context, runID string) (any, error) {
	return s.c.doData(ctx, "POST", simulationPath(runID)+"/draft", nil, timeoutDefault)
}

// UpdateDraft replaces the script of an unpublished draft (at most 32,000 characters).
func (s *AgentSimulations) UpdateDraft(ctx context.Context, draftID, script string) (any, error) {
	return s.c.doData(ctx, "PUT", "/agents/simulation-drafts/"+pathEscape(draftID), map[string]any{"script": script}, timeoutDefault)
}

// Rerun runs the same stories again (spends credits). body: draft_id or agent "current"; repeats,
// priority, max_concurrency, max_calls, max_minutes, max_credits, estimated_credits_per_call, baseline_run_id.
func (s *AgentSimulations) Rerun(ctx context.Context, runID string, body map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", simulationPath(runID)+"/rerun", body, timeoutDefault)
}

// RerunCall re-runs one call against the run's draft (body: draft_id plus the Rerun controls).
func (s *AgentSimulations) RerunCall(ctx context.Context, runID, callID string, body map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", simulationPath(runID)+"/calls/"+pathEscape(callID)+"/rerun", body, timeoutDefault)
}

// SuggestFix writes improved instructions from what failed (one paid model request; saves nothing).
// The result holds base_hash, before, after, changes, problems, settings_problems.
func (s *AgentSimulations) SuggestFix(ctx context.Context, runID string) (any, error) {
	return s.c.doData(ctx, "POST", simulationPath(runID)+"/fix", nil, timeoutLong)
}

// ApplyFix saves previewed instructions to the agent (params: base_hash, script); 409 when the agent changed.
func (s *AgentSimulations) ApplyFix(ctx context.Context, runID string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", simulationPath(runID)+"/fix/apply", params, timeoutDefault)
}

// UndoFix puts the previous instructions back (params: base_hash of the applied text, script = previous text).
func (s *AgentSimulations) UndoFix(ctx context.Context, runID string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", simulationPath(runID)+"/fix/undo", params, timeoutDefault)
}

// Evals lists the agent's built-in and custom evals with golden-set status.
func (s *AgentSimulations) Evals(ctx context.Context, agentID string) (any, error) {
	return s.c.doData(ctx, "GET", agentPath(agentID)+"/evals", nil, timeoutDefault)
}

// Metrics returns pooled latency and accuracy (params: source production|simulation|test and
// agent_version, both required; from, to as unix seconds).
func (s *AgentSimulations) Metrics(ctx context.Context, agentID string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "GET", withQuery(agentPath(agentID)+"/metrics", params), nil, timeoutDefault)
}

// CompareMetrics compares two agent versions (params: source, version_a, version_b, from, to).
func (s *AgentSimulations) CompareMetrics(ctx context.Context, agentID string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "GET", withQuery(agentPath(agentID)+"/metrics/compare", params), nil, timeoutDefault)
}

// RunGolden measures an eval against its golden set (one paid request per fixture).
func (s *AgentSimulations) RunGolden(ctx context.Context, agentID, evalName string) (any, error) {
	return s.c.doData(ctx, "POST", agentPath(agentID)+"/evals/"+pathEscape(evalName)+"/golden", nil, timeoutLong)
}

// GetGoldenRun reads one stored golden run.
func (s *AgentSimulations) GetGoldenRun(ctx context.Context, evalName, runID string) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/evals/"+pathEscape(evalName)+"/golden/runs/"+pathEscape(runID), nil, timeoutDefault)
}
