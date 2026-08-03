# Upgrading

Upgrade the operator and its Helm chart together. The chart version and the
operator image version are published as a tested pair in the release notes.

## Before upgrading

1. Read the release notes for API, chart, and behavior changes.
2. Check whether the release contains breaking changes to custom resources or
   controller flags.
3. Keep the desired custom resources in version control and, for production
   upgrades, take a backup of the cluster state you would need to restore.

## Upgrade the release

Update the chart reference in your HelmRelease (or Helm command) to the
documented chart version. The chart packages CRDs under `crds/`; Helm installs
those on the initial release, but does not automatically upgrade existing CRDs
on every `helm upgrade`. Confirm that your deployment tool applies the new CRD
manifests, or apply the versioned CRDs explicitly before rolling out a release
that changes the schema.

After the rollout, verify that:

- the operator Deployment is available;
- the CRDs are established and served at the expected API version; and
- managed resources become `Ready` again without unexpected changes in Harbor.

For a resource that reports a failure, see [Status and Conditions](../reference/status-and-conditions.md)
and [Troubleshooting](../reference/troubleshooting.md).

## Breaking API changes

Follow the migration instructions in the release notes for breaking changes.
Do not apply a new manifest blindly when a field has become immutable or when a
resource has moved to a new API version. Make the migration explicit in the
desired state, then verify the resulting Harbor objects before continuing with
the rest of the upgrade.

Publishing a new operator release is documented for maintainers in
[Releases](../contributing/releases.md).
