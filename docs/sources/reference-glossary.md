---
title: "Reference: Glossary"
description: ""
weight: 10000
---

# Reference: Glossary

## Blocks storage

Blocks storage is the Mimir storage engine based on the Prometheus TSDB.
Grafana Mimir stores blocks in object stores such as AWS S3, Google Cloud Storage (GCS), Azure blob storage, or OpenStack Object Storage (Swift).
For the full list of supported backends and more information, refer to [Blocks storage]({{<relref "./blocks-storage/_index.md" >}})

## Chunk

A chunk is an object containing encoded timestamp-value pairs for one series.

## Churn

Churn is the frequency at which series become idle.

A series becomes idle once it's no longer exported by the monitored targets.
Typically, series become idle when a monitored target process or node gets terminated.

## Flushing

Flushing is the operation run by ingesters to offload time series from memory and store them in the long-term storage.

## Gossip

Gossip is a protocol by which components coordinate without the need for a centralized [key-value store]({{<relref "#key-value-store" >}}).

## HA tracker

The HA tracker is a feature of the Grafana Mimir distributor.
It deduplicates time series received from two or more Prometheus servers that are configured to scrape the same targets.
To configure HA tracking, refer to [Configure HA deduplication]({{<relref "./operating-grafana-mimir/configure-ha-deduplication.md" >}}).

## Hash ring

The hash ring is a distributed data structure used by Grafana Mimir for sharding, replication, and service discovery.
Components use a [key-value store]({{<relref "#key-value-store" >}}) or [gossip]({{<relref "#gossip" >}}) to share the hash ring data structure.
For more information, refer to the [About the hash ring]({{<relref "./architecture/about-the-hash-ring.md" >}}).

## Key-value store

A key-value store is a database that associates keys with values.
To understand how Grafana Mimir uses key-value stores, refer to [About the key-value store]({{<relref "./architecture/about-the-key-value-store.md" >}}).

## Memberlist

Memberlist manages cluster membership and member failure detection using [gossip]({{<relref "#gossip" >}}).

## Org

Refer to [Tenant]({{<relref "#tenant" >}}).

## Ring

Refer to [Hash ring]({{<relref "#hash-ring" >}}).

## Sample

A sample is a single timestamped value in a time series.

Given the series `node_cpu_seconds_total{instance="10.0.0.1",mode="system"}` its stream of samples may look like:

```
# Display format: <value> @<timestamp>
11775 @1603812134
11790 @1603812149
11805 @1603812164
11819 @1603812179
11834 @1603812194
```

## Series

A series is a single stream of [samples]({{<relref "#sample" >}}) belonging to the same metric, with the same set of label key-value pairs.

Given a single metric `node_cpu_seconds_total` you may have multiple series, each one uniquely identified by the combination of metric name and unique label key-value pairs:

```
node_cpu_seconds_total{instance="10.0.0.1",mode="system"}
node_cpu_seconds_total{instance="10.0.0.1",mode="user"}
node_cpu_seconds_total{instance="10.0.0.2",mode="system"}
node_cpu_seconds_total{instance="10.0.0.2",mode="user"}
```

## Tenant

A tenant is the owner of a set of series written to and queried from Grafana Mimir.
Grafana Mimir isolates series and alerts belonging to different tenants.
To understand how Grafana Mimir authenticates tenants, refer to [About authentication and authorization]({{<relref "./about-authentication-and-authorization.md" >}}).

## Time series

Refer to [Series]({{<relref "#series" >}}).

## User

Refer to [Tenant]({{<relref "#tenant" >}}).

## Write-ahead log (WAL)

The write-ahead Log (WAL) is an append only log stored on disk by ingesters to recover their in-memory state after the process gets restarted.
For more information, refer to [The write path]({{<relref "./architecture/_index.md#the-write-path" >}}).