# Installation

The recommended way to install `harbor-operator` in a real cluster is the Helm chart published to GHCR.

## Prerequisites

- a running Harbor instance
- cluster access with permission to install CRDs, RBAC, and the operator deployment
- credentials and optional CA material for whichever Harbor instance you want the operator to manage

## Install from OCI

```sh
helm registry login ghcr.io
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \
  --namespace harbor-operator-system \
  --create-namespace \
  --version <chart-version>
```

## Common Overrides

```sh
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \
  --namespace harbor-operator-system \
  --create-namespace \
  --version <chart-version> \
  --set metrics.enabled=true
```

Useful overrides include:

- metrics enablement
- ServiceMonitor creation
- PodDisruptionBudget settings
- NetworkPolicy settings
- `watchNamespaces` to scope the operator to specific namespaces
- `harborConnection` to force a single `ClusterHarborConnection` for all Harbor-backed resources

See the chart documentation in [`charts/harbor-operator/README.md`](https://github.com/rkthtrifork/harbor-operator/blob/main/charts/harbor-operator/README.md) for the install flags that matter most.

## After Installation

The operator is only the control plane. You still need to create one of:

- a namespaced `HarborConnection`
- a cluster-scoped `ClusterHarborConnection`

Then create Harbor-backed resources that reference that connection object.
If you set `harborConnection` in the chart values, Harbor-backed resources can
omit `spec.harborConnectionRef` and will all use that cluster-scoped connection.

For shared-cluster setups, see [Multi-Tenancy](../reference/multi-tenancy.md).

## Local Development Stack

If you want a local Harbor plus operator environment for development or testing, use:

```sh
make kind-up
```

or:

```sh
make kind-up-cilium
```
