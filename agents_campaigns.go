package sixtydb

import "context"

// AgentCampaigns is available as Client.Agents.Campaigns. Creating a campaign (or adding recipients
// to one) dials real phones and spends credits.
type AgentCampaigns struct {
	c          *Client
	Recipients *CampaignRecipients
}

// CampaignRecipients is available as Client.Agents.Campaigns.Recipients.
type CampaignRecipients struct{ c *Client }

// AgentDoNotCall is available as Client.Agents.DoNotCall: numbers campaigns skip at upload and dial time.
type AgentDoNotCall struct{ c *Client }

func campaignPath(id string) string { return "/agents/campaigns/" + pathEscape(id) }

func recipientPath(campaignID, recipientID string) string {
	return campaignPath(campaignID) + "/recipients/" + pathEscape(recipientID)
}

// List returns a page of campaigns (params: status, q, limit, offset). status: draft, scheduled,
// running, paused, completed, cancelled.
func (s *AgentCampaigns) List(ctx context.Context, params map[string]any) (*Page, error) {
	return s.c.doPage(ctx, withQuery("/agents/campaigns", params), timeoutDefault)
}

// Summary returns campaigns running, recipients waiting and recipients on a call.
func (s *AgentCampaigns) Summary(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/campaigns/summary", nil, timeoutDefault)
}

// PreviewRecipients checks a recipient CSV before Create (params: csv_text, phone_line_id, agent_id):
// valid, invalid, duplicates, do-not-call.
func (s *AgentCampaigns) PreviewRecipients(ctx context.Context, params map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", "/agents/campaigns/recipients/preview", params, timeoutLong)
}

// Create makes a campaign and starts dialing per its schedule. body: name, agent_id, phone_line_id,
// timezone, csv_text (required); start, calling_windows, concurrency_limit, max_attempts,
// retry_delay_secs, max_consecutive_failures, variable_mapping, variable_defaults.
// idempotencyKey ("" for none) makes a retry safe.
func (s *AgentCampaigns) Create(ctx context.Context, body map[string]any, idempotencyKey string) (any, error) {
	return s.c.doDataWith(ctx, "POST", "/agents/campaigns", body, idempotencyHeader(idempotencyKey), timeoutLong)
}

// Get returns one campaign with counts, effective concurrency and the workspace limit.
func (s *AgentCampaigns) Get(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", campaignPath(id), nil, timeoutDefault)
}

// Analytics returns the campaign's calls by outcome, talk time, sentiments and a timeline
// (params: from, to as YYYY-MM-DD days in the campaign timezone).
func (s *AgentCampaigns) Analytics(ctx context.Context, id string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "GET", withQuery(campaignPath(id)+"/analytics", params), nil, timeoutDefault)
}

func (s *AgentCampaigns) Pause(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "POST", campaignPath(id)+"/pause", nil, timeoutDefault)
}

func (s *AgentCampaigns) Resume(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "POST", campaignPath(id)+"/resume", nil, timeoutDefault)
}

// Cancel stops new dials; calls already live finish.
func (s *AgentCampaigns) Cancel(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "POST", campaignPath(id)+"/cancel", nil, timeoutDefault)
}

// List returns a page of recipients (params: status, q, sentiment, limit, offset). status: pending,
// calling, completed, no_answer, failed, skipped, cancelled, or callback (waiting for a callback).
func (r *CampaignRecipients) List(ctx context.Context, campaignID string, params map[string]any) (*Page, error) {
	return r.c.doPage(ctx, withQuery(campaignPath(campaignID)+"/recipients", params), timeoutDefault)
}

// Add adds recipients (body: csv_text, or recipients [{phone, variables}]); they are dialed.
// idempotencyKey ("" for none) makes a retry safe. The result includes the campaign.
func (r *CampaignRecipients) Add(ctx context.Context, campaignID string, body map[string]any, idempotencyKey string) (any, error) {
	return r.c.doDataWith(ctx, "POST", campaignPath(campaignID)+"/recipients", body, idempotencyHeader(idempotencyKey), timeoutLong)
}

// Preview reports what Add would do with the same body, storing nothing.
func (r *CampaignRecipients) Preview(ctx context.Context, campaignID string, body map[string]any) (any, error) {
	return r.c.doData(ctx, "POST", campaignPath(campaignID)+"/recipients/preview", body, timeoutLong)
}

// Template returns a CSV to fill in for Add.
func (r *CampaignRecipients) Template(ctx context.Context, campaignID string) (string, error) {
	return r.c.doText(ctx, campaignPath(campaignID)+"/recipients/template", timeoutDefault)
}

// Export returns every recipient matching the filters as CSV (params: status, q, sentiment).
func (r *CampaignRecipients) Export(ctx context.Context, campaignID string, params map[string]any) (string, error) {
	return r.c.doText(ctx, withQuery(campaignPath(campaignID)+"/recipients/export", params), timeoutLong)
}

// Calls lists every call placed to one recipient, newest first.
func (r *CampaignRecipients) Calls(ctx context.Context, campaignID, recipientID string) (any, error) {
	return r.c.doData(ctx, "GET", recipientPath(campaignID, recipientID)+"/calls", nil, timeoutDefault)
}

// CancelCallback drops a recipient's scheduled callback (409 CALLBACK_NOT_SCHEDULED when none).
func (r *CampaignRecipients) CancelCallback(ctx context.Context, campaignID, recipientID string) (any, error) {
	return r.c.doData(ctx, "POST", recipientPath(campaignID, recipientID)+"/callback/cancel", nil, timeoutDefault)
}

// CallbackNow makes a recipient's scheduled callback due now.
func (r *CampaignRecipients) CallbackNow(ctx context.Context, campaignID, recipientID string) (any, error) {
	return r.c.doData(ctx, "POST", recipientPath(campaignID, recipientID)+"/callback/call-now", nil, timeoutDefault)
}

// List returns a page of do-not-call entries (params: q, limit, offset).
func (d *AgentDoNotCall) List(ctx context.Context, params map[string]any) (*Page, error) {
	return d.c.doPage(ctx, withQuery("/agents/do-not-call", params), timeoutDefault)
}

// Add adds a number (params: phone, reason).
func (d *AgentDoNotCall) Add(ctx context.Context, params map[string]any) (any, error) {
	return d.c.doData(ctx, "POST", "/agents/do-not-call", params, timeoutDefault)
}

// Remove deletes an entry by its id.
func (d *AgentDoNotCall) Remove(ctx context.Context, entryID string) (any, error) {
	return d.c.doData(ctx, "DELETE", "/agents/do-not-call/"+pathEscape(entryID), nil, timeoutDefault)
}

// RemoveByPhone deletes the entry for an E.164 number.
func (d *AgentDoNotCall) RemoveByPhone(ctx context.Context, phone string) (any, error) {
	return d.c.doData(ctx, "DELETE", withQuery("/agents/do-not-call", map[string]any{"phone": phone}), nil, timeoutDefault)
}
