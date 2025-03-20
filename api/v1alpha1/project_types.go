package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProjectSpec defines the desired state of Project.
type ProjectSpec struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef string `json:"harborConnectionRef"`

	// Name is the name of the project.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// RegistryName is the name of the registry that this project should use as a proxy cache.
	// It is recommended that this matches the metadata.name of the Registry custom resource.
	// +kubebuilder:validation:Required
	RegistryName string `json:"registryName"`

	// Public indicates if the project should be public.
	Public bool `json:"public"`

	// Metadata holds additional project settings.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`

	// CveAllowlist holds CVE allowlist settings.
	// +optional
	CveAllowlist *CveAllowlist `json:"cveAllowlist,omitempty"`

	// StorageLimit defines the storage limit in bytes.
	// +optional
	StorageLimit int64 `json:"storageLimit,omitempty"`
}

// CveAllowlist defines the CVE allowlist configuration.
type CveAllowlist struct {
	// ID of the CVE allowlist.
	ID int `json:"id,omitempty"`
	// ProjectID associated with the allowlist.
	ProjectID int `json:"project_id,omitempty"`
	// ExpiresAt is the expiration timestamp.
	ExpiresAt int64 `json:"expires_at,omitempty"`
	// Items is the list of allowed CVEs.
	Items []CveItem `json:"items,omitempty"`
}

// CveItem represents a single allowed CVE.
type CveItem struct {
	CveID string `json:"cve_id"`
}

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Project is the Schema for the projects API.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
