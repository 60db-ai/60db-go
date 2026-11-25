package sixtydb

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentBuilder is available as Client.Agents.Builder: it drafts an agent with AI from a description,
// pasted text, a website and documents. Drafts expire after 24 hours.
type AgentBuilder struct{ c *Client }

// AgentDraftParams are the inputs of Builder.CreateDraft. Description and Direction are required.
type AgentDraftParams struct {
	Description string // what the agent does (at most 2000 characters)
	Direction   string // "inbound" or "outbound"
	Language    string // optional language code
	Text        string // optional pasted text (at most 50,000 characters)
	WebsiteURL  string // optional https URL
	// Files and FilePaths are up to 5 documents in total (pdf, docx, doc, txt, md, rtf, html, csv,
	// xlsx, pptx; at most 20 MB each), sent under the form field "files".
	Files     []UploadFile
	FilePaths []string
}

var documentMIME = map[string]string{
	".doc": "application/msword", ".rtf": "application/rtf", ".html": "text/html", ".htm": "text/html",
	".csv":  "text/csv",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

func documentMIMEOf(filename string) string {
	if m, ok := documentMIME[strings.ToLower(filepath.Ext(filename))]; ok {
		return m
	}
	return mimeByExt(filename)
}

// CreateDraft starts building a draft (multipart upload); the result holds id and status.
func (b *AgentBuilder) CreateDraft(ctx context.Context, p AgentDraftParams) (any, error) {
	fields := map[string]string{}
	for k, v := range map[string]string{"description": p.Description, "direction": p.Direction,
		"language": p.Language, "text": p.Text, "website_url": p.WebsiteURL} {
		if v != "" {
			fields[k] = v
		}
	}
	files := make([]filePart, 0, len(p.Files)+len(p.FilePaths))
	for _, f := range p.Files {
		name := f.Filename
		if name == "" {
			name = "document.txt"
		}
		files = append(files, filePart{Field: "files", Filename: name, MIME: documentMIMEOf(name), Data: f.Data})
	}
	for _, path := range p.FilePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		name := filepath.Base(path)
		files = append(files, filePart{Field: "files", Filename: name, MIME: documentMIMEOf(name), Data: data})
	}
	buf, ct, err := buildMultipart(fields, files)
	if err != nil {
		return nil, err
	}
	m, err := b.c.doBody(ctx, "POST", "/agents/builder/drafts", buf, ct, timeoutLong)
	if err != nil {
		return nil, err
	}
	return m["data"], nil
}

// GetDraft returns id, status (queued, extracting, reading, writing, ready, failed), error and, when
// ready, draft{agent, voice, warnings, suggestions, sources}. draft.agent is a ready Agents.Create payload.
func (b *AgentBuilder) GetDraft(ctx context.Context, id string) (any, error) {
	return b.c.doData(ctx, "GET", "/agents/builder/drafts/"+pathEscape(id), nil, timeoutDefault)
}

// WaitForDraft polls GetDraft until status is ready or failed and returns the draft either way (a failed
// draft is not an error). 429, 5xx and network errors are retried; timeout returns an *Error with Code "TIMEOUT".
func (b *AgentBuilder) WaitForDraft(ctx context.Context, id string, opts *AgentWaitOptions) (map[string]any, error) {
	return waitFor(ctx, opts, 2*time.Second, 5*time.Minute, "Timed out waiting for the agent draft",
		func() (any, error) { return b.GetDraft(ctx, id) },
		func(d map[string]any) bool { st := asString(d["status"]); return st == "ready" || st == "failed" })
}
