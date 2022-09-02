---
title: "Migrate from single zone to zone aware replication with Helm"
menuTitle: "Migrate from single zone to zone aware replication with Helm"
description: "Learn how to migrate from having a single availability zone to full zone aware replication using the Grafana Mimir Helm chart"
weight: 10
---

# Migrate from single zone to zone aware replication with Helm

This document explains how to migrate stateful componens from single zone to zone aware replication with Helm. The three components in question are the [alertmanager]({{< relref "../operators-guide/architecture/components/alertmanager.md" >}}), the [store-gateway]({{< relref "../operators-guide/architecture/components/store-gateway.md" >}}) and the [ingester]({{< relref "../operators-guide/architecture/components/ingester.md" >}}).

The migration path of the alertmanager and store-gatway is straight forward, however migrating ingesters is more complicated.

This document is applicable to both Grafana Mimir and Grafana Enterprise Metrics.

## Prerequisite

1. Zone aware replication is turned off for the component in question

1. The installation is already upgraded to `mimir-distributed` Helm chart version 4.0.0 or later.

## Migrate alertmanager to zone aware replication

### Configure zone aware replication for alertmanagers

Follow the TBD operations-guide to set a desired configuration.

## Migrate store-gateways to zone aware replication

### Configure zone aware replication for store-gateways

Follow the TBD operations-guide to set a desired configuration.

## Migrate ingesters to zone aware replication

### Configure zone aware replication for ingesters

This section is about planning and configuring the availability zones defined in the array value `ingester.zone_aware_replication.zones`.

> **Note**: as this value is an array, you must copy and modify it to make changes to it, there is no way to overwrite just parts of the array!

There are two use cases in general:

1. Speeding up rollout of ingesters. In this case the default value for `ingester.zone_aware_replication.zones` can be used. The default value defines 3 "virtual" zones and sets affinity rules so that ingesters from different zones do not mix, but it allows multiple ingesters of the same zone on the same node.
1. Geographical redundancy. In this case you need to set a suitable [nodeSelector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/) value to choose where the pods of each zone are to be placed. For example:
   ```yaml
   ingester:
     zone_aware_replication:
       enabled: false # Do not turn on zone awareness without migration because of data loss
       zones:
         - name: zone-a
           nodeSelector:
             topology.kubernetes.io/zone: zone-a
           affinity:
             podAntiAffinity:
               requiredDuringSchedulingIgnoredDuringExecution:
                 - labelSelector:
                     matchExpressions:
                       - key: rollout-group
                         operator: In
                         values:
                           - store-gateway
                       - key: app.kubernetes.io/component
                         operator: NotIn
                         values:
                           - store-gateway-zone-a
                   topologyKey: "kubernetes.io/hostname"
         - name: zone-b
           nodeSelector:
             topology.kubernetes.io/zone: zone-b
           affinity:
             podAntiAffinity:
               requiredDuringSchedulingIgnoredDuringExecution:
                 - labelSelector:
                     matchExpressions:
                       - key: rollout-group
                         operator: In
                         values:
                           - store-gateway
                       - key: app.kubernetes.io/component
                         operator: NotIn
                         values:
                           - store-gateway-zone-b
                   topologyKey: "kubernetes.io/hostname"
         - name: zone-c
           nodeSelector:
             topology.kubernetes.io/zone: zone-c
           affinity:
             podAntiAffinity:
               requiredDuringSchedulingIgnoredDuringExecution:
                 - labelSelector:
                     matchExpressions:
                       - key: rollout-group
                         operator: In
                         values:
                           - store-gateway
                       - key: app.kubernetes.io/component
                         operator: NotIn
                         values:
                           - store-gateway-zone-c
                   topologyKey: "kubernetes.io/hostname"
   ```

Set the chosen configuration in your custom values (e.g. `custom.yaml`).

> **Note**: Do not turn on zone awareness without migration because of data loss, make sure that `ingester.zone_aware_replication.enabled` is set to false.

```yaml
ingester:
  zone_aware_replication:
    enabled: false
```

> **Note**: The number of ingester pods that will be started is derived from `ingester.replicas`. Each zone will start `ingester.replicas / number of zones` pods, rounded up to the nearest integer value. For example if you have 3 zones, then `ingester.replicas=3` will yield 1 ingester per zone, but `ingester.replicas=4` will yield 2 per zone, 6 in total.

### Decide which migration path to take

There are two ways to do the migration:

1. With downtime. In this procedure ingress is stopped to the cluster while ingesters are migrated. This is the quicker and simpler way.
1. Without downtime. This is a multi step process which requires additional hardware resources as the old and new ingesters run in parallel for some time. This is a complex migration.

### Migrate ingesters with downtime

1. Create a new empty YAML file called `migrate.yaml`

1. Enable flushing data from ingesters to storage on shutdown

   Copy the following into the `migrate.yaml`:

   ```yaml
   mimir:
     structuredConfig:
       blocks_storage:
         flush_blocks_on_shutdown: true
       ingester:
         ring:
           unregister_on_shutdown: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait for all ingesters to restart.

1. Turn off writes to the installation

   Copy the following into the `migrate.yaml`:

   ```yaml
   mimir:
     structuredConfig:
       blocks_storage:
         flush_blocks_on_shutdown: true
       ingester:
         ring:
           unregister_on_shutdown: true

   nginx:
     replicas: 0
   gateway:
     replicas: 0
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until there is no nginx or gateway running.

1. Scale the current ingesters to 0

   Copy the following into the `migrate.yaml`:

   ```yaml
   mimir:
     structuredConfig:
       blocks_storage:
         flush_blocks_on_shutdown: true
       ingester:
         ring:
           unregister_on_shutdown: true

   nginx:
     replicas: 0
   gateway:
     replicas: 0

   ingester:
     replicas: 0
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until no ingesters are running.

1. Start the new zone aware ingesters

   > **Note**: this step assumes that you set up your zones according to [Configure zone aware replication for ingesters](#configure-zone-aware-replication-for-ingesters)

   Copy the following into the `migrate.yaml`:

   ```yaml
   nginx:
     replicas: 0
   gateway:
     replicas: 0

   ingester:
     zone_aware_replication:
       enabled: true

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until all requested ingesters are running.

1. Enable writes to the installation

   **Merge** the following values into your regular `custom.yaml` file:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true

   rollout_operator:
     enabled: true
   ```

   > **Note**: these values are actually the default, which means that removing the values `ingester.zone_aware_replication.enabled` and `rollout_operator.enabled` from your `custom.yaml` is also a valid step.

1. Upgrade the installation with the `helm` command using only your regular command line flags.
