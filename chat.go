package sixtydb

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// DefaultChatModel is used when ChatRequest.Model is empty.
const DefaultChatModel = "vyas"

// ChatMessage is one chat turn.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is a chat (Messages) or text-correction (Text) request.
// Extra carries temperature, max_tokens, chat_id, save_chat, ...
type ChatRequest struct {
	Model      string
	Messages   []ChatMessage
	Text       string
	Dictionary any
	Style      any
	AppContext string
	Extra      map[string]any
}

// ChatService is available as Client.Chat.
type ChatService struct{ c *Client }

func (r ChatRequest) body(stream bool) (map[string]any, error) {
	if len(r.Messages) == 0 && r.Text == "" {
		return nil, errors.New("sixtydb: provide either Messages or Text")
	}
	model := r.Model
	if model == "" {
		model = DefaultChatModel
	}
	b := map[string]any{"model": model, "stream": stream}
	if len(r.Messages) > 0 {
		b["messages"] = r.Messages
	}
	if r.Text != "" {
		b["text"] = r.Text
	}
	if r.Dictionary != nil {
		b["dictionary"] = r.Dictionary
	}
	if r.Style != nil {
		b["style"] = r.Style
	}
	if r.AppContext != "" {
		b["appContext"] = r.AppContext
	}
	return mergeParams(b, r.Extra), nil
}

// Completion runs a non-streaming completion and returns the OpenAI-shaped JSON.
func (s *ChatService) Completion(ctx context.Context, req ChatRequest) (map[string]any, error) {
	body, err := req.body(false)
	if err != nil {
		return nil, err
	}
	return s.c.do(ctx, "POST", "/v1/chat/completions", body, timeoutLong)
}

// Stream runs a streaming completion, calling onChunk for every decoded SSE chunk
// (chat_id chunk first, then deltas). In-stream error chunks are returned as *Error.
func (s *ChatService) Stream(ctx context.Context, req ChatRequest, onChunk func(chunk map[string]any) error) error {
	body, err := req.body(true)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(body)
	resp, cancel, err := s.c.send(ctx, "POST", "/v1/chat/completions", bytesReader(payload), "application/json", timeoutLong)
	if err != nil {
		return err
	}
	defer cancel()
	defer resp.Body.Close()
	return parseSSE(resp.Body, onChunk)
}

// ChunkContent returns choices[0].delta.content of a chunk ("" if none).
func ChunkContent(chunk map[string]any) string {
	ch, _ := chunk["choices"].([]any)
	if len(ch) == 0 {
		return ""
	}
	c0, _ := ch[0].(map[string]any)
	d, _ := c0["delta"].(map[string]any)
	return asString(d["content"])
}

func parseSSE(r io.Reader, onChunk func(map[string]any) error) error {
	br := bufio.NewReader(r)
	for {
		line, rerr := br.ReadString('\n')
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				return nil
			}
			var m map[string]any
			if data != "" && json.Unmarshal([]byte(data), &m) == nil {
				if e := chunkError(m, data); e != nil {
					return e
				}
				if err := onChunk(m); err != nil {
					return err
				}
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return nil
			}
			return rerr
		}
	}
}

func chunkError(m map[string]any, raw string) error {
	_, hasObj := m["error"].(map[string]any)
	if asString(m["type"]) != "error" && !hasObj {
		return nil
	}
	msg := asString(m["message"])
	switch e := m["error"].(type) {
	case string:
		var inner map[string]any
		if json.Unmarshal([]byte(e), &inner) == nil {
			if ie, ok := inner["error"].(map[string]any); ok && asString(ie["message"]) != "" {
				msg = asString(ie["message"])
			}
		}
		if msg == "" {
			msg = e
		}
	case map[string]any:
		if msg == "" {
			msg = asString(e["message"])
		}
	}
	if msg == "" {
		msg = "chat stream error"
	}
	return &Error{Status: 200, Message: msg, Body: []byte(raw)}
}
