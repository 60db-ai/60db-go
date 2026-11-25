package sixtydb

import "context"

func (c *Client) get(ctx context.Context, path string) (map[string]any, error) {
	return c.do(ctx, "GET", path, nil, timeoutDefault)
}

func (c *Client) post(ctx context.Context, path string, body any) (map[string]any, error) {
	return c.do(ctx, "POST", path, body, timeoutDefault)
}

// GetProfile returns the current user.
func (c *Client) GetProfile(ctx context.Context) (map[string]any, error) {
	return c.get(ctx, "/auth/me")
}

// UpdateProfile updates the current user.
func (c *Client) UpdateProfile(ctx context.Context, params map[string]any) (map[string]any, error) {
	return c.do(ctx, "PUT", "/auth/profile", params, timeoutDefault)
}

// GetWorkspaces lists workspaces.
func (c *Client) GetWorkspaces(ctx context.Context) (map[string]any, error) {
	return c.get(ctx, "/workspaces")
}

// CreateWorkspace creates a workspace.
func (c *Client) CreateWorkspace(ctx context.Context, name string) (map[string]any, error) {
	return c.post(ctx, "/workspaces", map[string]any{"name": name})
}

// GetPlans lists plans.
func (c *Client) GetPlans(ctx context.Context) (map[string]any, error) { return c.get(ctx, "/plans") }

// GetCurrentPlan returns the current billing plan.
func (c *Client) GetCurrentPlan(ctx context.Context) (map[string]any, error) {
	return c.get(ctx, "/billing/current-plan")
}

// Subscribe subscribes to a plan.
func (c *Client) Subscribe(ctx context.Context, planID string) (map[string]any, error) {
	return c.post(ctx, "/billing/subscribe", map[string]any{"plan_id": planID})
}

// GetUsage returns usage analytics.
func (c *Client) GetUsage(ctx context.Context) (map[string]any, error) {
	return c.get(ctx, "/analytics/usage")
}

// GetAPIKeys lists API keys.
func (c *Client) GetAPIKeys(ctx context.Context) (map[string]any, error) {
	return c.get(ctx, "/developer/api")
}

// CreateAPIKey creates an API key.
func (c *Client) CreateAPIKey(ctx context.Context, name string) (map[string]any, error) {
	return c.post(ctx, "/developer/api", map[string]any{"name": name})
}

// DeleteAPIKey deletes an API key.
func (c *Client) DeleteAPIKey(ctx context.Context, id string) (map[string]any, error) {
	return c.do(ctx, "DELETE", "/developer/api/"+pathEscape(id), nil, timeoutDefault)
}

// GetWebhooks lists webhooks.
func (c *Client) GetWebhooks(ctx context.Context) (map[string]any, error) {
	return c.get(ctx, "/webhooks")
}

// CreateWebhook creates a webhook for the given events.
func (c *Client) CreateWebhook(ctx context.Context, url string, events []string) (map[string]any, error) {
	return c.post(ctx, "/webhooks", map[string]any{"url": url, "events": events})
}

// DeleteWebhook deletes a webhook.
func (c *Client) DeleteWebhook(ctx context.Context, id string) (map[string]any, error) {
	return c.do(ctx, "DELETE", "/webhooks/"+pathEscape(id), nil, timeoutDefault)
}

// SignUp registers a user (params: email, password, name, ...).
func (c *Client) SignUp(ctx context.Context, params map[string]any) (map[string]any, error) {
	return c.post(ctx, "/auth/signup", params)
}

// SignIn logs in a user (params: email, password).
func (c *Client) SignIn(ctx context.Context, params map[string]any) (map[string]any, error) {
	return c.post(ctx, "/auth/login", params)
}
