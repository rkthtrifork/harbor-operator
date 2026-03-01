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
  harborConnectionRef: "my-harbor"

  # Optional explicit name (defaults to metadata.name)
  name: "sample-replication"

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

- **spec.harborConnectionRef** (string, required)
  Name of the HarborConnection to use.

- **spec.sourceRegistryRef** / **spec.sourceRegistryID** (one required)
  Select the source registry for replication.

- **spec.destinationRegistryRef** / **spec.destinationRegistryID** (one required)
  Select the destination registry.

- **spec.trigger** (object, optional)
  Defines manual, event-based, or scheduled triggers.

- **spec.filters** (array, optional)
  Replication filters (name/tag/label scopes).

- **spec.replicateDeletion** (bool, optional)
  Whether to replicate deletions.

## Behavior

- **Create / Update**
  Creates or updates a replication policy in Harbor.

- **Delete**
  Deletes the policy in Harbor when the CR is deleted.

- **Adoption**
  If `allowTakeover` is true, a policy with the same name is adopted.
