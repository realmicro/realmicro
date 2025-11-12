package common

import (
	"context"
	"net"

	"github.com/realmicro/realmicro/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func GetLocalIp() string {
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

func NewTraceProvider(ctx context.Context, endpoint, token, serverName string) (*trace.TracerProvider, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint), // <endpoint>替换为上报地址
		otlptracegrpc.WithInsecure(),
	}
	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		logger.Fatal(err)
	}

	r, err := resource.New(ctx, []resource.Option{
		resource.WithAttributes(
			attribute.KeyValue{Key: "token", Value: attribute.StringValue(token)},             // <token>替换为业务系统Token
			attribute.KeyValue{Key: "service.name", Value: attribute.StringValue(serverName)}, // <serviceName>替换为应用名
			attribute.KeyValue{Key: "host.name", Value: attribute.StringValue(GetLocalIp())},  // <hostName>替换为IP地址
		),
	}...)
	if err != nil {
		logger.Fatal(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(r),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}
