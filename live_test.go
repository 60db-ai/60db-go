package sixtydb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func liveClient(t *testing.T) *Client {
	key := os.Getenv("SIXTYDB_API_KEY")
	if key == "" {
		t.Skip("SIXTYDB_API_KEY not set")
	}
	c, _ := NewClient(Config{APIKey: key})
	return c
}

func wantStatus(t *testing.T, err error, status int) *Error {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Status != status {
		t.Fatalf("want status %d, got %v", status, err)
	}
	return e
}

func TestLive(t *testing.T) {
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	var wav []byte
	var voiceID string

	t.Run("GetsAndAccount", func(t *testing.T) {
		for name, f := range map[string]func(context.Context) (map[string]any, error){
			"voices": c.GetVoices, "myvoices": c.GetMyVoices, "tts models": c.GetTTSModels,
			"stt models": c.GetSTTModels, "profile": c.GetProfile, "workspaces": c.GetWorkspaces,
			"plans": c.GetPlans, "current plan": c.GetCurrentPlan, "usage": c.GetUsage,
			"api keys": c.GetAPIKeys, "webhooks": c.GetWebhooks, "memory usage": c.Memory.Usage,
		} {
			if _, err := f(ctx); err != nil {
				t.Errorf("%s: %v", name, err)
			}
		}
		if l, err := c.GetLanguages(ctx); err != nil || len(l) == 0 {
			t.Errorf("stt languages: %v %d", err, len(l))
		}
		if l, err := c.GetTTSLanguages(ctx); err != nil || len(l) == 0 {
			t.Errorf("tts languages: %v %d", err, len(l))
		}
		v, err := c.GetVoices(ctx)
		if err != nil {
			t.Fatal(err)
		}
		list, _ := v["data"].([]any)
		if len(list) == 0 {
			t.Fatal("no voices")
		}
		voiceID = asString(list[0].(map[string]any)["voice_id"])
		if _, err := c.GetVoice(ctx, voiceID); err != nil {
			t.Errorf("get voice: %v", err)
		}
	})

	t.Run("TTS", func(t *testing.T) {
		var err error
		wav, err = c.TextToSpeech(ctx, "Hello from the Go SDK.", voiceID, map[string]any{"sample_rate": 16000})
		if err != nil || len(wav) < 44 || string(wav[:4]) != "RIFF" {
			t.Fatalf("tts: %v len=%d", err, len(wav))
		}
	})

	t.Run("TTSStream", func(t *testing.T) {
		var pcm []byte
		chunks := 0
		err := c.TextToSpeechStream(ctx, "Streaming from Go.", voiceID, map[string]any{"sample_rate": 16000},
			func(p []byte) error { chunks++; pcm = append(pcm, p...); return nil })
		if err != nil || len(pcm) == 0 || len(pcm)%2 != 0 {
			t.Fatalf("stream: %v chunks=%d bytes=%d", err, chunks, len(pcm))
		}
		if w := PCMToWAV(pcm, 16000); string(w[:4]) != "RIFF" {
			t.Fatal("wav")
		}
	})

	t.Run("STT", func(t *testing.T) {
		if wav == nil {
			t.Skip("no wav")
		}
		out, err := c.SpeechToText(ctx, wav, &STTOptions{Language: "en"})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := out["text"]; !ok {
			t.Fatalf("no text: %v", out)
		}
		t.Logf("transcript: %v", out["text"])
	})

	t.Run("Chat", func(t *testing.T) {
		msgs := []ChatMessage{{"user", "Reply with the single word: pong"}}
		out, err := c.Chat.Completion(ctx, ChatRequest{Messages: msgs})
		if err != nil || out["choices"] == nil {
			t.Fatalf("completion: %v", err)
		}
		var text string
		err = c.Chat.Stream(ctx, ChatRequest{Messages: msgs}, func(m map[string]any) error {
			text += ChunkContent(m)
			return nil
		})
		if err != nil || text == "" {
			t.Fatalf("stream: %v %q", err, text)
		}
		if _, err := c.Chat.Completion(ctx, ChatRequest{Model: "no-such-model-xyz", Messages: msgs}); err == nil {
			t.Fatal("unknown model should fail")
		}
	})

	t.Run("Memory", func(t *testing.T) {
		raw := func(method, path string, body any) (map[string]any, error) {
			return c.do(ctx, method, path, body, timeoutMemory)
		}
		created, err := raw("POST", "/memory/collections", map[string]any{"collection_id": fmt.Sprintf("sdk-test-go-%d", time.Now().Unix()), "label": "SDK Go test"})
		if err != nil {
			t.Fatal(err)
		}
		d, _ := created["data"].(map[string]any)
		coll := asString(d["collection_id"])
		if coll == "" {
			t.Fatalf("no collection id: %v", created)
		}
		defer func() {
			if _, err := raw("DELETE", "/memory/collections/"+pathEscape(coll), nil); err != nil {
				t.Errorf("cleanup: %v", err)
			}
		}()
		p := map[string]any{"collection": coll}
		if _, err := c.Memory.Ingest(ctx, "The Go SDK test user likes green tea.", mergeParams(p, map[string]any{"infer": false, "title": "go sdk"})); err != nil {
			t.Fatalf("ingest: %v", err)
		}
		if _, err := c.Memory.Search(ctx, "tea", p); err != nil {
			t.Fatalf("search: %v", err)
		}
		cx, err := c.Memory.Context(ctx, "tea", p)
		if err != nil {
			t.Fatalf("context: %v", err)
		}
		if dd, _ := cx["data"].(map[string]any); dd == nil {
			t.Fatalf("context shape: %v", cx)
		}
		if _, err := c.Memory.ExtractDocument(ctx, []byte("Hello document text for extraction."), "note.txt", nil); err != nil {
			t.Fatalf("extract document: %v", err)
		}
		_, err = c.Memory.Search(ctx, "x", map[string]any{"collection": "does_not_exist_zz"})
		if e := wantStatus(t, err, 404); e.Code != "COLLECTION_NOT_FOUND" {
			t.Errorf("code %s", e.Code)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		_, err := c.TextToSpeech(ctx, "", "", nil)
		if err == nil {
			t.Fatal("empty text")
		}
		// server-side empty-text check
		body, _ := json.Marshal(map[string]any{"text": ""})
		_, _, err = c.rawBody(ctx, http.MethodPost, "/tts-synthesize", bytes.NewReader(body), "application/json", timeoutDefault)
		if e := wantStatus(t, err, 400); !strings.Contains(e.Message, "text is required") {
			t.Errorf("msg %q", e.Message)
		}
		bad, _ := NewClient(Config{APIKey: "sk_invalid1234567890"})
		_, err = bad.GetProfile(ctx)
		wantStatus(t, err, 401)
	})
}
