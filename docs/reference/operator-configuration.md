# Operator Configuration

The Helm chart exposes the operator's runtime flags as values. These settings
control reconciliation scope and defaults; they do not replace Kubernetes
authorization or admission policies.

## Harbor connection

`harborConnection` configures one operator-wide `ClusterHarborConnection`.
When it is set, resources may omit `spec.harborConnectionRef` and use this
connection instead. An explicit resource reference still takes precedence.

## Reconciliation scope

`watchNamespaces` limits the namespaces cached and reconciled by the operator.
It is an operational scope and is not a tenant-isolation boundary. Use
namespace RBAC and admission policies, or set
`allowCrossNamespaceReferences: false`, when you need tenant boundaries.

Use `watchNamespaces` when one operator deployment should manage only a fixed
subset of namespaces, reduce its operational blast radius, or separate
operator instances by workload. Leaving it empty watches all namespaces.

With `allowCrossNamespaceReferences: false`, a resource may reference only
objects in its own namespace. The default is `true` for compatibility with
single-tenant and shared-service installations.

## Creation and drift defaults

`defaultCreationPolicy` supplies the creation policy when a resource does not
specify one. The supported policies are `Create`, `Adopt`, and `CreateOrAdopt`.
An explicit `spec.creationPolicy` always wins over the operator default.

`defaultDriftDetectionInterval` controls how often managed resources are
checked for drift when they are otherwise idle. Set it to `0s` to disable the
periodic check if that is appropriate for your installation.

## Metrics and network policy

Metrics are disabled by default. When enabled, the chart can create a
`ServiceMonitor` and a NetworkPolicy. Secure metrics use the chart's configured
certificate; for production, configure a certificate Secret trusted by your
Prometheus installation instead of relying on the development certificate.

See the chart's values and README for the complete list of values and their
defaults.
