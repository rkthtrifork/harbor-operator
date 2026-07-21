# harbor-operator

`harbor-operator` manages resources in an existing Harbor installation through Kubernetes custom resources. It does not install Harbor outside the repository's local development environment.

The operator provides namespaced `HarborConnection` and cluster-scoped `ClusterHarborConnection` resources for Harbor access. Other resources, including projects, registries, users, robots, replication policies, and schedules, reference one of those connections and reconcile their desired state through the Harbor API.

## Install

Install the published OCI Helm chart into a cluster that can reach Harbor:

```sh
helm registry login ghcr.io
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \
  --namespace harbor-operator-system \
  --create-namespace \
  --version <chart-version>
```

Create a `HarborConnection` or `ClusterHarborConnection`, then create Harbor resources that reference it. See the [quickstart](docs/quickstart.md) for the shortest complete path and the [installation guide](docs/introduction/installation.md) for chart configuration.

## Documentation

The [documentation site](https://rkthtrifork.github.io/harbor-operator/) contains:

- installation and core concepts
- resource guides and examples
- generated API reference
- connection, ownership, deletion, and multi-tenancy behavior

The Go API types and Kubebuilder markers under `api/v1alpha1` define the public Kubernetes API. The generated [API reference](docs/reference/api.md) documents that schema, while the hand-written guides explain behavior and Harbor semantics.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for the contributor contract and [local development](docs/contributing/local-development.md) for the Kind-based workflow.
