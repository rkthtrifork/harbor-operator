# Reference

The reference section has three complementary parts:

- **Resource Guides** for behavior, examples, and operator-specific notes
- **Common Spec Fields**, status behavior, and operator configuration
- **Generated API Reference** for the exact schema

Use the resource guides when you want to understand how a custom resource behaves in practice.

Use the generated reference when you want exact field definitions, defaults, enums, and validation markers.

Use [Common Spec Fields](common-spec-fields.md) for the shared
`HarborSpecBase` fields that appear on every Harbor-managed resource.

Use the [Guides](../reference/connection-patterns.md) for cross-cutting topics
such as connection patterns, multi-tenancy, lifecycle, upgrades, and
troubleshooting.

Use [Operator Configuration](operator-configuration.md) for runtime flags and
chart values, and [Status and Conditions](status-and-conditions.md) when a
resource is not becoming ready.

## Generated Reference

The generated reference is produced with `crd-ref-docs` from the API types in `api/v1alpha1`.

Regenerate it with:

```sh
make generate-api-reference
```

The generated page is checked into the repository and verified in CI so that the schema reference stays aligned with the API definitions.

Use the [Resource Index](resources.md) if you want to jump directly to a single
custom resource instead of scanning the full generated page.
