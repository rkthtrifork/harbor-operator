# Connection Patterns

Every Harbor-backed custom resource references either a `HarborConnection` or a `ClusterHarborConnection`.

## When to Use HarborConnection

Use `HarborConnection` when:

- the Harbor access should stay namespaced
- credentials are tenant-local
- you want each namespace to manage its own Harbor integration separately

This is the safer default in multi-tenant clusters.

## When to Use ClusterHarborConnection

Use `ClusterHarborConnection` when:

- multiple namespaces should share the same Harbor endpoint
- the platform team manages the connection centrally
- you want one shared Harbor definition reused across tenants or workloads

## Credentials

Connection credentials are stored in Kubernetes Secrets and referenced from the connection object.

Typical pattern:

- username in the CR
- password or token in a Secret

## CA Material

If Harbor uses a custom CA, provide either:

- `spec.caBundle`
- or `spec.caBundleSecretRef`

but not both.

## Update Behavior

When a `HarborConnection` or `ClusterHarborConnection` changes, dependent Harbor-backed resources are reconciled again.

That includes changes such as:

- base URL changes
- credential secret changes reflected through the connection object
- CA material changes

## Cross-Namespace Sharing

Cross-namespace sharing is not done through namespaced references. Instead:

- use `HarborConnection` for namespaced local use
- use `ClusterHarborConnection` when you intentionally want sharing
