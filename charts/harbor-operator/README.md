# Harbor Operator Helm Chart

This chart installs the harbor-operator controller into your cluster.

## Install (OCI)

```sh
helm registry login ghcr.io
helm install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator --version <chart-version>
```

## Template (render YAML)

```sh
helm template harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator --version <chart-version>
```

## Values

Default values live in `values.yaml`, and validation in `values.schema.json`.

Common overrides:

```sh
helm upgrade --install harbor-operator oci://ghcr.io/rkthtrifork/charts/harbor-operator \\
  --version <chart-version> \\
  --set image.tag=v0.3.0 \\
  --set metrics.enabled=true
```

## CRDs

CRDs are packaged in the chart under `crds/`. These are synced from `config/crd/bases`.
