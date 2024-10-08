package tracerprovider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	ProcessorBatch  = "batch"
	ProcessorSimple = "simple"

	ErrInvalidSamplePercentage = errors.New("invalid sample percentage")
	ErrInvalidProcessor        = errors.New("invalid processor")
)

type TracerProvider struct {
	provider         *trace.TracerProvider
	serviceName      attribute.KeyValue
	resource         *resource.Resource
	processorName    string
	batchProcessor   trace.SpanProcessor
	collectorHost    string
	collectorPort    int
	flushDuration    time.Duration
	samplePercentage float64
	credentials      credentials.TransportCredentials
	stopped          chan struct{}
	opts             []Opt
}

// New creates tracer provider module with given options.
func New(opts ...Opt) *TracerProvider {
	return &TracerProvider{opts: opts}
}

func (tp *TracerProvider) Init() error {
	tp.stopped = make(chan struct{})
	tp.flushDuration = 2 * time.Second

	for _, opt := range tp.opts {
		if err := opt(tp); err != nil {
			return fmt.Errorf("%s option error: %w", tp.Name(), err)
		}
	}

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			tp.serviceName,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}
	tp.resource = res

	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", tp.collectorHost, tp.collectorPort),
		grpc.WithTransportCredentials(tp.credentials),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	switch tp.processorName {
	case ProcessorBatch:
		tp.batchProcessor = trace.NewBatchSpanProcessor(exporter)
	case ProcessorSimple:
		tp.batchProcessor = trace.NewSimpleSpanProcessor(exporter)
		slog.Warn("using simple span processor, this should NOT be used in production")
	}

	tp.provider = trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(tp.samplePercentage)),
		trace.WithResource(tp.resource),
		trace.WithSpanProcessor(tp.batchProcessor),
	)

	return nil
}

func (tp *TracerProvider) Run() error {
	otel.SetTracerProvider(tp.provider)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	<-tp.stopped

	return nil
}

func (tp *TracerProvider) Stop() error {
	close(tp.stopped)
	ctx, cancel := context.WithTimeout(context.Background(), tp.flushDuration)
	defer cancel()
	return tp.provider.Shutdown(ctx)
}

func (tp *TracerProvider) Name() string {
	return "otel.TracerProvider"
}

type Opt func(*TracerProvider) error

func WithSamplePercentage(percentage int) Opt {
	return func(tp *TracerProvider) error {
		if percentage > 100 || percentage < 0 {
			return ErrInvalidSamplePercentage
		}
		tp.samplePercentage = float64(tp.samplePercentage) / 100
		return nil
	}
}

func WithCollector(host string, port int, credentials credentials.TransportCredentials) Opt {
	return func(tp *TracerProvider) error {
		tp.collectorHost = host
		tp.collectorPort = port
		tp.credentials = credentials
		return nil
	}
}

func WithServiceName(serviceName string) Opt {
	return func(tp *TracerProvider) error {
		tp.serviceName = semconv.ServiceNameKey.String(serviceName)
		return nil
	}
}

func WithProcessor(processorName string) Opt {
	return func(tp *TracerProvider) error {
		switch processorName {
		case ProcessorBatch, ProcessorSimple:
			tp.processorName = processorName
		default:
			return ErrInvalidProcessor
		}
		return nil
	}
}

// WithFlushDuration sets timeout for flushing spans.
func WithFlushDuration(d time.Duration) Opt {
	return func(tp *TracerProvider) error {
		tp.flushDuration = d
		return nil
	}
}
