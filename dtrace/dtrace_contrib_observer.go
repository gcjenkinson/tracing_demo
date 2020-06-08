package dtrace 

import (
	"io"
	stdlog "log"
	"reflect"
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
