package util

import (
	"strconv"
	"time"

	"github.com/arjantop/saola"
	"github.com/arjantop/saola/httpservice"
	"github.com/wadey/go-zipkin"
	gzipkin "github.com/wadey/go-zipkin/gen-go/zipkin"
	"golang.org/x/net/context"
)

type traceKey struct{}

func withTrace(ctx context.Context, trace *zipkin.Trace) context.Context {
	return context.WithValue(ctx, traceKey{}, trace)
}

func GetTrace(ctx context.Context) (*zipkin.Trace, bool) {
	t, ok := ctx.Value(traceKey{}).(*zipkin.Trace)
	return t, ok
}

func NewServerTracingFilter(port int, cs []zipkin.SpanCollector) saola.Filter {
	return saola.FuncFilter(func(ctx context.Context, s saola.Service) error {
		r := httpservice.GetServerRequest(ctx)
		trace := zipkin.NewTraceForHTTPHeader(r.Request.Method, r.Request.Header, cs)
		trace.Endpoint = &gzipkin.Endpoint{
			Ipv4:        0,
			Port:        int16(port),
			ServiceName: s.Name(),
		}
		trace.Record(zipkin.ServerRecvAnnotation(time.Now()))
		trace.RecordBinary(zipkin.NewStringAnnotation("http.uri", r.Request.RequestURI))

		err := s.Do(withTrace(ctx, trace))
		if si, ok := r.Writer.(httpservice.StatusCodeInterceptor); ok {
			trace.RecordBinary(zipkin.NewStringAnnotation("http.responsecode", strconv.Itoa(si.StatusCode())))
		}
		trace.Record(zipkin.ServerSendAnnotation(time.Now()))

		return err
	})
}

func NewClientTracingFilter(cs []zipkin.SpanCollector) saola.Filter {
	return saola.FuncFilter(func(ctx context.Context, s saola.Service) error {
		r := httpservice.GetClientRequest(ctx)
		parent, _ := GetTrace(ctx)
		trace := parent.Child(r.Request.Method)

		for h, v := range trace.HTTPHeader() {
			r.Request.Header.Set(h, v[0])
		}

		trace.Record(zipkin.ClientSendAnnotation(time.Now()))
		trace.RecordBinary(zipkin.NewStringAnnotation("http.uri", r.Request.URL.String()))

		err := s.Do(withTrace(ctx, trace))
		if err == nil {
			trace.RecordBinary(zipkin.NewStringAnnotation("http.responsecode", strconv.Itoa(r.Response.StatusCode)))
		}
		trace.Record(zipkin.ClientRecvAnnotation(time.Now()))

		return err
	})
}
