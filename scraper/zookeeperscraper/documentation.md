[comment]: <> (Code generated by mdatagen. DO NOT EDIT.)

# zookeeper

## Default Metrics

The following metrics are emitted by default. Each of them can be disabled by applying the following configuration:

```yaml
metrics:
  <metric_name>:
    enabled: false
```

### zookeeper.connection.active

Number of active clients connected to a ZooKeeper server.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {connections} | Sum | Int | Cumulative | false |

### zookeeper.data_tree.ephemeral_node.count

Number of ephemeral nodes that a ZooKeeper server has in its data tree.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {nodes} | Sum | Int | Cumulative | false |

### zookeeper.data_tree.size

Size of data in bytes that a ZooKeeper server has in its data tree.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| By | Sum | Int | Cumulative | false |

### zookeeper.file_descriptor.limit

Maximum number of file descriptors that a ZooKeeper server can open.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {file_descriptors} | Gauge | Int |

### zookeeper.file_descriptor.open

Number of file descriptors that a ZooKeeper server has open.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {file_descriptors} | Sum | Int | Cumulative | false |

### zookeeper.follower.count

The number of followers. Only exposed by the leader.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {followers} | Sum | Int | Cumulative | false |

#### Attributes

| Name | Description | Values | Optional |
| ---- | ----------- | ------ | -------- |
| state | State of followers | Str: ``synced``, ``unsynced`` | false |

### zookeeper.fsync.exceeded_threshold.count

Number of times fsync duration has exceeded warning threshold.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {events} | Sum | Int | Cumulative | true |

### zookeeper.latency.avg

Average time in milliseconds for requests to be processed.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| ms | Gauge | Int |

### zookeeper.latency.max

Maximum time in milliseconds for requests to be processed.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| ms | Gauge | Int |

### zookeeper.latency.min

Minimum time in milliseconds for requests to be processed.

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| ms | Gauge | Int |

### zookeeper.packet.count

The number of ZooKeeper packets received or sent by a server.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {packets} | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values | Optional |
| ---- | ----------- | ------ | -------- |
| direction | State of a packet based on io direction. | Str: ``received``, ``sent`` | false |

### zookeeper.request.active

Number of currently executing requests.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {requests} | Sum | Int | Cumulative | false |

### zookeeper.ruok

Response from zookeeper ruok command

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| 1 | Gauge | Int |

### zookeeper.sync.pending

The number of pending syncs from the followers. Only exposed by the leader.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {syncs} | Sum | Int | Cumulative | false |

### zookeeper.watch.count

Number of watches placed on Z-Nodes on a ZooKeeper server.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {watches} | Sum | Int | Cumulative | false |

### zookeeper.znode.count

Number of z-nodes that a ZooKeeper server has in its data tree.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {znodes} | Sum | Int | Cumulative | false |

## Resource Attributes

| Name | Description | Values | Enabled |
| ---- | ----------- | ------ | ------- |
| server.state | State of the Zookeeper server (leader, standalone or follower). | Any Str | true |
| zk.version | Zookeeper version of the instance. | Any Str | true |
