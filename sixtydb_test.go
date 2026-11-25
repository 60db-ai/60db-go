package sixtydb

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := NewClient(Config{APIKey: "k", BaseURL: srv.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestWebhookVector(t *testing.T) {
	want := "v1=1698a50bc74d1ff1db85c4e0a5297c2ad9fdba245d5737cdb789e4cc6e098940"
	body := []byte(`{"a":1}`)
	if got := ComputeWebhookSignature(body, "1700000000", "s3cret"); got != want {
		t.Fatalf("got %s", got)
	}
	if !VerifyWebhookSignature(body, want, "1700000000", "s3cret", 300, 1700000100) {
		t.Fatal("should verify")
	}
	if VerifyWebhookSignature(body, want, "1700000000", "s3cret", 300, 1700001000) {
		t.Fatal("stale timestamp must fail")
	}
	if VerifyWebhookSignature(body, want, "17000000000000", "s3cret", 300, 1700000100) ||
		VerifyWebhookSignature(body, "", "1700000000", "s3cret", 300, 1700000100) ||
		VerifyWebhookSignature(body, want, "1700000000", "", 300, 1700000100) ||
		VerifyWebhookSignature(body, "v1=bad", "1700000000", "s3cret", 300, 1700000100) {
		t.Fatal("invalid input must fail")
	}
}

func TestPCMToWAV(t *testing.T) {
	w := PCMToWAV(make([]byte, 100), 24000)
	if len(w) != 144 || string(w[0:4]) != "RIFF" || string(w[8:12]) != "WAVE" || string(w[36:40]) != "data" {
		t.Fatal("bad header")
	}
	if binary.LittleEndian.Uint32(w[24:]) != 24000 || binary.LittleEndian.Uint32(w[28:]) != 48000 ||
		binary.LittleEndian.Uint32(w[40:]) != 100 || binary.LittleEndian.Uint32(w[4:]) != 136 ||
		binary.LittleEndian.Uint16(w[22:]) != 1 || binary.LittleEndian.Uint16(w[34:]) != 16 {
		t.Fatal("bad header fields")
	}
}

func TestSniffMIME(t *testing.T) {
	cases := []struct {
		data []byte
		name string
		want string
	}{
		{[]byte("RIFF\x00\x00\x00\x00WAVEfmt "), "", "audio/wav"},
		{[]byte("ID3\x03"), "", "audio/mpeg"},
		{[]byte{0xFF, 0xFB, 0x90}, "", "audio/mpeg"},
		{[]byte("OggS\x00"), "", "audio/ogg"},
		{[]byte("fLaC"), "", "audio/flac"},
		{[]byte("\x00\x00\x00\x20ftypM4A "), "", "audio/mp4"},
		{[]byte{0x1A, 0x45, 0xDF, 0xA3}, "", "audio/webm"},
		{[]byte("xx"), "a.MP3", "audio/mpeg"},
		{[]byte("xx"), "a.zzz", "application/octet-stream"},
	}
	for _, tc := range cases {
		if got := sniffAudioMIME(tc.data, tc.name); got != tc.want {
			t.Errorf("%q: got %s want %s", tc.data, got, tc.want)
		}
	}
}

func TestSTTMultipart(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stt" || r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("bad request %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		fhs := r.MultipartForm.File["file"]
		if len(fhs) != 1 || fhs[0].Filename != "audio.wav" || fhs[0].Header.Get("Content-Type") != "audio/wav" {
			t.Errorf("bad file part: %+v", r.MultipartForm.File)
		}
		if r.FormValue("diarize") != "true" || r.FormValue("language") != "en" || r.FormValue("context") != "hi" {
			t.Errorf("bad fields %v", r.MultipartForm.Value)
		}
		w.Write([]byte(`{"text":"ok"}`))
	})
	out, err := c.SpeechToText(context.Background(), []byte("RIFF\x00\x00\x00\x00WAVE"),
		&STTOptions{Language: "en", Diarize: true, Context: "  hi "})
	if err != nil || out["text"] != "ok" {
		t.Fatalf("%v %v", out, err)
	}
}

