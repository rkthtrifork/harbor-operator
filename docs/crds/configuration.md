# Configuration CRD

A **Configuration** custom resource manages Harbor system configuration via the
`/api/v2.0/configurations` endpoint. You can use it to enable OIDC, tune robot
settings, and set other global options.

## Quick Start

```yaml
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Configuration
metadata:
  name: harbor-configuration
spec:
  harborConnectionRef:
    name: my-harbor
    kind: HarborConnection
  settings:
    auth_mode: "oidc_auth"
    oidc_name: "ExampleOIDC"
    oidc_endpoint: "https://oidc.example.com"
    oidc_client_id: "harbor"
    oidc_groups_claim: "groups"
    oidc_admin_group: "harbor-admins"
    oidc_scope: "openid,profile,email,groups"
    oidc_user_claim: "preferred_username"
    oidc_auto_onboard: true
    oidc_verify_cert: false
    robot_token_duration: 30
    robot_name_prefix: "robot$"
  secretSettings:
    oidc_client_secret:
      name: harbor-oidc-client
      key: clientSecret
```

## Key Fields

- **spec.harborConnectionRef** (object, required)
  Reference to the Harbor connection object to use. Set `name` and optional `kind` (`HarborConnection` by default or `ClusterHarborConnection`).

- **spec.settings** (map, optional)
  Map of Harbor configuration keys to values. Keys must match the
  `/configurations` schema in `swagger.yaml`. Values may be strings, numbers,
  booleans, or JSON objects.

- **spec.secretSettings** (map, optional)
  Map of Harbor configuration keys to secret references. The secret values are
  read and injected into the update payload. If `key` is omitted, the operator
  defaults to `value`.

## Common Fields

`Configuration` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create/Update**
  - Sends only the specified keys to Harbor (partial update).
  - Only one `Configuration` may manage a given Harbor instance. If multiple CRs target the same Harbor instance, the oldest CR remains the owner and later CRs report a conflict.
  - Sends only the specified keys to Harbor (partial update).
  - Skips reconciliation when no settings are provided.

- **Delete**

  - Removing the CR does not reset Harbor settings (Harbor has no delete API
    for system configuration). The CR is simply removed.
