package tracerprovider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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
	ErrInvalidToken            = errors.New("invalid token")
)

type ExporterHTTP struct {
	endpoint string
	token    string
	insecure bool
}

type ExporterGRPC struct {
	endpoint    string
	credentials credentials.TransportCredentials
}

type TracerProvider struct {
	provider         *trace.TracerProvider
	ctx              context.Context //nolint:containedctx
	serviceName      attribute.KeyValue
	environment      attribute.KeyValue
	resource         *resource.Resource
	processorName    string
	batchProcessor   trace.SpanProcessor
	grpc             ExporterGRPC
	http             ExporterHTTP
	samplePercentage float64
	stopped          chan struct{}
	opts             []Opt
}

// New creates tracer provider module with given options.
func New(opts ...Opt) *TracerProvider {
	return &TracerProvider{opts: opts}
}

func (tp *TracerProvider) Init() error {
	tp.stopped = make(chan struct{})

	for _, opt := range tp.opts {
		if err := opt(tp); err != nil {
			return fmt.Errorf("%s option error: %w", tp.Name(), err)
		}
	}

	res, err := resource.New(tp.ctx,
		resource.WithAttributes(
			tp.serviceName,
			tp.environment,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}
	tp.resource = res

	var exporter *otlptrace.Exporter
	// fallback to gRPC exporter if HTTP exporter is not configured
	if tp.http.endpoint != "" {
		exporter, err = tp.createHTTPExporter()
	} else {
		exporter, err = tp.createGRPCExporter()
	}
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

func (tp *TracerProvider) createHTTPExporter() (*otlptrace.Exporter, error) {
	slog.Debug("using http exporter",
		slog.String("endpoint", tp.http.endpoint),
		slog.Bool("insecure", tp.http.insecure),
	)
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(tp.http.endpoint),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	}
	if tp.http.insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	if tp.http.token != "" {
		opts = append(opts, otlptracehttp.WithHeaders(map[string]string{
			"Authorization": tp.http.token,
		}))
	}
	exporter, err := otlptracehttp.New(tp.ctx, opts...)
	return exporter, err
}

func (tp *TracerProvider) createGRPCExporter() (*otlptrace.Exporter, error) {
	slog.Debug("using grpc exporter",
		slog.String("endpoint", tp.grpc.endpoint),
	)
	conn, err := grpc.NewClient(
		tp.grpc.endpoint,
		grpc.WithTransportCredentials(tp.grpc.credentials),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc connection to collector: %w", err)
	}
	exporter, err := otlptracegrpc.New(tp.ctx, otlptracegrpc.WithGRPCConn(conn))
	return exporter, err
}

func (tp *TracerProvider) Run() error {
	otel.SetTracerProvider(tp.provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	<-tp.stopped

	return nil
}

func (tp *TracerProvider) Stop() error {
	close(tp.stopped)
	err := tp.provider.ForceFlush(tp.ctx)
	if err != nil {
		slog.Warn("failed to export remaining spans")
	}
	return tp.provider.Shutdown(tp.ctx)
}

func (tp *TracerProvider) Name() string {
	return "otel.TracerProvider"
}

type Opt func(*TracerProvider) error

// WithSamplePercentage sets the percentage of spans to sample. Allowed values are 0-100.
func WithSamplePercentage(percentage int) Opt {
	return func(tp *TracerProvider) error {
		if percentage > 100 || percentage < 0 {
			return ErrInvalidSamplePercentage
		}
		tp.samplePercentage = float64(percentage) / 100
		return nil
	}
}

// WithCollector is deprecated. Use WithGRPCExporter instead.
func WithCollector(host string, port int, credentials credentials.TransportCredentials) Opt {
	slog.Warn("WithCollector is deprecated, use WithGRPCExporter instead")
	return WithGRPCExporter(fmt.Sprintf("%s:%d", host, port), credentials)
}

// WithGRPCExporter sets gRPC collector endpoint for trace exporter.
// Endpoint is "host:port" of the collector and you can set transport credentials to "insecure.NewCredentials()"
// if secure connection is not required.
func WithGRPCExporter(endpoint string, credentials credentials.TransportCredentials) Opt {
	return func(tp *TracerProvider) error {
		tp.grpc.endpoint = endpoint
		tp.grpc.credentials = credentials
		return nil
	}
}

// WithHTTPExporter sets HTTP exporter endpoint for trace exporter with an authorization token.
// Endpoint is "host:port" of to the collector and you can set token as "ApiKey <token>" or "Bearer <token>".
// Set token to empty if not needed. Insecure disables TLS.
func WithHTTPExporter(endpoint string, token string, insecure bool) Opt {
	return func(tp *TracerProvider) error {
		if token != "" && !(strings.HasPrefix(strings.ToLower(token), "apikey ") || strings.HasPrefix(strings.ToLower(token), "bearer ")) {
			return ErrInvalidToken
		}
		tp.http.endpoint = endpoint
		tp.http.token = token
		tp.http.insecure = insecure
		return nil
	}
}

// WithContext sets the context for the tracer provider.
func WithContext(ctx context.Context) Opt {
	return func(tp *TracerProvider) error {
		tp.ctx = ctx
		return nil
	}
}

// WithServiceName sets the service name attribute for the tracer provider.
func WithServiceName(serviceName string) Opt {
	return func(tp *TracerProvider) error {
		tp.serviceName = semconv.ServiceNameKey.String(serviceName)
		return nil
	}
}

// WithEnvironment sets the deployment environment name attribute for the tracer provider.
func WithEnvironment(environment string) Opt {
	return func(tp *TracerProvider) error {
		if environment != "" {
			tp.environment = semconv.DeploymentEnvironment(environment)
		}
		return nil
	}
}

// WithProcessor sets the span processor for the tracer provider. Allowed values are "batch" and "simple",
// but "simple" should not be used in production.
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
