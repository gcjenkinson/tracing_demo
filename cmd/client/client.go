package client 

import (
	"log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/gcjenkinson/tracing_demo/chat"
)

func main() {

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}

	cfg.ServiceName = "this-will-be-the-service-name"
	log.Printf("Service name: %s", cfg.ServiceName);
	jLogger := jaegerlog.StdLogger
	cfg.Reporter.LogSpans = true
	cfg.Sampler.Type = "const" 
	cfg.Sampler.Param = 1

	tracer, closer, err:= cfg.NewTracer(jaegercfg.Logger(jLogger))
	if err != nil {

		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()
	
	sp := opentracing.StartSpan("operation_name")
	ext.SamplingPriority.Set(sp, 1)
        sp.Finish()

	opentracing.SetGlobalTracer(tracer)
	if err != nil {

		log.Printf("Could not initialize Jaeger tracer: %s", err.Error())
		return
	}

	conn , err := grpc.Dial(":9000", grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
        		otgrpc.OpenTracingClientInterceptor(tracer)),
    		grpc.WithStreamInterceptor(
        		otgrpc.OpenTracingStreamClientInterceptor(tracer)))
	if err != nil {

		log.Fatalf("did not connect: %s", err)
	}

	defer conn.Close()

	c := chat.NewChatServiceClient(conn)

	response, err := c.SayHello(context.Background(), &chat.Message{Body: "Hello from Client!"})
	if err != nil {

		log.Fatalf("Error when calling SayHello: %s", err)
	}

	log.Printf("Response from server: %s", response.Body)
}
