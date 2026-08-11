package encryption

import (
	"bufio"
	"compress/gzip"
	"mime"
	"net"
	"net/http"
	"strings"
)

var compressibleContentTypes = map[string]struct{}{
	"application/json": {},
	"text/html":        {},
}

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type compressWriter struct {
	w            http.ResponseWriter
	zw           *gzip.Writer
	method       string
	supportsGzip bool
	wroteHeader  bool
	compressed   bool
}

func newCompressWriter(w http.ResponseWriter, method string, supportsGzip bool) *compressWriter {
	return &compressWriter{
		w:            w,
		method:       method,
		supportsGzip: supportsGzip,
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		if c.Header().Get("Content-Type") == "" {
			c.Header().Set("Content-Type", http.DetectContentType(p))
		}
		c.WriteHeader(http.StatusOK)
	}
	if c.compressed {
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if c.wroteHeader {
		return
	}
	c.wroteHeader = true

	if c.canNegotiate(statusCode) {
		addVary(c.Header(), "Accept-Encoding")
	}
	if c.shouldCompress(statusCode) {
		c.w.Header().Set("Content-Encoding", "gzip")
		c.w.Header().Del("Content-Length")
		c.zw = gzip.NewWriter(c.w)
		c.compressed = true
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressWriter) Close() error {
	if c.zw == nil {
		return nil
	}
	return c.zw.Close()
}

func (c *compressWriter) Unwrap() http.ResponseWriter {
	return c.w
}

func (c *compressWriter) Flush() {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	if c.zw != nil {
		_ = c.zw.Flush()
	}
	if flusher, ok := c.w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (c *compressWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := c.w.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (c *compressWriter) Push(target string, options *http.PushOptions) error {
	pusher, ok := c.w.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, options)
}

func (c *compressWriter) canNegotiate(statusCode int) bool {
	return c.method != http.MethodHead &&
		statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices &&
		statusCode != http.StatusNoContent && statusCode != http.StatusResetContent &&
		c.Header().Get("Content-Encoding") == "" &&
		isCompressibleContentType(c.Header().Get("Content-Type"))
}

func (c *compressWriter) shouldCompress(statusCode int) bool {
	return c.supportsGzip && c.canNegotiate(statusCode)
}

func isCompressibleContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	_, ok := compressibleContentTypes[strings.ToLower(mediaType)]
	return ok
}

func addVary(header http.Header, value string) {
	for _, currentValue := range header.Values("Vary") {
		for _, item := range strings.Split(currentValue, ",") {
			if strings.EqualFold(strings.TrimSpace(item), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}
