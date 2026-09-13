package telemetry

import (
	"context"
	"errors"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

type Shutdown func(context.Context) error

func Setup(ctx context.Context, serviceName string) (Shutdown, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	defaultResource := resource.Default()
	serviceResource, err := resource.Merge(
		defaultResource,
		resource.NewWithAttributes(defaultResource.SchemaURL(), semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, err
	}
	providerOptions := []sdktrace.TracerProviderOption{sdktrace.WithResource(serviceResource)}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != "" {
		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, err
		}
		providerOptions = append(providerOptions, sdktrace.WithBatcher(exporter))
	}
	provider := sdktrace.NewTracerProvider(providerOptions...)
	otel.SetTracerProvider(provider)
	return func(shutdownCtx context.Context) error {
		return errors.Join(provider.ForceFlush(shutdownCtx), provider.Shutdown(shutdownCtx))
	}, nil
}
