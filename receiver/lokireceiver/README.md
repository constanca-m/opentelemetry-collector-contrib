# Loki Receiver

<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [alpha]: logs   |
| Distributions | [contrib] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Areceiver%2Floki%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Areceiver%2Floki) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Areceiver%2Floki%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Areceiver%2Floki) |
| Code coverage | [![codecov](https://codecov.io/github/open-telemetry/opentelemetry-collector-contrib/graph/main/badge.svg?component=receiver_loki)](https://app.codecov.io/gh/open-telemetry/opentelemetry-collector-contrib/tree/main/?components%5B0%5D=receiver_loki&displayType=list) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    | [@mar4uk](https://www.github.com/mar4uk) |

[alpha]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#alpha
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
<!-- end autogenerated section -->

The Loki receiver implements the [Loki push api](https://grafana.com/docs/loki/latest/clients/promtail/#loki-push-api) as specified in [Loki HTTP API: Ingest logs](https://grafana.com/docs/loki/latest/reference/loki-http-api/#ingest-logs).
It allows Promtail instances to specify the open telemetry collector as their lokiAddress.

This receiver runs HTTP and GRPC servers to ingest log entries in Loki format.

## Getting Started

The settings are:

- `endpoint` (required, default = localhost:3500 for HTTP protocol, localhost:3600 gRPC protocol): host:port to which the receiver is going to receive data. See our [security best practices doc](https://opentelemetry.io/docs/security/config-best-practices/#protect-against-denial-of-service-attacks) to understand how to set the endpoint in different environments.
- `use_incoming_timestamp` (optional, default = false) if set `true` the timestamp from Loki log entry is used

Example:
```yaml
receivers:
  loki:
    protocols:
      http:
        endpoint: 0.0.0.0:3500
      grpc:
        endpoint: 0.0.0.0:3600
    use_incoming_timestamp: true
```

## Advanced Configuration

Several helper files are leveraged to provide additional capabilities automatically:

- [gRPC settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configgrpc/README.md) including CORS
- [HTTP settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md)
- [TLS and mTLS settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md)
- [Auth settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configauth/README.md)
