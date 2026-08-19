package compression

import (
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

func GzipMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			contentEncoding := strings.TrimSpace(request.Header.Get("Content-Encoding"))
			if contentEncoding != "" && !strings.EqualFold(contentEncoding, "identity") {
				if !strings.EqualFold(contentEncoding, "gzip") {
					writer.Header().Set("Accept-Encoding", "gzip")
					http.Error(writer, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
					return
				}

				cr, err := newCompressReader(request.Body)
				if err != nil {
					http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
					return
				}
				request.Body = cr
				request.Header.Del("Content-Encoding")
				request.Header.Del("Content-Length")
				request.ContentLength = -1
				defer func() {
					if err = cr.Close(); err != nil {
						log.Error("cannot close compress reader", zap.Error(err))
					}
				}()
			}

			cw := newCompressWriter(writer, request.Method, acceptsGzip(request.Header))
			defer func() {
				if err := cw.Close(); err != nil {
					log.Error("cannot close compress writer", zap.Error(err))
				}
			}()

			handler.ServeHTTP(cw, request)
		})
	}
}

func acceptsGzip(header http.Header) bool {
	explicitGzip := false
	gzipQuality := -1.0
	wildcardQuality := -1.0

	for _, headerValue := range header.Values("Accept-Encoding") {
		for _, item := range strings.Split(headerValue, ",") {
			parts := strings.Split(item, ";")
			coding := strings.ToLower(strings.TrimSpace(parts[0]))
			if coding == "" {
				continue
			}

			quality := 1.0
			valid := true
			for _, parameter := range parts[1:] {
				name, value, found := strings.Cut(strings.TrimSpace(parameter), "=")
				if !found || !strings.EqualFold(strings.TrimSpace(name), "q") {
					valid = false
					break
				}
				parsedQuality, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				if err != nil || parsedQuality < 0 || parsedQuality > 1 {
					valid = false
					break
				}
				quality = parsedQuality
			}

			switch coding {
			case "gzip":
				explicitGzip = true
				if !valid {
					quality = 0
				}
				if quality > gzipQuality {
					gzipQuality = quality
				}
			case "*":
				if valid && quality > wildcardQuality {
					wildcardQuality = quality
				}
			}
		}
	}

	if explicitGzip {
		return gzipQuality > 0
	}

	return wildcardQuality > 0
}
