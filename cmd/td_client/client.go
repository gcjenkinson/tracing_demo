package main 

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/gcjenkinson/tracing_demo/chat"
)

func main() {
	target := flag.String("target", "localhost:9000", "Target address")
	srvName:= flag.String("service", "TracingDemo", "Service name")
	flag.Parse()

	logger := log.WithFields(log.Fields{"service": srvName})

	cfg, err := jaegercfg.FromEnv()
	if err != nil {

		// parsing errors might happen here, such as when we get a string where we expect a number
		logger.WithFields(log.Fields{"error": err.Error}).Error("Could not parse Jaeger env vars")
		return
	}

	cfg.ServiceName = *srvName
	jLogger := jaegerlog.StdLogger
	cfg.Reporter.LogSpans = true
	cfg.Sampler.Type = "const" 
	cfg.Sampler.Param = 1

	tracer, closer, err:= cfg.NewTracer(jaegercfg.Logger(jLogger))
	if err != nil {

		logger.WithFields(log.Fields{"error": err.Error}).Error("Could not initialize jaeger tracer")
		return
	}
	defer closer.Close()
	
	opentracing.SetGlobalTracer(tracer)
	if err != nil {

		logger.WithFields(log.Fields{"error": err.Error}).Error("Could not initialize Jaeger tracer")
		return
	}
		
	logger.WithFields(log.Fields{"target": *target}).Info("Connecting to service")

	conn , err := grpc.Dial(*target, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
        		otgrpc.OpenTracingClientInterceptor(tracer)),
    		grpc.WithStreamInterceptor(
        		otgrpc.OpenTracingStreamClientInterceptor(tracer)))
	if err != nil {

		logger.WithFields(log.Fields{"error": err.Error}).Error("Couldn't connect to service")
	}
	defer conn.Close()

	c := chat.NewChatServiceClient(conn)

	response, err := c.SayHello(context.Background(), &chat.Message{Body: "Hello from Client!"})
	if err != nil {

		logger.WithFields(log.Fields{"error": err.Error}).Error("Error when calling SayHello")
	} else {

		logger.WithFields(log.Fields{"response": response.Body}).Info("Response from serve")
	}
}
