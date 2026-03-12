# HarborConnection CRD

The **HarborConnection** custom resource describes how the operator should connect
to an existing Harbor instance using a namespaced connection object. All other
Harbor CRDs reference either a `HarborConnection` or a `ClusterHarborConnection`.

Typical use:

- Define one or more HarborConnection objects (e.g. one for dev Harbor, one for prod).
- Point Registry / Project / User / Member CRs at the right connection via
  `spec.harborConnectionRef.name` and optional `spec.harborConnectionRef.kind`.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: HarborConnection
metadata:
  name: my-harbor
spec:
  # Harbor API endpoint. Must include protocol (http:// or https://).
  baseURL: "https://harbor.example.com"

  # Optional credentials for API access.
  credentials:
    type: basic
    username: "my-username"
    passwordSecretRef:
      name: my-harbor-secret
      key: password

  # Optional CA bundle for self-signed Harbor TLS certs.
  # caBundleSecretRef and caBundle are mutually exclusive.
  caBundleSecretRef:
    name: my-harbor-ca
    key: ca.crt
```

## Key Fields

- **spec.baseURL** (string, required)
  Harbor API base URL. Must include scheme, e.g. `https://harbor.example.com`.

- **spec.credentials** (object, optional)

  - **type** (string) – currently `basic` is supported.
  - **username** (string) – username for Harbor.
  - **passwordSecretRef** (object) – Secret reference with `name`, optional
    `namespace`, and `key`.

- **spec.caBundle** (string, optional)
  PEM-encoded CA bundle.

- **spec.caBundleSecretRef** (object, optional)
  Secret reference containing a PEM-encoded CA bundle (defaults to `ca.crt`).
  Mutually exclusive with `spec.caBundle`.

## Behavior

- **Validation**

  - The operator checks that `baseURL` parses and includes a scheme.

- **Connectivity check**

  - Without credentials: calls `/api/v2.0/ping`.
  - With credentials: calls `/api/v2.0/users/current` to verify auth.

- **Dependent resources**

  - Updating a `HarborConnection` triggers reconciliation of Harbor-backed CRs that reference it.
  - This applies to changes such as base URL, credentials, and CA bundle configuration.

- **Error handling**

  - If the URL is invalid or Harbor returns an error status, the operator logs
    a clear error to help diagnose connectivity/auth issues.

- **Secret usage**

  - The secret referenced by `passwordSecretRef` is read at reconcile time.
  - Password is passed to Harbor via basic auth on each request.

## Related

- Use `HarborConnection` for tenant-local access scoped to one namespace.
- Use `ClusterHarborConnection` when multiple namespaces should share the same
  Harbor endpoint and credentials.
