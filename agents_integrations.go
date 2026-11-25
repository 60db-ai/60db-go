package sixtydb

import "context"

// AgentIntegrations is available as Client.Agents.Integrations: third-party connections whose
// actions agents can use as tools. Secrets are never returned. OAuth providers are connected from
// the 60db dashboard (the OAuth callback is bound to the browser that started it), not from the SDK.
type AgentIntegrations struct{ c *Client }

func integrationPath(id string) string { return "/agents/integrations/" + pathEscape(id) }

// Catalog lists the providers: connect method, credential fields and curated actions.
func (s *AgentIntegrations) Catalog(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/integrations/catalog", nil, timeoutDefault)
}

// List lists the workspace's connections.
func (s *AgentIntegrations) List(ctx context.Context) (any, error) {
	return s.c.doData(ctx, "GET", "/agents/integrations", nil, timeoutDefault)
}

// Create connects a key, webhook or MCP provider (params: provider, name, fields). Credentials are tested live.
func (s *AgentIntegrations) Create(ctx context.Context, params map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", "/agents/integrations", params, timeoutDefault)
}

// Update changes a connection (params: name, fields).
func (s *AgentIntegrations) Update(ctx context.Context, id string, params map[string]any) (any, error) {
	return s.c.doData(ctx, "PATCH", integrationPath(id), params, timeoutDefault)
}

func (s *AgentIntegrations) Delete(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "DELETE", integrationPath(id), nil, timeoutDefault)
}

// Test checks a connection's credentials.
func (s *AgentIntegrations) Test(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "POST", integrationPath(id)+"/test", nil, timeoutDefault)
}

// RotateSecret issues a new signing secret for a custom webhook connection (returned once).
func (s *AgentIntegrations) RotateSecret(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "POST", integrationPath(id)+"/rotate-secret", nil, timeoutDefault)
}

// Tools lists the tool keys this connection offers.
func (s *AgentIntegrations) Tools(ctx context.Context, id string) (any, error) {
	return s.c.doData(ctx, "GET", integrationPath(id)+"/tools", nil, timeoutDefault)
}

// AssignAgents sets exactly which tools each agent gets from this connection; agents is
// [{agent_id, tools}] and an agent left out loses this connection's tools. The result holds used_by.
func (s *AgentIntegrations) AssignAgents(ctx context.Context, id string, agents []map[string]any) (any, error) {
	if agents == nil {
		agents = []map[string]any{}
	}
	return s.c.doData(ctx, "PUT", integrationPath(id)+"/agents", map[string]any{"agents": agents}, timeoutDefault)
}

// RunAction runs one action of the connection with arguments (nil sends {}); the result holds ok and result.
func (s *AgentIntegrations) RunAction(ctx context.Context, id, action string, arguments map[string]any) (any, error) {
	if arguments == nil {
		arguments = map[string]any{}
	}
	return s.c.doData(ctx, "POST", integrationPath(id)+"/actions/"+pathEscape(action),
		map[string]any{"arguments": arguments}, timeoutDefault)
}

// Request asks 60db for a new integration (params: name, details).
func (s *AgentIntegrations) Request(ctx context.Context, params map[string]any) (any, error) {
	return s.c.doData(ctx, "POST", "/agents/integrations/requests", params, timeoutDefault)
}

// GoogleSheetTabs lists the tabs of a spreadsheet (spreadsheetID may be the sheet URL).
func (s *AgentIntegrations) GoogleSheetTabs(ctx context.Context, id, spreadsheetID string) (any, error) {
	return s.c.doData(ctx, "GET", withQuery(integrationPath(id)+"/google-sheets/tabs",
		map[string]any{"spreadsheet_id": spreadsheetID}), nil, timeoutDefault)
}

// GoogleSheetColumns lists the header columns of one tab.
func (s *AgentIntegrations) GoogleSheetColumns(ctx context.Context, id, spreadsheetID, sheet string) (any, error) {
	return s.c.doData(ctx, "GET", withQuery(integrationPath(id)+"/google-sheets/columns",
		map[string]any{"spreadsheet_id": spreadsheetID, "sheet": sheet}), nil, timeoutDefault)
}
