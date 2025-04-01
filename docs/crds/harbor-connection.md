# HarborConnection CRD

The **HarborConnection** custom resource lets you define the connection details for your existing Harbor instance. This resource is used by other Harbor CRDs to access your Harbor API.

## Quick Start

Create a HarborConnection resource by providing at least the Harbor API endpoint (baseURL). Optionally, you can include credentials for authentication.

#### Example

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: HarborConnection
metadata:
  name: my-harbor-connection
spec:
  # The Harbor API endpoint. Must include the protocol (http:// or https://).
  baseURL: "https://harbor.example.com"

  # Optional credentials for API access.
  credentials:
    # Currently, only "basic" auth is supported.
    type: basic
    # The username to use for authentication.
    accessKey: "my-username"
    # Name of the Kubernetes Secret that holds the password in the "access_secret" key.
    accessSecretRef: "my-harbor-secret"
```

## Field Descriptions

- **baseURL**:  
  The URL of your Harbor API. It must include the protocol (e.g., `https://`).

- **credentials** (optional):  
  Details used for authenticating to Harbor.
  - **type**: The type of authentication. Currently, only `basic` is supported.
  - **accessKey**: The username used to authenticate with Harbor.
  - **accessSecretRef**: A reference to a Kubernetes Secret containing the password (stored under the key `access_secret`).

## Advanced Details

- **Connectivity Checks:**  
  The operator validates the provided `baseURL` to ensure it’s a proper URL with a protocol.

  - If no credentials are provided, the operator performs a connectivity check by calling the `/api/v2.0/ping` endpoint.
  - If credentials are provided, it uses them to call `/api/v2.0/users/current` for authentication verification.

- **Error Handling:**  
  If the URL is invalid or the connectivity check fails (unexpected status codes), the operator logs detailed error messages. This can help diagnose issues like missing protocol schemes or network problems.

- **Credential Retrieval:**  
  The operator retrieves the secret referenced in `accessSecretRef` and expects a key named `access_secret` to obtain the authentication password.
