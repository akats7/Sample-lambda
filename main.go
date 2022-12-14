package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	otelresource "go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

func NewResource(resourceName string) *otelresource.Resource {
	resource := otelresource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(resourceName),
	)
	return resource
}

func newTraceExporter(ctx context.Context) (*otlptrace.Exporter, error) {

	client := otlptracegrpc.NewClient(otlptracegrpc.WithEndpoint("otlp.nr-data.net:443"),
		otlptracegrpc.WithHeaders(map[string]string{"api-key": ""}))

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatalf("creating OTLP exporter %v, err")
	}
	fmt.Println("created")
	return exporter, err
}

func newTraceProvider(exp trace.SpanExporter, resource *otelresource.Resource) *trace.TracerProvider {

	return trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource),
	)
}

func InstallTracePipeline(ctx context.Context, resource *otelresource.Resource) func() {
	exp, err := newTraceExporter(ctx)

	if err != nil {
		log.Fatalf("failed to initialize %v", err)
	}

	tp := newTraceProvider(exp, resource)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("Error stopping %v", err)
		}
	}
}

func hello() {
	ctx := context.Background()
	resource := NewResource("sample-lambda")
	traceShutdown := InstallTracePipeline(ctx, resource)
	defer traceShutdown()
	fmt.Println("hello from lambda2")
	_, span := otel.Tracer("SampleGo").Start(ctx, "Hello")
	defer span.End()
}

func main() {
	// ctx := context.Background()
	// resource := NewResource("sample-lambda")
	// traceShutdown := InstallTracePipeline(ctx, resource)
	// defer traceShutdown()
	lambda.Start(hello)
}
