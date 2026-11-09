package compression

import (
	"compress/gzip"
	"errors"
	"io"
)

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	r  io.ReadCloser
	zr io.ReadCloser
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read returns decompressed request bytes.
func (c *compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close releases both the gzip reader and the request body.
func (c *compressReader) Close() error {
	return errors.Join(c.zr.Close(), c.r.Close())
}
