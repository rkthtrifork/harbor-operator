package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// HarborSpecBase holds the fields that appear in every Harbor CR.
type HarborSpecBase struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef string `json:"harborConnectionRef"`

	// AllowTakeover indicates whether the operator is allowed to adopt an
	// existing object in Harbor with the same name/ID.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

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
	// Key inside the Secret data. Defaults to "access_secret".
	// +optional
	Key string `json:"key,omitempty"`
	// Namespace of the Secret. Omit to use the HarborConnection namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// GetDriftDetectionInterval returns the drift detection interval.
func (base *HarborSpecBase) GetDriftDetectionInterval() *metav1.Duration {
	return base.DriftDetectionInterval
}
