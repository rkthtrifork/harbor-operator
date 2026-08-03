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
    auth_mode:
      value: "oidc_auth"
    oidc_name:
      value: "ExampleOIDC"
    oidc_endpoint:
      value: "https://oidc.example.com"
    oidc_client_id:
      value: "harbor"
    oidc_groups_claim:
      value: "groups"
    oidc_admin_group:
      value: "harbor-admins"
    oidc_scope:
      value: "openid,profile,email,groups"
    oidc_user_claim:
      value: "preferred_username"
    oidc_auto_onboard:
      value: true
    oidc_verify_cert:
      value: false
    robot_token_duration:
      value: 30
    robot_name_prefix:
      value: "robot$"
    oidc_client_secret:
      valueFrom:
        secretKeyRef:
          name: harbor-oidc-client
          key: clientSecret
```

## Key Fields

- **spec.harborConnectionRef** (object, optional when the operator is configured with `--harbor-connection`)
  Reference to the Harbor connection object to use. Set `name` and optional `kind` (`HarborConnection` by default or `ClusterHarborConnection`).

- **spec.settings** (map, optional)
  Map of Harbor configuration keys to value sources. Keys must be recognized by
  Harbor's `/api/v2.0/configurations` endpoint. Each entry sets exactly one of a
  literal `value` or `valueFrom.secretKeyRef`. Literal values may be any valid
  JSON value. If the Secret key is omitted, the operator defaults it to `value`.

## Common Fields

`Configuration` embeds `HarborSpecBase`. See [Common Spec Fields](../reference/common-spec-fields.md)
for the shared connection, deletion, and reconciliation controls, or jump to the
generated [`HarborSpecBase` reference](../reference/api.md#harborspecbase).

## Behavior

- **Create/Update**
  - Sends only the specified keys to Harbor (partial update).
  - Only one `Configuration` may manage a given Harbor instance. If multiple CRs target the same Harbor instance, the oldest CR remains the owner and later CRs report a conflict.
  - Skips reconciliation when no settings are provided.

- **Delete**

  - Removing the CR does not reset Harbor settings (Harbor has no delete API
    for system configuration). The CR is simply removed.
