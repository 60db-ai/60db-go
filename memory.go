package sixtydb

import (
	"context"
	"os"
	"path/filepath"
)

// MemoryService is available as Client.Memory.
type MemoryService struct{ c *Client }

// Ingest stores text. params: title, type, collection, infer, session_id, metadata.
func (s *MemoryService) Ingest(ctx context.Context, text string, params map[string]any) (map[string]any, error) {
	return s.c.do(ctx, "POST", "/memory/ingest", mergeParams(map[string]any{"text": text}, params), timeoutMemory)
}

// Search queries memory. params: mode, max_results, alpha, recency_bias, collection, type.
func (s *MemoryService) Search(ctx context.Context, query string, params map[string]any) (map[string]any, error) {
	return s.c.do(ctx, "POST", "/memory/search", mergeParams(map[string]any{"query": query}, params), timeoutMemory)
}

// Context returns a prompt-ready context (data.prompt_ready).
func (s *MemoryService) Context(ctx context.Context, query string, params map[string]any) (map[string]any, error) {
	return s.c.do(ctx, "POST", "/memory/context", mergeParams(map[string]any{"query": query}, params), timeoutMemory)
}

// ExtractDocument uploads a document (multipart field "file") plus string fields.
func (s *MemoryService) ExtractDocument(ctx context.Context, file []byte, filename string, fields map[string]string) (map[string]any, error) {
	if filename == "" {
		filename = "document"
	}
	buf, ct, err := buildMultipart(fields, []filePart{{Field: "file", Filename: filename, MIME: mimeByExt(filename), Data: file}})
	if err != nil {
		return nil, err
	}
	return s.c.doBody(ctx, "POST", "/memory/documents/extract", buf, ct, timeoutLong)
}

// ExtractDocumentFile reads a document from disk and extracts it.
func (s *MemoryService) ExtractDocumentFile(ctx context.Context, path string, fields map[string]string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.ExtractDocument(ctx, data, filepath.Base(path), fields)
}

// Usage returns memory usage.
func (s *MemoryService) Usage(ctx context.Context) (map[string]any, error) {
	return s.c.do(ctx, "GET", "/memory/usage", nil, timeoutMemory)
}
