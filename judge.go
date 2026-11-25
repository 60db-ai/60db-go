package sixtydb

import (
	"context"
)

// JudgeService is available as Client.Judge. Methods return the API's `data`
// (a map[string]any or []any) unwrapped.
type JudgeService struct {
	c       *Client
	Rubrics *JudgeRubrics
	Runs    *JudgeRuns
}

func newJudgeService(c *Client) *JudgeService {
	return &JudgeService{c: c, Rubrics: &JudgeRubrics{c}, Runs: &JudgeRuns{c}}
}

// JudgeRubrics is available as Client.Judge.Rubrics.
type JudgeRubrics struct{ c *Client }

// JudgeRuns is available as Client.Judge.Runs.
type JudgeRuns struct{ c *Client }

// Evaluate asks rubric questions about `state` (params: state, questions|rubric_id, model, save, label).
func (s *JudgeService) Evaluate(ctx context.Context, params map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", "/judge/evaluate", params, timeoutMemory)
}

// Extract extracts intents/operations/entities (params: text, schema, profile, budget_ms, save, label).
func (s *JudgeService) Extract(ctx context.Context, params map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", "/judge/extract", params, timeoutMemory)
}

// Models lists judge models.
func (s *JudgeService) Models(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/judge/models", nil, timeoutDefault)
}

// Usage returns usage for period: current_month (default), last_30_days or all_time.
func (s *JudgeService) Usage(ctx context.Context, period string) (any, error) {
	if period == "" {
		period = "current_month"
	}
	return s.c.doData(ctx, "GET", withQuery("/judge/usage", map[string]any{"period": period}), nil, timeoutDefault)
}

// Health returns judge service health.
func (s *JudgeService) Health(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/judge/health", nil, timeoutDefault)
}

func (r *JudgeRubrics) List(ctx context.Context) (any, error) {
	return r.c.doData(ctx, "GET", "/judge/rubrics", nil, timeoutDefault)
}

// Create makes a rubric (params: name, description, model, questions, shared).
func (r *JudgeRubrics) Create(ctx context.Context, params map[string]any) (any, error) {
	return r.c.doData(ctx, "POST", "/judge/rubrics", params, timeoutDefault)
}

func (r *JudgeRubrics) Get(ctx context.Context, id string) (any, error) {
	return r.c.doData(ctx, "GET", "/judge/rubrics/"+pathEscape(id), nil, timeoutDefault)
}

func (r *JudgeRubrics) Update(ctx context.Context, id string, params map[string]any) (any, error) {
	return r.c.doData(ctx, "PATCH", "/judge/rubrics/"+pathEscape(id), params, timeoutDefault)
}

func (r *JudgeRubrics) Delete(ctx context.Context, id string) error {
	_, err := r.c.do(ctx, "DELETE", "/judge/rubrics/"+pathEscape(id), nil, timeoutDefault)
	return err
}

// List returns {"runs": data, "pagination": pagination}. params: limit, offset, kind,
// rubric_id, needs_review (bool; sent as "true" only when true).
func (r *JudgeRuns) List(ctx context.Context, params map[string]any) (map[string]any, error) {
	q := map[string]any{}
	for k, v := range params {
		q[k] = v
	}
	if nr, ok := q["needs_review"].(bool); ok {
		if nr {
			q["needs_review"] = "true"
		} else {
			delete(q, "needs_review")
		}
	}
	m, err := r.c.do(ctx, "GET", withQuery("/judge/evaluations", q), nil, timeoutDefault)
	if err != nil {
		return nil, err
	}
	return map[string]any{"runs": m["data"], "pagination": m["pagination"]}, nil
}

func (r *JudgeRuns) Get(ctx context.Context, id string) (any, error) {
	return r.c.doData(ctx, "GET", "/judge/evaluations/"+pathEscape(id), nil, timeoutDefault)
}

func (r *JudgeRuns) Delete(ctx context.Context, id string) error {
	_, err := r.c.do(ctx, "DELETE", "/judge/evaluations/"+pathEscape(id), nil, timeoutDefault)
	return err
}
