package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"syscall"
	"github.com/gcjenkinson/tracing_demo/chat"
	"google.golang.org/grpc"
	"golang.org/x/sync/errgroup"
	opentracing "github.com/opentracing/opentracing-go"
	ext "github.com/opentracing/opentracing-go/ext"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
)

const serviceName = "demosrv"

var (
	version = "0.1.0"
	grpcServer *grpc.Server
)

func main() {
	srvLogger := log.WithFields(log.Fields{"service": serviceName, "version": version})

	srvLogger.Info("gRPC tracing demo!")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	g, ctx := errgroup.WithContext(ctx)

	/* Create the Jaeger tracer from the env settings */
	cfg, err := jaegercfg.FromEnv()
	if err != nil {

		// parsing errors might happen here, such as when we get a string where we expect a number
		srvLogger.WithFields(log.Fields{"error": err.Error}).Error("Could not parse Jaeger env vars")
		return
	}

	jLogger := jaegerlog.StdLogger

	cfg.ServiceName = serviceName;
	cfg.Reporter.LogSpans = true
	cfg.Sampler.Type = "const" 
	cfg.Sampler.Param = 1
		
	dos, dosCloser, err := chat.NewDtraceObserver()
	defer dosCloser.Close()

	tracer, closer, err:= cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.ContribObserver(dos))
	if err != nil {

		srvLogger.WithFields(log.Fields{"error": err.Error}).Error("Could not initialize Jaeger tracer")
		return
	}
	defer closer.Close()

	sp := opentracing.StartSpan("operation_name")
	ext.SamplingPriority.Set(sp, 1)
        sp.Finish()

	g.Go(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 9000))
		if err != nil {

			srvLogger.WithFields(log.Fields{"error": err}).Error("Fialed to listen")
		}

		s := chat.Server{}

		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(
				otgrpc.OpenTracingServerInterceptor(tracer)),
			grpc.StreamInterceptor(
				otgrpc.OpenTracingStreamServerInterceptor(tracer)))

		chat.RegisterChatServiceServer(grpcServer, &s)
		
		return grpcServer.Serve(lis)
	})

	select {
	case <-interrupt:
		break
	case <-ctx.Done():
		break
	}

	cancel()

	if grpcServer != nil {

		grpcServer.GracefulStop()
	}

	srvLogger.Info("Shutdown...");
}
