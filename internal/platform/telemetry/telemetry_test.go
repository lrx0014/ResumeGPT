package telemetry_test

import (
	"context"
	"testing"

	"github.com/lrx0014/ResumeGPT/internal/platform/telemetry"
	"go.opentelemetry.io/otel"
)

func TestSetupCreatesTraceContextWithoutExporter(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
	shutdown, err := telemetry.Setup(context.Background(), "resumegpt-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})

	_, span := otel.Tracer("test").Start(context.Background(), "operation")
	defer span.End()
	if !span.SpanContext().TraceID().IsValid() {
		t.Fatal("expected a valid trace ID")
	}
}
