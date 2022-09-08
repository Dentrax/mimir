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

This section is about planning and configuring the availability zones defined in the array value `store_gateway.zone_aware_replication.zones`.

> **Note**: as this value is an array, you must copy and modify it to make changes to it, there is no way to overwrite just parts of the array!

There are two use cases in general:

1. Speeding up rollout of store gateways. In this case the default value for `store_gateway.zone_aware_replication.zones` can be used. The default value defines 3 "virtual" zones and sets affinity rules so that store gateways from different zones do not mix, but it allows multiple store gateways of the same zone on the same node.
1. Geographical redundancy. In this case you need to set a suitable [nodeSelector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/) value to choose where the pods of each zone are to be placed. For example:
   ```yaml
   store_gateway:
     zone_aware_replication:
       enabled: false # Do not turn on zone awareness without migration because of potential query errors
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

> **Note**: Do not turn on zone awareness without migration because of potential query errors, make sure that `store_gateway.zone_aware_replication.enabled` is set to false.

```yaml
store_gateway:
  zone_aware_replication:
    enabled: false
```

> **Note**: The number of store gateway pods that will be started is derived from `store_gateway.replicas`. Each zone will start `store_gateway.replicas / number of zones` pods, rounded up to the nearest integer value. For example if you have 3 zones, then `store_gateway.replicas=3` will yield 1 store gateway per zone, but `store_gateway.replicas=4` will yield 2 per zone, 6 in total.

### Decide which migration path to take for store gateways

There are two ways to do the migration:

1. With downtime. In this [procedure](#migrate-store-gateways-with-downtime) ingress is stopped to the cluster while store gateways are migrated. This is the quicker and simpler way.
1. Without downtime. This is a multi step [procedure](#migrate-store-gateways-without-downtime) which requires additional hardware resources as the old and new store gateways run in parallel for some time.

### Migrate store gateways with downtime

1. Create a new empty YAML file called `migrate.yaml`.

1. Turn off traffic to the installation.

   Copy the following into the `migrate.yaml`:

   ```yaml
   store_gateway:
     zone_aware_replication:
       enabled: false

   nginx:
     replicas: 0
   gateway:
     replicas: 0
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until there is no nginx or gateway running.

1. Scale the current store gateways to 0.

   Copy the following into the `migrate.yaml`:

   ```yaml
   store_gateway:
     replicas: 0
     zone_aware_replication:
       enabled: false

   nginx:
     replicas: 0
   gateway:
     replicas: 0
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until no store-gateways are running.

1. Start the new zone aware store gateways.

   > **Note**: this step assumes that you set up your zones according to [Configure zone aware replication for ingesters](#configure-zone-aware-replication-for-store-gateways)

   Copy the following into the `migrate.yaml`:

   ```yaml
   store_gateway:
     zone_aware_replication:
       enabled: true

   nginx:
     replicas: 0
   gateway:
     replicas: 0

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until all requested store gateways are running.

1. Enable traffic to the installation.

   **Merge** the following values into your regular `custom.yaml` file:

   ```yaml
   store_gateway:
     zone_aware_replication:
       enabled: true

   rollout_operator:
     enabled: true
   ```

   > **Note**: these values are actually the default, which means that removing the values `store_gateway.zone_aware_replication.enabled` and `rollout_operator.enabled` from your `custom.yaml` is also a valid step.

1. Upgrade the installation with the `helm` command using only your regular command line flags.

### Migrate store gateways without downtime

1. Create a new empty YAML file called `migrate.yaml`.

1. Create the new zone aware store gateways

   Copy the following into the `migrate.yaml`:

   ```yaml
   store_gateway:
     zone_aware_replication:
       enabled: true
       migration:
         enabled: true

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait for all new store gateways to start up.

1. Wait for store gateways to sync all blocks.

   The logs of the new store gateways should contain the line "successfully synchronized TSDB blocks for all users".

1. Make the read path use the new zone aware store gateways.

   Copy the following into the `migrate.yaml`:

   ```yaml
   store_gateway:
     zone_aware_replication:
       enabled: true
       migration:
         enabled: true
         read_path: true

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait for the `helm` command to finish.

1. Scale down non zone aware store gateways to 0.

