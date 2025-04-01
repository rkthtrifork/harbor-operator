package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RegistrySpec defines the desired state of Registry.
type RegistrySpec struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef string `json:"harborConnectionRef"`

	// Type of the registry, e.g., "github-ghcr".
	// +kubebuilder:validation:Enum=github-ghcr;other-types-if-needed
	Type string `json:"type"`

	// Name is the registry name.
	// It is recommended to leave this field empty so that the operator defaults it
	// to the custom resource's metadata name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Description is an optional description.
	// +optional
	Description string `json:"description,omitempty"`

	// URL is the registry URL.
	// +kubebuilder:validation:Format=url
	URL string `json:"url"`

	// Insecure indicates if remote certificates should be verified.
	Insecure bool `json:"insecure"`

	// AllowTakeover indicates whether the operator is allowed to adopt an existing registry
	// in Harbor if one with the same name already exists.
	// If true, the operator will search for an existing registry with the same name and, if found,
	// update its configuration to match the custom resource.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// DriftDetectionInterval is the interval at which the operator will check for drift.
	// A value of 0 (or omitted) disables periodic drift detection.
	// +optional
	DriftDetectionInterval metav1.Duration `json:"driftDetectionInterval,omitempty"`

	// ReconcileNonce is an optional field that, when updated, forces an immediate reconcile.
	// +optional
	ReconcileNonce string `json:"reconcileNonce,omitempty"`
}

// RegistryStatus defines the observed state of Registry.
type RegistryStatus struct {
	// HarborRegistryID is the ID of the registry in Harbor.
	HarborRegistryID int `json:"harborRegistryID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Registry is the Schema for the registries API.
type Registry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistrySpec   `json:"spec,omitempty"`
	Status RegistryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RegistryList contains a list of Registry.
type RegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Registry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Registry{}, &RegistryList{})
}
