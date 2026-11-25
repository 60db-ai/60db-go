package sixtydb

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"
)

// filePart is one binary part of a multipart form.
type filePart struct {
	Field    string
	Filename string
	MIME     string
	Data     []byte
}

// buildMultipart encodes string fields and file parts; returns body and content type.
func buildMultipart(fields map[string]string, files []filePart) (*bytes.Buffer, string, error) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}
	for _, f := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(f.Field), escapeQuotes(f.Filename)))
		h.Set("Content-Type", f.MIME)
		pw, err := w.CreatePart(h)
		if err != nil {
			return nil, "", err
		}
		if _, err := pw.Write(f.Data); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf, w.FormDataContentType(), nil
}

func escapeQuotes(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}

var extMIME = map[string]string{
	".wav": "audio/wav", ".mp3": "audio/mpeg", ".ogg": "audio/ogg", ".oga": "audio/ogg",
	".flac": "audio/flac", ".m4a": "audio/mp4", ".mp4": "audio/mp4", ".webm": "audio/webm",
	".pdf": "application/pdf", ".txt": "text/plain", ".md": "text/markdown",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
}

// sniffAudioMIME detects an audio MIME type from magic bytes, then the filename extension.
func sniffAudioMIME(data []byte, filename string) string {
	switch {
	case len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WAVE":
		return "audio/wav"
	case len(data) >= 3 && string(data[0:3]) == "ID3",
		len(data) >= 2 && data[0] == 0xFF && data[1]&0xE0 == 0xE0:
		return "audio/mpeg"
	case len(data) >= 4 && string(data[0:4]) == "OggS":
		return "audio/ogg"
	case len(data) >= 4 && string(data[0:4]) == "fLaC":
		return "audio/flac"
	case len(data) >= 8 && string(data[4:8]) == "ftyp":
		return "audio/mp4"
	case len(data) >= 4 && data[0] == 0x1A && data[1] == 0x45 && data[2] == 0xDF && data[3] == 0xA3:
		return "audio/webm"
	}
	return mimeByExt(filename)
}

func mimeByExt(filename string) string {
	if m, ok := extMIME[strings.ToLower(filepath.Ext(filename))]; ok {
		return m
	}
	return "application/octet-stream"
}

// extForMIME returns the extension (without dot) used for default filenames.
func extForMIME(mime string) string {
	switch mime {
	case "audio/wav":
		return "wav"
	case "audio/mpeg":
		return "mp3"
	case "audio/ogg":
		return "ogg"
	case "audio/flac":
		return "flac"
	case "audio/mp4":
		return "m4a"
	case "audio/webm":
		return "webm"
	}
	return "bin"
}

// UploadFile is a named blob for multipart uploads.
type UploadFile struct {
	Filename string
	Data     []byte
}
