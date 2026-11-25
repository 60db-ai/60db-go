package sixtydb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// STTOptions are optional speech-to-text fields.
type STTOptions struct {
	Filename string // defaults to audio.<ext> from the sniffed type
	Language string
	Diarize  bool
	Context  string
}

// SpeechToText transcribes audio bytes. Upload field is "file"; the MIME type is
// sniffed from magic bytes because the server validates it.
func (c *Client) SpeechToText(ctx context.Context, audio []byte, opts *STTOptions) (map[string]any, error) {
	if opts == nil {
		opts = &STTOptions{}
	}
	mime := sniffAudioMIME(audio, opts.Filename)
	name := opts.Filename
	if name == "" {
		name = "audio." + extForMIME(mime)
	}
	fields := map[string]string{}
	if opts.Language != "" {
		fields["language"] = opts.Language
	}
	if opts.Diarize {
		fields["diarize"] = "true"
	}
	if ctxt := strings.TrimSpace(opts.Context); ctxt != "" {
		fields["context"] = ctxt
	}
	buf, ct, err := buildMultipart(fields, []filePart{{Field: "file", Filename: name, MIME: mime, Data: audio}})
	if err != nil {
		return nil, err
	}
	return c.doBody(ctx, "POST", "/stt", buf, ct, timeoutLong)
}

// SpeechToTextFile reads a file from disk and transcribes it.
func (c *Client) SpeechToTextFile(ctx context.Context, path string, opts *STTOptions) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sixtydb: %w", err)
	}
	o := STTOptions{}
	if opts != nil {
		o = *opts
	}
	if o.Filename == "" {
		o.Filename = filepath.Base(path)
	}
	return c.SpeechToText(ctx, data, &o)
}
