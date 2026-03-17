# API

The `API` section is split into two parts:

- **Resource Guides** for behavior, examples, and operator-specific notes
- **API Reference** for generated schema documentation

Use the resource guides when you want to understand how a custom resource behaves in practice.

Use the generated reference when you want exact field definitions, defaults, enums, and validation markers.

Use [Common Spec Fields](common-spec-fields.md) for the shared
`HarborSpecBase` fields that appear on every Harbor-managed resource.

## Generated Reference

The generated reference is produced with `crd-ref-docs` from the API types in `api/v1alpha1`.

Regenerate it with:

```sh
make generate-docs
```

The generated page is checked into the repository and verified in CI so that the schema reference stays aligned with the API definitions.

Use the [Resource Index](resources.md) if you want to jump directly to a single custom resource instead of scanning the full generated page.
