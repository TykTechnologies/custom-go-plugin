# Jaeger | Open Telemetry with Tyk
Jaeger can demonstrate tracing via Open Telemetry. It has a [Dashboard](http://localhost:16686/) you can use to view traces.

It has been configured to use in-memory storage, so will not retain data once the container is restarted/removed.

- [Jaeger Dashboard](http://localhost:16686/)

## Setup
> [!IMPORTANT]
> Open Telemetry support is only available on Tyk Gateway versions `v5.2.0+`.

Run the `make` command:

```
make
```

## Usage 

To use Jaeger, open the [Jaeger Dashboard](http://localhost:16686/) in a browser. The *Search* page displays trace data based on filters:

- For *Service*, select `tyk-gateway` to see traces from the Tyk gateway, or select `jaeger-query` to see traces from the Jaeger application.
- The values for *Operation* change based on the *service*. Leave it on `all` to see everything.
- *Lookback* filters by time, by limiting displayed data to the selected time period. 

For more information please visit our official documentation on [How to instrument plugins with OpenTelemetry](https://deploy-preview-3184--tyk-docs.netlify.app/docs/nightly/product-stack/tyk-gateway/advanced-configurations/plugins/otel-plugins/).