1. Copy the following into the `migrate.yaml`:

   ```yaml
   store_gateway:
     zone_aware_replication:
       enabled: true
       migration:
         enabled: true
         read_path: true
         scale_down_default_zone: true

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait for non zone aware store gateways to stop.

1. Set the final configuration.

   **Merge** the last values in `migrate.yaml` into your regular `custom.yaml` file:

   ```yaml
   store_gateway:
     zone_aware_replication:
       enabled: true

   rollout_operator:
     enabled: true
   ```

   These values are actually the default, which means that removing the values `store_gateway.zone_aware_replication.enabled` and `rollout_operator.enabled` from your `custom.yaml` is also a valid step.

   > **Note**: if you have copied the `mimir.config` value for customizations, make sure to merge the latest version from the chart. That value should include this snippet:

   ```yaml
   store_gateway:
     sharding_ring:
       {{- if .Values.store_gateway.zone_aware_replication.enabled }}
       kvstore:
         prefix: multi-zone/
       {{- end }}
       tokens_file_path: /data/tokens
       {{- if .Values.store_gateway.zone_aware_replication.enabled }}
       zone_awareness_enabled: true
       {{- end }}
   ```

   If in doubt, set the following values:

   ```yaml
   mimir:
     structuredConfig:
       store_gateway:
         sharding_ring:
           kvstore:
             prefix: multi-zone/
           zone_awareness_enabled: true
   ```

1. Upgrade the installation with the `helm` command using only your regular command line flags.


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
                           - ingester
                       - key: app.kubernetes.io/component
                         operator: NotIn
                         values:
                           - ingester-zone-a
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
                           - ingester
                       - key: app.kubernetes.io/component
                         operator: NotIn
                         values:
                           - ingester-zone-b
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
                           - ingester
                       - key: app.kubernetes.io/component
                         operator: NotIn
                         values:
                           - ingester-zone-c
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

### Decide which migration path to take for ingesters

There are two ways to do the migration:

1. With downtime. In this [procedure](#migrate-ingesters-with-downtime) ingress is stopped to the cluster while ingesters are migrated. This is the quicker and simpler way.
1. Without downtime. This is a multi step [procedure](#migrate-ingesters-without-downtime) which requires additional hardware resources as the old and new ingesters run in parallel for some time. This is a complex migration that can take days.

### Migrate ingesters with downtime

1. Create a new empty YAML file called `migrate.yaml`.

1. Enable flushing data from ingesters to storage on shutdown.

   Copy the following into the `migrate.yaml`:

   ```yaml
   mimir:
     structuredConfig:
       blocks_storage:
         flush_blocks_on_shutdown: true
       ingester:
         ring:
           unregister_on_shutdown: true

   ingester:
     zone_aware_replication:
       enabled: false
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait for all ingesters to restart.

1. Turn off traffic to the installation.

   Copy the following into the `migrate.yaml`:

   ```yaml
   mimir:
     structuredConfig:
       blocks_storage:
         flush_blocks_on_shutdown: true
       ingester:
         ring:
           unregister_on_shutdown: true

   ingester:
     zone_aware_replication:
       enabled: false

   nginx:
     replicas: 0
   gateway:
     replicas: 0
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until there is no nginx or gateway running.

1. Scale the current ingesters to 0.

   Copy the following into the `migrate.yaml`:

   ```yaml
   mimir:
     structuredConfig:
       blocks_storage:
         flush_blocks_on_shutdown: true
       ingester:
         ring:
           unregister_on_shutdown: true

   ingester:
     replicas: 0
     zone_aware_replication:
       enabled: false

   nginx:
     replicas: 0
   gateway:
     replicas: 0
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until no ingesters are running.

