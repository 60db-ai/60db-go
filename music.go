package sixtydb

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// MusicService is available as Client.Music. Responses are the full JSON body.
type MusicService struct{ c *Client }

// Create generates a song (params: prompt, mode, lyrics, tags, ...). With a non-nil
// lyricsFile the request is multipart (fields stringified, file part "lyrics_file"),
// otherwise JSON. Starts paid generation.
func (s *MusicService) Create(ctx context.Context, params map[string]any, lyricsFile *UploadFile) (map[string]any, error) {
	if lyricsFile == nil {
		return s.c.do(ctx, "POST", "/songs", params, timeoutDefault)
	}
	fields := map[string]string{}
	for k, v := range params {
		if v != nil {
			fields[k] = fmt.Sprint(v)
		}
	}
	name := lyricsFile.Filename
	if name == "" {
		name = "lyrics.txt"
	}
	buf, ct, err := buildMultipart(fields, []filePart{{Field: "lyrics_file", Filename: name, MIME: mimeByExt(name), Data: lyricsFile.Data}})
	if err != nil {
		return nil, err
	}
	return s.c.doBody(ctx, "POST", "/songs", buf, ct, timeoutLong)
}

// List lists songs (params: limit, offset, q, status, type, liked, sort).
func (s *MusicService) List(ctx context.Context, params map[string]any) (map[string]any, error) {
	return s.c.do(ctx, "GET", withQuery("/songs", params), nil, timeoutDefault)
}

func (s *MusicService) Get(ctx context.Context, id string) (map[string]any, error) {
	return s.c.do(ctx, "GET", "/songs/"+pathEscape(id), nil, timeoutDefault)
}

// Update sets liked and/or played (params: liked, played).
func (s *MusicService) Update(ctx context.Context, id string, params map[string]any) (map[string]any, error) {
	return s.c.do(ctx, "PATCH", "/songs/"+pathEscape(id), params, timeoutDefault)
}

func (s *MusicService) Delete(ctx context.Context, id string) (map[string]any, error) {
	return s.c.do(ctx, "DELETE", "/songs/"+pathEscape(id), nil, timeoutDefault)
}

// Download returns the MP3 bytes of a finished song.
func (s *MusicService) Download(ctx context.Context, id string) ([]byte, error) {
	b, _, err := s.c.raw(ctx, "GET", "/songs/"+pathEscape(id)+"/download", nil, timeoutLong)
	return b, err
}

// MusicWaitOptions tunes WaitForCompletion. Zero values mean 4 s / 10 min.
type MusicWaitOptions struct{ Interval, Timeout time.Duration }

// WaitForCompletion polls until the song status is succeeded or failed and returns the song (data).
func (s *MusicService) WaitForCompletion(ctx context.Context, id string, opts *MusicWaitOptions) (map[string]any, error) {
	interval, timeout := 4*time.Second, 10*time.Minute
	if opts != nil {
		if opts.Interval > 0 {
			interval = opts.Interval
		}
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
	}
	return poll(ctx, interval, timeout,
		func() error {
			return &Error{Code: "TIMEOUT", Message: fmt.Sprintf("Timed out waiting for song %s to complete", id)}
		},
		func() (bool, map[string]any, error) {
			body, err := s.Get(ctx, id)
			if err != nil {
				return false, nil, err
			}
			song, ok := body["data"].(map[string]any)
			if !ok {
				song = body
			}
			st := asString(song["status"])
			return st == "succeeded" || st == "failed", song, nil
		})
}

func (s *MusicService) ListVoices(ctx context.Context) (map[string]any, error) {
	return s.c.do(ctx, "GET", "/songs/voices", nil, timeoutDefault)
}

// CreateVoice uploads a voice reference (fields name, consent=true; part voice_reference).
func (s *MusicService) CreateVoice(ctx context.Context, name string, file UploadFile) (map[string]any, error) {
	if file.Filename == "" {
		file.Filename = "voice.wav"
	}
	if len(file.Data) == 0 {
		return nil, errors.New("sixtydb: voice file is empty")
	}
	buf, ct, err := buildMultipart(map[string]string{"name": name, "consent": "true"},
		[]filePart{{Field: "voice_reference", Filename: file.Filename, MIME: sniffAudioMIME(file.Data, file.Filename), Data: file.Data}})
	if err != nil {
		return nil, err
	}
	return s.c.doBody(ctx, "POST", "/songs/voices", buf, ct, timeoutLong)
}

func (s *MusicService) Health(ctx context.Context) (map[string]any, error) {
	return s.c.do(ctx, "GET", "/songs/health", nil, timeoutDefault)
}
