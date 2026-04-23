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

- **spec.replicateDeletion** (bool, optional)
  Whether to replicate deletions.

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
  If `allowTakeover` is true, a policy with the same name is adopted.
