# OpenTelemetry wrappers

OpenTelemetry wrappers propagate traces (spans) accross services.

## Usage

```go
service := realmicro.NewService(
    realmicro.Name("realmicro.srv.greeter"),
    realmicro.WrapClient(opentelemetry.NewClientWrapper()),
    realmicro.WrapHandler(opentelemetry.NewHandlerWrapper()),
    realmicro.WrapSubscriber(opentelemetry.NewSubscriberWrapper()),
)
```