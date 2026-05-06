package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitTracer(serviceName string, jaegerEndpoint string) (*sdktrace.TracerProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	conn, err := grpc.NewClient(jaegerEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to Jaeger: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		fmt.Errorf("failed to create gRPC connection to OTLP: %w", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)))
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)

	// Set up context propagation (e.g., W3C Trace Context)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C Trace Context
		propagation.Baggage{},      // Add more propagators if needed
	))

	return tp, nil
}
