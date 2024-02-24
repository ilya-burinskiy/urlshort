package compress

import (
	"compress/gzip"
	"io"
	"net/http"
)

// Response writer with gzip compression
type Writer struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// Creates response writer with gzip compression
func NewWriter(w http.ResponseWriter) *Writer {
	return &Writer{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Header
func (cw *Writer) Header() http.Header {
	return cw.w.Header()
}

// Writes compressed data
func (cw *Writer) Write(p []byte) (int, error) {
	return cw.zw.Write(p)
}

// WriteHeader
func (cw *Writer) WriteHeader(statusCode int) {
	if statusCode < 300 {
		cw.w.Header().Set("Content-Encoding", "gzip")
	}
	cw.w.WriteHeader(statusCode)
}

// Close
func (cw *Writer) Close() error {
	return cw.zw.Close()
}

// Reader for compressed data
type Reader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// Creates reader for compressed data
func NewReader(r io.ReadCloser) (*Reader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &Reader{
		r:  r,
		zr: zr,
	}, nil
}

// Read uncompressed data
func (cr Reader) Read(p []byte) (int, error) {
	return cr.zr.Read(p)
}

// Close
func (cr *Reader) Close() error {
	if err := cr.r.Close(); err != nil {
		return err
	}
	return cr.zr.Close()
}
