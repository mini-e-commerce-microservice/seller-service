package infra

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/mini-e-commerce-microservice/product-service/generated/proto/secret_proto"
	"github.com/mini-e-commerce-microservice/product-service/internal/util/primitive"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"time"
)

func NewOtel(cred *secret_proto.Otel, tracerName string) primitive.CloseFn {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cred.Username, cred.Password)))
	traceCli := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithHeaders(map[string]string{
			"Authorization": authHeader,
		}),
		otlptracegrpc.WithEndpoint(cred.Endpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)

	traceExp, err := otlptrace.New(ctx, traceCli)
	if err != nil {
		panic(err)
	}

	traceProvide, closeFnTracer, err := start(traceExp, tracerName)

	if err != nil {
		log.Fatal().Err(err).Msgf("failed initializing the tracer provider")
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(traceProvide)

	log.Info().Msg("initialization opentelemetry successfully")

	return closeFnTracer
}

func start(exp trace.SpanExporter, name string) (*trace.TracerProvider, primitive.CloseFn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(name),
		),
	)
	if err != nil {
		err = fmt.Errorf("failed to created resource: %w", err)
		return nil, nil, err
	}

	bsp := trace.NewBatchSpanProcessor(exp)

	provider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)

	closeFn := func(ctx context.Context) (err error) {
		log.Info().Msg("starting shutdown export and provider")
		ctxClosure, cancelClosure := context.WithTimeout(ctx, 5*time.Second)
		defer cancelClosure()

		if err = exp.Shutdown(ctxClosure); err != nil {
			return err
		}

		if err = provider.Shutdown(ctxClosure); err != nil {
			return err
		}

		log.Info().Msg("shutdown export and provider successfully")

		return
	}

	return provider, closeFn, err
}