func TestTTS(t *testing.T) {
	wav := PCMToWAV([]byte{1, 2}, 8000)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if strings.Contains(string(b), `"stream"`) {
			t.Error("non-stream must not send stream")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"audio_base64":"` + base64.StdEncoding.EncodeToString(wav) + `"}`))
	})
	got, err := c.TextToSpeech(context.Background(), "hi", "v", nil)
	if err != nil || string(got[:4]) != "RIFF" {
		t.Fatalf("%v", err)
	}
	if _, err := c.TextToSpeech(context.Background(), "  ", "", nil); err == nil {
		t.Fatal("empty text must error")
	}
}

func TestTTSStream(t *testing.T) {
	enc := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"stream":true`) {
			t.Error("stream flag missing")
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		io.WriteString(w, `{"conditioning":{"a":1}}`+"\n")
		io.WriteString(w, `{"type":"meta"}`+"\n")
		io.WriteString(w, `{"result":{"audioContent":"`+enc([]byte{1, 2})+`"}}`+"\n")
		io.WriteString(w, `{"result":{"audioContent":"`+enc([]byte{3})+`"}}`+"\n")
		io.WriteString(w, `{"type":"error","message":"boom"}`+"\n")
	})
	var got []byte
	err := c.TextToSpeechStream(context.Background(), "hi", "", nil, func(p []byte) error {
		got = append(got, p...)
		return nil
	})
	var e *Error
	if !errors.As(err, &e) || e.Message != "boom" {
		t.Fatalf("err=%v", err)
	}
	if string(got) != "\x01\x02\x03" {
		t.Fatalf("got %v", got)
	}
}

func TestChatStreamAndErrors(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"type\":\"chat_id\",\"chat_id\":\"c1\"}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\" there\"}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	})
	var text string
	var ids int
	err := c.Chat.Stream(context.Background(), ChatRequest{Messages: []ChatMessage{{"user", "hi"}}},
		func(m map[string]any) error {
			if m["type"] == "chat_id" {
				ids++
			}
			text += ChunkContent(m)
			return nil
		})
	if err != nil || text != "Hi there" || ids != 1 {
		t.Fatalf("%q %d %v", text, ids, err)
	}

	inner := `{\"error\":{\"message\":\"The model x does not exist\"}}`
	c2 := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "data: {\"type\":\"error\",\"message\":\"outer\",\"error\":\""+inner+"\"}\n\n")
	})
	err = c2.Chat.Stream(context.Background(), ChatRequest{Text: "x"}, func(map[string]any) error { return nil })
	var e *Error
	if !errors.As(err, &e) || e.Message != "The model x does not exist" {
		t.Fatalf("err=%v", err)
	}
	if err := c.Chat.Stream(context.Background(), ChatRequest{}, nil); err == nil {
		t.Fatal("need messages or text")
	}
}

func TestChatCompletionBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		s := string(b)
		if !strings.Contains(s, `"model":"vyas"`) || !strings.Contains(s, `"stream":false`) || !strings.Contains(s, `"temperature":0.5`) {
			t.Errorf("body %s", s)
		}
		io.WriteString(w, `{"choices":[{"message":{"content":"yo"}}]}`)
	})
	out, err := c.Chat.Completion(context.Background(), ChatRequest{
		Messages: []ChatMessage{{"user", "hi"}}, Extra: map[string]any{"temperature": 0.5}})
	if err != nil || out["choices"] == nil {
		t.Fatal(err)
	}
}

func TestErrorShape(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/plain" {
			w.WriteHeader(502)
			return
		}
		w.WriteHeader(403)
		io.WriteString(w, `{"message":"nope","code":"POLICY_DENY","field":"scope"}`)
	})
	_, err := c.GetProfile(context.Background())
	var e *Error
	if !errors.As(err, &e) || e.Status != 403 || e.Code != "POLICY_DENY" || e.Field != "scope" || e.Message != "nope" || len(e.Body) == 0 {
		t.Fatalf("%+v", e)
	}
	_, err = c.do(context.Background(), "GET", "/plain", nil, timeoutDefault)
	if !errors.As(err, &e) || e.Message != "HTTP 502" {
		t.Fatalf("%+v", e)
	}
}

func TestMemoryAndPathEscape(t *testing.T) {
	var paths []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.EscapedPath())
		io.WriteString(w, `{"success":true}`)
	})
	ctx := context.Background()
	c.Memory.Ingest(ctx, "t", map[string]any{"collection": "x"})
	c.Memory.Search(ctx, "q", nil)
	c.Memory.Context(ctx, "q", nil)
	c.Memory.Usage(ctx)
	c.GetVoice(ctx, "a/b c")
	want := "POST /memory/ingest,POST /memory/search,POST /memory/context,GET /memory/usage,GET /voices/a%2Fb%20c"
	if strings.Join(paths, ",") != want {
		t.Fatalf("%v", paths)
	}
}
