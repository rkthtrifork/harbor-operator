# HarborConnection CRD

The **HarborConnection** custom resource describes how the operator should connect
to an existing Harbor instance. All other Harbor CRDs reference a HarborConnection.

Typical use:

- Define one or more HarborConnection objects (e.g. one for dev Harbor, one for prod).
- Point Registry / Project / User / Member CRs at the right connection via `harborConnectionRef`.

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
    accessKey: "my-username"
    # Name of a Secret in the same namespace with key "access_secret".
    accessSecretRef: "my-harbor-secret"
```

## Key Fields

- **spec.baseURL** (string, required)
  Harbor API base URL. Must include scheme, e.g. `https://harbor.example.com`.

- **spec.credentials** (object, optional)

  - **type** (string) – currently `basic` is supported.
  - **accessKey** (string) – username for Harbor.
  - **accessSecretRef** (string) – name of a `Secret` in the same namespace.
    The secret must contain a key `access_secret` with the password.

## Behavior

- **Validation**

  - The operator checks that `baseURL` parses and includes a scheme.

- **Connectivity check**

  - Without credentials: calls `/api/v2.0/ping`.
  - With credentials: calls `/api/v2.0/users/current` to verify auth.

- **Error handling**

  - If the URL is invalid or Harbor returns an error status, the operator logs
    a clear error to help diagnose connectivity/auth issues.

- **Secret usage**

  - The secret referenced by `accessSecretRef` is read at reconcile time.
  - Password is passed to Harbor via basic auth on each request.
