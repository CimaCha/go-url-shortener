package encryption

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type trackingReadCloser struct {
	name       string
	closeError error
	closed     *[]string
}

func (c *trackingReadCloser) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (c *trackingReadCloser) Close() error {
	*c.closed = append(*c.closed, c.name)
	return c.closeError
}

func TestCompressReaderClose(t *testing.T) {
	errGzipClose := errors.New("gzip close error")
	errSourceClose := errors.New("source close error")
	tests := []struct {
		name            string
		gzipError       error
		sourceError     error
		wantGzipError   bool
		wantSourceError bool
	}{
		{name: "both readers close successfully"},
		{name: "gzip close error is returned", gzipError: errGzipClose, wantGzipError: true},
		{name: "source close error is returned", sourceError: errSourceClose, wantSourceError: true},
		{name: "both close errors are returned", gzipError: errGzipClose, sourceError: errSourceClose, wantGzipError: true, wantSourceError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closed := make([]string, 0, 2)
			reader := &compressReader{
				zr: &trackingReadCloser{name: "gzip", closeError: tt.gzipError, closed: &closed},
				r:  &trackingReadCloser{name: "source", closeError: tt.sourceError, closed: &closed},
			}

			err := reader.Close()

			assert.Equal(t, []string{"gzip", "source"}, closed)
			assert.Equal(t, tt.wantGzipError, errors.Is(err, errGzipClose))
			assert.Equal(t, tt.wantSourceError, errors.Is(err, errSourceClose))
		})
	}
}
