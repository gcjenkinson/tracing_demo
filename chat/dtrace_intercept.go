package chat 

import (
	"context"
	"io"
	stdlog "log"
	"reflect"
	"strings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	jaeger "github.com/uber/jaeger-client-go"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/jen20/go-usdt"
)

const provider = "opentracing"
const module = "jaeger"
const function = "span"
const startName = "start"
const clearName = "clear"

type DtraceObserver struct {

	provider *usdt.Provider
	startSpanProbe *usdt.Probe
	finishSpanProbe *usdt.Probe
}

type DtraceSpanObserver struct {

	// Span context passed as arguments to the DTrace USDT probes
	spanContext string
	// DTrace USDT probe fire in OnFinish
	finishSpanProbe *usdt.Probe
}

func NewDtraceObserver() (*DtraceObserver, io.Closer, error) {


	provider, err := usdt.NewProvider(provider, module)
	if err != nil {

		stdlog.Fatalf("NewProvider: %s", err)
		return nil, nil, err
	}

	startProbe, err := usdt.NewProbe(function, startName, reflect.String, reflect.String)
	if err != nil {

		stdlog.Fatalf("NewProbe: %s", err)
		return nil, nil, err
	}

	err = provider.AddProbe(startProbe)
	if err != nil {

		stdlog.Fatalf("AddProbe: %s", err)
		return nil, nil, err
	}

	finishProbe, err := usdt.NewProbe(function, clearName, reflect.String, reflect.String)
	if err != nil {

		stdlog.Fatalf("NewProbe: %s", err)
		return nil, nil, err
	}

	err = provider.AddProbe(finishProbe)
	if err != nil {

		stdlog.Fatalf("AddProbe: %s", err)
		return nil, nil, err
	}

	err = provider.Enable()
	if err != nil {

		stdlog.Fatalf("Enable: %s", err)
		return nil, nil, err
	}

	do := &DtraceObserver{
		provider: provider,
		startSpanProbe: startProbe,
		finishSpanProbe: finishProbe,
	}

	return do, do, nil
}

func (do *DtraceObserver) Close() error {

	do.provider.Close()
	return nil	
}

func (do *DtraceObserver) OnStartSpan (sp opentracing.Span, operationName string, options opentracing.StartSpanOptions) (jaeger.ContribSpanObserver, bool) {

	if sc, ok := sp.Context().(jaeger.SpanContext); ok {

		stdlog.Printf("Start SpanContext: %s", sc.String())
		do.startSpanProbe.Fire("SpanContext", sc.String())

		t := &DtraceSpanObserver{
			spanContext: sc.String(),
			finishSpanProbe: do.finishSpanProbe,
		}

		return t, true
	}

	return nil, false
}

func (dso *DtraceSpanObserver) OnSetOperationName (operationName string) {

}

func (dso *DtraceSpanObserver) OnSetTag(key string, value interface{}) {

}

func (dso *DtraceSpanObserver) OnFinish (options opentracing.FinishOptions) {

	stdlog.Printf("Finish SpanContext: %s", dso.spanContext)
	dso.finishSpanProbe.Fire("SpanContext", dso.spanContext)
}

func DTraceServerInterceptor(tracer opentracing.Tracer) grpc.UnaryServerInterceptor {

	provider, err := usdt.NewProvider(provider, module)
	if err != nil {

		stdlog.Fatalf("NewProvider: %s", err)
	}
	defer provider.Close()

	setProbe, err := usdt.NewProbe(function, startName, reflect.String, reflect.String)
	if err != nil {
		stdlog.Fatalf("NewProbe: %s", err)
	}

	err = provider.AddProbe(setProbe)
	if err != nil {
		stdlog.Fatalf("AddProbe: %s", err)
	}

	clrProbe, err := usdt.NewProbe(function, clearName, reflect.String, reflect.String)
	if err != nil {
		stdlog.Fatalf("NewProbe: %s", err)
	}

	err = provider.AddProbe(clrProbe)
	if err != nil {
		stdlog.Fatalf("AddProbe: %s", err)
	}

	err = provider.Enable()
	if err != nil {
		stdlog.Fatalf("Enable: %s", err)
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,

	) (resp interface{}, err error) {

		spanContext, err := extractSpanContext(ctx, tracer)
		if err != nil && err != opentracing.ErrSpanContextNotFound {
			// TODO: establish some sort of error reporting mechanism here. We
			// don't know where to put such an error and must rely on Tracer
			// implementations to do something appropriate for the time being.
		}

		if sc, ok := spanContext.(jaeger.SpanContext); ok {

			setProbe.Fire("SpanContext", sc.String())

			stdlog.Printf("SpanContext: %s", sc.String())

			clrProbe.Fire("SpanContext", sc.String())
		}

		return resp, err
	}
}

func extractSpanContext(ctx context.Context, tracer opentracing.Tracer) (opentracing.SpanContext, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	return tracer.Extract(opentracing.TextMap, metadataReaderWriter{md})
}

// metadataReaderWriter satisfies both the opentracing.TextMapReader and
// opentracing.TextMapWriter interfaces.
type metadataReaderWriter struct {
	metadata.MD
}

func (w metadataReaderWriter) Set(key, val string) {
	// The GRPC HPACK implementation rejects any uppercase keys here.
	//
	// As such, since the HTTP_HEADERS format is case-insensitive anyway, we
	// blindly lowercase the key (which is guaranteed to work in the
	// Inject/Extract sense per the OpenTracing spec).
	key = strings.ToLower(key)
	w.MD[key] = append(w.MD[key], val)
}

func (w metadataReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vals := range w.MD {
		for _, v := range vals {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}

	return nil
}
