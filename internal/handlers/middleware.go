package handlers

import (
	"github.com/sebasttiano/Budgie/internal/common"
	"github.com/sebasttiano/Budgie/internal/logger"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

// WithLogging - middleware that logs request and response params
func WithLogging(next http.Handler) http.Handler {
	logFn := func(res http.ResponseWriter, req *http.Request) {

		start := time.Now()

		responseData := &responseData{
			status: 200,
			size:   0,
		}

		lw := loggingResponseWriter{
			ResponseWriter: res,
			responseData:   responseData,
		}

		next.ServeHTTP(&lw, req)

		duration := time.Since(start)
		sugar := logger.Log.Sugar()
		sugar.Infoln(
			zap.String("uri", req.RequestURI),
			zap.String("method", req.Method),
			zap.Int("status", responseData.status),
			zap.Duration("duration", duration),
			zap.Int("size", responseData.size),
		)

	}
	return http.HandlerFunc(logFn)
}

type (
	// for response data
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// GzipMiddleware handles compressed with gzip requests and responses
func GzipMiddleware(next http.Handler) http.Handler {

	gzipFn := func(w http.ResponseWriter, r *http.Request) {

		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			cw := common.NewGZIPWriter(w)
			ow = cw
			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := common.NewZIPReader(r.Body)
			if err != nil {
				logger.Log.Error("couldn`t decompress request")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		next.ServeHTTP(ow, r)
	}
	return http.HandlerFunc(gzipFn)
}
