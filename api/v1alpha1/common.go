package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type HarborConnectionReferenceKind string

const (
	HarborConnectionReferenceKindNamespaced HarborConnectionReferenceKind = "HarborConnection"
	HarborConnectionReferenceKindCluster    HarborConnectionReferenceKind = "ClusterHarborConnection"
)

type DeletionPolicy string

const (
	DeletionPolicyDelete DeletionPolicy = "Delete"
	DeletionPolicyOrphan DeletionPolicy = "Orphan"
)

// HarborConnectionReference identifies either a namespaced HarborConnection or a
// cluster-scoped ClusterHarborConnection.
type HarborConnectionReference struct {
	// Name of the referenced Harbor connection object.
	Name string `json:"name"`

	// Kind selects the Harbor connection object kind.
	// When omitted, controllers treat it as HarborConnection.
	// +kubebuilder:validation:Enum=HarborConnection;ClusterHarborConnection
	// +optional
	Kind HarborConnectionReferenceKind `json:"kind,omitempty"`
}

func (r HarborConnectionReference) IsZero() bool {
	return r.Name == "" && r.Kind == ""
}

// HarborSpecBase holds the fields that appear in every Harbor CR.
type HarborSpecBase struct {
	// HarborConnectionRef references the Harbor connection object to use.
	// When the operator is started with --harbor-connection, this field may be omitted.
	// +optional
	HarborConnectionRef *HarborConnectionReference `json:"harborConnectionRef,omitempty"`

	// DeletionPolicy controls what happens when the Kubernetes object is deleted.
	// Delete removes the corresponding Harbor resource before removing the finalizer.
	// Orphan skips Harbor-side deletion and removes the finalizer so the
	// Kubernetes object can be deleted while leaving the Harbor resource in place.
	// +kubebuilder:default=Delete
	// +kubebuilder:validation:Enum=Delete;Orphan
	// +optional
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`

	// DriftDetectionInterval is the interval at which the operator will check
	// for drift. A value of 0 (or omitted) disables periodic drift detection.
	// +optional
	DriftDetectionInterval *metav1.Duration `json:"driftDetectionInterval,omitempty"`

	// ReconcileNonce forces an immediate reconcile when updated.
	// +optional
	ReconcileNonce string `json:"reconcileNonce,omitempty"`
}

// SecretReference is similar to a corev1.SecretKeySelector but allows
// cross-namespace references when enabled in the operator RBAC.
type SecretReference struct {
	// Name of the Secret.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Key inside the Secret data. When omitted, the controller using this
	// reference will apply a sensible default.
	// +optional
	Key string `json:"key,omitempty"`
	// Namespace of the Secret. Omit to use the HarborConnection namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// ProjectReference identifies a Project custom resource.
type ProjectReference struct {
	// Name of the Project resource.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Namespace of the Project resource. Defaults to the referencing resource namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// RegistryReference identifies a Registry custom resource.
type RegistryReference struct {
	// Name of the Registry resource.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Namespace of the Registry resource. Defaults to the referencing resource namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// UserReference identifies a User custom resource.
type UserReference struct {
	// Name of the User resource.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Namespace of the User resource. Defaults to the referencing resource namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// UserGroupReference identifies a UserGroup custom resource.
type UserGroupReference struct {
	// Name of the UserGroup resource.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Namespace of the UserGroup resource. Defaults to the referencing resource namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// HarborStatusBase holds common status fields for Harbor resources.
type HarborStatusBase struct {
	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the resource's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// GetDriftDetectionInterval returns the drift detection interval.
func (base *HarborSpecBase) GetDriftDetectionInterval() *metav1.Duration {
	return base.DriftDetectionInterval
}

func (base *HarborSpecBase) GetDeletionPolicy() DeletionPolicy {
	if base.DeletionPolicy == "" {
		return DeletionPolicyDelete
	}
	return base.DeletionPolicy
}
