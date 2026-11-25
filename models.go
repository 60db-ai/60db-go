package sixtydb

import "context"

// GetLanguages returns the STT languages (data.languages[]).
func (c *Client) GetLanguages(ctx context.Context) ([]any, error) {
	m, err := c.do(ctx, "GET", "/stt/languages", nil, timeoutDefault)
	if err != nil {
		return nil, err
	}
	if d, ok := m["data"].(map[string]any); ok {
		l, _ := d["languages"].([]any)
		return l, nil
	}
	return nil, nil
}

// GetTTSLanguages returns the TTS languages (data[]).
func (c *Client) GetTTSLanguages(ctx context.Context) ([]any, error) {
	m, err := c.do(ctx, "GET", "/tts/languages", nil, timeoutDefault)
	if err != nil {
		return nil, err
	}
	l, _ := m["data"].([]any)
	return l, nil
}

// GetTTSModels returns the full TTS models response.
func (c *Client) GetTTSModels(ctx context.Context) (map[string]any, error) {
	return c.do(ctx, "GET", "/tts/models", nil, timeoutDefault)
}

// GetSTTModels returns the full STT models response.
func (c *Client) GetSTTModels(ctx context.Context) (map[string]any, error) {
	return c.do(ctx, "GET", "/stt/models", nil, timeoutDefault)
}