1. Start the new zone aware ingesters.

   > **Note**: this step assumes that you set up your zones according to [Configure zone aware replication for ingesters](#configure-zone-aware-replication-for-ingesters)

   Copy the following into the `migrate.yaml`:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true

   nginx:
     replicas: 0
   gateway:
     replicas: 0

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until all requested ingesters are running.

1. Enable traffic to the installation.

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

### Migrate ingesters without downtime

1. Create a new empty YAML file called `migrate.yaml`

1. Start the migration.

   > **Note**: this step assumes that you set up your zones according to [Configure zone aware replication for ingesters](#configure-zone-aware-replication-for-ingesters)

   Copy the following into the `migrate.yaml`:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true
       migration:
         enabled: true
         replicas: 0

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

   In this step new zone aware statefulsets are created - but no new pods are started yet. The parameter `ingester.ring.zone_awareness_enabled: true` is set in the Mimir configuration via the `mimir.config` value. The flag `-ingester.ring.zone-awareness-enabled=false` is set on distributors, rulers and queriers. The flags `-blocks-storage.tsdb.flush-blocks-on-shutdown` and `-ingester.ring.unregister-on-shutdown` are set to true for the ingesters.

1. Wait for all Mimir components to restart.

1. Progressively scale zone-aware ingesters up, maximum 21 at a time.

   Copy the following into the `migrate.yaml`:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true
       migration:
         enabled: true
         replicas: <N>

   rollout_operator:
     enabled: true
   ```

   > **Note**: replace `<N>` with the number of replicas in each step until `<N>` reaches the same number as in `ingester.replicas`, do not increase `<N>` with more than 21 in each step.

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Once the new ingesters are started, wait at least 3 hours.

   The 3 hours is calculated from `blocks_storage.tsdb.block_ranges_period` + `blocks_storage.tsdb.head_compaction_idle_timeout` Grafana Mimir parameters to give enough time for ingesters to remove stale series from memory. Stale series will be there due to series being moved between ingesters.

1. If the current `<N>` above in `ingester.zone_aware_replication.migration.replicas` is less than `ingester.replicas`, go back to step 8.

1. Enable zone awareness on the write path.

   Copy the following into the `migrate.yaml`:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true
       migration:
         enabled: true
         write_path: true

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

   In this step the flag `-ingester.ring.zone-awareness-enabled=false` is removed from distributors, rulers.

1. Once all distributors and rulers have restarted, wait 12 hours.

   The 12 hours is calculated from the `querier.query_store_after` Grafana Mimir parameter.

1. Enable zone awareness on the read path.

   Copy the following into the `migrate.yaml`:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true
       migration:
         enabled: true
         write_path: true
         read_path: true

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

   In this step the flag `-ingester.ring.zone-awareness-enabled=false` is removed from queriers.

1. Wait until all queriers have restarted.

1. Exclude non zone aware ingesters from the write path.

   Copy the following into the `migrate.yaml`:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true
       migration:
         enabled: true
         write_path: true
         read_path: true
         exclude_default_zone: true

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

   In this step the flag `-distributor.excluded-zones=zone-default` is added to distributors and rulers.

1. Wait until all distributors and rulers have restarted.

1. Scale down non zone aware ingesters to 0.

   Copy the following into the `migrate.yaml`:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true
       migration:
         enabled: true
         write_path: true
         read_path: true
         exclude_default_zone: true
         scale_down_default_zone: true

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until all non zone aware ingesters are terminated.

1. Delete the default zone.

   Copy the following into the `migrate.yaml`:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true

   rollout_operator:
     enabled: true
   ```

1. Upgrade the installation with the `helm` command and make sure to provide the flag `-f migrate.yaml` as the last flag.

1. Wait until all old ingesters have terminated.

1. Set the final configuration.

   **Merge** the last values in `migrate.yaml` into your regular `custom.yaml` file:

   ```yaml
   ingester:
     zone_aware_replication:
       enabled: true

   rollout_operator:
     enabled: true
   ```

   These values are actually the default, which means that removing the values `ingester.zone_aware_replication.enabled` and `rollout_operator.enabled` from your `custom.yaml` is also a valid step.

   > **Note**: if you have copied the `mimir.config` value for customizations, make sure to merge the latest version from the chart. That value should include this snippet:

   ```yaml
   ingester:
      ring:
        ...
        unregister_on_shutdown: false
        {{- if .Values.ingester.zone_aware_replication.enabled }}
        zone_awareness_enabled: true
        {{- end }}
   ```

   If in doubt, set the following value:

   ```yaml
   mimir:
     structuredConfig:
       ingester:
         ring:
           zone_awareness_enabled: true
   ```

1. Upgrade the installation with the `helm` command using only your regular command line flags.
