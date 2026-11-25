package sixtydb

import "context"

// VoiceFile is one audio sample for CreateVoice.
type VoiceFile = UploadFile

// GetVoices lists available voices.
func (c *Client) GetVoices(ctx context.Context) (map[string]any, error) {
	return c.do(ctx, "GET", "/voices", nil, timeoutDefault)
}

// GetVoice fetches one voice.
func (c *Client) GetVoice(ctx context.Context, id string) (map[string]any, error) {
	return c.do(ctx, "GET", "/voices/"+pathEscape(id), nil, timeoutDefault)
}

// GetMyVoices lists the caller's own voices.
func (c *Client) GetMyVoices(ctx context.Context) (map[string]any, error) {
	return c.do(ctx, "GET", "/myvoices", nil, timeoutDefault)
}

// CreateVoice clones a voice from audio samples (multipart: name + files parts).
func (c *Client) CreateVoice(ctx context.Context, name string, files []VoiceFile) (map[string]any, error) {
	parts := make([]filePart, 0, len(files))
	for _, f := range files {
		parts = append(parts, filePart{Field: "files", Filename: f.Filename, MIME: sniffAudioMIME(f.Data, f.Filename), Data: f.Data})
	}
	buf, ct, err := buildMultipart(map[string]string{"name": name}, parts)
	if err != nil {
		return nil, err
	}
	return c.doBody(ctx, "POST", "/voices", buf, ct, timeoutLong)
}

// UpdateVoice updates a voice.
func (c *Client) UpdateVoice(ctx context.Context, id string, params map[string]any) (map[string]any, error) {
	return c.do(ctx, "PUT", "/voices/"+pathEscape(id), params, timeoutDefault)
}

// DeleteVoice deletes a voice.
func (c *Client) DeleteVoice(ctx context.Context, id string) (map[string]any, error) {
	return c.do(ctx, "DELETE", "/voices/"+pathEscape(id), nil, timeoutDefault)
}
