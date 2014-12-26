package util

import (
	"fmt"
	"net/http"
	"time"

	"github.com/arjantop/saola"
	"github.com/arjantop/saola/httpservice"
	"github.com/quipo/statsd"
	"golang.org/x/net/context"
)

type responseLoggingWriter struct {
	status int
	writer http.ResponseWriter
}

func (w *responseLoggingWriter) Header() http.Header {
	return w.writer.Header()
}

func (w *responseLoggingWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *responseLoggingWriter) WriteHeader(statusCode int) {
	if w.status == 0 {
		w.status = statusCode
	}
	w.writer.WriteHeader(statusCode)
}

func (w *responseLoggingWriter) StatusCode() int {
	return w.status
}

func NewResponseStatsFilter(client statsd.Statsd) saola.Filter {
	return saola.FuncFilter(func(ctx context.Context, s saola.Service) error {
		req := httpservice.GetHttpRequest(ctx)
		responseWriter := &responseLoggingWriter{writer: req.Writer}

		start := time.Now()
		err := s.Do(httpservice.WithHttpRequest(ctx, responseWriter, req.Request))

		statusCode := responseStatusCode(responseWriter.StatusCode(), err)

		updateStats(client, statusCode, time.Now().Sub(start))

		return err
	})
}

func responseStatusCode(statusCode int, err error) int {
	if statusCode == 0 {
		if err != nil {
			return http.StatusInternalServerError
		}
		return http.StatusOK
	}
	return statusCode
}

func updateStats(client statsd.Statsd, statusCode int, d time.Duration) {
	statusCodeClass := fmt.Sprintf("%dxx", statusCode/100)
	client.Incr(fmt.Sprintf("http.status.%d", statusCode), 1)
	client.PrecisionTiming(fmt.Sprintf("http.time.%d", statusCode), d)
	client.Incr("http.status."+statusCodeClass, 1)
	client.PrecisionTiming("http.time."+statusCodeClass, d)
}
