# Local Development

The supported development environment runs on the host with Go, Docker, Helm, `kubectl`, and Kind. The repository does not maintain a devcontainer.

## Start the stack

Add the Harbor ingress name to the host file:

```text
127.0.0.1 core.harbor.domain
```

Create a Kind cluster, install Traefik and Harbor, build the operator, and deploy it:

```sh
make kind-up
```

To use Cilium and Hubble instead of Kind's default CNI:

```sh
make kind-up KIND_CNI=cilium
```

## Iterate

After changing controller code, API types, generated assets, RBAC, or chart wiring, rebuild and redeploy with:

```sh
make kind-refresh
```

Apply the sample resources when useful:

```sh
make apply-samples
```

`make delete-harbor-crs` deletes every custom resource in the `harbor.harbor-operator.io` API group, not only the checked-in samples.

## Reset or stop

Reset operator-managed resources and CRDs, then redeploy:

```sh
make kind-redeploy
```

Delete the Kind cluster completely:

```sh
make kind-down
```

Use `kind-redeploy` when testing first-install behavior or an intentionally incompatible CRD change. Avoid resetting a shared development cluster that another task may be using.

## Focused deployment targets

The high-level workflow is usually sufficient. For packaging or deployment work, the Makefile also exposes `generate-manifests`, `sync-chart-assets`, `apply-crds`, `docker-build`, `deploy`, and `undeploy`.
