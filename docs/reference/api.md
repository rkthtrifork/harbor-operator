# API Reference

## Packages
- [harbor.harbor-operator.io/v1alpha1](#harborharbor-operatoriov1alpha1)


## harbor.harbor-operator.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the harbor v1alpha1 API group.

### Resource Types
- [ClusterHarborConnection](#clusterharborconnection)
- [Configuration](#configuration)
- [GCSchedule](#gcschedule)
- [HarborConnection](#harborconnection)
- [ImmutableTagRule](#immutabletagrule)
- [Label](#label)
- [Member](#member)
- [Project](#project)
- [PurgeAuditSchedule](#purgeauditschedule)
- [Quota](#quota)
- [Registry](#registry)
- [ReplicationPolicy](#replicationpolicy)
- [RetentionPolicy](#retentionpolicy)
- [Robot](#robot)
- [ScanAllSchedule](#scanallschedule)
- [ScannerRegistration](#scannerregistration)
- [User](#user)
- [UserGroup](#usergroup)
- [WebhookPolicy](#webhookpolicy)



#### CVEAllowlist



CVEAllowlist defines the CVE allowlist configuration.



_Appears in:_
- [ProjectSpec](#projectspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `id` _integer_ |  |  |  |
| `project_id` _integer_ |  |  |  |
| `expires_at` _integer_ |  |  |  |
| `items` _[CVEAllowlistItem](#cveallowlistitem) array_ |  |  |  |
| `creation_time` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#time-v1-meta)_ |  |  |  |
| `update_time` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#time-v1-meta)_ |  |  |  |


#### CVEAllowlistItem



CVEAllowlistItem defines a single CVE allowlist entry.



_Appears in:_
- [CVEAllowlist](#cveallowlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `cve_id` _string_ |  |  |  |


#### ClusterHarborConnection



ClusterHarborConnection is the Schema for the clusterharborconnections API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `ClusterHarborConnection` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[HarborConnectionSpec](#harborconnectionspec)_ |  |  |  |


#### Configuration



Configuration is the Schema for the configurations API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `Configuration` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ConfigurationSpec](#configurationspec)_ |  |  |  |


#### ConfigurationSpec



ConfigurationSpec defines the desired state of Harbor system configuration.



_Appears in:_
- [Configuration](#configuration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `settings` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#json-v1-apiextensions-k8s-io))_ | Settings contains Harbor configuration keys and their desired values.<br />Values can be strings, numbers, booleans, or JSON objects. |  | Optional: \{\} <br /> |
| `secretSettings` _object (keys:string, values:[SecretReference](#secretreference))_ | SecretSettings references secret-backed configuration values such as<br />oidc_client_secret. The secret data is read and injected into Settings<br />during reconciliation. |  | Optional: \{\} <br /> |


#### Credentials



Credentials holds default authentication details.



_Appears in:_
- [HarborConnectionSpec](#harborconnectionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type of the credential, e.g., "basic". | basic | Enum: [basic] <br /> |
| `username` _string_ | Username for authentication. |  | MinLength: 1 <br /> |
| `passwordSecretRef` _[SecretReference](#secretreference)_ | PasswordSecretRef points to the Kubernetes Secret that stores the password / token. |  |  |


#### DeletionPolicy

_Underlying type:_ _string_





_Appears in:_
- [ConfigurationSpec](#configurationspec)
- [GCScheduleSpec](#gcschedulespec)
- [HarborSpecBase](#harborspecbase)
- [ImmutableTagRuleSpec](#immutabletagrulespec)
- [LabelSpec](#labelspec)
- [MemberSpec](#memberspec)
- [ProjectSpec](#projectspec)
- [PurgeAuditScheduleSpec](#purgeauditschedulespec)
- [QuotaSpec](#quotaspec)
- [RegistrySpec](#registryspec)
- [ReplicationPolicySpec](#replicationpolicyspec)
- [RetentionPolicySpec](#retentionpolicyspec)
- [RobotSpec](#robotspec)
- [ScanAllScheduleSpec](#scanallschedulespec)
- [ScannerRegistrationSpec](#scannerregistrationspec)
- [UserGroupSpec](#usergroupspec)
- [UserSpec](#userspec)
- [WebhookPolicySpec](#webhookpolicyspec)

| Field | Description |
| --- | --- |
| `Delete` |  |
| `Orphan` |  |


#### GCSchedule



GCSchedule is the Schema for the gcschedules API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `GCSchedule` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GCScheduleSpec](#gcschedulespec)_ |  |  |  |


#### GCScheduleSpec



GCScheduleSpec defines the desired schedule for garbage collection.



_Appears in:_
- [GCSchedule](#gcschedule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `schedule` _[ScheduleSpec](#schedulespec)_ | Schedule defines when GC runs. |  |  |
| `parameters` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#json-v1-apiextensions-k8s-io))_ | Parameters define GC settings passed to Harbor. |  | Optional: \{\} <br /> |


#### HarborConnection



HarborConnection is the Schema for the harborconnections API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `HarborConnection` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[HarborConnectionSpec](#harborconnectionspec)_ |  |  |  |


#### HarborConnectionReference



HarborConnectionReference identifies either a namespaced HarborConnection or a
cluster-scoped ClusterHarborConnection.



_Appears in:_
- [ConfigurationSpec](#configurationspec)
- [GCScheduleSpec](#gcschedulespec)
- [HarborSpecBase](#harborspecbase)
- [ImmutableTagRuleSpec](#immutabletagrulespec)
- [LabelSpec](#labelspec)
- [MemberSpec](#memberspec)
- [ProjectSpec](#projectspec)
- [PurgeAuditScheduleSpec](#purgeauditschedulespec)
- [QuotaSpec](#quotaspec)
- [RegistrySpec](#registryspec)
- [ReplicationPolicySpec](#replicationpolicyspec)
- [RetentionPolicySpec](#retentionpolicyspec)
- [RobotSpec](#robotspec)
- [ScanAllScheduleSpec](#scanallschedulespec)
- [ScannerRegistrationSpec](#scannerregistrationspec)
- [UserGroupSpec](#usergroupspec)
- [UserSpec](#userspec)
- [WebhookPolicySpec](#webhookpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the referenced Harbor connection object. |  |  |
| `kind` _[HarborConnectionReferenceKind](#harborconnectionreferencekind)_ | Kind selects the Harbor connection object kind.<br />When omitted, controllers treat it as HarborConnection. |  | Enum: [HarborConnection ClusterHarborConnection] <br />Optional: \{\} <br /> |


#### HarborConnectionReferenceKind

_Underlying type:_ _string_





_Appears in:_
- [HarborConnectionReference](#harborconnectionreference)

| Field | Description |
| --- | --- |
| `HarborConnection` |  |
| `ClusterHarborConnection` |  |


#### HarborConnectionSpec



HarborConnectionSpec defines the desired state of HarborConnection.



_Appears in:_
- [ClusterHarborConnection](#clusterharborconnection)
- [HarborConnection](#harborconnection)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `baseURL` _string_ | BaseURL is the Harbor API endpoint. |  | Format: url <br /> |
| `credentials` _[Credentials](#credentials)_ | Credentials holds the default credentials for Harbor API calls. |  |  |
| `caBundle` _string_ | CABundle is a PEM-encoded CA bundle for validating Harbor TLS certificates. |  | Optional: \{\} <br /> |
| `caBundleSecretRef` _[SecretReference](#secretreference)_ | CABundleSecretRef references a Secret containing a PEM-encoded CA bundle.<br />When set, it is mutually exclusive with caBundle. |  | Optional: \{\} <br /> |


#### HarborSpecBase



HarborSpecBase holds the fields that appear in every Harbor CR.



_Appears in:_
- [ConfigurationSpec](#configurationspec)
- [GCScheduleSpec](#gcschedulespec)
- [ImmutableTagRuleSpec](#immutabletagrulespec)
- [LabelSpec](#labelspec)
- [MemberSpec](#memberspec)
- [ProjectSpec](#projectspec)
- [PurgeAuditScheduleSpec](#purgeauditschedulespec)
- [QuotaSpec](#quotaspec)
- [RegistrySpec](#registryspec)
- [ReplicationPolicySpec](#replicationpolicyspec)
- [RetentionPolicySpec](#retentionpolicyspec)
- [RobotSpec](#robotspec)
- [ScanAllScheduleSpec](#scanallschedulespec)
- [ScannerRegistrationSpec](#scannerregistrationspec)
- [UserGroupSpec](#usergroupspec)
- [UserSpec](#userspec)
- [WebhookPolicySpec](#webhookpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |




#### ImmutableSelector



ImmutableSelector defines an immutable tag rule selector.



_Appears in:_
- [ImmutableTagRuleSpec](#immutabletagrulespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind defines selector kind. |  | Optional: \{\} <br /> |
| `decoration` _string_ | Decoration defines selector decoration. |  | Optional: \{\} <br /> |
| `pattern` _string_ | Pattern defines selector pattern. |  | Optional: \{\} <br /> |
| `extras` _string_ | Extras defines extra selector details. |  | Optional: \{\} <br /> |


#### ImmutableTagRule



ImmutableTagRule is the Schema for the immutabletagrules API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `ImmutableTagRule` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ImmutableTagRuleSpec](#immutabletagrulespec)_ |  |  |  |


#### ImmutableTagRuleSpec



ImmutableTagRuleSpec defines the desired state of ImmutableTagRule.



_Appears in:_
- [ImmutableTagRule](#immutabletagrule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing immutable tag rule in Harbor that matches this spec. |  | Optional: \{\} <br /> |
| `projectRef` _[ProjectReference](#projectreference)_ | ProjectRef references a Project CR to derive the Harbor project ID. |  | Optional: \{\} <br /> |
| `disabled` _boolean_ | Disabled indicates whether the rule is disabled. |  | Optional: \{\} <br /> |
| `action` _string_ | Action defines the rule action. |  | Optional: \{\} <br /> |
| `template` _string_ | Template defines the rule template. |  | Optional: \{\} <br /> |
| `params` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#json-v1-apiextensions-k8s-io))_ | Params holds template parameters. |  | Optional: \{\} <br /> |
| `tagSelectors` _[ImmutableSelector](#immutableselector) array_ | TagSelectors define tag selectors. |  | Optional: \{\} <br /> |
| `scopeSelectors` _object (keys:string, values:[ImmutableSelector](#immutableselector))_ | ScopeSelectors define scope selectors. |  | Optional: \{\} <br /> |
| `priority` _integer_ | Priority defines the rule priority. |  | Optional: \{\} <br /> |


#### Label



Label is the Schema for the labels API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `Label` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LabelSpec](#labelspec)_ |  |  |  |


#### LabelSpec



LabelSpec defines the desired state of Label.



_Appears in:_
- [Label](#label)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing label in Harbor with the same name. |  | Optional: \{\} <br /> |
| `description` _string_ | Description is an optional description. |  | Optional: \{\} <br /> |
| `color` _string_ | Color is the label color, e.g. #3366ff. |  | Optional: \{\} <br /> |
| `scope` _string_ | Scope is the label scope. Valid values are g (global) and p (project). |  | Enum: [g p] <br />Optional: \{\} <br /> |
| `projectRef` _[ProjectReference](#projectreference)_ | ProjectRef references a Project CR for project-scoped labels. |  | Optional: \{\} <br /> |


#### Member



Member is the Schema for the members API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `Member` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[MemberSpec](#memberspec)_ |  |  |  |


#### MemberGroup



MemberGroup defines a group-based member.



_Appears in:_
- [MemberSpec](#memberspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `groupRef` _[UserGroupReference](#usergroupreference)_ | GroupRef references the UserGroup to grant membership to. |  |  |


#### MemberSpec



MemberSpec defines the desired state of Member.



_Appears in:_
- [Member](#member)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing project membership in Harbor for the same identity. |  | Optional: \{\} <br /> |
| `projectRef` _[ProjectReference](#projectreference)_ | ProjectRef references the project where the member should be added. |  |  |
| `role` _string_ | Role is the human‑readable name of the role.<br />Allowed values: "admin", "maintainer", "developer", "guest" |  | Enum: [admin maintainer developer guest] <br />Required: \{\} <br /> |
| `memberUser` _[MemberUser](#memberuser)_ | MemberUser defines the member if it is a user. |  | Optional: \{\} <br /> |
| `memberGroup` _[MemberGroup](#membergroup)_ | MemberGroup defines the member if it is a group. |  | Optional: \{\} <br /> |


#### MemberUser



MemberUser defines a user-based member.



_Appears in:_
- [MemberSpec](#memberspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `userRef` _[UserReference](#userreference)_ | UserRef references the User to grant membership to. |  |  |


#### Project



Project is the Schema for the projects API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `Project` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ProjectSpec](#projectspec)_ |  |  |  |


#### ProjectMetadata



ProjectMetadata defines additional metadata for the project.



_Appears in:_
- [ProjectSpec](#projectspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `public` _string_ |  |  |  |
| `enable_content_trust` _string_ |  |  |  |
| `enable_content_trust_cosign` _string_ |  |  |  |
| `prevent_vul` _string_ |  |  |  |
| `severity` _string_ |  |  |  |
| `auto_scan` _string_ |  |  |  |
| `auto_sbom_generation` _string_ |  |  |  |
| `reuse_sys_cve_allowlist` _string_ |  |  |  |
| `retention_id` _string_ |  |  |  |
| `proxy_speed_kb` _string_ |  |  |  |


#### ProjectReference



ProjectReference identifies a Project custom resource.



_Appears in:_
- [ImmutableTagRuleSpec](#immutabletagrulespec)
- [LabelSpec](#labelspec)
- [MemberSpec](#memberspec)
- [QuotaSpec](#quotaspec)
- [RetentionPolicySpec](#retentionpolicyspec)
- [WebhookPolicySpec](#webhookpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Project resource. |  | MinLength: 1 <br /> |
| `namespace` _string_ | Namespace of the Project resource. Defaults to the referencing resource namespace. |  | Optional: \{\} <br /> |


#### ProjectSpec



ProjectSpec defines the desired state of Project.



_Appears in:_
- [Project](#project)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing project in Harbor with the same name. |  | Optional: \{\} <br /> |
| `public` _boolean_ | Public indicates whether the project is public. |  |  |
| `owner` _string_ | Owner is an optional field for the project owner. |  | Optional: \{\} <br /> |
| `metadata` _[ProjectMetadata](#projectmetadata)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `cve_allowlist` _[CVEAllowlist](#cveallowlist)_ | CVEAllowlist holds the configuration for the CVE allowlist. |  | Optional: \{\} <br /> |
| `storage_limit` _integer_ | StorageLimit is the storage limit for the project. |  | Optional: \{\} <br /> |
| `registryRef` _[RegistryReference](#registryreference)_ | RegistryRef references the Registry to use for proxy cache projects. |  | Optional: \{\} <br /> |


#### PurgeAuditParameters



PurgeAuditParameters defines parameters for purge audit schedules.



_Appears in:_
- [PurgeAuditScheduleSpec](#purgeauditschedulespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `auditRetentionHour` _integer_ | AuditRetentionHour is the retention period in hours. |  | Optional: \{\} <br /> |
| `includeEventTypes` _string_ | IncludeEventTypes is a comma-separated list of event types to include. |  | Optional: \{\} <br /> |
| `dryRun` _boolean_ | DryRun indicates whether to run in dry-run mode. |  | Optional: \{\} <br /> |


#### PurgeAuditSchedule



PurgeAuditSchedule is the Schema for the purgeauditschedules API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `PurgeAuditSchedule` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PurgeAuditScheduleSpec](#purgeauditschedulespec)_ |  |  |  |


#### PurgeAuditScheduleSpec



PurgeAuditScheduleSpec defines the desired schedule for audit purge.



_Appears in:_
- [PurgeAuditSchedule](#purgeauditschedule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `schedule` _[ScheduleSpec](#schedulespec)_ | Schedule defines when purge runs. |  |  |
| `parameters` _[PurgeAuditParameters](#purgeauditparameters)_ | Parameters define purge settings. |  | Optional: \{\} <br /> |


#### Quota



Quota is the Schema for the quotas API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `Quota` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[QuotaSpec](#quotaspec)_ |  |  |  |


#### QuotaSpec



QuotaSpec defines the desired state of Quota.



_Appears in:_
- [Quota](#quota)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `projectRef` _[ProjectReference](#projectreference)_ | ProjectRef references a Project CR to derive the Harbor project ID. |  | Optional: \{\} <br /> |
| `hard` _object (keys:string, values:integer)_ | Hard defines the quota hard limits (resource name -> limit). |  | Optional: \{\} <br /> |


#### Registry



Registry is the Schema for the registries API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `Registry` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RegistrySpec](#registryspec)_ |  |  |  |


#### RegistryCredentialSpec



RegistryCredentialSpec defines registry authentication details.



_Appears in:_
- [RegistrySpec](#registryspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type of the credential, e.g. "basic" or "oauth". |  | Enum: [basic oauth] <br /> |
| `accessKeySecretRef` _[SecretReference](#secretreference)_ | AccessKeySecretRef references the secret key holding the access key (username). |  |  |
| `accessSecretSecretRef` _[SecretReference](#secretreference)_ | AccessSecretSecretRef references the secret key holding the access secret (password/token). |  |  |


#### RegistryReference



RegistryReference identifies a Registry custom resource.



_Appears in:_
- [ProjectSpec](#projectspec)
- [ReplicationPolicySpec](#replicationpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Registry resource. |  | MinLength: 1 <br /> |
| `namespace` _string_ | Namespace of the Registry resource. Defaults to the referencing resource namespace. |  | Optional: \{\} <br /> |


#### RegistrySpec



RegistrySpec defines the desired state of Registry.



_Appears in:_
- [Registry](#registry)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing registry in Harbor with the same name. |  | Optional: \{\} <br /> |
| `type` _string_ | Type of the registry, e.g., "github-ghcr". |  | Enum: [github-ghcr ali-acr aws-ecr azure-acr docker-hub docker-registry google-gcr harbor huawei-SWR jfrog-artifactory tencent-tcr volcengine-cr] <br /> |
| `description` _string_ | Description is an optional description. |  | Optional: \{\} <br /> |
| `url` _string_ | URL is the registry URL. |  | Format: url <br /> |
| `credential` _[RegistryCredentialSpec](#registrycredentialspec)_ | Credential holds authentication details for the registry. |  | Optional: \{\} <br /> |
| `caCertificate` _string_ | CACertificate is the PEM-encoded CA certificate for this registry endpoint. |  | Optional: \{\} <br /> |
| `caCertificateRef` _[SecretReference](#secretreference)_ | CACertificateRef references a secret value holding the PEM-encoded CA certificate.<br />If set, it overrides CACertificate. |  | Optional: \{\} <br /> |
| `insecure` _boolean_ | Insecure indicates if remote certificates should be verified. |  |  |


#### ReplicationFilterSpec



ReplicationFilterSpec defines a replication filter.



_Appears in:_
- [ReplicationPolicySpec](#replicationpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type defines the filter type. |  | Optional: \{\} <br /> |
| `value` _[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#json-v1-apiextensions-k8s-io)_ | Value defines the filter value. |  | Optional: \{\} <br /> |
| `decoration` _string_ | Decoration defines how to interpret the filter. |  | Optional: \{\} <br /> |


#### ReplicationPolicy



ReplicationPolicy is the Schema for the replicationpolicies API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `ReplicationPolicy` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ReplicationPolicySpec](#replicationpolicyspec)_ |  |  |  |


#### ReplicationPolicySpec



ReplicationPolicySpec defines the desired state of ReplicationPolicy.



_Appears in:_
- [ReplicationPolicy](#replicationpolicy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing replication policy in Harbor with the same name. |  | Optional: \{\} <br /> |
| `description` _string_ | Description is an optional policy description. |  | Optional: \{\} <br /> |
| `sourceRegistryRef` _[RegistryReference](#registryreference)_ | SourceRegistryRef references a Registry CR to use as the source. |  | Optional: \{\} <br /> |
| `destinationRegistryRef` _[RegistryReference](#registryreference)_ | DestinationRegistryRef references a Registry CR to use as the destination. |  | Optional: \{\} <br /> |
| `destNamespace` _string_ | DestNamespace is the destination namespace. |  | Optional: \{\} <br /> |
| `destNamespaceReplaceCount` _integer_ | DestNamespaceReplaceCount controls namespace replacement behavior. |  | Optional: \{\} <br /> |
| `trigger` _[ReplicationTriggerSpec](#replicationtriggerspec)_ | Trigger defines when the replication policy runs. |  | Optional: \{\} <br /> |
| `filters` _[ReplicationFilterSpec](#replicationfilterspec) array_ | Filters defines the replication filters. |  | Optional: \{\} <br /> |
| `replicateDeletion` _boolean_ | ReplicateDeletion indicates whether delete operations are replicated. |  | Optional: \{\} <br /> |
| `override` _boolean_ | Override indicates whether to overwrite destination resources. |  | Optional: \{\} <br /> |
| `enabled` _boolean_ | Enabled indicates whether the policy is enabled. |  | Optional: \{\} <br /> |
| `speed` _integer_ | Speed is the speed limit for each task. |  | Optional: \{\} <br /> |
| `copyByChunk` _boolean_ | CopyByChunk indicates whether to enable copy by chunk. |  | Optional: \{\} <br /> |
| `singleActiveReplication` _boolean_ | SingleActiveReplication avoids overlapping executions. |  | Optional: \{\} <br /> |


#### ReplicationTriggerSettings



ReplicationTriggerSettings defines settings for a replication trigger.



_Appears in:_
- [ReplicationTriggerSpec](#replicationtriggerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `cron` _string_ | Cron is the cron expression for scheduled triggers. |  | Optional: \{\} <br /> |


#### ReplicationTriggerSpec



ReplicationTriggerSpec defines when the replication policy runs.



_Appears in:_
- [ReplicationPolicySpec](#replicationpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type defines the trigger type (manual, event_based, scheduled). |  | Enum: [manual event_based scheduled] <br />Optional: \{\} <br /> |
| `settings` _[ReplicationTriggerSettings](#replicationtriggersettings)_ | Settings holds trigger settings. |  | Optional: \{\} <br /> |


#### RetentionPolicy



RetentionPolicy is the Schema for the retentionpolicies API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `RetentionPolicy` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RetentionPolicySpec](#retentionpolicyspec)_ |  |  |  |


#### RetentionPolicySpec



RetentionPolicySpec defines the desired state of a retention policy.



_Appears in:_
- [RetentionPolicy](#retentionpolicy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `projectRef` _[ProjectReference](#projectreference)_ | ProjectRef references a Project CR to derive the Harbor project ID.<br />When set, scope.ref is resolved from the Project status and scope.level is forced to "project". |  | Optional: \{\} <br /> |
| `algorithm` _string_ | Algorithm defines the retention algorithm, e.g. "or". |  | Optional: \{\} <br /> |
| `rules` _[RetentionRule](#retentionrule) array_ | Rules defines the retention rules. |  | MinItems: 1 <br /> |
| `trigger` _[RetentionTrigger](#retentiontrigger)_ | Trigger defines when the retention policy runs. |  | Optional: \{\} <br /> |
| `scope` _[RetentionScope](#retentionscope)_ | Scope defines the policy scope. |  | Optional: \{\} <br /> |


#### RetentionRule



RetentionRule defines a retention rule.



_Appears in:_
- [RetentionPolicySpec](#retentionpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `disabled` _boolean_ | Disabled indicates whether the rule is disabled. |  | Optional: \{\} <br /> |
| `action` _string_ | Action defines the rule action, e.g. "delete". |  | Optional: \{\} <br /> |
| `template` _string_ | Template defines the rule template. |  | Optional: \{\} <br /> |
| `params` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#json-v1-apiextensions-k8s-io))_ | Params holds template parameters. |  | Optional: \{\} <br /> |
| `tagSelectors` _[RetentionSelector](#retentionselector) array_ | TagSelectors define the tag selectors. |  | Optional: \{\} <br /> |
| `scopeSelectors` _object (keys:string, values:[RetentionSelector](#retentionselector))_ | ScopeSelectors define the scope selectors. |  | Optional: \{\} <br /> |


#### RetentionScope



RetentionScope defines policy scope.



_Appears in:_
- [RetentionPolicySpec](#retentionpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `level` _string_ | Level defines scope level, e.g. "project". |  | Optional: \{\} <br /> |
| `ref` _integer_ | Ref is the scope reference. |  | Optional: \{\} <br /> |


#### RetentionSelector



RetentionSelector defines a selector.



_Appears in:_
- [RetentionRule](#retentionrule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind defines selector kind. |  | Optional: \{\} <br /> |
| `decoration` _string_ | Decoration defines selector decoration. |  | Optional: \{\} <br /> |
| `pattern` _string_ | Pattern defines selector pattern. |  | Optional: \{\} <br /> |
| `extras` _string_ | Extras defines extra selector details. |  | Optional: \{\} <br /> |


#### RetentionTrigger



RetentionTrigger defines when a policy runs.



_Appears in:_
- [RetentionPolicySpec](#retentionpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind defines trigger kind. |  | Optional: \{\} <br /> |
| `settings` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#json-v1-apiextensions-k8s-io))_ | Settings holds trigger settings. |  | Optional: \{\} <br /> |
| `references` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#json-v1-apiextensions-k8s-io))_ | References holds trigger references. |  | Optional: \{\} <br /> |


#### Robot



Robot is the Schema for the robots API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `Robot` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RobotSpec](#robotspec)_ |  |  |  |


#### RobotAccess



RobotAccess defines a single access rule for a robot account.



_Appears in:_
- [RobotPermission](#robotpermission)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resource` _[RobotResource](#robotresource)_ | Resource defines the resource to grant access to. |  | Enum: [* configuration label log ldap-user member metadata quota repository tag-retention immutable-tag robot notification-policy scan sbom scanner artifact tag accessory artifact-addition artifact-label preheat-policy preheat-instance audit-log catalog project user user-group registry replication distribution garbage-collection replication-adapter replication-policy scan-all system-volumes purge-audit export-cve jobservice-monitor security-hub] <br /> |
| `action` _[RobotAction](#robotaction)_ | Action defines the action to permit. |  | Enum: [* pull push create read update delete list operate scanner-pull stop] <br /> |
| `effect` _string_ | Effect defines the effect of the access rule, typically "allow". |  | Optional: \{\} <br /> |


#### RobotAction

_Underlying type:_ _string_

RobotAction is the action of a robot permission access rule.



_Appears in:_
- [RobotAccess](#robotaccess)

| Field | Description |
| --- | --- |
| `*` |  |
| `pull` |  |
| `push` |  |
| `create` |  |
| `read` |  |
| `update` |  |
| `delete` |  |
| `list` |  |
| `operate` |  |
| `scanner-pull` |  |
| `stop` |  |


#### RobotPermission



RobotPermission defines a permission block for a robot account.



_Appears in:_
- [RobotSpec](#robotspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind defines the permission scope, such as "project" or "system". |  | MinLength: 1 <br /> |
| `namespace` _string_ | Namespace is the Harbor project name for project-scoped permissions. |  | Optional: \{\} <br /> |
| `access` _[RobotAccess](#robotaccess) array_ | Access lists the access rules for this permission. |  | MinItems: 1 <br /> |


#### RobotResource

_Underlying type:_ _string_

RobotResource is the resource of a robot permission access rule.



_Appears in:_
- [RobotAccess](#robotaccess)

| Field | Description |
| --- | --- |
| `*` |  |
| `configuration` |  |
| `label` |  |
| `log` |  |
| `ldap-user` |  |
| `member` |  |
| `metadata` |  |
| `quota` |  |
| `repository` |  |
| `tag-retention` |  |
| `immutable-tag` |  |
| `robot` |  |
| `notification-policy` |  |
| `scan` |  |
| `sbom` |  |
| `scanner` |  |
| `artifact` |  |
| `tag` |  |
| `accessory` |  |
| `artifact-addition` |  |
| `artifact-label` |  |
| `preheat-policy` |  |
| `preheat-instance` |  |
| `audit-log` |  |
| `catalog` |  |
| `project` |  |
| `user` |  |
| `user-group` |  |
| `registry` |  |
| `replication` |  |
| `distribution` |  |
| `garbage-collection` |  |
| `replication-adapter` |  |
| `replication-policy` |  |
| `scan-all` |  |
| `system-volumes` |  |
| `purge-audit` |  |
| `export-cve` |  |
| `jobservice-monitor` |  |
| `security-hub` |  |


#### RobotSpec



RobotSpec defines the desired state of Robot.



_Appears in:_
- [Robot](#robot)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing robot in Harbor with the same name. |  | Optional: \{\} <br /> |
| `description` _string_ | Description of the robot account. |  | Optional: \{\} <br /> |
| `level` _string_ | Level is the scope of the robot account.<br />Allowed values: "system", "project". |  | Enum: [system project] <br /> |
| `permissions` _[RobotPermission](#robotpermission) array_ | Permissions define the access granted to the robot account. |  | MinItems: 1 <br /> |
| `disable` _boolean_ | Disable indicates whether the robot account is disabled. |  | Optional: \{\} <br /> |
| `duration` _integer_ | Duration is the token duration in days. Use -1 for never expires.<br />If omitted, it defaults to -1. | -1 |  |
| `secretRef` _[SecretReference](#secretreference)_ | SecretRef references the operator-managed secret key holding the robot secret.<br />The operator writes the generated robot secret to this location and expects<br />the Secret to either not exist yet or already be managed by this Robot.<br />If omitted, the operator will create a Secret named "<metadata.name>-secret"<br />in the same namespace with key "secret". |  | Optional: \{\} <br /> |


#### ScanAllSchedule



ScanAllSchedule is the Schema for the scanallschedules API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `ScanAllSchedule` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ScanAllScheduleSpec](#scanallschedulespec)_ |  |  |  |


#### ScanAllScheduleSpec



ScanAllScheduleSpec defines the desired schedule for scan all.



_Appears in:_
- [ScanAllSchedule](#scanallschedule)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `schedule` _[ScheduleSpec](#schedulespec)_ | Schedule defines when scan all runs. |  |  |
| `parameters` _object (keys:string, values:[JSON](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#json-v1-apiextensions-k8s-io))_ | Parameters define scan all settings passed to Harbor. |  | Optional: \{\} <br /> |


#### ScannerRegistration



ScannerRegistration is the Schema for the scannerregistrations API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `ScannerRegistration` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ScannerRegistrationSpec](#scannerregistrationspec)_ |  |  |  |


#### ScannerRegistrationSpec



ScannerRegistrationSpec defines the desired state of ScannerRegistration.



_Appears in:_
- [ScannerRegistration](#scannerregistration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing scanner registration in Harbor with the same name. |  | Optional: \{\} <br /> |
| `description` _string_ | Description is an optional description. |  | Optional: \{\} <br /> |
| `url` _string_ | URL is the scanner adapter base URL. |  | Format: uri <br /> |
| `auth` _string_ | Auth defines the authentication approach (e.g. Basic, Bearer, X-ScannerAdapter-API-Key). |  | Optional: \{\} <br /> |
| `accessCredential` _string_ | AccessCredential is the credential value sent in the auth header. |  | Optional: \{\} <br /> |
| `accessCredentialSecretRef` _[SecretReference](#secretreference)_ | AccessCredentialSecretRef references a secret value holding the credential. |  | Optional: \{\} <br /> |
| `skipCertVerify` _boolean_ | SkipCertVerify indicates whether to skip certificate verification. |  | Optional: \{\} <br /> |
| `useInternalAddr` _boolean_ | UseInternalAddr indicates whether the scanner uses Harbor's internal address. |  | Optional: \{\} <br /> |
| `disabled` _boolean_ | Disabled indicates whether the registration is disabled. |  | Optional: \{\} <br /> |
| `default` _boolean_ | Default indicates whether this scanner should be set as system default. |  | Optional: \{\} <br /> |


#### ScheduleSpec



ScheduleSpec defines the schedule configuration.



_Appears in:_
- [GCScheduleSpec](#gcschedulespec)
- [PurgeAuditScheduleSpec](#purgeauditschedulespec)
- [ScanAllScheduleSpec](#scanallschedulespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type defines the schedule type.<br />Valid values: Hourly, Daily, Weekly, Custom, Manual, None, Schedule. |  | Enum: [Hourly Daily Weekly Custom Manual None Schedule] <br /> |
| `cron` _string_ | Cron is the cron expression when Type is not Manual or None. |  | Optional: \{\} <br /> |


#### SecretReference



SecretReference is similar to a corev1.SecretKeySelector but allows
cross-namespace references when enabled in the operator RBAC.



_Appears in:_
- [ConfigurationSpec](#configurationspec)
- [Credentials](#credentials)
- [HarborConnectionSpec](#harborconnectionspec)
- [RegistryCredentialSpec](#registrycredentialspec)
- [RegistrySpec](#registryspec)
- [RobotSpec](#robotspec)
- [ScannerRegistrationSpec](#scannerregistrationspec)
- [WebhookTargetSpec](#webhooktargetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Secret. |  | MinLength: 1 <br /> |
| `key` _string_ | Key inside the Secret data. When omitted, the controller using this<br />reference will apply a sensible default. |  | Optional: \{\} <br /> |
| `namespace` _string_ | Namespace of the Secret. Omit to use the HarborConnection namespace. |  | Optional: \{\} <br /> |


#### User



User is the Schema for the users API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `User` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UserSpec](#userspec)_ |  |  |  |


#### UserGroup



UserGroup is the Schema for the usergroups API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `UserGroup` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UserGroupSpec](#usergroupspec)_ |  |  |  |


#### UserGroupReference



UserGroupReference identifies a UserGroup custom resource.



_Appears in:_
- [MemberGroup](#membergroup)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the UserGroup resource. |  | MinLength: 1 <br /> |
| `namespace` _string_ | Namespace of the UserGroup resource. Defaults to the referencing resource namespace. |  | Optional: \{\} <br /> |


#### UserGroupSpec



UserGroupSpec defines the desired state of UserGroup.



_Appears in:_
- [UserGroup](#usergroup)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing user group in Harbor with the same name. |  | Optional: \{\} <br /> |
| `groupType` _integer_ | GroupType is the group type (1=LDAP, 2=HTTP, 3=OIDC). |  | Enum: [1 2 3] <br /> |
| `ldapGroupDN` _string_ | LDAPGroupDN is the DN of the LDAP group when GroupType is LDAP. |  | Optional: \{\} <br /> |


#### UserReference



UserReference identifies a User custom resource.



_Appears in:_
- [MemberUser](#memberuser)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the User resource. |  | MinLength: 1 <br /> |
| `namespace` _string_ | Namespace of the User resource. Defaults to the referencing resource namespace. |  | Optional: \{\} <br /> |


#### UserSpec



UserSpec defines the desired state of User.



_Appears in:_
- [User](#user)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing user in Harbor with the same username. |  | Optional: \{\} <br /> |
| `email` _string_ | Email address of the user. |  | Format: email <br /> |
| `realname` _string_ | Realname is an optional full name. |  | Optional: \{\} <br /> |
| `comment` _string_ | Comment is an optional comment for the user. |  | Optional: \{\} <br /> |
| `passwordSecretRef` _[SecretKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#secretkeyselector-v1-core)_ | PasswordSecretRef references a secret key that contains the password for the user. |  |  |


#### WebhookPolicy



WebhookPolicy is the Schema for the webhookpolicies API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `harbor.harbor-operator.io/v1alpha1` | | |
| `kind` _string_ | `WebhookPolicy` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[WebhookPolicySpec](#webhookpolicyspec)_ |  |  |  |


#### WebhookPolicySpec



WebhookPolicySpec defines the desired state of WebhookPolicy.



_Appears in:_
- [WebhookPolicy](#webhookpolicy)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `harborConnectionRef` _[HarborConnectionReference](#harborconnectionreference)_ | HarborConnectionRef references the Harbor connection object to use.<br />When the operator is started with --harbor-connection, this field may be omitted. |  | Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls what happens when the Kubernetes object is deleted.<br />Delete removes the corresponding Harbor resource before removing the finalizer.<br />Orphan skips Harbor-side deletion and removes the finalizer so the<br />Kubernetes object can be deleted while leaving the Harbor resource in place. | Delete | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.35/#duration-v1-meta)_ | DriftDetectionInterval is the interval at which the operator will check<br />for drift. A value of 0 (or omitted) disables periodic drift detection. |  | Optional: \{\} <br /> |
| `reconcileNonce` _string_ | ReconcileNonce forces an immediate reconcile when updated. |  | Optional: \{\} <br /> |
| `allowTakeover` _boolean_ | AllowTakeover indicates whether the operator is allowed to adopt an<br />existing webhook policy in Harbor with the same name. |  | Optional: \{\} <br /> |
| `projectRef` _[ProjectReference](#projectreference)_ | ProjectRef references a Project CR to derive the Harbor project ID. |  | Optional: \{\} <br /> |
| `description` _string_ | Description is an optional policy description. |  | Optional: \{\} <br /> |
| `enabled` _boolean_ | Enabled indicates whether the policy is enabled. | true | Optional: \{\} <br /> |
| `eventTypes` _string array_ | EventTypes lists the webhook event types. |  | MinItems: 1 <br /> |
| `targets` _[WebhookTargetSpec](#webhooktargetspec) array_ | Targets lists the webhook targets. |  | MinItems: 1 <br /> |


#### WebhookTargetSpec



WebhookTargetSpec defines a single webhook target.



_Appears in:_
- [WebhookPolicySpec](#webhookpolicyspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type defines the webhook notify type. |  | Optional: \{\} <br /> |
| `address` _string_ | Address is the webhook target address. |  | Optional: \{\} <br /> |
| `authHeader` _string_ | AuthHeader is the auth header to send to the webhook target. |  | Optional: \{\} <br /> |
| `authHeaderSecretRef` _[SecretReference](#secretreference)_ | AuthHeaderSecretRef references a secret value holding the auth header. |  | Optional: \{\} <br /> |
| `payloadFormat` _string_ | PayloadFormat is the payload format (e.g. CloudEvents). |  | Optional: \{\} <br /> |
| `skipCertVerify` _boolean_ | SkipCertVerify indicates whether to skip TLS certificate verification. |  | Optional: \{\} <br /> |


