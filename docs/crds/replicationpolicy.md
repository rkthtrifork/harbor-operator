# Replication Policy CRD

A **ReplicationPolicy** custom resource manages Harbor replication policies via
`/api/v2.0/replication/policies`.

Replication policies control automated (or manual) synchronization between
registries.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: ReplicationPolicy
metadata:
  name: sample-replication
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection

  # Source / destination registries
  sourceRegistryRef:
    name: src-registry
  destinationRegistryRef:
    name: dest-registry

  # Trigger settings
  trigger:
    type: scheduled
    settings:
      cron: "0 0 * * * *"

  # Optional filters
  filters:
    - type: name
      decoration: matches
      value: "library/**"

  replicateDeletion: true
  enabled: true
```

## Key Fields

- **spec.harborConnectionRef** (object, required)
  Reference to the Harbor connection object to use. Set `name` and optional `kind` (`HarborConnection` by default or `ClusterHarborConnection`).

- **metadata.name** (string, required)
  The Harbor replication policy name managed by this CR.

- **spec.sourceRegistryRef** (object, required)
  Select the source registry for replication.

- **spec.destinationRegistryRef** (object, required)
  Select the destination registry.

- **spec.trigger** (object, optional)
  Defines manual, event-based, or scheduled triggers.

- **spec.filters** (array, optional)
  Replication filters (name/tag/label scopes).

- **spec.destNamespaceReplaceCount** (int, optional)
  Number of source path components replaced by `destNamespace`. Defaults to
  `-1`, which selects Harbor's legacy replacement behavior.

- **spec.replicateDeletion** (bool, optional)
  Whether to replicate deletions. When omitted, the operator leaves it unset for
  Harbor to interpret.

- **spec.override**, **spec.enabled**, **spec.speed**, **spec.copyByChunk**, and
  **spec.singleActiveReplication** (optional)
  The operator has no defaults for these controls. When omitted, they are left
  unset for Harbor to interpret.

- **spec.creationPolicy** (string, optional)
  Controls whether the policy is created, adopted, or either. When omitted, uses the operator's default creation policy (`Create` unless configured otherwise).

## Common Fields

`ReplicationPolicy` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create / Update**
  Creates or updates a replication policy in Harbor.

- **Delete**
  Deletes the policy in Harbor when the CR is deleted.

- **Adoption**
  A policy with the same name is adopted when `creationPolicy` permits adoption.
