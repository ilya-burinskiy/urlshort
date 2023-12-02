package compress

import (
	"compress/gzip"
	"io"
	"net/http"
)

type Writer struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func NewWriter(w http.ResponseWriter) *Writer {
	return &Writer{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (cw *Writer) Header() http.Header {
	return cw.w.Header()
}

func (cw *Writer) Write(p []byte) (int, error) {
	return cw.zw.Write(p)
}

func (cw *Writer) WriteHeader(statusCode int) {
	if statusCode < 300 {
		cw.w.Header().Set("Content-Encoding", "gzip")
	}
	cw.w.WriteHeader(statusCode)
}

func (cw *Writer) Close() error {
	return cw.zw.Close()
}

type Reader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

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

func (cr Reader) Read(p []byte) (int, error) {
	return cr.zr.Read(p)
}

func (cr *Reader) Close() error {
	if err := cr.r.Close(); err != nil {
		return err
	}
	return cr.zr.Close()
}
