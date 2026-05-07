## Kafka Group Balancer extension

The `kafkagroupbalancer` package defines the `GroupBalancer` extension interface.
Implement it in a custom Collector distribution to supply a custom Kafka
consumer-group partition-assignment strategy.

The extension is selected by setting `group_rebalance_strategy` in `kafkareceiver` to the extension's component ID instead of one of the built-in
values (`range`, `roundrobin`, `sticky`, `cooperative-sticky`).

## Example configuration

Set `group_rebalance_strategy` to the component ID of any registered `GroupBalancer` extension:

```yaml
extensions:
  my_custom_balancer:

receivers:
  kafka:
    group_rebalance_strategy: my_custom_balancer
    traces:
      topics: ["^signals\\.traces\\..*"]
    metrics:
      topics: ["^signals\\.metrics\\..*"]
    logs:
      topics: ["^signals\\.logs\\..*"]

service:
  extensions: [my_custom_balancer]
  pipelines:
    traces:
      receivers: [kafka]
    metrics:
      receivers: [kafka]
    logs:
      receivers: [kafka]
```
