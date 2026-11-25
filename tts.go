package sixtydb

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// DefaultStreamSampleRate is the PCM rate of streamed TTS when none is requested.
const DefaultStreamSampleRate = 48000

func ttsBody(text, voiceID string, params map[string]any, stream bool) (map[string]any, error) {
	if strings.TrimSpace(text) == "" {
		return nil, &Error{Status: 400, Message: "text is required"}
	}
	base := map[string]any{"text": text}
	if voiceID != "" {
		base["voice_id"] = voiceID
	}
	if stream {
		base["stream"] = true
	}
	return mergeParams(base, params), nil
}

func asString(v any) string { s, _ := v.(string); return s }

// TextToSpeech synthesizes text and returns a complete WAV file (headerless PCM if the
// server only sends the audioContent fallback; wrap it with PCMToWAV).
// params may carry speed, sample_rate, audio_enhance and any other API field.
func (c *Client) TextToSpeech(ctx context.Context, text, voiceID string, params map[string]any) ([]byte, error) {
	body, err := ttsBody(text, voiceID, params, false)
	if err != nil {
		return nil, err
	}
	b, ct, err := c.raw(ctx, "POST", "/tts-synthesize", body, timeoutDefault)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(ct, "audio/") || strings.Contains(ct, "octet-stream") {
		return b, nil
	}
	m, err := parseJSON(b)
	if err != nil {
		return nil, err
	}
	if ok, has := m["success"].(bool); has && !ok {
		msg := asString(m["message"])
		if msg == "" {
			msg = "TTS failed"
		}
		return nil, &Error{Status: 200, Message: msg, Body: b}
	}
	sources := []map[string]any{m}
	if be, ok := m["backendResponse"].(map[string]any); ok {
		sources = append(sources, be)
	}
	for _, s := range sources {
		if enc := asString(s["audio_base64"]); enc != "" {
			return base64.StdEncoding.DecodeString(enc)
		}
	}
	for _, s := range sources {
		if enc := asString(s["audioContent"]); enc != "" { // headerless PCM
			return base64.StdEncoding.DecodeString(enc)
		}
	}
	return nil, &Error{Status: 200, Message: "TTS response contained no audio", Body: b}
}

// TextToSpeechStream streams 16-bit LE mono PCM chunks to onChunk. The PCM rate is
// params["sample_rate"] or DefaultStreamSampleRate. Return an error from onChunk to stop.
func (c *Client) TextToSpeechStream(ctx context.Context, text, voiceID string, params map[string]any, onChunk func(pcm []byte) error) error {
	body, err := ttsBody(text, voiceID, params, true)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(body)
	resp, cancel, err := c.send(ctx, "POST", "/tts-synthesize", bytesReader(payload), "application/json", timeoutLong)
	if err != nil {
		return err
	}
	defer cancel()
	defer resp.Body.Close()
	return parseNDJSONAudio(resp.Body, onChunk)
}

// parseNDJSONAudio reads NDJSON lines and emits decoded PCM per audio line.
func parseNDJSONAudio(r io.Reader, onChunk func([]byte) error) error {
	br := bufio.NewReader(r)
	for {
		line, rerr := br.ReadBytes('\n')
		if s := strings.TrimSpace(string(line)); s != "" {
			var m map[string]any
			if json.Unmarshal([]byte(s), &m) == nil {
				if t := asString(m["type"]); t == "meta" {
					goto next
				} else if ok, has := m["success"].(bool); t == "error" || (has && !ok) {
					msg := asString(m["message"])
					if msg == "" {
						msg = "TTS stream error"
					}
					return &Error{Status: 200, Message: msg, Body: []byte(s)}
				}
				if res, ok := m["result"].(map[string]any); ok {
					if enc := asString(res["audioContent"]); enc != "" {
						pcm, err := base64.StdEncoding.DecodeString(enc)
						if err != nil {
							return fmt.Errorf("sixtydb: bad audio chunk: %w", err)
						}
						if err := onChunk(pcm); err != nil {
							return err
						}
					}
				}
			}
		}
	next:
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return nil
			}
			return rerr
		}
	}
}
