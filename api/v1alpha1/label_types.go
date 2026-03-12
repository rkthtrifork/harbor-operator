package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// LabelSpec defines the desired state of Label.
// +kubebuilder:validation:XValidation:rule="!has(self.projectRef) || self.scope == 'p'",message="scope must be 'p' when projectRef is set"
// +kubebuilder:validation:XValidation:rule="self.scope != 'p' || has(self.projectRef)",message="projectRef is required when scope is 'p'"
type LabelSpec struct {
	HarborSpecBase `json:",inline"`

	// AllowTakeover indicates whether the operator is allowed to adopt an
	// existing label in Harbor with the same name.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// Name is the label name.
	// Defaults to metadata.name when omitted.
	// +optional
	Name string `json:"name,omitempty"`

	// Description is an optional description.
	// +optional
	Description string `json:"description,omitempty"`

	// Color is the label color, e.g. #3366ff.
	// +optional
	Color string `json:"color,omitempty"`

	// Scope is the label scope. Valid values are g (global) and p (project).
	// +kubebuilder:validation:Enum=g;p
	// +optional
	Scope string `json:"scope,omitempty"`

	// ProjectRef references a Project CR for project-scoped labels.
	// +optional
	ProjectRef *ProjectReference `json:"projectRef,omitempty"`
}

// LabelStatus defines the observed state of Label.
type LabelStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborLabelID is the ID of the label in Harbor.
	HarborLabelID int `json:"harborLabelID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=harbor
// +kubebuilder:printcolumn:name="Scope",type=string,JSONPath=`.spec.scope`
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Label is the Schema for the labels API.
type Label struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LabelSpec   `json:"spec,omitempty"`
	Status LabelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LabelList contains a list of Label.
type LabelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Label `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Label{}, &LabelList{})
}
