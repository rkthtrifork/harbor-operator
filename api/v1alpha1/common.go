package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// HarborSpecBase holds the fields that appear in every Harbor CR.
type HarborSpecBase struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef string `json:"harborConnectionRef"`

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
	Name string `json:"name"`
	// Namespace of the Project resource. Defaults to the referencing resource namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// RegistryReference identifies a Registry custom resource.
type RegistryReference struct {
	// Name of the Registry resource.
	Name string `json:"name"`
	// Namespace of the Registry resource. Defaults to the referencing resource namespace.
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
