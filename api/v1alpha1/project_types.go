package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of Project.
type ProjectSpec struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef string `json:"harborConnectionRef"`

	// Name is the name of the project.
	// It is recommended to leave this field empty so that the operator defaults it
	// to the custom resource’s metadata name.
	// +optional
	Name string `json:"projectName,omitempty"`

	// Public indicates whether the project is public.
	// +kubebuilder:default:=true
	Public bool `json:"public"`

	// Description is an optional description of the project.
	// +optional
	Description string `json:"description,omitempty"`

	// Owner is an optional field for the project owner.
	// +optional
	Owner string `json:"owner,omitempty"`

	// AllowTakeover indicates whether the operator is allowed to adopt an existing
	// project in Harbor with the same name.
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

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	// HarborProjectID is the ID of the project in Harbor.
	HarborProjectID int `json:"harborProjectID,omitempty"`
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

func (p *Project) GetDriftDetectionInterval() metav1.Duration {
	return p.Spec.DriftDetectionInterval
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
