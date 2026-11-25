package sixtydb

import (
	"context"
)

// AgentsService is available as Client.Agents. Methods return the API's `data`
// (unwrapped); failures are *Error with Status, Code (e.g. POLICY_DENY) and Field.
type AgentsService struct {
	c *Client
	// Calls and PhoneLines are sub-resources.
	Calls      *AgentCalls
	PhoneLines *AgentPhoneLines
	// Settings holds the workspace webhooks and limits; Builder drafts agents with AI.
	Settings *AgentSettings
	Builder  *AgentBuilder
	// Campaigns dial recipient lists; DoNotCall is the numbers campaigns skip.
	Campaigns *AgentCampaigns
	DoNotCall *AgentDoNotCall
	// Integrations are third-party connections for agent tools.
	Integrations *AgentIntegrations
	// Simulations covers simulation runs, evals, metrics and golden runs.
	Simulations *AgentSimulations
}

func newAgentsService(c *Client) *AgentsService {
	return &AgentsService{c: c, Calls: &AgentCalls{c}, PhoneLines: &AgentPhoneLines{c},
		Settings: &AgentSettings{c}, Builder: &AgentBuilder{c},
		Campaigns: &AgentCampaigns{c: c, Recipients: &CampaignRecipients{c}}, DoNotCall: &AgentDoNotCall{c},
		Integrations: &AgentIntegrations{c}, Simulations: &AgentSimulations{c}}
}

func agentPath(id string) string { return "/agents/" + pathEscape(id) }

// List lists agents (params: limit, offset).
func (s *AgentsService) List(ctx context.Context, params map[string]any) (any, error) {
	return s.c.doData(ctx, "GET", withQuery("/agents", params), nil, timeoutDefault)
}

func (s *AgentsService) Get(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", agentPath(id), nil, timeoutDefault)
}

// Create makes an agent (required: name, script, voice_id). The result carries webhook_secret once.
func (s *AgentsService) Create(ctx context.Context, agent map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", "/agents", agent, timeoutDefault)
}

func (s *AgentsService) Update(ctx context.Context, id string, patch map[string]any) (any, error) {
	return s.c.doData(ctx, "PATCH", agentPath(id), patch, timeoutDefault)
}

func (s *AgentsService) Delete(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "DELETE", agentPath(id), nil, timeoutDefault)
}

func (s *AgentsService) RotateWebhookSecret(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "POST", agentPath(id)+"/rotate-webhook-secret", nil, timeoutDefault)
}

func (s *AgentsService) ToolEvents(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", agentPath(id)+"/tool-events", nil, timeoutDefault)
}

// PlaceCallParams are the inputs of Agents.Call. Variables fill {{key}} in the agent's script.
type PlaceCallParams struct {
	To          string
	PhoneLineID string
	Overrides   map[string]any
	Metadata    map[string]any
	Variables   map[string]string
}

// Call places a call as the agent (rings a real phone).
func (s *AgentsService) Call(ctx context.Context, agentID string, p PlaceCallParams) (any, error) {
	body := map[string]any{"to": p.To, "phone_line_id": p.PhoneLineID}
	if p.Overrides != nil {
		body["overrides"] = p.Overrides
	}
	if p.Metadata != nil {
		body["metadata"] = p.Metadata
	}
	if p.Variables != nil {
		body["variables"] = p.Variables
	}
	return s.c.doData(ctx, "POST", agentPath(agentID)+"/calls", body, timeoutDefault)
}

// StreamCall starts a WebSocket stream call; the result holds call, url and expires_in.
// overrides and metadata may be nil. Use StreamCallWithParams to send variables.
func (s *AgentsService) StreamCall(ctx context.Context, agentID string, overrides, metadata map[string]any) (any, error) {
	return s.StreamCallWithParams(ctx, agentID, StreamCallParams{Overrides: overrides, Metadata: metadata})
}

// StreamCallParams are the inputs of Agents.StreamCallWithParams; every field is optional.
type StreamCallParams struct {
	Overrides map[string]any
	Metadata  map[string]any
	Variables map[string]string // fill {{key}} in the agent's script
}

// StreamCallWithParams starts a WebSocket stream call; the result holds call, url and expires_in.
func (s *AgentsService) StreamCallWithParams(ctx context.Context, agentID string, p StreamCallParams) (any, error) {
	body := map[string]any{}
	if p.Overrides != nil {
		body["overrides"] = p.Overrides
	}
	if p.Metadata != nil {
		body["metadata"] = p.Metadata
	}
	if p.Variables != nil {
		body["variables"] = p.Variables
	}
	return s.c.doData(ctx, "POST", agentPath(agentID)+"/stream-calls", body, timeoutDefault)
}

// WebCall talks to the agent from a browser. offer is the finished WebRTC SDP (starts "v=0");
// the result holds call and answer (the SDP answer).
func (s *AgentsService) WebCall(ctx context.Context, agentID, offer string) (any, error) {
	return s.c.doData(ctx, "POST", agentPath(agentID)+"/web-calls", map[string]any{"offer": offer}, timeoutDefault)
}

// BackgroundSounds lists the names usable as call_defaults.background_sound.
func (s *AgentsService) BackgroundSounds(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/background-sounds", nil, timeoutDefault)
}

// AgentSettings is available as Client.Agents.Settings: the workspace webhooks used by calls
// without a stored agent, and the limits in force.
type AgentSettings struct{ c *Client }

// Get returns event_webhook_url, tool_webhook_url and limits.
func (s *AgentSettings) Get(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/settings", nil, timeoutDefault)
}

// Update sets event_webhook_url and/or tool_webhook_url (an https URL, or nil to clear).
// The result carries webhook_secret.
func (s *AgentSettings) Update(ctx context.Context, params map[string]any) (any, error) {
	return s.c.doData(ctx, "PUT", "/agents/settings", params, timeoutDefault)
}

// RotateWebhookSecret issues a new workspace webhook signing secret (returned once).
func (s *AgentSettings) RotateWebhookSecret(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "POST", "/agents/settings/rotate-webhook-secret", nil, timeoutDefault)
}
