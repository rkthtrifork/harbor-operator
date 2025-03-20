package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RegistrySpec defines the desired state of Registry.
type RegistrySpec struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef string `json:"harborConnectionRef"`

	// Type of the registry, e.g., "github-ghcr"
	// +kubebuilder:validation:Enum=github-ghcr;other-types-if-needed
	Type string `json:"type"`

	// Name is the registry name.
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
}

// RegistryStatus defines the observed state of Registry.
type RegistryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
