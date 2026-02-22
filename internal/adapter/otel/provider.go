package otel

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config holds OpenTelemetry provider configuration.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string // "development" or "production"
	Exporter       string // "stdout" or "otlp"
	Insecure       bool   // use HTTP instead of HTTPS for OTLP
}

// ConfigFromEnv builds Config from environment variables with sensible defaults.
func ConfigFromEnv() Config {
	env := envOrDefault("OTEL_ENVIRONMENT", "development")
	return Config{
		ServiceName:    envOrDefault("OTEL_SERVICE_NAME", "tenantiq"),
		ServiceVersion: envOrDefault("OTEL_SERVICE_VERSION", "0.1.0"),
		Environment:    env,
		Exporter:       envOrDefault("OTEL_EXPORTER", "stdout"),
		Insecure:       env == "development",
	}
}

// Providers holds initialized OTel providers and their shutdown function.
type Providers struct {
	Shutdown func(ctx context.Context) error
}

// Setup initializes TracerProvider and MeterProvider based on Config.
// It registers them globally and returns a Providers whose Shutdown must
// be called on application exit to flush pending telemetry.
func Setup(ctx context.Context, cfg Config) (*Providers, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating otel resource: %w", err)
	}

	tp, err := newTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("creating tracer provider: %w", err)
	}

	mp, err := newMeterProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("creating meter provider: %w", err)
	}

	// Register globally so any package can obtain a tracer via otel.Tracer("name").
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func(ctx context.Context) error {
		var errs []error
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
		if err := mp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
		if len(errs) > 0 {
			return fmt.Errorf("otel shutdown: %v", errs)
		}
		return nil
	}

	return &Providers{Shutdown: shutdown}, nil
}

func newTracerProvider(ctx context.Context, cfg Config, res *resource.Resource) (*trace.TracerProvider, error) {
	var exporter trace.SpanExporter
	var err error

	switch cfg.Exporter {
	case "otlp":
		var opts []otlptracehttp.Option
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	default:
		return nil, fmt.Errorf("unsupported exporter: %q (use \"stdout\" or \"otlp\")", cfg.Exporter)
	}

	if err != nil {
		return nil, err
	}

	return trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(exporter),
	), nil
}

func newMeterProvider(ctx context.Context, cfg Config, res *resource.Resource) (*metric.MeterProvider, error) {
	var exporter metric.Exporter
	var err error

	switch cfg.Exporter {
	case "otlp":
		var opts []otlpmetrichttp.Option
		if cfg.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
	case "stdout":
		exporter, err = stdoutmetric.New()
	default:
		return nil, fmt.Errorf("unsupported exporter: %q (use \"stdout\" or \"otlp\")", cfg.Exporter)
	}

	if err != nil {
		return nil, err
	}

	return metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter)),
	), nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
