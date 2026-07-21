# Quickstart

This operator manages an existing Harbor installation. You need a reachable Harbor instance, credentials, and permission to install CRDs and the operator in Kubernetes.

## Install the operator

```sh
helm registry login ghcr.io
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \
  --namespace harbor-operator-system \
  --create-namespace \
  --version <chart-version>
```

See [Installation](introduction/installation.md) for chart settings such as namespace scoping, an operator-wide connection, metrics, and request defaults.

## Connect Harbor

Create either:

- a namespaced `HarborConnection` for tenant-local use
- a cluster-scoped `ClusterHarborConnection` for shared access

The connection contains Harbor's base URL and may reference credentials and custom CA material. Follow the [connection and project example](examples/connection-and-project.md) for complete manifests.

## Create resources

Create Harbor-backed custom resources that reference the connection. The operator reports progress and failures through status conditions:

```sh
kubectl get projects.harbor.harbor-operator.io
kubectl get project <name> -o yaml
```

Continue with [Concepts](introduction/concepts.md), the [API overview](reference/index.md), and the [resource guides](crds/index.md).

For a local contributor stack that installs Harbor and the operator in Kind, use the [local development guide](contributing/local-development.md).
